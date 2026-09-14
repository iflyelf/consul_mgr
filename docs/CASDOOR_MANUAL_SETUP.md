# Casdoor 手动配置指南

本文档提供详细的 Casdoor 手动配置步骤，作为自动配置脚本的备选方案。

---

## 📋 前置条件

1. ✅ Casdoor 服务已启动并运行在 `http://localhost:8000`
2. ✅ 可以访问 Casdoor 管理后台
3. ✅ 知道默认管理员账号：`admin / 123`

---

## 🚀 配置步骤

### 步骤 1: 登录 Casdoor 管理后台

1. **打开浏览器访问：** http://localhost:8000

2. **使用默认账号登录：**
   - 用户名：`admin`
   - 密码：`123`

3. **首次登录后，强烈建议修改密码：**
   - 点击右上角用户头像
   - 选择 "Edit Profile"
   - 修改密码并保存

---

### 步骤 2: 创建组织（Organization）

1. **导航到组织管理：**
   - 点击左侧菜单 "Organizations"
   - 点击右上角 "Add" 按钮

2. **填写组织信息：**
   ```
   Name:         consul_mgr
   Display name: Consul Manager
   Website URL:  http://localhost:8080
   Favicon:      https://cdn.casbin.org/img/casbin.svg
   ```

3. **密码配置：**
   ```
   Password type: plain
   Password salt: (留空)
   ```

4. **其他配置：**
   ```
   Phone prefix:    86
   Default avatar:  https://cdn.casbin.org/img/casbin.svg
   ```

5. **点击 "Save" 保存**

✅ **验证：** 在组织列表中应该看到 `consul_mgr`

---

### 步骤 3: 创建应用（Application）

1. **导航到应用管理：**
   - 点击左侧菜单 "Applications"
   - 点击右上角 "Add" 按钮

2. **填写基本信息：**
   ```
   Name:         consul_manager
   Display name: Consul Manager 应用
   Organization: consul_mgr (选择刚创建的组织)
   Logo:         https://cdn.casbin.org/img/casbin.svg
   Homepage URL: http://localhost:8080
   ```

3. **配置认证信息：**
   
   **重要：** Client ID 和 Client Secret 会自动生成
   
   ```
   Client ID:     (自动生成，记下来！)
   Client Secret: (自动生成，记下来！)
   ```

4. **配置回调 URL：**
   ```
   Redirect URIs: http://localhost:8080/api/auth/callback
   ```
   
   **注意：** 可以添加多个，一行一个：
   ```
   http://localhost:8080/api/auth/callback
   http://localhost:8080/callback
   ```

5. **Token 配置：**
   ```
   Token format:           JWT
   Token expire in hours:  24
   Refresh token expire:   168 (7天)
   ```

6. **启用 OAuth 提供商：**
   - 确保 "Enable password" 已勾选
   - 可选：启用其他第三方登录（GitHub、Google 等）

7. **点击 "Save" 保存**

✅ **验证：** 
- 应用列表中出现 `consul_manager`
- **记录 Client ID 和 Client Secret**（后续需要使用）

---

### 步骤 4: 创建角色（Roles）

#### 4.1 创建管理员角色

1. **导航到角色管理：**
   - 点击左侧菜单 "Roles"
   - 点击右上角 "Add" 按钮

2. **填写角色信息：**
   ```
   Name:         admin
   Display name: 管理员
   Organization: consul_mgr
   Description:  系统管理员，拥有所有权限
   ```

3. **启用角色：**
   - 勾选 "Is enabled"

4. **点击 "Save" 保存**

#### 4.2 创建运维角色

重复上述步骤，创建运维角色：

```
Name:         operator
Display name: 运维人员
Organization: consul_mgr
Description:  可以管理服务组、服务和实例
```

#### 4.3 创建访客角色

重复上述步骤，创建访客角色：

```
Name:         viewer
Display name: 访客
Organization: consul_mgr
Description:  只能查看，无法修改
```

✅ **验证：** 角色列表中应该有 3 个角色：`admin`、`operator`、`viewer`

---

### 步骤 5: 创建权限（Permissions）

#### 5.1 服务组管理权限

1. **导航到权限管理：**
   - 点击左侧菜单 "Permissions"
   - 点击右上角 "Add" 按钮

2. **填写权限信息：**
   ```
   Name:         consul_group_all
   Display name: 服务组管理权限
   Organization: consul_mgr
   Description:  服务组的所有操作权限
   ```

3. **配置资源和动作：**
   ```
   Resource type: consul_group
   Resources:     * (所有资源)
   Actions:       read, write, delete, admin
   ```

4. **关联角色：**
   - 勾选 `admin`
   - 勾选 `operator`

5. **设置效果：**
   ```
   Effect:    Allow
   Is enabled: ✓
   ```

6. **点击 "Save" 保存**

#### 5.2 服务管理权限

重复上述步骤，创建服务管理权限：

```
Name:          consul_service_all
Display name:  服务管理权限
Organization:  consul_mgr
Description:   服务的所有操作权限
Resource type: consul_service
Resources:     *
Actions:       read, write, delete, admin
Roles:         admin, operator
Effect:        Allow
```

#### 5.3 实例管理权限

创建实例管理权限：

```
Name:          consul_instance_all
Display name:  实例管理权限
Organization:  consul_mgr
Description:   实例的所有操作权限
Resource type: consul_instance
Resources:     *
Actions:       read, write, delete, import, export, admin
Roles:         admin, operator
Effect:        Allow
```

#### 5.4 审计日志查看权限

创建审计日志权限：

```
Name:          audit_log_read
Display name:  审计日志查看权限
Organization:  consul_mgr
Description:   查看审计日志
Resource type: audit_log
Resources:     *
Actions:       read, export
Roles:         admin, operator
Effect:        Allow
```

#### 5.5 只读权限（访客）

创建只读权限：

```
Name:          consul_read_only
Display name:  只读权限
Organization:  consul_mgr
Description:   只能查看，不能修改
Resource type: consul
Resources:     *
Actions:       read
Roles:         viewer
Effect:        Allow
```

✅ **验证：** 权限列表中应该有 5 个权限

---

### 步骤 6: 创建测试用户

1. **导航到用户管理：**
   - 点击左侧菜单 "Users"
   - 点击右上角 "Add" 按钮

2. **填写用户信息：**
   ```
   Name:         test_admin
   Display name: 测试管理员
   Organization: consul_mgr
   Email:        test@example.com
   Password:     Test@123456
   ```

3. **分配角色：**
   - 在 "Roles" 下拉框中选择 `admin`

4. **启用用户：**
   - 确保 "Is admin" 未勾选（组织级别管理员）
   - 勾选 "Is enabled"

5. **点击 "Save" 保存**

✅ **验证：** 
- 用户列表中出现 `test_admin`
- 可以用该账号登录 Casdoor

---

### 步骤 7: 配置 Consul Manager

#### 7.1 记录配置信息

从步骤 3 中记录的信息：

```
CASDOOR_ENDPOINT=http://localhost:8000
CASDOOR_CLIENT_ID=<your_client_id>
CASDOOR_CLIENT_SECRET=<your_client_secret>
CASDOOR_ORGANIZATION=consul_mgr
CASDOOR_APPLICATION=consul_manager
```

#### 7.2 创建 .env 文件

在项目根目录创建 `.env` 文件：

```bash
# Casdoor 配置
CASDOOR_ENDPOINT=http://localhost:8000
CASDOOR_CLIENT_ID=<替换为实际的 Client ID>
CASDOOR_CLIENT_SECRET=<替换为实际的 Client Secret>
CASDOOR_ORGANIZATION=consul_mgr
CASDOOR_APPLICATION=consul_manager

# 数据库配置
DATABASE_URL=postgresql://iflyelf:1q23l@Yc45j@10.0.51.88:6000/consul_mgr?sslmode=disable

# JWT 配置
JWT_SECRET=consul_mgr_jwt_secret_key_for_testing_min_32_chars
```

#### 7.3 更新 config.yaml

在 `etc/config.yaml` 中添加 Casdoor 配置：

```yaml
# Casdoor 配置
Casdoor:
  Endpoint: ${CASDOOR_ENDPOINT:http://localhost:8000}
  ClientId: ${CASDOOR_CLIENT_ID}
  ClientSecret: ${CASDOOR_CLIENT_SECRET}
  Certificate: ${CASDOOR_CERTIFICATE:}
  OrganizationName: ${CASDOOR_ORGANIZATION:consul_mgr}
  ApplicationName: ${CASDOOR_APPLICATION:consul_manager}

# 权限配置
Permission:
  EnableServiceGroupAuth: true
  DefaultPermissions: ["read"]
```

---

## ✅ 验证配置

### 1. 验证 Casdoor 配置

访问 Casdoor API：

```bash
curl http://localhost:8000/api/get-organization?id=admin/consul_mgr
```

应该返回组织信息。

### 2. 验证应用配置

```bash
curl http://localhost:8000/api/get-application?id=consul_mgr/consul_manager
```

应该返回应用信息（包含 Client ID）。

### 3. 验证角色和权限

登录 Casdoor 管理后台，检查：
- Organizations: 有 `consul_mgr`
- Applications: 有 `consul_manager`
- Roles: 有 `admin`、`operator`、`viewer`
- Permissions: 有 5 个权限
- Users: 有测试用户

---

## 🔧 常见问题

### Q1: 创建组织时提示"已存在"

**解决方法：**
- 使用不同的名称
- 或删除已有的同名组织

### Q2: Client ID 和 Secret 忘记了怎么办？

**解决方法：**
1. 登录 Casdoor 管理后台
2. 进入 Applications
3. 找到 `consul_manager`
4. 点击编辑，查看 Client ID 和 Secret

### Q3: 回调 URL 配置错误

**症状：** OAuth 登录后无法回调到 Consul Manager

**解决方法：**
1. 检查应用配置中的 Redirect URIs
2. 确保包含：`http://localhost:8080/api/auth/callback`
3. 注意端口号要一致

### Q4: 权限不生效

**解决方法：**
1. 检查权限是否已启用（Is enabled）
2. 检查角色是否关联到权限
3. 检查用户是否分配了角色
4. 重新登录以刷新 Token

### Q5: 无法创建用户

**解决方法：**
1. 检查组织是否已创建
2. 检查密码策略（默认需要大小写字母+数字）
3. 检查邮箱格式是否正确

---

## 📊 配置检查清单

在继续下一步之前，请确认：

- [ ] Casdoor 服务运行正常
- [ ] 组织 `consul_mgr` 已创建
- [ ] 应用 `consul_manager` 已创建
- [ ] Client ID 和 Secret 已记录
- [ ] 回调 URL 已正确配置
- [ ] 3 个角色已创建（admin、operator、viewer）
- [ ] 5 个权限已创建并关联到角色
- [ ] 至少创建了一个测试用户
- [ ] .env 文件已创建并配置
- [ ] config.yaml 已更新

---

## 🔗 相关资源

- **Casdoor 官方文档：** https://casdoor.org/docs/overview
- **OAuth 2.0 RFC：** https://datatracker.ietf.org/doc/html/rfc6749
- **JWT 介绍：** https://jwt.io/introduction

---

## 📝 配置示例

### 完整的 .env 文件示例

```bash
# ==========================================
# Casdoor 配置
# ==========================================
CASDOOR_ENDPOINT=http://localhost:8000
CASDOOR_CLIENT_ID=consul_manager_1234567890
CASDOOR_CLIENT_SECRET=abcdefghijklmnopqrstuvwxyz123456
CASDOOR_ORGANIZATION=consul_mgr
CASDOOR_APPLICATION=consul_manager

# ==========================================
# 数据库配置
# ==========================================
DATABASE_URL=postgresql://iflyelf:1q23l@Yc45j@10.0.51.88:6000/consul_mgr?sslmode=disable

# ==========================================
# JWT 配置
# ==========================================
JWT_SECRET=consul_mgr_jwt_secret_key_for_testing_min_32_chars

# ==========================================
# 服务配置
# ==========================================
SERVER_HOST=0.0.0.0
SERVER_PORT=8080
```

### 完整的 config.yaml 示例

```yaml
Name: consul_mgr
Host: 0.0.0.0
Port: 8080

# 数据库配置
Database:
  Host: ${DATABASE_HOST:10.0.51.88}
  Port: ${DATABASE_PORT:6000}
  User: ${DATABASE_USER:iflyelf}
  Password: ${DATABASE_PASSWORD:1q23l@Yc45j}
  DBName: ${DATABASE_NAME:consul_mgr}
  SSLMode: ${DATABASE_SSLMODE:disable}

# JWT 配置
JWT:
  Secret: ${JWT_SECRET:your-secret-key}
  Expire: 86400

# Casdoor 配置
Casdoor:
  Endpoint: ${CASDOOR_ENDPOINT:http://localhost:8000}
  ClientId: ${CASDOOR_CLIENT_ID}
  ClientSecret: ${CASDOOR_CLIENT_SECRET}
  Certificate: ${CASDOOR_CERTIFICATE:}
  OrganizationName: ${CASDOOR_ORGANIZATION:consul_mgr}
  ApplicationName: ${CASDOOR_APPLICATION:consul_manager}

# 权限配置
Permission:
  EnableServiceGroupAuth: true
  DefaultPermissions: ["read"]

# 审计日志配置
Audit:
  Enabled: true
  RetentionDays: 90
  CleanupInterval: 86400

# 日志配置
Log:
  Mode: console
  Level: info
  Path: logs
```

---

## 🎉 完成

配置完成后，您可以：

1. **启动 Consul Manager：**
   ```bash
   source .env
   go build -o consul_mgr ./cmd/api
   ./consul_mgr -c etc/config.yaml
   ```

2. **访问 Consul Manager：**
   - 地址：http://localhost:8080
   - 点击登录，会跳转到 Casdoor
   - 使用创建的测试用户登录

3. **开始使用！**

---

**最后更新：** 2024-01-15  
**文档版本：** v1.0
