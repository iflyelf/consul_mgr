-- ============================================================
-- 初始权限数据
-- ============================================================

-- 认证权限
INSERT INTO permissions (name, code, resource, action, description) VALUES
('登录', 'auth:login', '/api/auth/login', 'POST', '用户登录'),
('登出', 'auth:logout', '/api/auth/logout', 'POST', '用户登出'),
('刷新Token', 'auth:refresh', '/api/auth/refresh', 'POST', '刷新访问令牌'),
('获取用户信息', 'auth:info', '/api/auth/info', 'GET', '获取当前用户信息')
ON CONFLICT (code) DO NOTHING;

-- 用户管理权限
INSERT INTO permissions (name, code, resource, action, description) VALUES
('查看用户列表', 'user:list', '/api/users', 'GET', '查看所有用户'),
('查看用户详情', 'user:read', '/api/users/*', 'GET', '查看用户详细信息'),
('创建用户', 'user:create', '/api/users', 'POST', '创建新用户'),
('更新用户', 'user:update', '/api/users/*', 'PUT', '更新用户信息'),
('删除用户', 'user:delete', '/api/users/*', 'DELETE', '删除用户'),
('分配角色', 'user:role', '/api/users/*/roles', 'POST', '为用户分配角色'),
('修改密码', 'user:password', '/api/users/*/password', 'PUT', '修改用户密码')
ON CONFLICT (code) DO NOTHING;

-- 角色管理权限
INSERT INTO permissions (name, code, resource, action, description) VALUES
('查看角色列表', 'role:list', '/api/roles', 'GET', '查看所有角色'),
('查看角色详情', 'role:read', '/api/roles/*', 'GET', '查看角色详细信息'),
('创建角色', 'role:create', '/api/roles', 'POST', '创建新角色'),
('更新角色', 'role:update', '/api/roles/*', 'PUT', '更新角色信息'),
('删除角色', 'role:delete', '/api/roles/*', 'DELETE', '删除角色'),
('分配权限', 'role:permission', '/api/roles/*/permissions', 'POST', '为角色分配权限'),
('授权服务组', 'role:group', '/api/roles/*/groups', 'POST', '授权角色访问服务组')
ON CONFLICT (code) DO NOTHING;

-- 权限查询
INSERT INTO permissions (name, code, resource, action, description) VALUES
('查看权限列表', 'permission:list', '/api/permissions', 'GET', '查看所有权限')
ON CONFLICT (code) DO NOTHING;

-- 服务组管理权限
INSERT INTO permissions (name, code, resource, action, description) VALUES
('查看服务组列表', 'group:list', '/api/groups', 'GET', '查看所有服务组'),
('查看服务组详情', 'group:read', '/api/groups/*', 'GET', '查看服务组详细信息'),
('创建服务组', 'group:create', '/api/groups', 'POST', '创建新服务组'),
('更新服务组', 'group:update', '/api/groups/*', 'PUT', '更新服务组信息'),
('删除服务组', 'group:delete', '/api/groups/*', 'DELETE', '删除服务组'),
('测试连接', 'group:test', '/api/groups/*/test', 'POST', '测试Consul连接'),
('查看服务组服务', 'group:services', '/api/groups/*/services', 'GET', '查看服务组下的服务')
ON CONFLICT (code) DO NOTHING;

-- Consul 服务管理权限
INSERT INTO permissions (name, code, resource, action, description) VALUES
('查看服务列表', 'consul:service:list', '/api/consul/services', 'GET', '查看Consul服务列表'),
('查看服务详情', 'consul:service:read', '/api/consul/services/*', 'GET', '查看服务详细信息'),
('删除服务', 'consul:service:delete', '/api/consul/services/*', 'DELETE', '删除Consul服务'),
('批量删除服务', 'consul:service:batch_delete', '/api/consul/services/batch-delete', 'POST', '批量删除服务'),
('查看服务健康状态', 'consul:service:health', '/api/consul/services/*/health', 'GET', '查看服务健康状态'),
('查看服务实例', 'consul:service:instances', '/api/consul/services/*/instances', 'GET', '查看服务的所有实例')
ON CONFLICT (code) DO NOTHING;

-- Consul 实例管理权限
INSERT INTO permissions (name, code, resource, action, description) VALUES
('查看实例列表', 'consul:instance:list', '/api/consul/instances', 'GET', '查看Consul实例列表'),
('查看实例详情', 'consul:instance:read', '/api/consul/instances/*', 'GET', '查看实例详细信息'),
('注册实例', 'consul:instance:create', '/api/consul/instances', 'POST', '注册新实例'),
('更新实例', 'consul:instance:update', '/api/consul/instances/*', 'PUT', '更新实例信息'),
('注销实例', 'consul:instance:delete', '/api/consul/instances/*', 'DELETE', '注销实例'),
('批量删除实例', 'consul:instance:batch_delete', '/api/consul/instances/batch-delete', 'POST', '批量删除实例'),
('导入实例', 'consul:instance:import', '/api/consul/instances/import', 'POST', '批量导入实例'),
('导出实例', 'consul:instance:export', '/api/consul/instances/export', 'GET', '批量导出实例'),
('更新Tags', 'consul:instance:tags', '/api/consul/instances/*/tags', 'PUT', '更新实例标签'),
('更新Meta', 'consul:instance:meta', '/api/consul/instances/*/meta', 'PUT', '更新实例元数据'),
('更新健康检查', 'consul:instance:health', '/api/consul/instances/*/health', 'PUT', '更新健康检查配置')
ON CONFLICT (code) DO NOTHING;

-- 审计日志权限
INSERT INTO permissions (name, code, resource, action, description) VALUES
('查看审计日志', 'audit:list', '/api/audit/logs', 'GET', '查看审计日志'),
('查看日志详情', 'audit:read', '/api/audit/logs/*', 'GET', '查看日志详细信息'),
('查看统计信息', 'audit:stats', '/api/audit/stats', 'GET', '查看审计统计信息')
ON CONFLICT (code) DO NOTHING;

-- 仪表盘权限
INSERT INTO permissions (name, code, resource, action, description) VALUES
('查看仪表盘', 'dashboard:overview', '/api/dashboard/overview', 'GET', '查看仪表盘概览'),
('查看健康统计', 'dashboard:health', '/api/dashboard/health', 'GET', '查看健康状态统计')
ON CONFLICT (code) DO NOTHING;

-- ============================================================
-- 初始角色数据
-- ============================================================

-- 超级管理员角色
INSERT INTO roles (name, code, description, status) VALUES
('超级管理员', 'admin', '拥有所有权限的超级管理员，可以管理用户、角色、服务组和Consul', 1),
('运维人员', 'operator', '可以管理Consul服务和实例，查看审计日志，但不能管理用户和角色', 1),
('只读用户', 'viewer', '只能查看Consul服务和实例信息，不能进行修改操作', 1)
ON CONFLICT (code) DO NOTHING;

-- 给超级管理员分配所有权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT 
    (SELECT id FROM roles WHERE code = 'admin'),
    id 
FROM permissions
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- 给运维人员分配 Consul 相关权限和查看权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT 
    (SELECT id FROM roles WHERE code = 'operator'),
    id 
FROM permissions 
WHERE code LIKE 'consul:%' 
   OR code LIKE 'group:list' 
   OR code LIKE 'group:read'
   OR code LIKE 'group:services'
   OR code LIKE 'audit:%'
   OR code LIKE 'dashboard:%'
   OR code LIKE 'auth:%'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- 给只读用户分配查看权限
INSERT INTO role_permissions (role_id, permission_id)
SELECT 
    (SELECT id FROM roles WHERE code = 'viewer'),
    id 
FROM permissions 
WHERE code LIKE '%:list' 
   OR code LIKE '%:read'
   OR code LIKE '%:health'
   OR code LIKE '%:instances'
   OR code LIKE '%:services'
   OR code LIKE 'dashboard:%'
   OR code LIKE 'auth:%'
ON CONFLICT (role_id, permission_id) DO NOTHING;

-- ============================================================
-- 默认服务组
-- ============================================================

-- 测试环境服务组（使用管理员用户 ID，需要先创建管理员）
INSERT INTO service_groups (name, code, description, consul_address, consul_token, consul_datacenter, status, created_by) 
SELECT 
    '测试环境',
    'test',
    '测试环境 Consul 集群',
    'http://10.0.56.118:8510',
    'a597a2fa-fbcb-9aba-e743-5043ea6cd673',
    'dc1',
    1,
    (SELECT id FROM users WHERE username = 'iflyelf' LIMIT 1)
WHERE NOT EXISTS (SELECT 1 FROM service_groups WHERE code = 'test');

-- 给超级管理员授权默认服务组
INSERT INTO role_group_permissions (role_id, group_id, permissions)
SELECT 
    (SELECT id FROM roles WHERE code = 'admin'),
    (SELECT id FROM service_groups WHERE code = 'test'),
    '["read", "write", "delete"]'::jsonb
WHERE NOT EXISTS (
    SELECT 1 FROM role_group_permissions 
    WHERE role_id = (SELECT id FROM roles WHERE code = 'admin')
    AND group_id = (SELECT id FROM service_groups WHERE code = 'test')
);

-- 给运维人员授权默认服务组
INSERT INTO role_group_permissions (role_id, group_id, permissions)
SELECT 
    (SELECT id FROM roles WHERE code = 'operator'),
    (SELECT id FROM service_groups WHERE code = 'test'),
    '["read", "write", "delete"]'::jsonb
WHERE NOT EXISTS (
    SELECT 1 FROM role_group_permissions 
    WHERE role_id = (SELECT id FROM roles WHERE code = 'operator')
    AND group_id = (SELECT id FROM service_groups WHERE code = 'test')
);

-- 给只读用户授权默认服务组（只读）
INSERT INTO role_group_permissions (role_id, group_id, permissions)
SELECT 
    (SELECT id FROM roles WHERE code = 'viewer'),
    (SELECT id FROM service_groups WHERE code = 'test'),
    '["read"]'::jsonb
WHERE NOT EXISTS (
    SELECT 1 FROM role_group_permissions 
    WHERE role_id = (SELECT id FROM roles WHERE code = 'viewer')
    AND group_id = (SELECT id FROM service_groups WHERE code = 'test')
);
