-- ============================================================
-- 学生会值班管理系统 — 数据库初始化脚本  init.sql
-- 数据库: PostgreSQL 15+   |   设计文档版本: v1.9
-- 时区: Asia/Shanghai
-- ============================================================
--
-- ┌─────────────────────────────────────────────────────────┐
-- │  初始化脚本设计决策记录  (2026-02-24)                   │
-- ├─────┬───────────────────────────────────────────────────┤
-- │ Q1  │ B — 不插入任何默认部门数据，由运行时管理后台创建  │
-- │ Q2  │ A — 不创建初始管理员，由部署脚本/管理后台处理       │
-- │ Q3  │ A — 不创建 deleted_at IS NOT NULL 回收站索引，     │
-- │     │     后续按需 CREATE INDEX CONCURRENTLY 添加        │
-- │ Q4  │ A — 创建 §5.2.2 两个待评估索引（宽索引策略起步）  │
-- │     │     idx_user_semester_assignments_semester_id      │
-- │     │     idx_notifications_created_at                  │
-- │ Q5  │ A — 仅保留会话级 SET timezone，不含 CREATE DATABASE│
-- └─────┴───────────────────────────────────────────────────┘
--

BEGIN;

-- ============================================================
-- 0. 扩展 & 自定义类型
-- ============================================================

-- btree_gist: 用于 EXCLUDE 约束防止日期/时间范围重叠
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- 自定义 TIME 范围类型（PostgreSQL 无内建 time range）
-- 用于 time_slots EXCLUDE 约束的时间段重叠检测
CREATE TYPE timerange AS RANGE (subtype = time);

-- 会话时区（ALTER DATABASE 级别设置见待确认问题）
SET timezone = 'Asia/Shanghai';

-- ============================================================
-- 1. departments（部门表）
--    审计 FK (created_by/updated_by/deleted_by → users) 延后添加
--    原因：与 users 存在循环引用
-- ============================================================

CREATE TABLE departments (
    department_id  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    name           VARCHAR(50)  NOT NULL,
    description    TEXT,
    is_active      BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at     TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by     UUID,
    updated_at     TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by     UUID,
    deleted_at     TIMESTAMPTZ,
    deleted_by     UUID,
    version        INT          NOT NULL DEFAULT 1,

    CONSTRAINT ck_departments_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL))
);

CREATE UNIQUE INDEX uk_departments_name
    ON departments (name) WHERE deleted_at IS NULL;

-- ============================================================
-- 2. users（用户表）
-- ============================================================

CREATE TABLE users (
    user_id              UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name                 VARCHAR(100)  NOT NULL,
    student_id           VARCHAR(20)   NOT NULL,
    email                VARCHAR(255)  NOT NULL,
    password_hash        VARCHAR(255)  NOT NULL,
    role                 VARCHAR(20)   NOT NULL DEFAULT 'member',
    department_id        UUID          NOT NULL,
    must_change_password BOOLEAN       NOT NULL DEFAULT FALSE,
    created_at           TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by           UUID,
    updated_at           TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by           UUID,
    deleted_at           TIMESTAMPTZ,
    deleted_by           UUID,
    version              INT           NOT NULL DEFAULT 1,

    CONSTRAINT ck_users_role
        CHECK (role IN ('admin', 'leader', 'member')),
    CONSTRAINT ck_users_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    CONSTRAINT fk_users_department
        FOREIGN KEY (department_id) REFERENCES departments(department_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    -- 自引用审计 FK（PG 默认 NO ACTION）
    CONSTRAINT fk_users_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_users_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_users_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE UNIQUE INDEX uk_users_student_id
    ON users (student_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uk_users_email
    ON users (email) WHERE deleted_at IS NULL;
CREATE INDEX idx_users_department_id
    ON users (department_id) WHERE deleted_at IS NULL;

-- ============================================================
-- 3. 回填 departments 审计 FK（解决循环引用）
-- ============================================================

ALTER TABLE departments
    ADD CONSTRAINT fk_departments_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    ADD CONSTRAINT fk_departments_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    ADD CONSTRAINT fk_departments_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id);

-- ============================================================
-- 4. semesters（学期表）
-- ============================================================

CREATE TABLE semesters (
    semester_id     UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(100)  NOT NULL,
    start_date      DATE          NOT NULL,
    end_date        DATE          NOT NULL,
    first_week_type VARCHAR(10)   NOT NULL,
    is_active       BOOLEAN       NOT NULL DEFAULT FALSE,
    status          VARCHAR(20)   NOT NULL DEFAULT 'active',
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by      UUID,
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by      UUID,
    deleted_at      TIMESTAMPTZ,
    deleted_by      UUID,
    version         INT           NOT NULL DEFAULT 1,

    CONSTRAINT ck_semesters_first_week_type
        CHECK (first_week_type IN ('odd', 'even')),
    CONSTRAINT ck_semesters_status
        CHECK (status IN ('active', 'archived')),
    CONSTRAINT ck_semesters_dates
        CHECK (end_date > start_date),
    CONSTRAINT ck_semesters_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    -- 防止学期日期范围重叠（需 btree_gist）
    CONSTRAINT excl_semesters_date_overlap
        EXCLUDE USING gist (
            daterange(start_date, end_date, '[]') WITH &&
        ) WHERE (deleted_at IS NULL),

    CONSTRAINT fk_semesters_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_semesters_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_semesters_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

-- 保证同一时间只有一个活动学期
CREATE UNIQUE INDEX uk_semesters_active
    ON semesters (is_active)
    WHERE is_active = TRUE AND deleted_at IS NULL;

-- ============================================================
-- 5. notification_preferences（通知偏好表）
--    与 users 1:1 绑定，不单独软删除
-- ============================================================

CREATE TABLE notification_preferences (
    user_id             UUID         PRIMARY KEY,
    schedule_published  BOOLEAN      NOT NULL DEFAULT TRUE,
    duty_reminder       BOOLEAN      NOT NULL DEFAULT TRUE,
    swap_notification   BOOLEAN      NOT NULL DEFAULT TRUE,
    absent_notification BOOLEAN      NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by          UUID,
    updated_at          TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by          UUID,

    CONSTRAINT fk_notif_pref_user
        FOREIGN KEY (user_id) REFERENCES users(user_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_notif_pref_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_notif_pref_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id)
);

-- ============================================================
-- 6. time_slots（时间段配置表）
-- ============================================================

CREATE TABLE time_slots (
    time_slot_id UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name         VARCHAR(50) NOT NULL,
    semester_id  UUID,
    start_time   TIME        NOT NULL,
    end_time     TIME        NOT NULL,
    day_of_week  SMALLINT    NOT NULL,
    is_active    BOOLEAN     NOT NULL DEFAULT TRUE,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by   UUID,
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by   UUID,
    deleted_at   TIMESTAMPTZ,
    deleted_by   UUID,
    version      INT         NOT NULL DEFAULT 1,

    CONSTRAINT ck_time_slots_day_of_week
        CHECK (day_of_week BETWEEN 1 AND 5),
    CONSTRAINT ck_time_slots_times
        CHECK (end_time > start_time),
    CONSTRAINT ck_time_slots_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    -- 同一学期（或全局默认）同一 day_of_week 下时段不可重叠
    CONSTRAINT excl_time_slots_overlap
        EXCLUDE USING gist (
            (COALESCE(semester_id::text, '__GLOBAL__')) WITH =,
            day_of_week WITH =,
            timerange(start_time, end_time) WITH &&
        ) WHERE (deleted_at IS NULL),

    CONSTRAINT fk_time_slots_semester
        FOREIGN KEY (semester_id) REFERENCES semesters(semester_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_time_slots_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_time_slots_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_time_slots_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE INDEX idx_time_slots_day_of_week
    ON time_slots (day_of_week) WHERE deleted_at IS NULL;
CREATE INDEX idx_time_slots_semester_id
    ON time_slots (semester_id)
    WHERE deleted_at IS NULL AND semester_id IS NOT NULL;

-- ============================================================
-- 7. locations（值班地点表 — 预留扩展）
-- ============================================================

CREATE TABLE locations (
    location_id UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100)  NOT NULL,
    address     VARCHAR(200),
    is_default  BOOLEAN       NOT NULL DEFAULT FALSE,
    is_active   BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by  UUID,
    updated_at  TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by  UUID,
    deleted_at  TIMESTAMPTZ,
    deleted_by  UUID,

    CONSTRAINT ck_locations_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    CONSTRAINT fk_locations_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_locations_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_locations_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE UNIQUE INDEX uk_locations_default
    ON locations (is_default)
    WHERE is_default = TRUE AND deleted_at IS NULL;

-- ============================================================
-- 8. schedule_rules（排班规则配置表）
-- ============================================================

CREATE TABLE schedule_rules (
    rule_id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    rule_code       VARCHAR(20)   NOT NULL,
    rule_name       VARCHAR(100)  NOT NULL,
    description     VARCHAR(500),
    is_enabled      BOOLEAN       NOT NULL DEFAULT TRUE,
    is_configurable BOOLEAN       NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by      UUID,
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by      UUID,
    deleted_at      TIMESTAMPTZ,
    deleted_by      UUID,
    version         INT           NOT NULL DEFAULT 1,

    CONSTRAINT ck_schedule_rules_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    CONSTRAINT fk_schedule_rules_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_schedule_rules_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_schedule_rules_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE UNIQUE INDEX uk_schedule_rules_code
    ON schedule_rules (rule_code) WHERE deleted_at IS NULL;

-- ============================================================
-- 9. system_config（系统配置表 — 单行强类型）
-- ============================================================

CREATE TABLE system_config (
    singleton               BOOLEAN      PRIMARY KEY DEFAULT TRUE,
    swap_deadline_hours     INT          NOT NULL DEFAULT 24,
    duty_reminder_time      TIME         NOT NULL DEFAULT '09:00',
    default_location        VARCHAR(200) NOT NULL DEFAULT '学生会办公室',
    sign_in_window_minutes  INT          NOT NULL DEFAULT 15,
    sign_out_window_minutes INT          NOT NULL DEFAULT 15,
    created_at              TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by              UUID,
    updated_at              TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by              UUID,

    CONSTRAINT ck_system_config_singleton
        CHECK (singleton = TRUE),
    CONSTRAINT ck_system_config_swap_deadline
        CHECK (swap_deadline_hours > 0),
    CONSTRAINT ck_system_config_sign_in_window
        CHECK (sign_in_window_minutes > 0),
    CONSTRAINT ck_system_config_sign_out_window
        CHECK (sign_out_window_minutes > 0),

    CONSTRAINT fk_system_config_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_system_config_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id)
);

-- ============================================================
-- 10. user_semester_assignments（用户-学期分配表）
-- ============================================================

CREATE TABLE user_semester_assignments (
    assignment_id          UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id                UUID         NOT NULL,
    semester_id            UUID         NOT NULL,
    duty_required          BOOLEAN      NOT NULL DEFAULT FALSE,
    timetable_status       VARCHAR(20)  NOT NULL DEFAULT 'not_submitted',
    timetable_submitted_at TIMESTAMPTZ,
    created_at             TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by             UUID,
    updated_at             TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by             UUID,
    deleted_at             TIMESTAMPTZ,
    deleted_by             UUID,
    version                INT          NOT NULL DEFAULT 1,

    CONSTRAINT ck_usa_timetable_status
        CHECK (timetable_status IN ('not_submitted', 'submitted')),
    CONSTRAINT ck_usa_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    CONSTRAINT fk_usa_user
        FOREIGN KEY (user_id) REFERENCES users(user_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_usa_semester
        FOREIGN KEY (semester_id) REFERENCES semesters(semester_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_usa_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_usa_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_usa_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE UNIQUE INDEX uk_user_semester_assignments_user_semester
    ON user_semester_assignments (user_id, semester_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_user_semester_assignments_semester_id
    ON user_semester_assignments (semester_id)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_user_semester_assignments_duty_required
    ON user_semester_assignments (semester_id, duty_required)
    WHERE deleted_at IS NULL;

-- ============================================================
-- 11. course_schedules（课表表）
-- ============================================================

CREATE TABLE course_schedules (
    course_schedule_id UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id            UUID          NOT NULL,
    semester_id        UUID          NOT NULL,
    course_name        VARCHAR(100)  NOT NULL,
    day_of_week        SMALLINT      NOT NULL,
    start_time         TIME          NOT NULL,
    end_time           TIME          NOT NULL,
    week_type          VARCHAR(10)   NOT NULL DEFAULT 'all',
    weeks              INT[],
    source             VARCHAR(20)   NOT NULL DEFAULT 'ics',
    created_at         TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by         UUID,
    updated_at         TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by         UUID,
    deleted_at         TIMESTAMPTZ,
    deleted_by         UUID,
    version            INT           NOT NULL DEFAULT 1,

    CONSTRAINT ck_course_schedules_week_type
        CHECK (week_type IN ('all', 'odd', 'even')),
    CONSTRAINT ck_course_schedules_source
        CHECK (source IN ('ics', 'manual')),
    CONSTRAINT ck_course_schedules_day_of_week
        CHECK (day_of_week BETWEEN 1 AND 7),
    CONSTRAINT ck_course_schedules_times
        CHECK (end_time > start_time),
    CONSTRAINT ck_course_schedules_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    CONSTRAINT fk_course_schedules_user
        FOREIGN KEY (user_id) REFERENCES users(user_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_course_schedules_semester
        FOREIGN KEY (semester_id) REFERENCES semesters(semester_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_course_schedules_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_course_schedules_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_course_schedules_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE INDEX idx_course_schedules_user_day_time
    ON course_schedules (user_id, semester_id, day_of_week, start_time, end_time)
    WHERE deleted_at IS NULL;
CREATE INDEX idx_course_schedules_weeks
    ON course_schedules USING gin (weeks)
    WHERE deleted_at IS NULL;

-- ============================================================
-- 12. unavailable_times（不可用时间表）
-- ============================================================

CREATE TABLE unavailable_times (
    unavailable_time_id UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             UUID          NOT NULL,
    semester_id         UUID          NOT NULL,
    day_of_week         SMALLINT      NOT NULL,
    start_time          TIME          NOT NULL,
    end_time            TIME          NOT NULL,
    reason              VARCHAR(200),
    repeat_type         VARCHAR(20)   NOT NULL DEFAULT 'weekly',
    specific_date       DATE,
    week_type           VARCHAR(10)   NOT NULL DEFAULT 'all',
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by          UUID,
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by          UUID,
    deleted_at          TIMESTAMPTZ,
    deleted_by          UUID,
    version             INT           NOT NULL DEFAULT 1,

    CONSTRAINT ck_unavailable_times_repeat_type
        CHECK (repeat_type IN ('once', 'weekly', 'biweekly')),
    CONSTRAINT ck_unavailable_times_week_type
        CHECK (week_type IN ('all', 'odd', 'even')),
    CONSTRAINT ck_unavailable_times_day_of_week
        CHECK (day_of_week BETWEEN 1 AND 7),
    CONSTRAINT ck_unavailable_times_times
        CHECK (end_time > start_time),
    -- 单次必须指定日期，每周/双周重复不应指定日期
    CONSTRAINT ck_unavailable_times_specific_date
        CHECK ((repeat_type = 'once' AND specific_date IS NOT NULL)
            OR (repeat_type IN ('weekly', 'biweekly') AND specific_date IS NULL)),
    -- 单次事件无需区分单双周，双周必须指定 odd/even
    CONSTRAINT ck_unavailable_times_once_week_type
        CHECK (repeat_type = 'weekly' OR repeat_type = 'biweekly' OR week_type = 'all'),
    CONSTRAINT ck_unavailable_times_biweekly_week_type
        CHECK (repeat_type != 'biweekly' OR week_type IN ('odd', 'even')),
    CONSTRAINT ck_unavailable_times_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    CONSTRAINT fk_unavailable_times_user
        FOREIGN KEY (user_id) REFERENCES users(user_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_unavailable_times_semester
        FOREIGN KEY (semester_id) REFERENCES semesters(semester_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_unavailable_times_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_unavailable_times_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_unavailable_times_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE INDEX idx_unavailable_times_user_day_time
    ON unavailable_times (user_id, semester_id, day_of_week, start_time, end_time)
    WHERE deleted_at IS NULL;

-- ============================================================
-- 13. schedules（排班表）
-- ============================================================

CREATE TABLE schedules (
    schedule_id  UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    semester_id  UUID         NOT NULL,
    status       VARCHAR(20)  NOT NULL DEFAULT 'draft',
    published_at TIMESTAMPTZ,
    created_by   UUID         NOT NULL,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by   UUID,
    deleted_at   TIMESTAMPTZ,
    deleted_by   UUID,
    version      INT          NOT NULL DEFAULT 1,

    CONSTRAINT ck_schedules_status
        CHECK (status IN ('draft', 'published', 'need_regen', 'archived')),
    CONSTRAINT ck_schedules_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    CONSTRAINT fk_schedules_semester
        FOREIGN KEY (semester_id) REFERENCES semesters(semester_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_schedules_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_schedules_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_schedules_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE INDEX idx_schedules_semester_id
    ON schedules (semester_id) WHERE deleted_at IS NULL;

-- 同一学期只允许一个活跃排班表（非归档状态唯一约束）
CREATE UNIQUE INDEX uk_schedules_active_per_semester
    ON schedules (semester_id) WHERE status IN ('draft', 'published', 'need_regen') AND deleted_at IS NULL;

-- ============================================================
-- 14. schedule_member_snapshots（排班成员快照表）
--     依附 schedules 生命周期，不单独软删除
-- ============================================================

CREATE TABLE schedule_member_snapshots (
    snapshot_id   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id   UUID         NOT NULL,
    user_id       UUID         NOT NULL,
    department_id UUID         NOT NULL,
    snapshot_at   TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_sms_schedule
        FOREIGN KEY (schedule_id) REFERENCES schedules(schedule_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_sms_user
        FOREIGN KEY (user_id) REFERENCES users(user_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_sms_department
        FOREIGN KEY (department_id) REFERENCES departments(department_id)
        ON DELETE RESTRICT ON UPDATE CASCADE
);

CREATE UNIQUE INDEX uk_schedule_member_snapshots_schedule_user
    ON schedule_member_snapshots (schedule_id, user_id);

-- ============================================================
-- 15. schedule_items（排班明细表）
-- ============================================================

CREATE TABLE schedule_items (
    schedule_item_id UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id      UUID         NOT NULL,
    week_number      SMALLINT     NOT NULL,
    time_slot_id     UUID         NOT NULL,
    member_id        UUID         NOT NULL,
    location_id      UUID,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by       UUID,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    deleted_by       UUID,
    version          INT          NOT NULL DEFAULT 1,

    CONSTRAINT ck_schedule_items_week_number
        CHECK (week_number IN (1, 2)),
    CONSTRAINT ck_schedule_items_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    CONSTRAINT fk_schedule_items_schedule
        FOREIGN KEY (schedule_id) REFERENCES schedules(schedule_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_schedule_items_time_slot
        FOREIGN KEY (time_slot_id) REFERENCES time_slots(time_slot_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_schedule_items_member
        FOREIGN KEY (member_id) REFERENCES users(user_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_schedule_items_location
        FOREIGN KEY (location_id) REFERENCES locations(location_id)
        ON DELETE SET NULL ON UPDATE CASCADE,
    CONSTRAINT fk_schedule_items_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_schedule_items_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_schedule_items_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE INDEX idx_schedule_items_schedule_id
    ON schedule_items (schedule_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_schedule_items_member_id
    ON schedule_items (member_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_schedule_items_time_slot_id
    ON schedule_items (time_slot_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_schedule_items_member_schedule
    ON schedule_items (member_id, schedule_id) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uk_schedule_items_slot
    ON schedule_items (schedule_id, week_number, time_slot_id)
    WHERE deleted_at IS NULL;

-- ============================================================
-- 16. schedule_change_logs（排班变更记录表 — 纯审计日志，只追加不删除）
-- ============================================================

CREATE TABLE schedule_change_logs (
    change_log_id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_id           UUID          NOT NULL,
    schedule_item_id      UUID          NOT NULL,
    original_member_id    UUID          NOT NULL,
    new_member_id         UUID          NOT NULL,
    original_time_slot_id UUID,
    new_time_slot_id      UUID,
    change_type           VARCHAR(20)   NOT NULL,
    reason                VARCHAR(500),
    operator_id           UUID          NOT NULL,
    created_at            TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT ck_scl_change_type
        CHECK (change_type IN ('manual_adjust', 'swap', 'admin_modify')),
    -- 时段变更字段成对出现
    CONSTRAINT ck_scl_time_slot_pair
        CHECK ((original_time_slot_id IS NULL AND new_time_slot_id IS NULL)
            OR (original_time_slot_id IS NOT NULL AND new_time_slot_id IS NOT NULL)),

    CONSTRAINT fk_scl_schedule
        FOREIGN KEY (schedule_id) REFERENCES schedules(schedule_id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_scl_schedule_item
        FOREIGN KEY (schedule_item_id) REFERENCES schedule_items(schedule_item_id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_scl_original_member
        FOREIGN KEY (original_member_id) REFERENCES users(user_id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_scl_new_member
        FOREIGN KEY (new_member_id) REFERENCES users(user_id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_scl_original_time_slot
        FOREIGN KEY (original_time_slot_id) REFERENCES time_slots(time_slot_id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_scl_new_time_slot
        FOREIGN KEY (new_time_slot_id) REFERENCES time_slots(time_slot_id)
        ON UPDATE CASCADE ON DELETE RESTRICT,
    CONSTRAINT fk_scl_operator
        FOREIGN KEY (operator_id) REFERENCES users(user_id)
        ON UPDATE CASCADE ON DELETE RESTRICT
);

CREATE INDEX idx_schedule_change_logs_schedule_id
    ON schedule_change_logs (schedule_id);
CREATE INDEX idx_schedule_change_logs_created_at
    ON schedule_change_logs (created_at DESC);

-- ============================================================
-- 17. swap_requests（换班申请表）
-- ============================================================

CREATE TABLE swap_requests (
    swap_request_id     UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_item_id    UUID          NOT NULL,
    applicant_id        UUID          NOT NULL,
    target_member_id    UUID          NOT NULL,
    reason              VARCHAR(500),
    status              VARCHAR(20)   NOT NULL DEFAULT 'pending',
    target_responded_at TIMESTAMPTZ,
    approved_at         TIMESTAMPTZ,
    approved_by         UUID,
    reject_reason       VARCHAR(500),
    created_at          TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by          UUID,
    updated_at          TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by          UUID,
    deleted_at          TIMESTAMPTZ,
    deleted_by          UUID,
    version             INT           NOT NULL DEFAULT 1,

    CONSTRAINT ck_swap_requests_status
        CHECK (status IN ('pending', 'reviewing', 'completed', 'rejected', 'cancelled')),
    CONSTRAINT ck_swap_requests_self_swap
        CHECK (applicant_id != target_member_id),
    CONSTRAINT ck_swap_requests_approved_at
        CHECK (approved_at IS NULL OR approved_at >= created_at),
    CONSTRAINT ck_swap_requests_target_responded_at
        CHECK (target_responded_at IS NULL OR target_responded_at >= created_at),
    CONSTRAINT ck_swap_requests_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    CONSTRAINT fk_swap_requests_schedule_item
        FOREIGN KEY (schedule_item_id) REFERENCES schedule_items(schedule_item_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_swap_requests_applicant
        FOREIGN KEY (applicant_id) REFERENCES users(user_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_swap_requests_target_member
        FOREIGN KEY (target_member_id) REFERENCES users(user_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_swap_requests_approved_by
        FOREIGN KEY (approved_by) REFERENCES users(user_id),
    CONSTRAINT fk_swap_requests_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_swap_requests_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_swap_requests_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE INDEX idx_swap_requests_applicant_id
    ON swap_requests (applicant_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_swap_requests_schedule_item_id
    ON swap_requests (schedule_item_id) WHERE deleted_at IS NULL;
CREATE INDEX idx_swap_requests_target_status
    ON swap_requests (target_member_id, status) WHERE deleted_at IS NULL;

-- ============================================================
-- 18. duty_records（值班记录表）
-- ============================================================

CREATE TABLE duty_records (
    duty_record_id   UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    schedule_item_id UUID         NOT NULL,
    member_id        UUID         NOT NULL,
    duty_date        DATE         NOT NULL,
    status           VARCHAR(20)  NOT NULL DEFAULT 'pending',
    sign_in_time     TIMESTAMPTZ,
    sign_out_time    TIMESTAMPTZ,
    is_late          BOOLEAN      NOT NULL DEFAULT FALSE,
    make_up_time     TIMESTAMPTZ,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by       UUID,
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by       UUID,
    deleted_at       TIMESTAMPTZ,
    deleted_by       UUID,
    version          INT          NOT NULL DEFAULT 1,

    CONSTRAINT ck_duty_records_status
        CHECK (status IN ('pending', 'on_duty', 'completed', 'absent', 'absent_made_up', 'no_sign_out')),
    CONSTRAINT ck_duty_records_sign_out
        CHECK (sign_out_time IS NULL OR sign_out_time > sign_in_time),
    CONSTRAINT ck_duty_records_make_up
        CHECK (make_up_time IS NULL OR status = 'absent_made_up'),
    CONSTRAINT ck_duty_records_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    CONSTRAINT fk_duty_records_schedule_item
        FOREIGN KEY (schedule_item_id) REFERENCES schedule_items(schedule_item_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_duty_records_member
        FOREIGN KEY (member_id) REFERENCES users(user_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_duty_records_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_duty_records_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_duty_records_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE INDEX idx_duty_records_date_status
    ON duty_records (duty_date, status) WHERE deleted_at IS NULL;
CREATE INDEX idx_duty_records_member_date_status
    ON duty_records (member_id, duty_date, status) WHERE deleted_at IS NULL;
CREATE UNIQUE INDEX uk_duty_records_schedule_item_date
    ON duty_records (schedule_item_id, duty_date) WHERE deleted_at IS NULL;

-- ============================================================
-- 19. notifications（通知消息表）
-- ============================================================

CREATE TABLE notifications (
    notification_id UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID          NOT NULL,
    type            VARCHAR(50)   NOT NULL,
    title           VARCHAR(200)  NOT NULL,
    content         TEXT          NOT NULL,
    is_read         BOOLEAN       NOT NULL DEFAULT FALSE,
    related_type    VARCHAR(20),
    related_id      UUID,
    created_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by      UUID,
    updated_at      TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by      UUID,
    deleted_at      TIMESTAMPTZ,
    deleted_by      UUID,

    CONSTRAINT ck_notifications_type
        CHECK (type IN (
            'schedule_published', 'schedule_changed', 'duty_reminder',
            'swap_request', 'swap_accepted', 'swap_rejected',
            'swap_approved', 'swap_denied',
            'absent_alert', 'make_up_alert', 'no_sign_out_alert'
        )),
    CONSTRAINT ck_notifications_related_type
        CHECK (related_type IS NULL
            OR related_type IN ('schedule', 'schedule_item', 'swap_request', 'duty_record')),
    CONSTRAINT ck_notifications_related_pair
        CHECK ((related_type IS NULL AND related_id IS NULL)
            OR (related_type IS NOT NULL AND related_id IS NOT NULL)),
    CONSTRAINT ck_notifications_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    CONSTRAINT fk_notifications_user
        FOREIGN KEY (user_id) REFERENCES users(user_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_notifications_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id),
    CONSTRAINT fk_notifications_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_notifications_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE INDEX idx_notifications_user_unread
    ON notifications (user_id, created_at DESC)
    WHERE is_read = FALSE AND deleted_at IS NULL;
CREATE INDEX idx_notifications_user_read
    ON notifications (user_id, is_read) WHERE deleted_at IS NULL;
CREATE INDEX idx_notifications_related
    ON notifications (related_type, related_id)
    WHERE deleted_at IS NULL AND related_type IS NOT NULL;
CREATE INDEX idx_notifications_created_at
    ON notifications (created_at DESC)
    WHERE deleted_at IS NULL;

-- ============================================================
-- 20. invite_codes（邀请码表）
-- ============================================================

CREATE TABLE invite_codes (
    invite_code_id UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    code           VARCHAR(50)   NOT NULL,
    created_by     UUID          NOT NULL,
    expires_at     TIMESTAMPTZ   NOT NULL,
    used_at        TIMESTAMPTZ,
    used_by        UUID,
    created_at     TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMPTZ   NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_by     UUID,
    deleted_at     TIMESTAMPTZ,
    deleted_by     UUID,
    version        INT           NOT NULL DEFAULT 1,

    CONSTRAINT ck_invite_codes_expires
        CHECK (expires_at > created_at),
    -- 过期码不允许使用
    CONSTRAINT ck_invite_codes_used
        CHECK (used_at IS NULL OR used_at <= expires_at),
    CONSTRAINT ck_invite_codes_soft_delete
        CHECK ((deleted_at IS NULL AND deleted_by IS NULL)
            OR (deleted_at IS NOT NULL AND deleted_by IS NOT NULL)),

    CONSTRAINT fk_invite_codes_created_by
        FOREIGN KEY (created_by) REFERENCES users(user_id)
        ON DELETE RESTRICT ON UPDATE CASCADE,
    CONSTRAINT fk_invite_codes_used_by
        FOREIGN KEY (used_by) REFERENCES users(user_id),
    CONSTRAINT fk_invite_codes_updated_by
        FOREIGN KEY (updated_by) REFERENCES users(user_id),
    CONSTRAINT fk_invite_codes_deleted_by
        FOREIGN KEY (deleted_by) REFERENCES users(user_id)
);

CREATE UNIQUE INDEX uk_invite_codes_code
    ON invite_codes (code) WHERE deleted_at IS NULL;
CREATE INDEX idx_invite_codes_expires_at
    ON invite_codes (expires_at) WHERE deleted_at IS NULL;

-- ============================================================
-- 21. 种子数据（文档明确定义的部分）
-- ============================================================

-- 21.1 系统配置（单行，使用列默认值）
INSERT INTO system_config (singleton) VALUES (TRUE);

-- 21.2 默认地点
INSERT INTO locations (name, address, is_default)
VALUES ('学生会办公室', '学生活动中心201', TRUE);

-- 21.3 排班规则（预置6条）
INSERT INTO schedule_rules (rule_code, rule_name, description, is_enabled, is_configurable) VALUES
    ('R1', '课表冲突',         '有课的时段不能排班',                          TRUE, FALSE),
    ('R2', '不可用时间冲突',   '用户标记的不可用时段不能排班',                TRUE, FALSE),
    ('R6', '同人同日不重复',   '同一成员同一天最多安排一个班次',              TRUE, FALSE),
    ('R3', '同日部门不重复',   '同一天的不同时段不能有来自同一部门的人',      TRUE, TRUE),
    ('R4', '相邻班次部门不重复', '相邻两个时段不能来自同一部门',              TRUE, TRUE),
    ('R5', '单双周早八不重复', '单周和双周的早八不能是同一人',                TRUE, TRUE);

-- 21.4 默认时间段（全局默认，semester_id = NULL）
-- 周一至周四 (day_of_week=1~4) 每天4个时段
-- 周五 (day_of_week=5) 3个时段
INSERT INTO time_slots (name, semester_id, day_of_week, start_time, end_time) VALUES
    -- 周一
    ('第一时段', NULL, 1, '08:10', '10:05'),
    ('第二时段', NULL, 1, '10:20', '12:15'),
    ('第三时段', NULL, 1, '14:00', '16:00'),
    ('第四时段', NULL, 1, '16:10', '18:00'),
    -- 周二
    ('第一时段', NULL, 2, '08:10', '10:05'),
    ('第二时段', NULL, 2, '10:20', '12:15'),
    ('第三时段', NULL, 2, '14:00', '16:00'),
    ('第四时段', NULL, 2, '16:10', '18:00'),
    -- 周三
    ('第一时段', NULL, 3, '08:10', '10:05'),
    ('第二时段', NULL, 3, '10:20', '12:15'),
    ('第三时段', NULL, 3, '14:00', '16:00'),
    ('第四时段', NULL, 3, '16:10', '18:00'),
    -- 周四
    ('第一时段', NULL, 4, '08:10', '10:05'),
    ('第二时段', NULL, 4, '10:20', '12:15'),
    ('第三时段', NULL, 4, '14:00', '16:00'),
    ('第四时段', NULL, 4, '16:10', '18:00'),
    -- 周五
    ('第一时段', NULL, 5, '08:10', '10:05'),
    ('第二时段', NULL, 5, '10:20', '12:15'),
    ('第三时段', NULL, 5, '14:00', '16:00');

COMMIT;
