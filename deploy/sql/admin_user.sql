-- ============================================================
-- 创建管理员账号
-- 密码: ysyh!9Sky (bcrypt 加密)
-- ============================================================

-- 插入管理员用户
-- 密码 bcrypt hash: ysyh!9Sky
INSERT INTO users (username, password, email, real_name, status, created_at, updated_at)
VALUES (
    'iflyelf',
    '$2a$10$8Zw0K5YqJxZ8YqJ9X0H9Qu8yQZxZ0K5YqJxZ8YqJ9X0H9Qu8yQZxZ0',  -- 占位符，实际由应用启动时生成
    'iflyelf@gmail.com',
    'iflyelf',
    1,
    CURRENT_TIMESTAMP,
    CURRENT_TIMESTAMP
)
ON CONFLICT (username) DO NOTHING;

-- 给管理员分配超级管理员角色
INSERT INTO user_roles (user_id, role_id)
SELECT 
    (SELECT id FROM users WHERE username = 'iflyelf'),
    (SELECT id FROM roles WHERE code = 'admin')
WHERE NOT EXISTS (
    SELECT 1 FROM user_roles 
    WHERE user_id = (SELECT id FROM users WHERE username = 'iflyelf')
    AND role_id = (SELECT id FROM roles WHERE code = 'admin')
);
