# Consul Manager + Casdoor 快速开始指南

## 🚀 快速部署

### 步骤 1: 启动 Casdoor 和数据库

```bash
# 克隆项目
cd /path/to/consul_mgr

# 启动服务
docker-compose -f docker-compose.casdoor.yml up -d

# 查看日志
docker-compose -f docker-compose.casdoor.yml logs -f casdoor
```

### 步骤 2: 配置 Casdoor

1. **访问 Casdoor 管理界面**
   - URL: http://localhost:8000
   - 默认账号: admin / 123

2. **创建组织**
   - 名称: `consul_mgr`
   - 显示名: `Consul Manager`

3. **创建应用**
   - 名称: `consul_manager`
   - 组织: `consul_mgr`
   - 回调 URL: `http://localhost:8080/api/auth/callback`
   - 主页: `http://localhost:8080`
   - Logo: (可选)

4. **记录凭证**
   ```bash
   Client ID: <your-client-id>
   Client Secret: <your-client-secret>
   ```

5. **配置权限**
   
   创建以下资源和权限：
   
   **资源 (Resources):**
   - `consul_group` - Consul 服务组
   - `consul_service` - Consul 服务
   - `consul_instance` - Consul 实例
   - `audit_log` - 审计日志
   - `user_management` - 用户管理

   **操作 (Actions):**
   - `read` - 读取
   - `write` - 写入
   - `delete` - 删除
   - `admin` - 管理

6. **创建角色**

   **管理员角色 (admin):**
   ```json
   {
     "name": "admin",
     "displayName": "系统管理员",
     "permissions": [
       {"resource": "*", "action": "*"}
     ]
   }
   ```

   **运维角色 (operator):**
   ```json
   {
     "name": "operator",
     "displayName": "运维人员",
     "permissions": [
       {"resource": "consul_group", "action": "*"},
       {"resource": "consul_service", "action": "*"},
       {"resource": "consul_instance", "action": "*"},
       {"resource": "audit_log", "action": "read"}
     ]
   }
   ```

   **只读角色 (viewer):**
   ```json
   {
     "name": "viewer",
     "displayName": "访客",
     "permissions": [
       {"resource": "consul_group", "action": "read"},
       {"resource": "consul_service", "action": "read"},
       {"resource": "consul_instance", "action": "read"}
     ]
   }
   ```

### 步骤 3: 更新 Consul Manager 配置

编辑 `etc/config.yaml`:

```yaml
Name: consul_mgr
Host: 0.0.0.0
Port: 8080

Database:
  Host: ${DATABASE_HOST:localhost}
  Port: ${DATABASE_PORT:5432}
  User: ${DATABASE_USER:consul_mgr}
  Password: ${DATABASE_PASSWORD:}
  DBName: ${DATABASE_NAME:consul_mgr}
  SSLMode: ${DATABASE_SSLMODE:disable}

JWT:
  Secret: ${JWT_SECRET:your-secret-key}
  Expire: 86400 # 24小时

# Casdoor 配置
Casdoor:
  Endpoint: ${CASDOOR_ENDPOINT:http://localhost:8000}
  ClientId: ${CASDOOR_CLIENT_ID:}
  ClientSecret: ${CASDOOR_CLIENT_SECRET:}
  Certificate: ${CASDOOR_CERTIFICATE:}
  OrganizationName: ${CASDOOR_ORGANIZATION:consul_mgr}
  ApplicationName: ${CASDOOR_APPLICATION:consul_manager}
  
# 服务组权限控制
ServiceGroupAuth:
  Enabled: true
  DefaultPermissions: ["read"] # 新用户默认权限
  
# 审计日志
Audit:
  Enabled: true
  LogLevel: info # debug, info, warn, error
  RetentionDays: 90 # 日志保留天数
```

### 步骤 4: 安装依赖

```bash
# 安装 Casdoor Go SDK
go get github.com/casdoor/casdoor-go-sdk

# 前端安装 Casdoor JS SDK
cd web
npm install casdoor-js-sdk
```

### 步骤 5: 启动 Consul Manager

```bash
# 设置环境变量
export DATABASE_URL="postgresql://consul_mgr:password@localhost:5432/consul_mgr?sslmode=disable"
export CASDOOR_ENDPOINT="http://localhost:8000"
export CASDOOR_CLIENT_ID="your-client-id"
export CASDOOR_CLIENT_SECRET="your-client-secret"
export JWT_SECRET="your-secret-key-min-32-chars"

# 编译运行
go build -o consul_mgr ./cmd/api
./consul_mgr -c etc/config.yaml
```

### 步骤 6: 测试集成

1. **访问前端**
   ```
   http://localhost:8080
   ```

2. **点击"使用 Casdoor 登录"**
   - 跳转到 Casdoor 登录页
   - 使用 admin / 123 登录
   - 授权后自动跳回

3. **测试 API**
   ```bash
   # 获取 Token（从浏览器控制台复制）
   TOKEN="your-casdoor-token"
   
   # 测试服务组列表
   curl -H "Authorization: Bearer $TOKEN" \
        http://localhost:8080/api/groups
   ```

## 🔐 权限配置示例

### 场景 1: 为用户分配服务组权限

```sql
-- 给用户 user123 分配服务组 1 的读写权限
INSERT INTO service_group_users 
  (group_id, user_id, username, permissions, created_by)
VALUES 
  (1, 'user123', 'Zhang San', ARRAY['read', 'write'], 'admin');
```

### 场景 2: 为角色分配服务组权限

```sql
-- 给 operator 角色分配所有服务组的读写权限
INSERT INTO service_group_roles 
  (group_id, role_name, permissions)
VALUES 
  (1, 'operator', ARRAY['read', 'write']),
  (2, 'operator', ARRAY['read', 'write']);
```

### 场景 3: 服务组授权 API

```bash
# 创建服务组时指定授权用户
curl -X POST http://localhost:8080/api/groups \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "production",
    "consul_address": "http://consul.prod:8500",
    "authorized_users": [
      {"user_id": "user123", "permissions": ["read", "write"]},
      {"user_id": "user456", "permissions": ["read"]}
    ],
    "authorized_roles": [
      {"role_name": "operator", "permissions": ["read", "write"]}
    ]
  }'
```

## 📊 审计日志查询

```bash
# 查询某用户的操作日志
curl "http://localhost:8080/api/audit/logs?user_id=user123&page=1&page_size=20" \
  -H "Authorization: Bearer $TOKEN"

# 查询某服务组的操作日志
curl "http://localhost:8080/api/audit/logs?group_id=1&page=1&page_size=20" \
  -H "Authorization: Bearer $TOKEN"

# 查询某时间段的日志
curl "http://localhost:8080/api/audit/logs?start_time=2024-01-01&end_time=2024-01-31" \
  -H "Authorization: Bearer $TOKEN"
```

## 🎯 开发任务清单

### Phase 1 - Casdoor 基础集成 ✅

- [x] 部署 Casdoor
- [x] 配置 Docker Compose
- [ ] 实现 Casdoor 客户端
- [ ] 实现 OAuth2 登录流程
- [ ] 实现 Token 验证中间件
- [ ] 前端登录页面改造

### Phase 2 - 权限系统

- [ ] 设计服务组权限表
- [ ] 实现权限检查中间件
- [ ] 实现服务组授权 API
- [ ] 前端权限管理界面
- [ ] 权限测试用例

### Phase 3 - 审计日志

- [ ] 设计审计日志表
- [ ] 实现审计日志中间件
- [ ] 实现审计日志查询 API
- [ ] 前端审计日志界面
- [ ] 日志自动清理

### Phase 4 - 测试和文档

- [ ] 单元测试
- [ ] 集成测试
- [ ] 性能测试
- [ ] 用户文档
- [ ] API 文档

## 🛠️ 开发工具

### 推荐 IDE 插件

- **VS Code:**
  - Go (官方)
  - Vetur (Vue 3)
  - REST Client (API 测试)

### 调试技巧

```bash
# 查看 Casdoor 日志
docker logs -f casdoor

# 查看数据库
docker exec -it casdoor_db psql -U casdoor -d casdoor

# 测试 Token 验证
curl -X POST http://localhost:8000/api/login/oauth/introspect \
  -d "token=your-token" \
  -d "client_id=your-client-id" \
  -d "client_secret=your-client-secret"
```

## 📚 参考文档

- [Casdoor 官方文档](https://casdoor.org/docs/overview)
- [Casdoor Go SDK](https://github.com/casdoor/casdoor-go-sdk)
- [Casdoor JS SDK](https://github.com/casdoor/casdoor-js-sdk)
- [Go-Zero 文档](https://go-zero.dev/)
- [OAuth 2.0 RFC](https://datatracker.ietf.org/doc/html/rfc6749)

## 🆘 常见问题

### Q1: Casdoor 启动失败？
```bash
# 检查端口占用
netstat -tuln | grep 8000

# 查看日志
docker logs casdoor

# 重置数据库
docker-compose -f docker-compose.casdoor.yml down -v
docker-compose -f docker-compose.casdoor.yml up -d
```

### Q2: Token 验证失败？
- 检查 Client ID 和 Secret 是否正确
- 确认 Token 没有过期
- 检查 Casdoor Endpoint 配置

### Q3: 权限检查不生效？
- 确认用户已分配正确的角色
- 检查角色是否配置了相应权限
- 查看审计日志确认请求流程

### Q4: 前端无法跳转到 Casdoor？
- 检查 Casdoor SDK 配置
- 确认回调 URL 正确
- 查看浏览器控制台错误

## 🎉 下一步

1. **评审集成方案** - 与团队讨论技术方案
2. **准备环境** - 部署 Casdoor 测试环境
3. **启动开发** - 按 Phase 顺序实施
4. **测试验证** - 完整的功能和性能测试
5. **上线部署** - 生产环境部署和监控

---

**需要帮助？**
- 查看详细文档: `CASDOOR_INTEGRATION_PLAN.md`
- 提交 Issue: GitHub Issues
- 联系开发团队

祝开发顺利！🚀
