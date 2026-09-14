# Consul Manager 使用手册

## 快速入门

### 第一步：启动服务

```bash
# 设置必需的环境变量
export DATABASE_URL="postgresql://iflyelf:password@localhost:5432/consul_mgr?sslmode=disable"
export ADMIN_PASSWORD="your_strong_password"

# 启动服务
./start.sh
```

启动成功后，你会看到：

```
======================================
   Consul Manager 启动脚本
======================================

停止已有进程...
启动后端服务...
✓ 后端服务启动成功 (PID: 12345)
  API 地址: http://localhost:8080
  健康检查: http://localhost:8080/health
  日志文件: /tmp/consul_mgr.log

启动前端服务...
✓ 前端服务启动成功 (PID: 12346)
  访问地址: http://localhost:5173
  日志文件: /tmp/vite.log

======================================
启动完成！
======================================

管理员账号: admin
访问地址: http://localhost:5173
```

### 第二步：登录系统

1. 打开浏览器访问 http://localhost:5173
2. 输入管理员账号和密码
3. 点击"登录"

![登录界面](docs/images/login.png)

### 第三步：管理服务组

#### 添加服务组

1. 登录后，点击"添加服务组"按钮
2. 填写表单：
   - **名称**: 测试环境
   - **代码**: test
   - **描述**: 测试环境 Consul 集群
   - **Consul 地址**: http://consul.test.com:8500
   - **Consul Token**: (可选)
   - **数据中心**: dc1

3. 点击"确定"保存

#### 编辑服务组

1. 在列表中找到要编辑的服务组
2. 点击"编辑"按钮
3. 修改需要的字段
4. 点击"确定"保存

#### 测试连接

点击"测试"按钮，系统会尝试连接到 Consul 服务器并验证配置是否正确。

#### 删除服务组

点击"删除"按钮，确认后即可删除服务组。

## 界面功能

### 主题切换

点击右上角的太阳图标 ☀️，可以切换三种主题：

1. **浅色主题** - 适合白天使用，明亮清爽
2. **深色主题** - 适合夜间使用，保护眼睛
3. **蓝色主题** - 清新蓝色风格

主题设置会自动保存，下次打开自动应用。

### 退出登录

点击右上角的用户头像，选择"退出登录"。

## 移动端使用

Consul Manager 完全支持手机访问：

1. 在手机浏览器打开 http://your-server:5173
2. 布局会自动适配手机屏幕
3. 侧边栏在手机上会自动收起，只显示图标
4. 表格在手机上会优化显示

## 数据管理

### 服务组字段说明

| 字段 | 必填 | 说明 |
|------|------|------|
| 名称 | ✅ | 服务组的显示名称，如"生产环境" |
| 代码 | ✅ | 唯一标识符，只能使用字母、数字、下划线 |
| 描述 | ❌ | 服务组的详细描述 |
| Consul 地址 | ✅ | Consul 服务器地址，格式：http://host:port |
| Consul Token | ❌ | Consul ACL Token，如果启用了 ACL 则必填 |
| 数据中心 | ❌ | Consul 数据中心名称，默认为 dc1 |
| 状态 | - | 启用/禁用，编辑时可修改 |

### 数据验证规则

- **名称**: 不能为空
- **代码**: 不能为空，必须唯一
- **Consul 地址**: 必须是有效的 URL 格式

## 故障排查

### 无法登录

**原因**：用户名或密码错误

**解决方案**：
1. 检查环境变量 ADMIN_USERNAME 和 ADMIN_PASSWORD
2. 重启服务：`./start.sh`

### 无法添加服务组

**原因**：字段验证失败

**解决方案**：
1. 检查所有必填字段是否填写
2. 检查 Consul 地址格式是否正确（必须以 http:// 或 https:// 开头）
3. 检查代码是否已存在（代码必须唯一）

### 测试连接失败

**原因**：无法连接到 Consul 服务器

**解决方案**：
1. 检查 Consul 地址是否正确
2. 检查网络连接
3. 如果 Consul 启用了 ACL，检查 Token 是否正确
4. 查看后端日志：`tail -f /tmp/consul_mgr.log`

### 页面无法加载

**原因**：前端服务未启动或后端服务未启动

**解决方案**：
1. 检查前端服务：`curl http://localhost:5173`
2. 检查后端服务：`curl http://localhost:8080/health`
3. 查看日志：
   - 前端：`tail -f /tmp/vite.log`
   - 后端：`tail -f /tmp/consul_mgr.log`

## 高级配置

### 自定义端口

#### 修改后端端口

编辑 `etc/config.yaml`：

```yaml
Server:
  Host: 0.0.0.0
  Port: 9000  # 修改为你需要的端口
```

#### 修改前端端口

编辑 `web/vite.config.js`：

```javascript
export default defineConfig({
  server: {
    port: 4000,  // 修改为你需要的端口
    proxy: {
      '/api': {
        target: 'http://localhost:9000',  // 对应后端端口
        changeOrigin: true
      }
    }
  }
})
```

### 配置 HTTPS

使用 Nginx 反向代理：

```nginx
server {
    listen 443 ssl http2;
    server_name consul-mgr.example.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:5173;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }

    location /api {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

### 数据库维护

#### 备份数据库

```bash
pg_dump -h localhost -U user -d consul_mgr > backup.sql
```

#### 恢复数据库

```bash
psql -h localhost -U user -d consul_mgr < backup.sql
```

#### 清理审计日志

```sql
-- 删除 30 天前的审计日志
DELETE FROM audit_logs WHERE created_at < NOW() - INTERVAL '30 days';
```

## 性能优化

### 数据库连接池

编辑 `etc/config.yaml`：

```yaml
Database:
  MaxOpenConns: 100  # 最大连接数
  MaxIdleConns: 10   # 最大空闲连接数
```

### JWT Token 过期时间

编辑 `etc/config.yaml`：

```yaml
JWT:
  AccessExpire: 7200    # 访问令牌过期时间（秒），默认 2 小时
  RefreshExpire: 604800 # 刷新令牌过期时间（秒），默认 7 天
```

## 安全建议

1. **使用强密码**：管理员密码至少 12 位，包含大小写字母、数字和特殊字符
2. **定期更换密钥**：定期更换 JWT_SECRET
3. **启用 HTTPS**：生产环境必须使用 HTTPS
4. **限制访问**：使用防火墙限制只允许特定 IP 访问
5. **定期备份**：定期备份数据库
6. **监控日志**：定期检查审计日志，发现异常行为

## 常见问题

### Q: 如何修改管理员密码？

A: 目前只能通过环境变量或配置文件修改，修改后需要重启服务。后续版本会支持在界面中修改。

### Q: 支持多用户吗？

A: 数据库已包含完整的用户、角色、权限表结构，但界面暂未实现。后续版本会添加。

### Q: 可以管理多个 Consul 集群吗？

A: 可以，每个服务组对应一个 Consul 集群，可以添加任意数量的服务组。

### Q: 数据存储在哪里？

A: 所有数据存储在 PostgreSQL 数据库中。

### Q: 如何升级？

A: 
1. 备份数据库
2. 停止服务
3. 替换二进制文件
4. 启动服务（会自动执行数据库迁移）

## 获取帮助

- GitHub Issues: https://github.com/iflyelf/consul_mgr/issues
- Email: iflyelf@gmail.com

---

更新时间: 2026-09-14
版本: v0.3.0
