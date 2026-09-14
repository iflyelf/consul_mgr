-- Consul 实例持久化表
CREATE TABLE IF NOT EXISTS consul_instances (
    id BIGSERIAL PRIMARY KEY,
    instance_id VARCHAR(255) NOT NULL,
    service_name VARCHAR(100) NOT NULL,
    group_id BIGINT NOT NULL,
    address VARCHAR(100) NOT NULL,
    port INT NOT NULL,
    tags TEXT[] DEFAULT '{}',
    meta JSONB,
    health_check JSONB,
    status VARCHAR(20) DEFAULT 'passing',
    datacenter VARCHAR(50) DEFAULT 'dc1',
    node_name VARCHAR(100),
    created_by VARCHAR(100),
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    FOREIGN KEY (group_id) REFERENCES service_groups(id) ON DELETE CASCADE,
    UNIQUE(group_id, instance_id)
);

CREATE INDEX IF NOT EXISTS idx_consul_instances_group ON consul_instances(group_id);
CREATE INDEX IF NOT EXISTS idx_consul_instances_service ON consul_instances(service_name);
CREATE INDEX IF NOT EXISTS idx_consul_instances_status ON consul_instances(status);
CREATE INDEX IF NOT EXISTS idx_consul_instances_created ON consul_instances(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_consul_instances_datacenter ON consul_instances(datacenter);
CREATE INDEX IF NOT EXISTS idx_consul_instances_meta ON consul_instances USING GIN (meta);

COMMENT ON TABLE consul_instances IS 'Consul 实例持久化表';
COMMENT ON COLUMN consul_instances.meta IS 'JSON 格式的元数据';
COMMENT ON COLUMN consul_instances.created_by IS 'Casdoor 用户 ID';
