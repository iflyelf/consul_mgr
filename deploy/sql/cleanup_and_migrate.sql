-- ============================================================
-- Consul Manager 数据库清理和 Casdoor 集成迁移脚本
-- 警告：此脚本会删除现有用户、角色、权限表
-- ============================================================

-- 开始事务
BEGIN;

-- ============================================================
-- Phase 1: 删除旧的用户权限相关表
-- ============================================================

DROP TABLE IF EXISTS role_group_permissions CASCADE;
DROP TABLE IF EXISTS role_permissions CASCADE;
DROP TABLE IF EXISTS user_roles CASCADE;
DROP TABLE IF EXISTS permissions CASCADE;
DROP TABLE IF EXISTS roles CASCADE;
DROP TABLE IF EXISTS users CASCADE;

-- ============================================================
-- Phase 2: 调整 service_groups 表
-- ============================================================

-- 删除外键约束
ALTER TABLE service_groups 
    DROP CONSTRAINT IF EXISTS service_groups_created_by_fkey CASCADE;

-- 修改字段类型为字符串（支持 Casdoor 用户 ID）
ALTER TABLE service_groups 
    ALTER COLUMN created_by DROP NOT NULL,
    ALTER COLUMN created_by TYPE VARCHAR(100);

-- 添加注释
COMMENT ON COLUMN service_groups.created_by IS 'Casdoor user ID';

-- ============================================================
-- Phase 3: 调整 audit_logs 表
-- ============================================================

-- 删除外键约束
ALTER TABLE audit_logs 
    DROP CONSTRAINT IF EXISTS audit_logs_user_id_fkey CASCADE,
    DROP CONSTRAINT IF EXISTS audit_logs_group_id_fkey CASCADE;

-- 修改用户 ID 为字符串类型
ALTER TABLE audit_logs 
    ALTER COLUMN user_id DROP NOT NULL,
    ALTER COLUMN user_id TYPE VARCHAR(100);

-- 添加注释
COMMENT ON COLUMN audit_logs.user_id IS 'Casdoor user ID';
COMMENT ON TABLE audit_logs IS 'Consul Manager audit logs';

-- ============================================================
-- Phase 4: 创建新的服务组权限表
-- ============================================================

-- 服务组用户权限表
CREATE TABLE IF NOT EXISTS service_group_users (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    user_id VARCHAR(100) NOT NULL,
    username VARCHAR(100) NOT NULL,
    permissions VARCHAR(50)[] DEFAULT ARRAY['read'],
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(100),
    FOREIGN KEY (group_id) REFERENCES service_groups(id) ON DELETE CASCADE,
    UNIQUE(group_id, user_id)
);

CREATE INDEX idx_service_group_users_group ON service_group_users(group_id);
CREATE INDEX idx_service_group_users_user ON service_group_users(user_id);
CREATE INDEX idx_service_group_users_created ON service_group_users(created_at DESC);

COMMENT ON TABLE service_group_users IS 'Service group user permissions mapping';
COMMENT ON COLUMN service_group_users.user_id IS 'Casdoor user ID';
COMMENT ON COLUMN service_group_users.username IS 'Casdoor username (redundant for query)';
COMMENT ON COLUMN service_group_users.permissions IS 'User permissions: read, write, admin';

-- 服务组角色权限表
CREATE TABLE IF NOT EXISTS service_group_roles (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    role_name VARCHAR(100) NOT NULL,
    permissions VARCHAR(50)[] DEFAULT ARRAY['read'],
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(100),
    FOREIGN KEY (group_id) REFERENCES service_groups(id) ON DELETE CASCADE,
    UNIQUE(group_id, role_name)
);

CREATE INDEX idx_service_group_roles_group ON service_group_roles(group_id);
CREATE INDEX idx_service_group_roles_role ON service_group_roles(role_name);
CREATE INDEX idx_service_group_roles_created ON service_group_roles(created_at DESC);

COMMENT ON TABLE service_group_roles IS 'Service group role permissions mapping';
COMMENT ON COLUMN service_group_roles.role_name IS 'Casdoor role name';
COMMENT ON COLUMN service_group_roles.permissions IS 'Role permissions: read, write, admin';

-- ============================================================
-- Phase 5: 清理 audit_logs 历史数据（可选）
-- ============================================================

-- 清空审计日志（因为旧的 user_id 已经无效）
TRUNCATE TABLE audit_logs;

-- ============================================================
-- 完成
-- ============================================================

COMMIT;

-- 查看最终表结构
SELECT 
    table_name,
    pg_size_pretty(pg_total_relation_size(quote_ident(table_name)::regclass)) as size
FROM information_schema.tables 
WHERE table_schema = 'public' 
AND table_type = 'BASE TABLE'
AND table_name NOT LIKE 'casdoor_%'
ORDER BY table_name;
