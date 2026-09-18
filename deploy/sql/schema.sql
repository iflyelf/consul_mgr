-- =============================================================================
-- Consul Manager 数据库结构（快照，请勿手工编辑）
--
-- ⚠️ 权威来源：程序启动时按代码内嵌 DDL 自动建表，并自动维护
--    （建库/加列/改类型/清理废弃表列），无需手工执行任何 SQL。
--
-- 重新生成本快照：
--     make schema            # 用 pg_dump 从运行中的数据库导出
--   或
--     pg_dump --schema-only "$DATABASE_URL" > deploy/sql/schema.sql
--
--   认证与用户体系由外部 FlyIAM（内置 Casdoor）负责，本项目不创建用户密码表。
--   Casdoor 自身的表（casdoor_*）由 FlyIAM 管理，不在本文件范围内。
-- =============================================================================

-- ---------------------------------------------------------------------------
-- 服务组：一个服务组对应一个 Consul 集群配置
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS service_groups (
    id                BIGSERIAL PRIMARY KEY,
    name              VARCHAR(100) NOT NULL UNIQUE,
    code              VARCHAR(100) NOT NULL DEFAULT '',
    consul_address    VARCHAR(255) NOT NULL,
    consul_token      VARCHAR(255),
    consul_datacenter VARCHAR(50) DEFAULT 'dc1',
    description       TEXT,
    status            SMALLINT DEFAULT 1,
    created_by        VARCHAR(100),
    created_at        TIMESTAMP DEFAULT NOW(),
    updated_at        TIMESTAMP DEFAULT NOW()
);

-- 兼容旧库：补齐可能缺失的列
ALTER TABLE service_groups ADD COLUMN IF NOT EXISTS code VARCHAR(100) NOT NULL DEFAULT '';
ALTER TABLE service_groups ADD COLUMN IF NOT EXISTS consul_datacenter VARCHAR(50) DEFAULT 'dc1';
ALTER TABLE service_groups ADD COLUMN IF NOT EXISTS status SMALLINT DEFAULT 1;
ALTER TABLE service_groups ADD COLUMN IF NOT EXISTS created_by VARCHAR(100);
ALTER TABLE service_groups ADD COLUMN IF NOT EXISTS datacenter VARCHAR(50);

-- ---------------------------------------------------------------------------
-- 服务组用户授权（user_id 为 Casdoor 用户 ID）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS service_group_users (
    id          BIGSERIAL PRIMARY KEY,
    group_id    BIGINT NOT NULL,
    user_id     VARCHAR(100) NOT NULL,
    permissions TEXT[] DEFAULT '{}',
    created_at  TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (group_id) REFERENCES service_groups(id) ON DELETE CASCADE,
    UNIQUE (group_id, user_id)
);

-- ---------------------------------------------------------------------------
-- 服务组角色授权（role_name 为 Casdoor 角色名）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS service_group_roles (
    id          BIGSERIAL PRIMARY KEY,
    group_id    BIGINT NOT NULL,
    role_name   VARCHAR(100) NOT NULL,
    permissions TEXT[] DEFAULT '{}',
    created_at  TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (group_id) REFERENCES service_groups(id) ON DELETE CASCADE,
    UNIQUE (group_id, role_name)
);

-- ---------------------------------------------------------------------------
-- Consul 实例持久化（用于批量导入导出与本地参考，列表以 Consul 实时数据为准）
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS consul_instances (
    id           BIGSERIAL PRIMARY KEY,
    instance_id  VARCHAR(255) NOT NULL,
    service_name VARCHAR(100) NOT NULL,
    group_id     BIGINT NOT NULL,
    address      VARCHAR(100) NOT NULL,
    port         INT NOT NULL,
    tags         TEXT[] DEFAULT '{}',
    meta         JSONB,
    health_check JSONB,
    status       VARCHAR(20) DEFAULT 'passing',
    datacenter   VARCHAR(50) DEFAULT 'dc1',
    node_name    VARCHAR(100),
    created_by   VARCHAR(100),
    created_at   TIMESTAMP DEFAULT NOW(),
    updated_at   TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (group_id) REFERENCES service_groups(id) ON DELETE CASCADE,
    UNIQUE (group_id, instance_id)
);

-- ---------------------------------------------------------------------------
-- 审计日志
-- ---------------------------------------------------------------------------
CREATE TABLE IF NOT EXISTS audit_logs (
    id            BIGSERIAL PRIMARY KEY,
    user_id       VARCHAR(100),
    username      VARCHAR(100),
    action        VARCHAR(50),
    resource_type VARCHAR(50),
    resource_id   VARCHAR(100),
    resource_name VARCHAR(255),
    group_id      BIGINT,
    details       JSONB,
    ip_address    VARCHAR(50),
    user_agent    TEXT,
    status        VARCHAR(20),
    error_message TEXT,
    created_at    TIMESTAMP DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- 索引
-- ---------------------------------------------------------------------------
CREATE INDEX IF NOT EXISTS idx_service_groups_code             ON service_groups(code);
CREATE INDEX IF NOT EXISTS idx_service_group_users_group       ON service_group_users(group_id);
CREATE INDEX IF NOT EXISTS idx_service_group_users_user        ON service_group_users(user_id);
CREATE INDEX IF NOT EXISTS idx_service_group_roles_group       ON service_group_roles(group_id);
CREATE INDEX IF NOT EXISTS idx_consul_instances_group          ON consul_instances(group_id);
CREATE INDEX IF NOT EXISTS idx_consul_instances_service        ON consul_instances(service_name);
CREATE INDEX IF NOT EXISTS idx_consul_instances_status         ON consul_instances(status);
CREATE INDEX IF NOT EXISTS idx_consul_instances_created        ON consul_instances(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_consul_instances_datacenter     ON consul_instances(datacenter);
CREATE INDEX IF NOT EXISTS idx_consul_instances_meta           ON consul_instances USING GIN (meta);
CREATE INDEX IF NOT EXISTS idx_audit_logs_user                 ON audit_logs(user_id);
CREATE INDEX IF NOT EXISTS idx_audit_logs_created              ON audit_logs(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_audit_logs_group                ON audit_logs(group_id);
