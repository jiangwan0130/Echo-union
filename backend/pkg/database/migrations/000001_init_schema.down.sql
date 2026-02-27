-- ============================================================
-- 回滚初始 schema：按依赖关系反向删除所有表
-- ============================================================

BEGIN;

-- 先删除有外键依赖的表
DROP TABLE IF EXISTS schedule_change_logs CASCADE;
DROP TABLE IF EXISTS schedule_member_snapshots CASCADE;
DROP TABLE IF EXISTS schedule_items CASCADE;
DROP TABLE IF EXISTS schedules CASCADE;
DROP TABLE IF EXISTS swap_requests CASCADE;
DROP TABLE IF EXISTS duty_records CASCADE;
DROP TABLE IF EXISTS notifications CASCADE;
DROP TABLE IF EXISTS notification_preferences CASCADE;
DROP TABLE IF EXISTS unavailable_times CASCADE;
DROP TABLE IF EXISTS course_schedules CASCADE;
DROP TABLE IF EXISTS user_semester_assignments CASCADE;
DROP TABLE IF EXISTS invite_codes CASCADE;
DROP TABLE IF EXISTS schedule_rules CASCADE;
DROP TABLE IF EXISTS system_config CASCADE;
DROP TABLE IF EXISTS time_slots CASCADE;
DROP TABLE IF EXISTS locations CASCADE;
DROP TABLE IF EXISTS semesters CASCADE;
DROP TABLE IF EXISTS users CASCADE;
DROP TABLE IF EXISTS departments CASCADE;

-- 删除自定义类型和扩展
DROP TYPE IF EXISTS timerange CASCADE;
DROP EXTENSION IF EXISTS btree_gist;

COMMIT;
