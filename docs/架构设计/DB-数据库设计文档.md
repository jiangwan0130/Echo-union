# 数据库设计文档 (Database Design)
# 学生会值班管理系统

| 文档信息 | |
|----------|----------|
| 版本号 | v1.0 |
| 创建日期 | 2026-01-29 |
| 最后更新 | 2026-01-29 |
| 文档状态 | 初稿 |
| 数据库 | PostgreSQL 15+ |

---

## 一、文档概述

### 1.1 目的

本文档定义学生会值班管理系统的数据库设计，包括实体关系图（ERD）、表结构定义、字段说明、索引设计及数据约束。

### 1.2 设计原则

- 使用 UUID 作为主键（便于分布式扩展）
- 所有表包含 `created_at`、`updated_at` 审计字段
- 软删除使用 `deleted_at` 字段
- 外键使用数据库约束（FK），并结合软删除策略保证一致性
- 合理使用索引优化查询性能

**时区与时间类型约定：**

- 数据库时区：`Asia/Shanghai`
- 所有时间戳字段（`*_at`、`sign_in_time`、`sign_out_time` 等）统一使用 **TIMESTAMPTZ**（带时区）
- 业务日期字段（如 `duty_date`）使用 **DATE**

### 1.3 命名规范

- 表名：小写下划线分隔，复数形式（如 `users`、`departments`）
- 字段名：小写下划线分隔（如 `created_at`、`department_id`）
- 索引名：`idx_表名_字段名`
- 唯一索引：`uk_表名_字段名`

---

## 二、实体关系图 (ERD)

### 2.1 核心实体关系

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              实体关系图 (ERD)                                     │
├─────────────────────────────────────────────────────────────────────────────────┤
│                                                                                 │
│                                                                                 │
│   ┌─────────────┐         ┌─────────────┐         ┌─────────────┐             │
│   │             │    1    │             │    N    │             │             │
│   │   semesters │◄────────│  schedules  │────────►│schedule_items│             │
│   │   学期      │         │  排班表      │         │  排班明细    │             │
│   │             │         │             │         │             │             │
│   └─────────────┘         └─────────────┘         └──────┬──────┘             │
│         │                                                │                     │
│         │ 1                                              │ 1                   │
│         │                                                │                     │
│         ▼ N                                              ▼ 1                   │
│   ┌─────────────┐                                 ┌─────────────┐             │
│   │             │                                 │             │             │
│   │ time_slots  │                                 │duty_records │             │
│   │ 时间段配置   │                                 │  值班记录    │             │
│   │             │                                 │             │             │
│   └─────────────┘                                 └─────────────┘             │
│                                                                                 │
│                                                                                 │
│   ┌─────────────┐    N    ┌─────────────┐    1    ┌─────────────┐             │
│   │             │◄────────│             │────────►│             │             │
│   │ departments │         │    users    │         │   roles     │             │
│   │   部门      │         │    用户     │         │   角色      │             │
│   │             │         │             │         │ (枚举/逻辑)  │             │
│   └─────────────┘         └──────┬──────┘         └─────────────┘             │
│                                  │                                              │
│                                  │ 1                                            │
│                                  │                                              │
│              ┌───────────────────┼───────────────────┐                         │
│              │                   │                   │                         │
│              ▼ N                 ▼ N                 ▼ N                        │
│   ┌─────────────────┐   ┌─────────────────┐   ┌─────────────┐                 │
│   │                 │   │                 │   │             │                 │
│   │ course_schedules│   │unavailable_times│   │swap_requests│                 │
│   │   课表          │   │  不可用时间      │   │  换班申请    │                 │
│   │                 │   │                 │   │             │                 │
│   └─────────────────┘   └─────────────────┘   └─────────────┘                 │
│                                                                                 │
│                                                                                 │
│   ┌─────────────┐         ┌─────────────┐         ┌─────────────┐             │
│   │             │         │             │         │             │             │
│   │notifications│         │ invite_codes│         │schedule_    │             │
│   │  通知消息    │         │  邀请码      │         │change_logs  │             │
│   │             │         │ (Redis/表)   │         │  变更记录    │             │
│   └─────────────┘         └─────────────┘         └─────────────┘             │
│                                                                                 │
└─────────────────────────────────────────────────────────────────────────────────┘
```

> 说明：`roles` 在图中仅表示 **用户角色枚举域**（`users.role`），V1 不单独建立 `roles` 表。

### 2.2 实体关系说明

| 关系 | 说明 |
|------|------|
| departments ↔ users | 一对多：一个部门有多个用户 |
| semesters ↔ schedules | 一对多：一个学期可有多个排班表（实际1个） |
| schedules ↔ schedule_items | 一对多：一个排班表有多个排班项 |
| schedule_items ↔ duty_records | 一对一：一个排班项对应一条值班记录 |
| users ↔ course_schedules | 一对多：一个用户有多条课表记录 |
| users ↔ unavailable_times | 一对多：一个用户有多条不可用时间 |
| users ↔ swap_requests | 一对多：一个用户可发起/接收多个换班申请 |
| users ↔ notifications | 一对多：一个用户有多条通知 |

---

## 三、表结构设计

> 类型说明：下列表结构中若出现 `TIMESTAMP`，均按本项目约定视为 **TIMESTAMPTZ（带时区）**，数据库时区为 `Asia/Shanghai`。

### 3.1 用户与组织

#### 3.1.1 departments (部门表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| name | VARCHAR(50) | NOT NULL, UNIQUE | | 部门名称 |
| description | VARCHAR(200) | | | 部门描述 |
| is_active | BOOLEAN | NOT NULL | TRUE | 是否启用 |
| sort_order | INT | | 0 | 排序序号 |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| deleted_at | TIMESTAMP | | NULL | 删除时间（软删除） |

**索引：**
- `uk_departments_name` UNIQUE (name) WHERE deleted_at IS NULL

---

#### 3.1.2 users (用户表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| name | VARCHAR(50) | NOT NULL | | 姓名 |
| student_id | VARCHAR(20) | NOT NULL, UNIQUE | | 学号 |
| email | VARCHAR(100) | NOT NULL, UNIQUE | | 邮箱 |
| password_hash | VARCHAR(255) | NOT NULL | | 密码哈希 |
| role | VARCHAR(20) | NOT NULL | 'member' | 角色：admin/leader/member |
| department_id | UUID | NOT NULL | | 部门ID |
| duty_required | BOOLEAN | NOT NULL | FALSE | 是否需要值班 |
| timetable_status | VARCHAR(20) | NOT NULL | 'not_submitted' | 时间表状态 |
| timetable_submitted_at | TIMESTAMP | | NULL | 时间表提交时间 |
| must_change_password | BOOLEAN | NOT NULL | FALSE | 是否需要修改密码 |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |
| deleted_at | TIMESTAMP | | NULL | 删除时间 |

**字段枚举值：**
- `role`: admin(排班管理员), leader(部门负责人), member(值班成员)
- `timetable_status`: not_submitted(未提交), submitted(已提交)

**索引：**
- `uk_users_student_id` UNIQUE (student_id) WHERE deleted_at IS NULL
- `uk_users_email` UNIQUE (email) WHERE deleted_at IS NULL
- `idx_users_department_id` (department_id)
- `idx_users_role` (role)
- `idx_users_duty_required` (duty_required)

---

#### 3.1.3 notification_preferences (通知偏好表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| user_id | UUID | NOT NULL, UNIQUE | | 用户ID |
| schedule_published | BOOLEAN | NOT NULL | TRUE | 排班发布通知 |
| duty_reminder | BOOLEAN | NOT NULL | TRUE | 值班提醒 |
| swap_notification | BOOLEAN | NOT NULL | TRUE | 换班通知 |
| absent_notification | BOOLEAN | NOT NULL | TRUE | 缺席通知 |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |

---

### 3.2 学期与配置

#### 3.2.1 semesters (学期表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| name | VARCHAR(100) | NOT NULL | | 学期名称 |
| start_date | DATE | NOT NULL | | 开始日期 |
| end_date | DATE | NOT NULL | | 结束日期 |
| first_week_type | VARCHAR(10) | NOT NULL | | 首周类型：odd/even |
| is_active | BOOLEAN | NOT NULL | FALSE | 是否为当前学期 |
| status | VARCHAR(20) | NOT NULL | 'active' | 状态：active/archived |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |

**字段枚举值：**
- `first_week_type`: odd(单周), even(双周)
- `status`: active(活动), archived(已归档)

**索引：**
- `idx_semesters_is_active` (is_active) WHERE is_active = TRUE

**约束：**
- 同一时间只能有一个 is_active = TRUE 的学期

---

#### 3.2.2 time_slots (时间段配置表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| name | VARCHAR(50) | NOT NULL | | 时段名称 |
| start_time | TIME | NOT NULL | | 开始时间 |
| end_time | TIME | NOT NULL | | 结束时间 |
| day_type | VARCHAR(20) | NOT NULL | | 适用日类型：weekday/friday |
| sort_order | INT | NOT NULL | 0 | 排序序号 |
| is_active | BOOLEAN | NOT NULL | TRUE | 是否启用 |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |

**字段枚举值：**
- `day_type`: weekday(周一至周四), friday(周五)

---

#### 3.2.3 schedule_rules (排班规则配置表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| rule_code | VARCHAR(20) | NOT NULL, UNIQUE | | 规则代码：R1-R6 |
| rule_name | VARCHAR(100) | NOT NULL | | 规则名称 |
| description | VARCHAR(500) | | | 规则描述 |
| is_enabled | BOOLEAN | NOT NULL | TRUE | 是否启用 |
| is_configurable | BOOLEAN | NOT NULL | TRUE | 是否可配置 |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |

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

#### 3.2.4 system_settings (系统设置表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| key | VARCHAR(50) | NOT NULL, UNIQUE | | 配置键 |
| value | VARCHAR(500) | NOT NULL | | 配置值 |
| description | VARCHAR(200) | | | 配置说明 |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |

**预置数据：**

| key | value | description |
|-----|-------|-------------|
| swap_deadline_hours | 24 | 换班截止时间（小时） |
| duty_reminder_time | 09:00 | 值班提醒发送时间 |
| default_location | 学生会办公室 | 默认值班地点 |
| sign_in_window_minutes | 15 | 签到时间窗口（分钟） |
| sign_out_window_minutes | 15 | 签退时间窗口（分钟） |

---

#### 3.2.5 locations (值班地点表) - 预留扩展

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| name | VARCHAR(100) | NOT NULL | | 地点名称 |
| address | VARCHAR(200) | | | 详细地址 |
| is_default | BOOLEAN | NOT NULL | FALSE | 是否默认地点 |
| is_active | BOOLEAN | NOT NULL | TRUE | 是否启用 |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |

**说明：** V1版本仅使用一条默认地点记录，预留多地点扩展。

---

### 3.3 时间表

#### 3.3.1 course_schedules (课表表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| user_id | UUID | NOT NULL | | 用户ID |
| semester_id | UUID | NOT NULL | | 学期ID |
| course_name | VARCHAR(100) | NOT NULL | | 课程名称 |
| day_of_week | SMALLINT | NOT NULL | | 星期几（1-7） |
| start_time | TIME | NOT NULL | | 开始时间 |
| end_time | TIME | NOT NULL | | 结束时间 |
| week_type | VARCHAR(10) | NOT NULL | 'all' | 周类型：all/odd/even |
| weeks | VARCHAR(100) | | | 适用周次（如"1-16"） |
| source | VARCHAR(20) | NOT NULL | 'ics' | 来源：ics |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |

**字段枚举值：**
- `week_type`: all(每周), odd(单周), even(双周)

**索引：**
- `idx_course_schedules_user_semester` (user_id, semester_id)
- `idx_course_schedules_day_time` (day_of_week, start_time, end_time)

---

#### 3.3.2 unavailable_times (不可用时间表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| user_id | UUID | NOT NULL | | 用户ID |
| semester_id | UUID | NOT NULL | | 学期ID |
| day_of_week | SMALLINT | NOT NULL | | 星期几（1-7） |
| start_time | TIME | NOT NULL | | 开始时间 |
| end_time | TIME | NOT NULL | | 结束时间 |
| reason | VARCHAR(200) | | | 原因 |
| repeat_type | VARCHAR(20) | NOT NULL | 'weekly' | 重复类型：once/weekly |
| specific_date | DATE | | NULL | 特定日期（单次时使用） |
| week_type | VARCHAR(10) | NOT NULL | 'all' | 周类型：all/odd/even |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |

**索引：**
- `idx_unavailable_times_user_semester` (user_id, semester_id)
- `idx_unavailable_times_day_time` (day_of_week, start_time, end_time)

---

### 3.4 排班

#### 3.4.1 schedules (排班表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| semester_id | UUID | NOT NULL | | 学期ID |
| status | VARCHAR(20) | NOT NULL | 'draft' | 状态 |
| published_at | TIMESTAMP | | NULL | 发布时间 |
| created_by | UUID | NOT NULL | | 创建人ID |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |

**字段枚举值：**
- `status`: draft(草稿), published(已发布), need_regen(需重新排班), archived(已归档)

**索引：**
- `idx_schedules_semester_id` (semester_id)
- `idx_schedules_status` (status)

---

#### 3.4.2 schedule_items (排班明细表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| schedule_id | UUID | NOT NULL | | 排班表ID |
| week_number | SMALLINT | NOT NULL | | 周次（1或2，表示单/双周模板） |
| day_of_week | SMALLINT | NOT NULL | | 星期几（1-5） |
| time_slot_id | UUID | NOT NULL | | 时间段ID |
| member_id | UUID | NOT NULL | | 值班人员ID |
| location_id | UUID | | NULL | 地点ID（预留） |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |

**索引：**
- `idx_schedule_items_schedule_id` (schedule_id)
- `idx_schedule_items_member_id` (member_id)
- `uk_schedule_items_slot` UNIQUE (schedule_id, week_number, day_of_week, time_slot_id)

---

#### 3.4.3 schedule_change_logs (排班变更记录表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| schedule_id | UUID | NOT NULL | | 排班表ID |
| schedule_item_id | UUID | NOT NULL | | 排班项ID |
| original_member_id | UUID | NOT NULL | | 原值班人员ID |
| new_member_id | UUID | NOT NULL | | 新值班人员ID |
| change_type | VARCHAR(20) | NOT NULL | | 变更类型 |
| reason | VARCHAR(500) | | | 变更原因 |
| operator_id | UUID | NOT NULL | | 操作人ID |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |

**字段枚举值：**
- `change_type`: manual_adjust(手动调整), swap(换班), admin_modify(发布后修改)

**索引：**
- `idx_schedule_change_logs_schedule_id` (schedule_id)
- `idx_schedule_change_logs_created_at` (created_at)

---

### 3.5 换班

#### 3.5.1 swap_requests (换班申请表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| schedule_item_id | UUID | NOT NULL | | 排班项ID |
| applicant_id | UUID | NOT NULL | | 申请人ID |
| target_member_id | UUID | NOT NULL | | 目标成员ID |
| reason | VARCHAR(500) | | | 换班原因 |
| status | VARCHAR(20) | NOT NULL | 'pending' | 状态 |
| target_responded_at | TIMESTAMP | | NULL | 目标成员响应时间 |
| approved_at | TIMESTAMP | | NULL | 审批时间 |
| approved_by | UUID | | NULL | 审批人ID |
| reject_reason | VARCHAR(500) | | | 拒绝/驳回原因 |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |

**字段枚举值：**
- `status`: pending(待同意), reviewing(待审核), completed(已完成), rejected(已拒绝)

**索引：**
- `idx_swap_requests_applicant_id` (applicant_id)
- `idx_swap_requests_target_member_id` (target_member_id)
- `idx_swap_requests_status` (status)
- `idx_swap_requests_schedule_item_id` (schedule_item_id)

---

### 3.6 签到

#### 3.6.1 duty_records (值班记录表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| schedule_item_id | UUID | NOT NULL | | 排班项ID |
| member_id | UUID | NOT NULL | | 值班人员ID |
| duty_date | DATE | NOT NULL | | 值班日期 |
| status | VARCHAR(20) | NOT NULL | 'pending' | 状态 |
| sign_in_time | TIMESTAMP | | NULL | 签到时间 |
| sign_out_time | TIMESTAMP | | NULL | 签退时间 |
| is_late | BOOLEAN | NOT NULL | FALSE | 是否迟到 |
| make_up_time | TIMESTAMP | | NULL | 补签时间 |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 更新时间 |

**字段枚举值：**
- `status`: pending(待值班), on_duty(值班中), completed(已完成), absent(缺席), absent_made_up(缺席已补签), no_sign_out(未签退)

**索引：**
- `idx_duty_records_member_id` (member_id)
- `idx_duty_records_duty_date` (duty_date)
- `idx_duty_records_status` (status)
- `uk_duty_records_schedule_item_date` UNIQUE (schedule_item_id, duty_date)

---

### 3.7 通知

#### 3.7.1 notifications (通知消息表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| user_id | UUID | NOT NULL | | 用户ID |
| type | VARCHAR(50) | NOT NULL | | 通知类型 |
| title | VARCHAR(200) | NOT NULL | | 标题 |
| content | TEXT | NOT NULL | | 内容 |
| is_read | BOOLEAN | NOT NULL | FALSE | 是否已读 |
| related_id | UUID | | NULL | 关联ID（排班/换班等） |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |

**字段枚举值：**
- `type`: schedule_published(排班发布), schedule_changed(排班变更), duty_reminder(值班提醒), swap_request(换班申请), swap_accepted(换班同意), swap_rejected(换班拒绝), swap_approved(换班审核通过), swap_denied(换班审核驳回), absent_alert(缺席提醒), make_up_alert(补签提醒), no_sign_out_alert(未签退提醒)

**索引：**
- `idx_notifications_user_id` (user_id)
- `idx_notifications_is_read` (is_read)
- `idx_notifications_created_at` (created_at DESC)

---

### 3.8 邀请码（可选，也可用Redis）

#### 3.8.1 invite_codes (邀请码表)

| 字段名 | 类型 | 约束 | 默认值 | 说明 |
|--------|------|------|--------|------|
| id | UUID | PRIMARY KEY | gen_random_uuid() | 主键 |
| code | VARCHAR(50) | NOT NULL, UNIQUE | | 邀请码 |
| created_by | UUID | NOT NULL | | 创建人ID |
| expires_at | TIMESTAMP | NOT NULL | | 过期时间 |
| used_at | TIMESTAMP | | NULL | 使用时间 |
| used_by | UUID | | NULL | 使用人ID |
| created_at | TIMESTAMP | NOT NULL | CURRENT_TIMESTAMP | 创建时间 |

**索引：**
- `uk_invite_codes_code` UNIQUE (code)
- `idx_invite_codes_expires_at` (expires_at)

---

## 四、数据关系约束

### 4.1 外键关系（数据库约束）

> 说明：本项目默认在 PostgreSQL 中建立外键（FK）以增强数据一致性。由于采用软删除（`deleted_at`），删除行为建议以 **RESTRICT** 为主，避免级联误删。

**推荐的删除/更新策略：**

- 大多数 FK：`ON UPDATE RESTRICT ON DELETE RESTRICT`
- 可空关联（如 `schedule_items.location_id`）：`ON DELETE SET NULL`
- 业务需要“级联清理”的场景（如学期归档清理）建议通过后台任务显式执行，而非数据库级联删除

| 表 | 字段 | 关联表 | 关联字段 | 说明 |
|----|------|--------|----------|------|
| users | department_id | departments | id | 用户所属部门 |
| course_schedules | user_id | users | id | 课表所属用户 |
| course_schedules | semester_id | semesters | id | 课表所属学期 |
| unavailable_times | user_id | users | id | 不可用时间所属用户 |
| unavailable_times | semester_id | semesters | id | 不可用时间所属学期 |
| schedules | semester_id | semesters | id | 排班表所属学期 |
| schedules | created_by | users | id | 排班表创建人 |
| schedule_items | schedule_id | schedules | id | 排班项所属排班表 |
| schedule_items | time_slot_id | time_slots | id | 排班项对应时间段 |
| schedule_items | member_id | users | id | 排班项值班人员 |
| schedule_items | location_id | locations | id | 排班项值班地点 |
| schedule_change_logs | schedule_id | schedules | id | 变更记录所属排班表 |
| schedule_change_logs | schedule_item_id | schedule_items | id | 变更记录对应排班项 |
| schedule_change_logs | original_member_id | users | id | 原值班人员 |
| schedule_change_logs | new_member_id | users | id | 新值班人员 |
| schedule_change_logs | operator_id | users | id | 操作人 |
| swap_requests | schedule_item_id | schedule_items | id | 换班对应排班项 |
| swap_requests | applicant_id | users | id | 换班申请人 |
| swap_requests | target_member_id | users | id | 换班目标成员 |
| swap_requests | approved_by | users | id | 审批人 |
| duty_records | schedule_item_id | schedule_items | id | 值班记录对应排班项 |
| duty_records | member_id | users | id | 值班人员 |
| notifications | user_id | users | id | 通知接收用户 |
| notification_preferences | user_id | users | id | 通知偏好所属用户 |
| invite_codes | created_by | users | id | 邀请码创建人 |
| invite_codes | used_by | users | id | 邀请码使用人 |

### 4.2 数据完整性规则

| 规则 | 说明 |
|------|------|
| 学号唯一 | 同一时间不存在重复学号的用户 |
| 邮箱唯一 | 同一时间不存在重复邮箱的用户 |
| 部门名称唯一 | 同一时间不存在重复名称的部门 |
| 单一活动学期 | 同一时间最多一个活动学期 |
| 排班时段唯一 | 同一排班表、同一周次、同一天、同一时段只有一条记录 |
| 值班记录唯一 | 同一排班项、同一日期只有一条值班记录 |

---

## 五、索引设计

### 5.1 索引汇总

| 表 | 索引名 | 类型 | 字段 | 说明 |
|----|--------|------|------|------|
| departments | uk_departments_name | UNIQUE | name | 部门名称唯一 |
| users | uk_users_student_id | UNIQUE | student_id | 学号唯一 |
| users | uk_users_email | UNIQUE | email | 邮箱唯一 |
| users | idx_users_department_id | INDEX | department_id | 部门筛选 |
| users | idx_users_role | INDEX | role | 角色筛选 |
| users | idx_users_duty_required | INDEX | duty_required | 值班状态筛选 |
| course_schedules | idx_course_schedules_user_semester | INDEX | user_id, semester_id | 用户学期联合查询 |
| unavailable_times | idx_unavailable_times_user_semester | INDEX | user_id, semester_id | 用户学期联合查询 |
| schedules | idx_schedules_semester_id | INDEX | semester_id | 学期筛选 |
| schedule_items | idx_schedule_items_schedule_id | INDEX | schedule_id | 排班表筛选 |
| schedule_items | idx_schedule_items_member_id | INDEX | member_id | 值班人员筛选 |
| schedule_items | uk_schedule_items_slot | UNIQUE | schedule_id, week_number, day_of_week, time_slot_id | 时段唯一 |
| swap_requests | idx_swap_requests_applicant_id | INDEX | applicant_id | 申请人筛选 |
| swap_requests | idx_swap_requests_target_member_id | INDEX | target_member_id | 目标成员筛选 |
| swap_requests | idx_swap_requests_status | INDEX | status | 状态筛选 |
| duty_records | idx_duty_records_member_id | INDEX | member_id | 值班人员筛选 |
| duty_records | idx_duty_records_duty_date | INDEX | duty_date | 日期筛选 |
| duty_records | idx_duty_records_status | INDEX | status | 状态筛选 |
| notifications | idx_notifications_user_id | INDEX | user_id | 用户筛选 |
| notifications | idx_notifications_created_at | INDEX | created_at DESC | 时间排序 |

---

## 六、数据字典

### 6.1 枚举值定义

#### 用户角色 (users.role)

| 值 | 说明 |
|----|------|
| admin | 排班管理员 |
| leader | 部门负责人 |
| member | 值班成员 |

#### 时间表状态 (users.timetable_status)

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

#### 日类型 (time_slots.day_type)

| 值 | 说明 |
|----|------|
| weekday | 周一至周四 |
| friday | 周五 |

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

#### 变更类型 (schedule_change_logs.change_type)

| 值 | 说明 |
|----|------|
| manual_adjust | 手动调整 |
| swap | 换班 |
| admin_modify | 发布后修改 |

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

---

## 七、数据初始化

### 7.1 系统初始化数据

#### 默认部门

以下为 **初始化示例数据（非 SQL 脚本）**，用于说明建议的默认配置项；实际初始化可通过迁移工具、管理后台或启动脚本完成。

| name | description | sort_order |
|------|-------------|-----------|
| 秘书部 | 负责排班管理 | 1 |
| 宣传部 | 负责宣传工作 | 2 |
| … | … | … |

#### 默认时间段

**周一至周四 (day_type=weekday)：**

| name | start_time | end_time | sort_order |
|------|------------|----------|-----------|
| 第一时段 | 08:10 | 10:05 | 1 |
| 第二时段 | 10:20 | 12:15 | 2 |
| 第三时段 | 14:00 | 16:00 | 3 |
| 第四时段 | 16:10 | 18:00 | 4 |

**周五 (day_type=friday)：**

| name | start_time | end_time | sort_order |
|------|------------|----------|-----------|
| 第一时段 | 08:10 | 10:05 | 1 |
| 第二时段 | 10:20 | 12:15 | 2 |
| 第三时段 | 14:00 | 16:00 | 3 |

#### 排班规则

| rule_code | rule_name | description | is_enabled | is_configurable |
|-----------|-----------|-------------|-----------|-----------------|
| R1 | 课表冲突 | 有课的时段不能排班 | TRUE | FALSE |
| R2 | 不可用时间冲突 | 用户标记的不可用时段不能排班 | TRUE | FALSE |
| R6 | 同人同日不重复 | 同一成员同一天最多安排一个班次 | TRUE | FALSE |
| R3 | 同日部门不重复 | 同一天的不同时段不能有来自同一部门的人 | TRUE | TRUE |
| R4 | 相邻班次部门不重复 | 相邻两个时段不能来自同一部门 | TRUE | TRUE |
| R5 | 单双周早八不重复 | 单周和双周的早八不能是同一人 | TRUE | TRUE |

#### 系统设置

| key | value | description |
|-----|-------|-------------|
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

