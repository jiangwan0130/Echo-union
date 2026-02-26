# 数据库设计文档 (Database Design)
# 学生会值班管理系统

| 文档信息 | |
|----------|----------|
| 版本号 | v1.9 |
| 创建日期 | 2026-01-29 |
| 最后更新 | 2026-02-24 |
| 文档状态 | 终审定稿版 |
| 数据库 | PostgreSQL 15+ |

---

## 一、文档概述

### 1.1 目的

本文档定义学生会值班管理系统的数据库设计，包括实体关系图（ERD）、表结构定义、字段说明、索引设计及数据约束。

### 1.2 设计原则

- 使用 UUID 作为主键（便于分布式扩展）
- 所有业务表包含完整审计字段：`created_at`、`created_by`、`updated_at`、`updated_by`
- 软删除使用 `deleted_at`、`deleted_by` 字段（除纯日志表与依附主表生命周期的从表外）
- 软删除一致性约束（工程规范）：对所有包含软删除字段的表，要求 `deleted_at` 与 `deleted_by` **要么同时为空，要么同时非空**（避免出现“有删除时间但无法追溯删除人”或“标记删除人但未标记删除时间”的脏数据）
- 可能发生并发修改的表使用 `version` 字段实现乐观锁
- 外键使用数据库约束（FK），**默认 `ON UPDATE CASCADE ON DELETE RESTRICT`**，不使用 `ON DELETE CASCADE`（因全局采用软删除，物理级联永远不会触发，避免语义冲突）
- 枚举字段 **全部** 使用 CHECK 约束保证数据完整性
- 合理使用 partial index 优化软删除场景下的查询性能
- 需要 `btree_gist` 扩展（用于 EXCLUDE 约束防止日期/时间范围重叠）
- 需要自定义 `timerange` 类型：`CREATE TYPE timerange AS RANGE (subtype = time)`（用于 `time_slots` EXCLUDE 约束的时间段重叠检测）

**审计字段 FK 级联策略约定：**

所有审计字段（`created_by`、`updated_by`、`deleted_by`）的 FK 声明均使用 PostgreSQL 默认行为 `ON UPDATE NO ACTION ON DELETE NO ACTION`（语义等同于 RESTRICT 的延迟版本）。因审计字段仅记录操作人快照，不需要 CASCADE 传播更新。各表 FK 声明中省略级联子句即表示采用此默认行为。

**受控反范式化声明：**

以下字段为查询性能优化而保留的冗余派生字段，**不构成唯一事实源**，应用层需在源数据变更时同步更新：
- `duty_records.is_late`：可由 `sign_in_time` 与 `time_slots.start_time` 计算，保留以避免签到列表查询 JOIN
- `duty_records.member_id`：可由 `schedule_item_id` JOIN `schedule_items.member_id` 获得，保留作为值班时间点的成员快照（换班后 `schedule_items.member_id` 会更新，但已生成的 `duty_records` 应保留原始分配人）
- `course_schedules.week_type`：可由 `weeks` 数组内容推导（全奇数→odd、全偶数→even、混合/NULL→all），保留以避免查询时遍历数组。应用层写入时负责校验与 `weeks` 的一致性

**多态外键声明：**

`notifications` 表的 `related_type` + `related_id` 为多态关联模式，无法建立数据库级 FK 约束。**应用层负责引用完整性**，展示通知时需做 defensive query（若关联实体已删除则降级展示）。

**邀请码存储方案定位：**

`invite_codes` 表为邀请码的持久化存储方案。Redis 可作为可选的短期缓存加速层（校验高频读），但不替代数据库作为唯一事实源。

**循环外键初始化顺序：**

`departments.created_by → users.user_id` 与 `users.department_id → departments.department_id` 形成循环引用。初始化顺序：
1. 创建 `departments` 记录（`created_by = NULL`）
2. 创建初始 `users` 记录
3. 回填 `departments.created_by`

**时区与时间类型约定：**

- 数据库时区：`Asia/Shanghai`
- 所有时间戳字段（`*_at`、`sign_in_time`、`sign_out_time` 等）统一使用 **TIMESTAMPTZ**（带时区）
- 业务日期字段（如 `duty_date`）使用 **DATE**

**乐观锁约定：**

- 可能发生并发修改的表添加 `version INT` 字段
- 应用层更新时必须校验版本号：`WHERE id = ? AND version = ?`
- 更新成功后版本号自动递增：`SET version = version + 1`

### 1.3 命名规范

- 表名：小写下划线分隔，复数形式（如 `users`、`departments`）
- 字段名：小写下划线分隔（如 `created_at`、`department_id`）
- 索引名：`idx_表名_字段名`
- 唯一索引：`uk_表名_字段名`

---

## 二、实体关系图 (ERD)

### 2.1 核心实体关系

```
┌────────────────────────────────────────────────────────────────────────────────────┐
│                          Entity Relationship Diagram                               │
├────────────────────────────────────────────────────────────────────────────────────┤
│                                                                                    │
│  ┌────────────┐        ┌────────────┐         ┌─────────────────┐                 │
│  │ semesters  │<--1--  │ schedules  │  --N--> │ schedule_items  │                 │
│  └─────┬──────┘        └─────┬──────┘         └────────┬────────┘                 │
│        |1                    |1                        |1                          │
│        vN                    vN                        vN                          │
│  ┌────────────┐  ┌───────────────────┐  ┌──────────────┐                          │
│  │ time_slots │  │ schedule_member_ │  │ duty_records │                          │
│  └────────────┘  │ snapshots        │  └──────────────┘                          │
│                  └───────────────────┘                                             │
│                                                                                    │
│  ┌─────────────┐   N   ┌──────────┐   1   ┌──────────────┐                       │
│  │ departments │<------│  users   │------>│ roles (enum) │                       │
│  └─────────────┘       └────┬─────┘       └──────────────┘                       │
│                             |1                                                    │
│     ┌─────────┬─────────────┼─────────────┬──────────┐                            │
│     vN        vN            vN            vN         vN                           │
│  ┌─────────┐ ┌──────────┐ ┌───────────┐ ┌─────────┐ ┌──────────────────┐         │
│  │user_sem-│ │ course_  │ │unavail-   │ │ swap_   │ │ notification_    │         │
│  │ester_as-│ │schedules │ │able_times │ │requests │ │ preferences      │         │
│  │signments│ │          │ │           │ │         │ │                  │         │
│  └─────────┘ └──────────┘ └───────────┘ └─────────┘ └──────────────────┘         │
│                                                                                    │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────────────┐               │
│  │notifications │     │ invite_codes │     │schedule_change_logs  │               │
│  └──────────────┘     └──────────────┘     └──────────────────────┘               │
│                                                                                    │
│  ┌──────────────┐     ┌──────────────┐     ┌────────────────┐                     │
│  │  locations   │     │system_config │     │ schedule_rules │                     │
│  └──────────────┘     └──────────────┘     └────────────────┘                     │
│                                                                                    │
└────────────────────────────────────────────────────────────────────────────────────┘
```

> 说明：`roles (enum)` 在图中仅表示 **用户角色枚举域**（`users.role`），V1 不单独建立 `roles` 表。

### 2.2 实体关系说明

| 关系 | 说明 |
|------|------|
| departments ↔ users | 一对多：一个部门有多个用户 |
| users ↔ user_semester_assignments | 一对多：一个用户在不同学期有不同的值班分配 |
| semesters ↔ user_semester_assignments | 一对多：一个学期下有多个用户分配 |
| semesters ↔ schedules | 一对多：一个学期可有多个排班表（实际1个） |
| semesters ↔ time_slots | 一对多：时间段可关联学期（NULL 表示全局默认） |
| schedules ↔ schedule_items | 一对多：一个排班表有多个排班项 |
| schedules ↔ schedule_member_snapshots | 一对多：一个排班表对应多条成员快照 |
| schedule_items ↔ duty_records | 一对多：一个排班项在不同日期产生多条值班记录 |
| users ↔ course_schedules | 一对多：一个用户有多条课表记录 |
| users ↔ unavailable_times | 一对多：一个用户有多条不可用时间 |
| users ↔ swap_requests | 一对多：一个用户可发起/接收多个换班申请 |
| users ↔ notifications | 一对多：一个用户有多条通知 |
| users ↔ notification_preferences | 一对一：一个用户有一条通知偏好配置 |
| schedule_items ↔ locations | 多对一：排班项关联值班地点（预留，可为 NULL） |

---

## 三、表结构设计

> 类型说明：下列表结构中若出现 `TIMESTAMP`，均按本项目约定视为 **TIMESTAMPTZ（带时区）**，数据库时区为 `Asia/Shanghai`。

### 3.1 用户与组织

#### 3.1.1 departments (部门表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| department_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| name | VARCHAR(50) | NOT NULL | | 部门名称 |
| description | TEXT | | | 部门描述 |
| is_active | BOOLEAN | NOT NULL | TRUE | 是否启用 |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**CHECK约束：**
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- `uk_departments_name` UNIQUE (name) WHERE deleted_at IS NULL
- （可选）`idx_departments_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

---

#### 3.1.2 users (用户表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| user_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| name | VARCHAR(100) | NOT NULL | | 姓名 |
| student_id | VARCHAR(20) | NOT NULL | | 学号 |
| email | VARCHAR(255) | NOT NULL | | 邮箱 |
| password_hash | VARCHAR(255) | NOT NULL | | 密码哈希 |
| role | VARCHAR(20) | NOT NULL | 'member' | 角色：admin/leader/member |
| department_id | UUID | NOT NULL | | 部门ID |
| must_change_password | BOOLEAN | NOT NULL | FALSE | 是否需要修改密码 |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**字段枚举值：**
- `role`: admin(排班管理员), leader(部门负责人), member(值班成员)

**CHECK约束：**
- `role IN ('admin', 'leader', 'member')`
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `department_id` REFERENCES departments(department_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- `uk_users_student_id` UNIQUE (student_id) WHERE deleted_at IS NULL
- `uk_users_email` UNIQUE (email) WHERE deleted_at IS NULL
- `idx_users_department_id` (department_id) WHERE deleted_at IS NULL
- （可选）`idx_users_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

---

#### 3.1.3 user_semester_assignments (用户-学期分配表)

> **设计说明**：将原 `users` 表中的 `duty_required`、`timetable_status`、`timetable_submitted_at` 抽取至此表，使其具备学期维度，支持多学期独立管理。

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| assignment_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| user_id | UUID | NOT NULL | | 用户ID |
| semester_id | UUID | NOT NULL | | 学期ID |
| duty_required | BOOLEAN | NOT NULL | FALSE | 是否需要值班 |
| timetable_status | VARCHAR(20) | NOT NULL | 'not_submitted' | 时间表状态 |
| timetable_submitted_at | TIMESTAMPTZ | | NULL | 时间表提交时间 |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**字段枚举值：**
- `timetable_status`: not_submitted(未提交), submitted(已提交)

**CHECK约束：**
- `timetable_status IN ('not_submitted', 'submitted')`
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `user_id` REFERENCES users(user_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `semester_id` REFERENCES semesters(semester_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- `uk_user_semester_assignments_user_semester` UNIQUE (user_id, semester_id) WHERE deleted_at IS NULL
- `idx_user_semester_assignments_semester_id` (semester_id) WHERE deleted_at IS NULL
- `idx_user_semester_assignments_duty_required` (semester_id, duty_required) WHERE deleted_at IS NULL
- （可选）`idx_user_semester_assignments_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

---

#### 3.1.4 notification_preferences (通知偏好表)

> **设计说明**：此表与 `users` 生命周期强绑定（1:1），不单独实施软删除——查询时通过 JOIN users 过滤已删除用户即可。

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| user_id | UUID | PRIMARY KEY | | 用户ID（同时为主键，与 users 1:1） |
| schedule_published | BOOLEAN | NOT NULL | TRUE | 排班发布通知 |
| duty_reminder | BOOLEAN | NOT NULL | TRUE | 值班提醒 |
| swap_notification | BOOLEAN | NOT NULL | TRUE | 换班通知 |
| absent_notification | BOOLEAN | NOT NULL | TRUE | 缺席通知 |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |

**外键：**
- `user_id` REFERENCES users(user_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)

---

### 3.2 学期与配置

#### 3.2.1 semesters (学期表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| semester_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| name | VARCHAR(100) | NOT NULL | | 学期名称 |
| start_date | DATE | NOT NULL | | 开始日期 |
| end_date | DATE | NOT NULL | | 结束日期 |
| first_week_type | VARCHAR(10) | NOT NULL | | 首周类型：odd/even |
| is_active | BOOLEAN | NOT NULL | FALSE | 是否为当前学期 |
| status | VARCHAR(20) | NOT NULL | 'active' | 状态：active/archived |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**字段枚举值：**
- `first_week_type`: odd(单周), even(双周)
- `status`: active(活动), archived(已归档)

**CHECK约束：**
- `first_week_type IN ('odd', 'even')`
- `status IN ('active', 'archived')`
- `end_date > start_date`
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**EXCLUDE约束：**
- `EXCLUDE USING gist (daterange(start_date, end_date, '[]') WITH &&) WHERE (deleted_at IS NULL)` —— 防止学期日期范围重叠（需启用 `btree_gist` 扩展）

**外键：**
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- `uk_semesters_active` UNIQUE (is_active) WHERE is_active = TRUE AND deleted_at IS NULL
- （可选）`idx_semesters_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

**业务约束：**
- 同一时间只能有一个 is_active = TRUE 的学期

---

#### 3.2.2 time_slots (时间段配置表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| time_slot_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| name | VARCHAR(50) | NOT NULL | | 时段名称 |
| semester_id | UUID | | NULL | 学期ID（NULL 表示全局默认配置） |
| start_time | TIME | NOT NULL | | 开始时间 |
| end_time | TIME | NOT NULL | | 结束时间 |
| day_of_week | SMALLINT | NOT NULL | | 星期几（1=周一, 2=周二, 3=周三, 4=周四, 5=周五） |
| is_active | BOOLEAN | NOT NULL | TRUE | 是否启用 |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**字段枚举值：**
- `day_of_week`: 1(周一), 2(周二), 3(周三), 4(周四), 5(周五)

**CHECK约束：**
- `day_of_week BETWEEN 1 AND 5`
- `end_time > start_time`
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**EXCLUDE约束：**
- `EXCLUDE USING gist (COALESCE(semester_id::text, '__GLOBAL__') WITH =, day_of_week WITH =, timerange(start_time, end_time) WITH &&) WHERE (deleted_at IS NULL)` —— 同一学期（或同为全局默认）的同一 `day_of_week` 下时段不可重叠（需启用 `btree_gist` 扩展及自定义 `timerange` 类型）

**外键：**
- `semester_id` REFERENCES semesters(semester_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- `idx_time_slots_day_of_week` (day_of_week) WHERE deleted_at IS NULL
- `idx_time_slots_semester_id` (semester_id) WHERE deleted_at IS NULL AND semester_id IS NOT NULL
- （可选）`idx_time_slots_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

---

#### 3.2.3 schedule_rules (排班规则配置表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| rule_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| rule_code | VARCHAR(20) | NOT NULL | | 规则代码：R1-R6 |
| rule_name | VARCHAR(100) | NOT NULL | | 规则名称 |
| description | VARCHAR(500) | | | 规则描述 |
| is_enabled | BOOLEAN | NOT NULL | TRUE | 是否启用 |
| is_configurable | BOOLEAN | NOT NULL | TRUE | 是否可配置 |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**CHECK约束：**
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- `uk_schedule_rules_code` UNIQUE (rule_code) WHERE deleted_at IS NULL
- （可选）`idx_schedule_rules_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

**预置数据：**

| rule_code | rule_name | is_configurable |
|-----------|-----------|-----------------|
| R1 | 课表冲突 | FALSE |
| R2 | 不可用时间冲突 | FALSE |
| R6 | 同人同日不重复 | FALSE |
| R3 | 同日部门不重复 | TRUE |
| R4 | 相邻班次部门不重复 | TRUE |
| R5 | 单双周早八不重复 | TRUE |

---

#### 3.2.4 system_config (系统配置表)

> **设计说明**：采用单行强类型表替代原 EAV 模式的 `system_settings`，符合 1NF，具备类型安全和 NOT NULL 约束能力。通过 `CHECK (singleton = TRUE)` + UNIQUE 保证全表仅一行。

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| singleton | BOOLEAN | PRIMARY KEY | TRUE | 单例主键（CHECK singleton = TRUE 保证全表仅一行） |
| swap_deadline_hours | INT | NOT NULL | 24 | 换班截止时间（小时） |
| duty_reminder_time | TIME | NOT NULL | '09:00' | 值班提醒发送时间 |
| default_location | VARCHAR(200) | NOT NULL | '学生会办公室' | 默认值班地点 |
| sign_in_window_minutes | INT | NOT NULL | 15 | 签到时间窗口（分钟） |
| sign_out_window_minutes | INT | NOT NULL | 15 | 签退时间窗口（分钟） |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |

**CHECK约束：**
- `singleton = TRUE`（配合 PRIMARY KEY 约束确保全表仅一行）
- `swap_deadline_hours > 0`
- `sign_in_window_minutes > 0`
- `sign_out_window_minutes > 0`

**外键：**
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)

---

#### 3.2.5 locations (值班地点表) - 预留扩展

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| location_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| name | VARCHAR(100) | NOT NULL | | 地点名称 |
| address | VARCHAR(200) | | | 详细地址 |
| is_default | BOOLEAN | NOT NULL | FALSE | 是否默认地点 |
| is_active | BOOLEAN | NOT NULL | TRUE | 是否启用 |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |

**CHECK约束：**
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- `uk_locations_default` UNIQUE (is_default) WHERE is_default = TRUE AND deleted_at IS NULL
- （可选）`idx_locations_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

**说明：** V1版本仅使用一条默认地点记录，预留多地点扩展。

---

### 3.3 时间表

#### 3.3.1 course_schedules (课表表)

> **设计说明**：`weeks` 字段使用 `INT[]`（PostgreSQL 原生数组）存储适用周次，配合 GIN 索引支持高效的 `@>` / `&&` 操作符查询。周次取值范围有限（1-25），此方案在实用性与查询性能间取得良好平衡。
>
> **`week_type` 与 `weeks` 的关系约定**：`weeks` 为唯一事实源，`week_type` 为概要性提示字段。当 `weeks` 不为 NULL 时，应用层应确保 `week_type` 与 `weeks` 内容一致（如 `week_type='odd'` 则 `weeks` 仅包含奇数周）。应用层写入时负责校验一致性，数据库层不做强制约束（CHECK 不支持子查询）。

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| course_schedule_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| user_id | UUID | NOT NULL | | 用户ID |
| semester_id | UUID | NOT NULL | | 学期ID |
| course_name | VARCHAR(100) | NOT NULL | | 课程名称 |
| day_of_week | SMALLINT | NOT NULL | | 星期几（1-7） |
| start_time | TIME | NOT NULL | | 开始时间 |
| end_time | TIME | NOT NULL | | 结束时间 |
| week_type | VARCHAR(10) | NOT NULL | 'all' | 周类型：all/odd/even |
| weeks | INT[] | | NULL | 适用周次数组（如 {1,2,3,...,16}） |
| source | VARCHAR(20) | NOT NULL | 'ics' | 来源：ics/manual |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**字段枚举值：**
- `week_type`: all(每周), odd(单周), even(双周)
- `source`: ics(导入), manual(手动录入)

**CHECK约束：**
- `week_type IN ('all', 'odd', 'even')`
- `source IN ('ics', 'manual')`
- `day_of_week BETWEEN 1 AND 7`
- `end_time > start_time`
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `user_id` REFERENCES users(user_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `semester_id` REFERENCES semesters(semester_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- ~~`idx_course_schedules_user_semester` (user_id, semester_id) WHERE deleted_at IS NULL~~ —— 已确认冗余，被 `idx_course_schedules_user_day_time` 覆盖（见 §5.2.1） (user_id, semester_id) WHERE deleted_at IS NULL
- `idx_course_schedules_user_day_time` (user_id, semester_id, day_of_week, start_time, end_time) WHERE deleted_at IS NULL
- `idx_course_schedules_weeks` GIN (weeks) WHERE deleted_at IS NULL
- （可选）`idx_course_schedules_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

---

#### 3.3.2 unavailable_times (不可用时间表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| unavailable_time_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| user_id | UUID | NOT NULL | | 用户ID |
| semester_id | UUID | NOT NULL | | 学期ID |
| day_of_week | SMALLINT | NOT NULL | | 星期几（1-7） |
| start_time | TIME | NOT NULL | | 开始时间 |
| end_time | TIME | NOT NULL | | 结束时间 |
| reason | VARCHAR(200) | | | 原因 |
| repeat_type | VARCHAR(20) | NOT NULL | 'weekly' | 重复类型：once/weekly/biweekly |
| specific_date | DATE | | NULL | 特定日期（单次时使用） |
| week_type | VARCHAR(10) | NOT NULL | 'all' | 周类型：all/odd/even |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**字段枚举值：**
- `repeat_type`: once(单次), weekly(每周重复), biweekly(双周重复)
- `week_type`: all(每周), odd(单周), even(双周)

**CHECK约束：**
- `repeat_type IN ('once', 'weekly', 'biweekly')`
- `week_type IN ('all', 'odd', 'even')`
- `day_of_week BETWEEN 1 AND 7`
- `end_time > start_time`
- `(repeat_type = 'once' AND specific_date IS NOT NULL) OR (repeat_type IN ('weekly', 'biweekly') AND specific_date IS NULL)` —— 单次必须指定日期，每周/双周重复不应指定日期
- `repeat_type = 'weekly' OR repeat_type = 'biweekly' OR week_type = 'all'` —— 单次事件无需区分单双周
- `repeat_type != 'biweekly' OR week_type IN ('odd', 'even')` —— 双周重复时 week_type 必须为 odd 或 even
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `user_id` REFERENCES users(user_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `semester_id` REFERENCES semesters(semester_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- ~~`idx_unavailable_times_user_semester` (user_id, semester_id) WHERE deleted_at IS NULL~~ —— 已确认冗余，被 `idx_unavailable_times_user_day_time` 覆盖（见 §5.2.1） (user_id, semester_id) WHERE deleted_at IS NULL
- `idx_unavailable_times_user_day_time` (user_id, semester_id, day_of_week, start_time, end_time) WHERE deleted_at IS NULL
- （可选）`idx_unavailable_times_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

---

### 3.4 排班

#### 3.4.1 schedules (排班表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| schedule_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| semester_id | UUID | NOT NULL | | 学期ID |
| status | VARCHAR(20) | NOT NULL | 'draft' | 状态 |
| published_at | TIMESTAMPTZ | | NULL | 发布时间 |
| created_by | UUID | NOT NULL | | 创建人ID |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**字段枚举值：**
- `status`: draft(草稿), published(已发布), need_regen(需重新排班), archived(已归档)

**CHECK约束：**
- `status IN ('draft', 'published', 'need_regen', 'archived')`
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `semester_id` REFERENCES semesters(semester_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- `idx_schedules_semester_id` (semester_id) WHERE deleted_at IS NULL
- （可选）`idx_schedules_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

---

#### 3.4.2 schedule_member_snapshots (排班成员快照表)

> **设计说明**：记录排班生成时的成员范围基准（duty_required=TRUE 且 timetable_status=submitted 的成员集合）。用于检测排班范围变化，支持 SRS AS-005"需重新排班"检测。此表依附于 `schedules` 生命周期，不单独实施软删除。

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| snapshot_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| schedule_id | UUID | NOT NULL | | 排班表ID |
| user_id | UUID | NOT NULL | | 成员用户ID |
| department_id | UUID | NOT NULL | | 快照时的部门ID |
| snapshot_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 快照时间 |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |

**外键：**
- `schedule_id` REFERENCES schedules(schedule_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `user_id` REFERENCES users(user_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `department_id` REFERENCES departments(department_id) ON DELETE RESTRICT ON UPDATE CASCADE

**索引：**
- `uk_schedule_member_snapshots_schedule_user` UNIQUE (schedule_id, user_id)
- ~~`idx_schedule_member_snapshots_schedule_id` (schedule_id)~~ —— 已确认冗余，被唯一索引 `uk_schedule_member_snapshots_schedule_user` 覆盖（见 §5.2.1）

---

#### 3.4.3 schedule_items (排班明细表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| schedule_item_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| schedule_id | UUID | NOT NULL | | 排班表ID |
| week_number | SMALLINT | NOT NULL | | 周次（1或2，表示单/双周模板） |
| time_slot_id | UUID | NOT NULL | | 时间段ID（星期几由 time_slots.day_of_week 确定） |
| member_id | UUID | NOT NULL | | 值班人员ID |
| location_id | UUID | | NULL | 地点ID（预留） |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**CHECK约束：**
- `week_number IN (1, 2)`
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `schedule_id` REFERENCES schedules(schedule_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `time_slot_id` REFERENCES time_slots(time_slot_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `member_id` REFERENCES users(user_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `location_id` REFERENCES locations(location_id) ON DELETE SET NULL ON UPDATE CASCADE
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- `idx_schedule_items_schedule_id` (schedule_id) WHERE deleted_at IS NULL
- `idx_schedule_items_member_id` (member_id) WHERE deleted_at IS NULL
- `idx_schedule_items_time_slot_id` (time_slot_id) WHERE deleted_at IS NULL
- `idx_schedule_items_member_schedule` (member_id, schedule_id) WHERE deleted_at IS NULL
- `uk_schedule_items_slot` UNIQUE (schedule_id, week_number, time_slot_id) WHERE deleted_at IS NULL
- （可选）`idx_schedule_items_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

---

#### 3.4.4 schedule_change_logs (排班变更记录表)

> **设计说明**：此表为纯审计日志表，只追加不删除，因此无需软删除字段。

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| change_log_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| schedule_id | UUID | NOT NULL | | 排班表ID |
| schedule_item_id | UUID | NOT NULL | | 排班项ID |
| original_member_id | UUID | NOT NULL | | 原值班人员ID |
| new_member_id | UUID | NOT NULL | | 新值班人员ID |
| original_time_slot_id | UUID | | NULL | 原时间段ID（时段变更时记录） |
| new_time_slot_id | UUID | | NULL | 新时间段ID（时段变更时记录） |
| change_type | VARCHAR(20) | NOT NULL | | 变更类型 |
| reason | VARCHAR(500) | | | 变更原因 |
| operator_id | UUID | NOT NULL | | 操作人ID |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |

**字段枚举值：**
- `change_type`: manual_adjust(手动调整), swap(换班), admin_modify(发布后修改)

**CHECK约束：**
- `change_type IN ('manual_adjust', 'swap', 'admin_modify')`
- `(original_time_slot_id IS NULL AND new_time_slot_id IS NULL) OR (original_time_slot_id IS NOT NULL AND new_time_slot_id IS NOT NULL)` —— 时段变更字段成对出现

**外键：**
- `schedule_id` REFERENCES schedules(schedule_id) ON UPDATE CASCADE ON DELETE RESTRICT
- `schedule_item_id` REFERENCES schedule_items(schedule_item_id) ON UPDATE CASCADE ON DELETE RESTRICT
- `original_member_id` REFERENCES users(user_id) ON UPDATE CASCADE ON DELETE RESTRICT
- `new_member_id` REFERENCES users(user_id) ON UPDATE CASCADE ON DELETE RESTRICT
- `original_time_slot_id` REFERENCES time_slots(time_slot_id) ON UPDATE CASCADE ON DELETE RESTRICT
- `new_time_slot_id` REFERENCES time_slots(time_slot_id) ON UPDATE CASCADE ON DELETE RESTRICT
- `operator_id` REFERENCES users(user_id) ON UPDATE CASCADE ON DELETE RESTRICT

**索引：**
- `idx_schedule_change_logs_schedule_id` (schedule_id)
- `idx_schedule_change_logs_created_at` (created_at DESC)

---

### 3.5 换班

> **⚠️ 二期工程内容** — 换班表结构已建立，后端业务逻辑待二期实现。

#### 3.5.1 swap_requests (换班申请表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| swap_request_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| schedule_item_id | UUID | NOT NULL | | 排班项ID |
| applicant_id | UUID | NOT NULL | | 申请人ID |
| target_member_id | UUID | NOT NULL | | 目标成员ID |
| reason | VARCHAR(500) | | | 换班原因 |
| status | VARCHAR(20) | NOT NULL | 'pending' | 状态 |
| target_responded_at | TIMESTAMPTZ | | NULL | 目标成员响应时间 |
| approved_at | TIMESTAMPTZ | | NULL | 审批时间 |
| approved_by | UUID | | NULL | 审批人ID |
| reject_reason | VARCHAR(500) | | | 拒绝/驳回原因 |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID（与 applicant_id 相同，保持审计字段统一） |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**字段枚举值：**
- `status`: pending(待同意), reviewing(待审核), completed(已完成), rejected(已拒绝), cancelled(已取消)

**CHECK约束：**
- `status IN ('pending', 'reviewing', 'completed', 'rejected', 'cancelled')`
- `applicant_id != target_member_id`
- `approved_at IS NULL OR approved_at >= created_at`
- `target_responded_at IS NULL OR target_responded_at >= created_at`
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `schedule_item_id` REFERENCES schedule_items(schedule_item_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `applicant_id` REFERENCES users(user_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `target_member_id` REFERENCES users(user_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `approved_by` REFERENCES users(user_id)
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- `idx_swap_requests_applicant_id` (applicant_id) WHERE deleted_at IS NULL
- ~~`idx_swap_requests_target_member_id` (target_member_id) WHERE deleted_at IS NULL~~ —— 已确认冗余，被 `idx_swap_requests_target_status` 覆盖（见 §5.2.1）
- `idx_swap_requests_schedule_item_id` (schedule_item_id) WHERE deleted_at IS NULL
- `idx_swap_requests_target_status` (target_member_id, status) WHERE deleted_at IS NULL
- （可选）`idx_swap_requests_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

---

### 3.6 签到

> **⚠️ 二期工程内容** — 签到表结构已建立，后端业务逻辑待二期实现。

#### 3.6.1 duty_records (值班记录表)

> **设计说明**：`is_late` 为受控反范式化字段（可由 `sign_in_time` 与 `time_slots.start_time` 计算），保留以避免签到列表查询时的 JOIN 开销。应用层需在签到时同步计算并写入。

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| duty_record_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| schedule_item_id | UUID | NOT NULL | | 排班项ID |
| member_id | UUID | NOT NULL | | 值班人员ID |
| duty_date | DATE | NOT NULL | | 值班日期 |
| status | VARCHAR(20) | NOT NULL | 'pending' | 状态 |
| sign_in_time | TIMESTAMPTZ | | NULL | 签到时间 |
| sign_out_time | TIMESTAMPTZ | | NULL | 签退时间 |
| is_late | BOOLEAN | NOT NULL | FALSE | 是否迟到（受控反范式化） |
| make_up_time | TIMESTAMPTZ | | NULL | 补签时间 |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**字段枚举值：**
- `status`: pending(待值班), on_duty(值班中), completed(已完成), absent(缺席), absent_made_up(缺席已补签), no_sign_out(未签退)

**CHECK约束：**
- `status IN ('pending', 'on_duty', 'completed', 'absent', 'absent_made_up', 'no_sign_out')`
- `sign_out_time IS NULL OR sign_out_time > sign_in_time`
- `make_up_time IS NULL OR status = 'absent_made_up'`
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `schedule_item_id` REFERENCES schedule_items(schedule_item_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `member_id` REFERENCES users(user_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- ~~`idx_duty_records_member_id` (member_id) WHERE deleted_at IS NULL~~ —— 已确认冗余，被 `idx_duty_records_member_date_status` 覆盖（见 §5.2.1）
- ~~`idx_duty_records_duty_date` (duty_date) WHERE deleted_at IS NULL~~ —— 已确认冗余，被 `idx_duty_records_date_status` 覆盖（见 §5.2.1）
- `idx_duty_records_date_status` (duty_date, status) WHERE deleted_at IS NULL
- `idx_duty_records_member_date_status` (member_id, duty_date, status) WHERE deleted_at IS NULL
- `uk_duty_records_schedule_item_date` UNIQUE (schedule_item_id, duty_date) WHERE deleted_at IS NULL
- （可选）`idx_duty_records_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

---

### 3.7 通知

> **⚠️ 二期工程内容** — 通知表结构已建立，后端业务逻辑待二期实现。

#### 3.7.1 notifications (通知消息表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| notification_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| user_id | UUID | NOT NULL | | 用户ID |
| type | VARCHAR(50) | NOT NULL | | 通知类型 |
| title | VARCHAR(200) | NOT NULL | | 标题 |
| content | TEXT | NOT NULL | | 内容 |
| is_read | BOOLEAN | NOT NULL | FALSE | 是否已读 |
| related_type | VARCHAR(20) | | NULL | 关联实体类型（鉴别器） |
| related_id | UUID | | NULL | 关联实体ID |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| created_by | UUID | | NULL | 创建人ID（系统触发时为 NULL） |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间（标记已读等操作） |
| updated_by | UUID | | NULL | 最后修改人ID（系统批处理时为 NULL） |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |

**字段枚举值：**
- `type`: schedule_published(排班发布), schedule_changed(排班变更), duty_reminder(值班提醒), swap_request(换班申请), swap_accepted(换班同意), swap_rejected(换班拒绝), swap_approved(换班审核通过), swap_denied(换班审核驳回), absent_alert(缺席提醒), make_up_alert(补签提醒), no_sign_out_alert(未签退提醒)
- `related_type`: schedule(排班表), schedule_item(排班项), swap_request(换班申请), duty_record(值班记录)

**CHECK约束：**
- `type IN ('schedule_published', 'schedule_changed', 'duty_reminder', 'swap_request', 'swap_accepted', 'swap_rejected', 'swap_approved', 'swap_denied', 'absent_alert', 'make_up_alert', 'no_sign_out_alert')`
- `related_type IS NULL OR related_type IN ('schedule', 'schedule_item', 'swap_request', 'duty_record')`
- `(related_type IS NULL AND related_id IS NULL) OR (related_type IS NOT NULL AND related_id IS NOT NULL)`
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `user_id` REFERENCES users(user_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `created_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- ~~`idx_notifications_user_id` (user_id) WHERE deleted_at IS NULL~~ —— 已确认冗余，被 `idx_notifications_user_read` 覆盖（见 §5.2.1）
- `idx_notifications_user_unread` (user_id, created_at DESC) WHERE is_read = FALSE AND deleted_at IS NULL
- `idx_notifications_user_read` (user_id, is_read) WHERE deleted_at IS NULL
- `idx_notifications_created_at` (created_at DESC) WHERE deleted_at IS NULL
- `idx_notifications_related` (related_type, related_id) WHERE deleted_at IS NULL AND related_type IS NOT NULL —— 支持按关联实体反向查询通知
- （可选）`idx_notifications_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

---

### 3.8 邀请码（可选，也可用Redis）

#### 3.8.1 invite_codes (邀请码表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| invite_code_id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| code | VARCHAR(50) | NOT NULL | | 邀请码 |
| created_by | UUID | NOT NULL | | 创建人ID |
| expires_at | TIMESTAMPTZ | NOT NULL | | 过期时间 |
| used_at | TIMESTAMPTZ | | NULL | 使用时间 |
| used_by | UUID | | NULL | 使用人ID |
| created_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMPTZ | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| updated_by | UUID | | NULL | 最后修改人ID |
| deleted_at | TIMESTAMPTZ | | NULL | 删除时间（软删除） |
| deleted_by | UUID | | NULL | 删除人ID |
| version | INT | NOT NULL | 1 | 乐观锁版本号 |

**CHECK约束：**
- `expires_at > created_at`
- `used_at IS NULL OR used_at <= expires_at` —— 过期码不允许使用，应用层校验过期后直接拒绝，不写入 `used_at`
- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)` —— 软删除一致性

**外键：**
- `created_by` REFERENCES users(user_id) ON DELETE RESTRICT ON UPDATE CASCADE
- `used_by` REFERENCES users(user_id)
- `updated_by` REFERENCES users(user_id)
- `deleted_by` REFERENCES users(user_id)

**索引：**
- `uk_invite_codes_code` UNIQUE (code) WHERE deleted_at IS NULL
- `idx_invite_codes_expires_at` (expires_at) WHERE deleted_at IS NULL
- （可选）`idx_invite_codes_deleted_at` (deleted_at) WHERE deleted_at IS NOT NULL —— 仅用于“回收站列表/按删除时间清理”等场景

---

## 四、数据关系约束

### 4.1 外键关系（数据库约束）

> 说明：本项目默认在 PostgreSQL 中建立外键（FK）以增强数据一致性。由于采用软删除（`deleted_at`），删除行为建议以 **RESTRICT** 为主，避免级联误删。

**推荐的删除/更新策略：**

- 所有 FK 统一：`ON UPDATE CASCADE ON DELETE RESTRICT`（因全局采用软删除，物理删除不会发生，CASCADE 永远不触发）
- 可空关联（如 `schedule_items.location_id`）：`ON DELETE SET NULL`
- 业务需要“级联清理”的场景（如学期归档清理）建议通过后台任务显式执行，而非数据库级联删除

| 表 | 字段 | 关联表 | 关联字段 | 说明 |
|----|------|--------|----------|------|
| users | department_id | departments | department_id | 用户所属部门 |
| user_semester_assignments | user_id | users | user_id | 用户-学期分配所属用户 |
| user_semester_assignments | semester_id | semesters | semester_id | 用户-学期分配所属学期 |
| notification_preferences | user_id | users | user_id | 通知偏好所属用户 |
| time_slots | semester_id | semesters | semester_id | 时间段关联学期（可为 NULL） |
| course_schedules | user_id | users | user_id | 课表所属用户 |
| course_schedules | semester_id | semesters | semester_id | 课表所属学期 |
| unavailable_times | user_id | users | user_id | 不可用时间所属用户 |
| unavailable_times | semester_id | semesters | semester_id | 不可用时间所属学期 |
| schedules | semester_id | semesters | semester_id | 排班表所属学期 |
| schedules | created_by | users | user_id | 排班表创建人 |
| schedule_member_snapshots | schedule_id | schedules | schedule_id | 快照所属排班表 |
| schedule_member_snapshots | user_id | users | user_id | 快照成员 |
| schedule_member_snapshots | department_id | departments | department_id | 快照时部门 |
| schedule_items | schedule_id | schedules | schedule_id | 排班项所属排班表 |
| schedule_items | time_slot_id | time_slots | time_slot_id | 排班项对应时间段 |
| schedule_items | member_id | users | user_id | 排班项值班人员 |
| schedule_items | location_id | locations | location_id | 排班项值班地点 |
| schedule_change_logs | schedule_id | schedules | schedule_id | 变更记录所属排班表 |
| schedule_change_logs | schedule_item_id | schedule_items | schedule_item_id | 变更记录对应排班项 |
| schedule_change_logs | original_member_id | users | user_id | 原值班人员 |
| schedule_change_logs | new_member_id | users | user_id | 新值班人员 |
| schedule_change_logs | original_time_slot_id | time_slots | time_slot_id | 原时间段（时段变更时） |
| schedule_change_logs | new_time_slot_id | time_slots | time_slot_id | 新时间段（时段变更时） |
| schedule_change_logs | operator_id | users | user_id | 操作人 |
| swap_requests | schedule_item_id | schedule_items | schedule_item_id | 换班对应排班项 |
| swap_requests | applicant_id | users | user_id | 换班申请人 |
| swap_requests | target_member_id | users | user_id | 换班目标成员 |
| swap_requests | approved_by | users | user_id | 审批人 |
| swap_requests | created_by | users | user_id | 换班申请创建人 |
| duty_records | schedule_item_id | schedule_items | schedule_item_id | 值班记录对应排班项 |
| duty_records | member_id | users | user_id | 值班人员 |
| notifications | user_id | users | user_id | 通知接收用户 |
| notifications | created_by | users | user_id | 通知创建人 |
| invite_codes | created_by | users | user_id | 邀请码创建人 |
| invite_codes | used_by | users | user_id | 邀请码使用人 |

### 4.2 数据完整性规则

#### 4.2.1 软删除一致性约束（通用）

为避免软删除数据出现不可追溯或状态不完整，所有包含 `deleted_at`、`deleted_by` 的表应满足：

- `(deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)`

> 说明：该约束已在所有含软删除字段的表中通过 CHECK 约束物化实现，保证“删除时间”与“删除人”成对出现。

| 规则 | 说明 |
|------|------|
| 学号唯一 | 同一时间不存在重复学号的用户 |
| 邮箱唯一 | 同一时间不存在重复邮箱的用户 |
| 部门名称唯一 | 同一时间不存在重复名称的部门 |
| 单一活动学期 | 同一时间最多一个活动学期 |
| 学期日期不重叠 | 不同学期的日期范围不得重叠（EXCLUDE 约束） |
| 时段不重叠 | 同一学期（或全局默认）的同一 day_of_week 下时间段不得重叠（EXCLUDE 约束） |
| 用户-学期唯一 | 同一用户在同一学期只有一条分配记录 |
| 排班时段唯一 | 同一排班表、同一周次、同一时段只有一条记录（星期几由 time_slot_id 确定） |
| 排班成员快照唯一 | 同一排班表、同一用户只有一条快照记录 |
| 值班记录唯一 | 同一排班项、同一日期只有一条值班记录 |
| 换班自引用禁止 | 申请人不得与目标成员相同 |
| 软删除一致性 | 所有含软删除字段的表均已通过 CHECK 约束保证 deleted_at/deleted_by 成对出现 |
| 时段变更成对 | schedule_change_logs 的 original/new_time_slot_id 必须同时为空或同时非空 |

---

## 五、索引设计

### 5.1 索引汇总

> 说明：此处为“核心索引”汇总（便于快速评审）。已确认冗余的索引以 ~~删除线~~ 标注。每张表的完整索引请以各表结构章节中的“索引”小节为准。

| 表 | 索引名 | 类型 | 字段 | 说明 |
|----|--------|------|------|------|
| departments | uk_departments_name | UNIQUE | name | 部门名称唯一 |
| users | uk_users_student_id | UNIQUE | student_id | 学号唯一 |
| users | uk_users_email | UNIQUE | email | 邮箱唯一 |
| users | idx_users_department_id | INDEX | department_id | 部门筛选 |
| user_semester_assignments | uk_user_semester_assignments_user_semester | UNIQUE | user_id, semester_id | 用户-学期唯一 |
| user_semester_assignments | idx_user_semester_assignments_duty_required | INDEX | semester_id, duty_required | 学期值班状态筛选 |
| time_slots | idx_time_slots_day_of_week | INDEX | day_of_week | 星期筛选 |
| time_slots | idx_time_slots_semester_id | INDEX | semester_id | 学期筛选（非空时） |
| ~~course_schedules~~ | ~~idx_course_schedules_user_semester~~ | ~~INDEX~~ | ~~user_id, semester_id~~ | ~~冗余，见 §5.2.1~~ |
| course_schedules | idx_course_schedules_weeks | GIN | weeks (INT[]) | 周次数组查询 |
| ~~unavailable_times~~ | ~~idx_unavailable_times_user_semester~~ | ~~INDEX~~ | ~~user_id, semester_id~~ | ~~冗余，见 §5.2.1~~ |
| schedules | idx_schedules_semester_id | INDEX | semester_id | 学期筛选 |
| schedule_member_snapshots | uk_schedule_member_snapshots_schedule_user | UNIQUE | schedule_id, user_id | 排班-成员唯一 |
| ~~schedule_member_snapshots~~ | ~~idx_schedule_member_snapshots_schedule_id~~ | ~~INDEX~~ | ~~schedule_id~~ | ~~冗余，见 §5.2.1~~ |
| schedule_items | idx_schedule_items_schedule_id | INDEX | schedule_id | 排班表筛选 |
| schedule_items | idx_schedule_items_member_id | INDEX | member_id | 值班人员筛选 |
| schedule_items | idx_schedule_items_time_slot_id | INDEX | time_slot_id | 时段筛选 |
| schedule_items | idx_schedule_items_member_schedule | INDEX | member_id, schedule_id | 用户排班表复合查询 |
| schedule_items | uk_schedule_items_slot | UNIQUE | schedule_id, week_number, time_slot_id | 时段唯一 |
| swap_requests | idx_swap_requests_applicant_id | INDEX | applicant_id | 申请人筛选 |
| ~~swap_requests~~ | ~~idx_swap_requests_target_member_id~~ | ~~INDEX~~ | ~~target_member_id~~ | ~~冗余，见 §5.2.1~~ |
| swap_requests | idx_swap_requests_target_status | INDEX | target_member_id, status | 目标成员+状态复合查询 |
| ~~duty_records~~ | ~~idx_duty_records_member_id~~ | ~~INDEX~~ | ~~member_id~~ | ~~冗余，见 §5.2.1~~ |
| ~~duty_records~~ | ~~idx_duty_records_duty_date~~ | ~~INDEX~~ | ~~duty_date~~ | ~~冗余，见 §5.2.1~~ |
| duty_records | idx_duty_records_date_status | INDEX | duty_date, status | 日期+状态复合查询（定时任务） |
| duty_records | idx_duty_records_member_date_status | INDEX | member_id, duty_date, status | 人员+日期+状态复合查询 |
| duty_records | uk_duty_records_schedule_item_date | UNIQUE | schedule_item_id, duty_date | 排班项+日期唯一 |
| ~~notifications~~ | ~~idx_notifications_user_id~~ | ~~INDEX~~ | ~~user_id~~ | ~~冗余，见 §5.2.1~~ |
| notifications | idx_notifications_user_unread | INDEX | user_id, created_at DESC | 未读消息查询 |
| notifications | idx_notifications_user_read | INDEX | user_id, is_read | 已读/未读计数 |
| notifications | idx_notifications_created_at | INDEX | created_at DESC | 时间排序 |
| notifications | idx_notifications_related | INDEX | related_type, related_id | 多态关联反向查询 |

---

### 5.2 索引整改建议（保守策略，默认建议）

本节目标是减少“重复覆盖索引”带来的写放大与维护成本，同时不牺牲核心查询性能。

**判定依据（PostgreSQL btree 常识规则）：**

- 对于 btree 复合索引 `(a, b, c)`，通常可以用于仅按 `a` 或按 `(a, b)` 过滤的查询（左前缀匹配）。
- 因此，若已存在以同一前缀开头的复合索引，单列/短复合索引往往是冗余的。

#### 5.2.1 已移除（确定冗余，已在各表索引定义中标注删除线）

以下索引已在 v1.9 中标记为冗余，建议在初始化 SQL 脚本中不创建：

- `course_schedules.idx_course_schedules_user_semester`：已被 `idx_course_schedules_user_day_time (user_id, semester_id, day_of_week, start_time, end_time)` 覆盖。
- `unavailable_times.idx_unavailable_times_user_semester`：已被 `idx_unavailable_times_user_day_time (user_id, semester_id, day_of_week, start_time, end_time)` 覆盖。
- `schedule_member_snapshots.idx_schedule_member_snapshots_schedule_id`：已被唯一索引 `uk_schedule_member_snapshots_schedule_user (schedule_id, user_id)` 覆盖。
- `swap_requests.idx_swap_requests_target_member_id`：已被 `idx_swap_requests_target_status (target_member_id, status)` 覆盖。
- `duty_records.idx_duty_records_member_id`：通常可被 `idx_duty_records_member_date_status (member_id, duty_date, status)` 覆盖。
- `duty_records.idx_duty_records_duty_date`：通常可被 `idx_duty_records_date_status (duty_date, status)` 覆盖。
- `notifications.idx_notifications_user_id`：通常可被 `idx_notifications_user_read (user_id, is_read)` 覆盖。

> 若存在“只按前缀字段过滤但强依赖极窄索引尺寸”的场景，上述索引也可能仍有价值；但在默认业务系统中，这类收益通常不足以抵消维护成本。

#### 5.2.2 建议评估后决定（依赖查询画像）

- `user_semester_assignments.idx_user_semester_assignments_semester_id`：若主要查询都是“按学期 + duty_required”筛选，则可由 `idx_user_semester_assignments_duty_required (semester_id, duty_required)` 覆盖；但若经常仅按 `semester_id` 拉全量分配列表，也可保留。
- `notifications.idx_notifications_created_at`：如果没有“全站按时间倒序扫通知”的后台任务/审计需求，该索引可去掉；若存在按时间清理或全局检索，建议保留。

#### 5.2.3 执行状态

上述“确定冗余”索引已在各表的索引定义中标注删除线，并在 §5.1 索引汇总中同步标记。初始化 SQL 脚本应跳过这些索引。若生产环境出现慢查询回归，可按需恢复。

---

## 六、数据字典

### 6.1 枚举值定义

#### 用户角色 (users.role)

| 值 | 说明 |
|----|------|
| admin | 排班管理员 |
| leader | 部门负责人 |
| member | 值班成员 |

#### 时间表状态 (user_semester_assignments.timetable_status)

| 值 | 说明 |
|----|------|
| not_submitted | 未提交 |
| submitted | 已提交 |

#### 周类型 (first_week_type / week_type)

| 值 | 说明 |
|----|------|
| all | 每周 |
| odd | 单周 |
| even | 双周 |

#### 日类型 (day_of_week)

> **值域说明**：本项目存在两组不同值域的 `day_of_week` 字段：
> - **值班相关表**（`time_slots.day_of_week`）：范围 **1-5**（仅工作日），因值班仅在周一至周五进行
> - **时间表相关表**（`course_schedules.day_of_week`、`unavailable_times.day_of_week`）：范围 **1-7**（含周末），因课程和个人事务可发生在任意一天

**time_slots.day_of_week (1-5)**

| 值 | 说明 |
|----|------|
| 1 | 周一 |
| 2 | 周二 |
| 3 | 周三 |
| 4 | 周四 |
| 5 | 周五 |

**course_schedules / unavailable_times 的 day_of_week (1-7)**

| 值 | 说明 |
|----|------|
| 1 | 周一 |
| 2 | 周二 |
| 3 | 周三 |
| 4 | 周四 |
| 5 | 周五 |
| 6 | 周六 |
| 7 | 周日 |

#### 排班表状态 (schedules.status)

| 值 | 说明 |
|----|------|
| draft | 草稿 |
| published | 已发布 |
| need_regen | 需重新排班 |
| archived | 已归档 |

#### 换班状态 (swap_requests.status)

| 值 | 说明 |
|----|------|
| pending | 待同意 |
| reviewing | 待审核 |
| completed | 已完成 |
| rejected | 已拒绝 |
| cancelled | 已取消 |

#### 值班记录状态 (duty_records.status)

| 值 | 说明 |
|----|------|
| pending | 待值班 |
| on_duty | 值班中 |
| completed | 已完成 |
| absent | 缺席 |
| absent_made_up | 缺席已补签 |
| no_sign_out | 未签退 |

#### 重复类型 (unavailable_times.repeat_type)

| 值 | 说明 |
|----|------|
| once | 单次 |
| weekly | 每周重复 |
| biweekly | 双周重复 |

#### 变更类型 (schedule_change_logs.change_type)

| 值 | 说明 |
|----|------|
| manual_adjust | 手动调整 |
| swap | 换班 |
| admin_modify | 发布后修改 |

#### 课表来源 (course_schedules.source)

| 值 | 说明 |
|----|------|
| ics | ICS 文件导入 |
| manual | 手动录入 |

#### 通知类型 (notifications.type)

| 值 | 说明 |
|----|------|
| schedule_published | 排班发布 |
| schedule_changed | 排班变更 |
| duty_reminder | 值班提醒 |
| swap_request | 收到换班申请 |
| swap_accepted | 换班被同意 |
| swap_rejected | 换班被拒绝 |
| swap_approved | 换班审核通过 |
| swap_denied | 换班审核驳回 |
| absent_alert | 缺席提醒 |
| make_up_alert | 补签提醒 |
| no_sign_out_alert | 未签退提醒 |

#### 通知关联实体类型 (notifications.related_type)

| 值 | 说明 |
|----|------|
| schedule | 排班表 |
| schedule_item | 排班项 |
| swap_request | 换班申请 |
| duty_record | 值班记录 |

---

## 七、数据初始化

### 7.1 系统初始化数据

#### 默认部门

以下为 **初始化示例数据（非 SQL 脚本）**，用于说明建议的默认配置项；实际初始化可通过迁移工具、管理后台或启动脚本完成。

| name | description |
|------|-------------|
| 秘书部 | 负责排班管理 |
| 宣传部 | 负责宣传工作 |
| … | … |

#### 默认时间段

> 每天独立配置时间段，`day_of_week` 取值 1-5 对应周一至周五。周一至周四共用相同时段配置（各4个），周五单独配置（3个）。

**周一至周四 (day_of_week=1/2/3/4)，每天各配置以下4个时段：**

| name | day_of_week | start_time | end_time |
|------|-------------|------------|----------|
| 第一时段 | 1-4 | 08:10 | 10:05 |
| 第二时段 | 1-4 | 10:20 | 12:15 |
| 第三时段 | 1-4 | 14:00 | 16:00 |
| 第四时段 | 1-4 | 16:10 | 18:00 |

**周五 (day_of_week=5)：**

| name | day_of_week | start_time | end_time |
|------|-------------|------------|----------|
| 第一时段 | 5 | 08:10 | 10:05 |
| 第二时段 | 5 | 10:20 | 12:15 |
| 第三时段 | 5 | 14:00 | 16:00 |

#### 排班规则

| rule_code | rule_name | description | is_enabled | is_configurable |
|-----------|-----------|-------------|-----------|-----------------|
| R1 | 课表冲突 | 有课的时段不能排班 | TRUE | FALSE |
| R2 | 不可用时间冲突 | 用户标记的不可用时段不能排班 | TRUE | FALSE |
| R6 | 同人同日不重复 | 同一成员同一天最多安排一个班次 | TRUE | FALSE |
| R3 | 同日部门不重复 | 同一天的不同时段不能有来自同一部门的人 | TRUE | TRUE |
| R4 | 相邻班次部门不重复 | 相邻两个时段不能来自同一部门 | TRUE | TRUE |
| R5 | 单双周早八不重复 | 单周和双周的早八不能是同一人 | TRUE | TRUE |

#### 系统配置

> 系统配置已重构为强类型单行表 `system_config`，以下为默认值：

| 字段 | 默认值 | 说明 |
|------|--------|------|
| swap_deadline_hours | 24 | 换班截止时间（小时） |
| duty_reminder_time | 09:00 | 值班提醒发送时间 |
| default_location | 学生会办公室 | 默认值班地点 |
| sign_in_window_minutes | 15 | 签到时间窗口（分钟） |
| sign_out_window_minutes | 15 | 签退时间窗口（分钟） |

#### 默认地点

| name | address | is_default |
|------|---------|-----------|
| 学生会办公室 | 学生活动中心201 | TRUE |

#### 初始管理员

初始管理员建议在部署阶段通过“初始化命令/管理后台”创建：

| name | student_id | email | role | must_change_password |
|------|------------|-------|------|----------------------|
| 系统管理员 | admin001 | admin@example.com | admin | TRUE |

---

## 八、数据备份与恢复

### 8.1 备份策略

| 数据类型 | 备份频率 | 保留周期 | 备份方式 |
|----------|----------|----------|----------|
| 全量备份 | 每日凌晨 | 7天 | pg_dump |
| 增量备份 | 每小时 | 24小时 | WAL归档 |

### 8.2 恢复流程

1. 停止应用服务
2. 恢复数据库备份
3. 应用WAL日志（如需要）
4. 验证数据完整性
5. 重启应用服务

---

## 九、性能优化建议

### 9.1 查询优化

| 场景 | 优化建议 |
|------|----------|
| 用户列表查询 | 使用分页、部门索引筛选 |
| 排班表查询 | 预加载关联数据，避免N+1 |
| 时间表冲突检查 | 使用时间范围索引 |
| 签到状态查询 | 按日期分区或索引 |

### 9.2 数据清理

| 数据类型 | 清理策略 |
|----------|----------|
| 历史学期数据 | 仅保留当前和上一学期 |
| 过期邀请码 | 定时清理过期记录 |
| 已读通知 | 保留30天内的记录 |
| 日志记录 | 定期归档超过90天的记录 |

---

## 版本历史

| 版本 | 日期 | 修改内容 | 修改人 |
|------|------|----------|--------|
| v1.0 | 2026-01-29 | 初稿 | 系统架构师 |
| v1.1 | 2026-02-24 | 完善数据字典设计：<br>1. 所有表添加完整审计字段（created_by, updated_by, deleted_by）<br>2. 统一实施软删除方案（deleted_at, deleted_by）<br>3. 核心表添加乐观锁字段（version）<br>4. 修正所有TIMESTAMP为TIMESTAMPTZ<br>5. 补充CHECK约束定义（枚举值、范围校验）<br>6. 优化索引定义（适配软删除、复合索引）<br>7. 明确外键级联策略<br>8. 标注设计缺陷（course_schedules.weeks, system_settings）供v1.2重构 | 数据库架构师 |
| v1.2 | 2026-02-24 | 范式化整改与工程规范统一：<br>1. **抽取 `user_semester_assignments` 表**：将 `duty_required`/`timetable_status`/`timetable_submitted_at` 从 `users` 表移至新表，增加学期维度<br>2. **重构 `system_settings` 为 `system_config`**：消除 EAV 反模式，采用强类型单行表<br>3. **修复 `course_schedules.weeks`**：VARCHAR(100) 改为 INT[]，消除 1NF 违规<br>4. **统一 FK 策略**：所有外键统一为 `ON DELETE RESTRICT`，消除 CASCADE 与软删除的语义冲突<br>5. **补全审计字段**：修复 8 张表的 created_by/updated_by 缺失<br>6. **增强约束**：添加 `applicant_id != target_member_id`、`notifications.type` CHECK、`make_up_time` 状态约束<br>7. **修正 ERD**：schedule_items↔duty_records 从 1:1 修正为 1:N<br>8. **添加 `notifications.related_type`** 鉴别器列，解决多态外键问题<br>9. **优化索引**：添加复合索引、GIN 索引，移除低选择性索引<br>10. **标注受控反范式化**：明确 `is_late` 为冗余派生字段<br>11. **文档化循环 FK 初始化顺序** | 数据库架构师 |
| v1.3 | 2026-02-24 | 全面审查整改版：<br>**P0 修复：**<br>1. **`course_schedules.weeks` 类型落定**：表结构定义统一为 `INT[]`，修正与 v1.2 变更日志的矛盾，删除过时设计说明，确认 GIN 索引可用<br>2. **`course_schedules` 补 `updated_by` 列**：修复列定义与 FK 声明不一致（建表会报错）<br>3. **`semesters` 添加 EXCLUDE 约束**：防止学期日期范围重叠（需 `btree_gist` 扩展）<br>4. **新增 `schedule_member_snapshots` 表**：记录排班生成时的成员范围基准，支持 SRS AS-005"需重新排班"检测<br>**P1 修复：**<br>5. **`user_semester_assignments` 添加 `version` 字段**：防止 duty_required 切换与时间表提交的并发丢失更新<br>6. **`swap_requests` 增加 `cancelled` 状态**：支持申请人撤回换班申请<br>7. **`time_slots` 增加可选 `semester_id`**：支持不同学期的时段配置版本化（NULL 表示全局默认）<br>8. **`course_schedules.source` CHECK 扩展**：`'ics'` → `IN ('ics', 'manual')` 预留手动录入<br>**P2 修复：**<br>9. **`notifications` 补 `created_by` 审计字段**：系统触发时为 NULL<br>10. **`time_slots` 添加 EXCLUDE 约束**：防止同 day_type 下时段重叠<br>11. **`notifications.related_id` 多态 FK 声明**：文档明确应用层负责引用完整性<br>12. **`schedule_change_logs` 扩展字段**：增加 `original_time_slot_id`/`new_time_slot_id` 支持时段变更记录<br>**P3 修复：**<br>13. **`departments` 补 `code` 字段**：对齐 SRS 部门代码需求，含 partial unique 约束<br>14. **邀请码方案定位**：明确 DB 为持久化方案，Redis 为可选缓存层<br>15. **更新 ERD**：补充 `schedule_member_snapshots`、`time_slots→semesters` 关系<br>16. **更新数据字典**：同步所有新增枚举值（cancelled、manual、source） | 数据库架构师 |
| v1.4 | 2026-02-24 | 范式与工程规范修缮版：<br>**P0 修复：**<br>1. **`time_slots` EXCLUDE 约束修复**：原约束 `semester_id IS NOT DISTINCT FROM semester_id` 为自身比较恒真，修正为 `COALESCE(semester_id::text, '__GLOBAL__')` 按学期分区检测重叠<br>2. **新增自定义 `timerange` 类型**：PostgreSQL 无内建 TIME range 类型，需 `CREATE TYPE timerange AS RANGE (subtype = time)`<br>3. **`course_schedules` week_type/weeks 关系约定**：明确 `weeks` 为唯一事实源，`week_type` 为概要提示，应用层负责一致性校验<br>**P1 修复：**<br>4. **`unavailable_times` 一致性约束**：添加 `repeat_type↔specific_date`、`repeat_type↔week_type` CHECK 约束<br>5. **`duty_records.member_id` 反范式化声明**：补充至受控反范式化声明（换班后保留原始分配人快照）<br>6. **`notifications` 补 `updated_at`**：标记已读为 UPDATE 操作，需追踪更新时间<br>7. **`swap_requests` 补 `created_by`**：与全局审计字段模式统一（值与 `applicant_id` 相同）<br>8. **`schedule_change_logs` FK 级联策略**：显式声明 `ON UPDATE CASCADE ON DELETE RESTRICT`，与全局约定一致<br>**P2 修复：**<br>9. **移除低选择性单列索引**：删除 `idx_users_role`、`idx_schedules_status`、`idx_swap_requests_status`、`idx_duty_records_status`<br>10. **新增 `idx_duty_records_date_status`**：支持定时任务（缺席自动标记）按日期+状态批量查询<br>11. **新增 `idx_schedule_items_time_slot_id`**：支持按时段查找排班项的冲突检查<br>12. **`schedule_items.day_of_week` 注释修正**：修正错别字"课表瞄"→"课表的"<br>13. **`time_slots`/`schedule_rules` 补 `version` 字段**：防止管理员并发编辑配置的丢失更新<br>14. **ERD 重绘**：修复对齐问题，补充 `locations`、`system_config`、`schedule_rules`、`notification_preferences` 实体 | 数据库架构师 |
| v1.5 | 2026-02-24 | 命名规范与字段整理：<br>1. **删除 `departments.code` 字段**：业务无部门代码需求，移除该字段及 `uk_departments_code` 索引<br>2. **所有表主键加表前缀**：`id` → `{table}_id`（如 `department_id`、`user_id`、`semester_id` 等），消除多表 JOIN 时的歧义<br>3. **`time_slots.day_type` 改为 `day_of_week`**：从 `VARCHAR(20)` 分组枚举（weekday/friday）改为 `SMALLINT`（1-5 对应周一至周五），消除命名歧义，支持按天独立配置时段<br>4. 同步更新所有 FK 引用、FK 汇总表、索引汇总表、数据字典及默认数据 | 数据库架构师 |
| v1.6 | 2026-02-24 | 精简设计版：<br>1. **删除 `departments.sort_order` 字段**：部门排序应由应用层控制，不需数据库字段<br>2. **删除 `time_slots.sort_order` 字段**：时间段排序信息冗余，删除相关索引 `idx_time_slots_day_of_week` 复合键中的 `sort_order`<br>3. 更新默认部门数据表（移除 sort_order 列）<br>4. 更新默认时间段数据表（移除 sort_order 列及数据） | 数据库架构师 |
| v1.7 | 2026-02-24 | 工程规范收敛版：<br>1. **补齐审计字段**：`notifications`、`invite_codes` 增加 `updated_by`，与 `updated_at` 语义对齐<br>2. **软删除一致性约束**：补充通用规则，要求 `deleted_at` 与 `deleted_by` 成对出现<br>3. **索引分层**：将各表 `deleted_at IS NOT NULL` 的“回收站/清理”索引标注为可选，避免默认索引膨胀 | 数据库架构师 |
| v1.8 | 2026-02-24 | 索引收敛建议版：<br>1. **明确索引汇总范围**：第5章索引汇总为核心索引，完整索引以各表章节为准<br>2. **新增索引整改建议（保守策略）**：给出“确定冗余可移除”清单与“需结合查询画像评估”清单，降低写放大与维护成本 | 数据库架构师 |
| v1.9 | 2026-02-24 | 终审定稿版（準备进入 SQL 初始化脚本阶段）：<br>**P0 修复：**<br>1. **移除 `schedule_items.day_of_week`**：消除与 `time_slots.day_of_week` 的传递依赖/不一致风险，星期几统一由 `time_slot_id` 确定；唯一索引更新为 `(schedule_id, week_number, time_slot_id)`<br>2. **软删除 CHECK 约束物化**：所有 15 张含软删除字段的表均添加 `CHECK ((deleted_at IS NULL AND deleted_by IS NULL) OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL))`，将工程规范下沉为数据库级约束<br>**P1 修复：**<br>3. **`course_schedules.week_type` 登记为受控反范式化**：补充至 §1.2 受控反范式化声明<br>4. **审计字段 FK 级联策略明确**：新增 §1.2 约定，说明审计 FK 省略级联子句即采用 PG 默认 `NO ACTION`<br>5. **补充乐观锁字段**：`course_schedules`、`unavailable_times`、`invite_codes` 添加 `version INT`<br>6. **`notification_preferences` 主键精简**：移除冗余代理主键 `preference_id`，改用 `user_id` 作为 PK<br>7. **索引冗余标注同步**：§5.2.1 已确认的 7 个冗余索引在各表索引定义及 §5.1 汇总中标注删除线<br>**P2 修复：**<br>8. **`schedule_change_logs` 时段成对约束**：添加 CHECK 约束保证 `original_time_slot_id`/`new_time_slot_id` 成对出现<br>9. **`day_of_week` 值域说明**：数据字典区分值班相关(1-5)与时间表相关(1-7)的不同值域<br>10. **`system_config` 主键精简**：移除冗余 `config_id`，`singleton` 直接作为 PK<br>11. **`invite_codes.used_at` 约束意图说明**：补充注释明确过期码不允许使用的设计意图<br>12. **`notifications` 多态索引**：新增 `idx_notifications_related (related_type, related_id)` 支持反向查询 | 数据库架构师 |

---

## 附录：init.sql 初始化脚本设计决策记录

> 以下决策在生成 `backend/init.sql` 脚本时确认，记录时间：2026-02-24。

| 编号 | 问题 | 选定方案 | 说明 |
|------|------|----------|------|
| Q1 | 默认部门种子数据 | **B — 不插入** | 不在 init.sql 中插入任何默认部门数据，完全由运行时管理后台创建。文档 §7.1 标注为"初始化示例数据（非 SQL 脚本）" |
| Q2 | 初始管理员账号 | **A — 不创建** | 不在 init.sql 中创建初始管理员，由部署脚本或管理后台处理。与文档 §7.1 建议一致 |
| Q3 | `deleted_at IS NOT NULL` 回收站索引 | **A — 初始化时全部不创建** | 15 张含软删除的表均标注为"可选"，初始化时不创建以避免索引膨胀；后续按需使用 `CREATE INDEX CONCURRENTLY` 添加 |
| Q4 | §5.2.2 两个待评估索引 | **A — 全部创建** | 采用宽索引策略起步：创建 `idx_user_semester_assignments_semester_id` 与 `idx_notifications_created_at`，后续按监控数据决定是否删除 |
| Q5 | 脚本前置设施 | **A — 仅会话级 SET timezone** | 仅保留 `SET timezone = 'Asia/Shanghai'`（会话级），不包含 `CREATE DATABASE` / `CREATE ROLE` / `ALTER DATABASE`。数据库和角色创建由基础设施层或部署脚本负责 |

