-- =============================================================================
-- Consul Manager 初始化数据（可选）
--
-- 说明:
--   应用启动会自动建表，无需手动执行本文件。
--   本文件仅用于「需要预置一个可用服务组」的场景。
--
--   注意:
--     - 认证由 Casdoor 负责，本文件不创建任何用户/角色。
--     - 服务组可完全通过 Web 界面「服务组管理 → 添加服务组」创建。
--     - 下方示例地址/Token 请按实际环境替换，切勿直接用于生产。
-- =============================================================================

-- 预置一个示例 Consul 服务组（存在同 code 时跳过）
INSERT INTO service_groups (name, code, consul_address, consul_token, consul_datacenter, description, status)
VALUES (
    '默认集群',
    'default',
    'http://localhost:8500',
    '',
    'dc1',
    '示例服务组，可通过界面修改或删除',
    1
)
ON CONFLICT (code) DO NOTHING;
