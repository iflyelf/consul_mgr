━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
🎯 Consul Manager + Casdoor 快速参考
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## 🌐 访问地址

Casdoor 管理后台：http://localhost:8000
  账号：admin
  密码：123

Consul Manager：http://localhost:8080
  状态：待配置 Casdoor 后启动

GitHub 仓库：https://github.com/iflyelf/consul_mgr

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## 🚀 常用命令

# 查看 Casdoor 状态
docker ps | grep casdoor

# 查看 Casdoor 日志
docker logs casdoor

# 重启 Casdoor
docker restart casdoor

# 停止所有服务
docker-compose -f docker-compose.casdoor-external-db.yml down

# 启动 Casdoor
docker-compose -f docker-compose.casdoor-external-db.yml up -d casdoor

# 启动完整服务（需先配置 Casdoor）
export CASDOOR_CLIENT_ID="your-client-id"
export CASDOOR_CLIENT_SECRET="your-client-secret"
docker-compose -f docker-compose.casdoor-external-db.yml up -d

# 检查数据库表
cd /xiaonuo/workspace/code/consul_mgr
go run /tmp/check_casdoor_tables.go

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## 📊 数据库信息

连接字符串：postgresql://iflyelf:1q23l@Yc45j@10.0.51.88:6000/consul_mgr?sslmode=disable

表统计：
  - Consul Manager：7 张表
  - Casdoor：41 张表
  - 总计：48 张表 (~3 MB)

核心表：
  - casdoor_user          # 用户
  - casdoor_organization  # 组织
  - casdoor_application   # 应用
  - casdoor_role          # 角色
  - casdoor_permission    # 权限
  - service_groups        # 服务组
  - service_group_users   # 服务组用户权限
  - service_group_roles   # 服务组角色权限
  - audit_logs            # 审计日志

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## 📝 配置清单

### Casdoor 配置步骤（首次使用必做）

1. ✅ 访问 http://localhost:8000
2. ✅ 登录（admin / 123）
3. ⏳ 修改管理员密码
4. ⏳ 创建组织（consul_mgr）
5. ⏳ 创建应用（consul_manager）
   - 回调 URL: http://localhost:8080/api/auth/callback
6. ⏳ 创建角色
   - admin（管理员）
   - operator（运维）
   - viewer（访客）
7. ⏳ 配置权限资源
   - consul_group
   - consul_service
   - consul_instance
   - audit_log
8. ⏳ 记录 Client ID 和 Secret

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## 📚 文档索引

项目根目录：/xiaonuo/workspace/code/consul_mgr

重要文档：
  - CASDOOR_INTEGRATION_PLAN.md    # 详细技术方案（20+ 页）
  - QUICK_START_CASDOOR.md         # 快速开始指南
  - INTEGRATION_COMPLETE.md         # 阶段性完成报告
  - THIS_FILE.md                   # 快速参考（本文件）

配置文件：
  - casdoor/conf/app.conf                    # Casdoor 配置
  - docker-compose.casdoor-external-db.yml   # Docker 编排
  - etc/config.yaml                          # Consul Manager 配置

工具脚本：
  - execute_cleanup.sh              # 数据库清理脚本
  - tools/cleanup_db.go            # Go 数据库清理工具
  - deploy/sql/cleanup_and_migrate.sql  # SQL 迁移脚本

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## 🔧 故障排除

### Casdoor 无法启动
```bash
# 查看日志
docker logs casdoor

# 检查端口占用
netstat -tuln | grep 8000

# 重启容器
docker restart casdoor
```

### 数据库连接失败
```bash
# 检查数据库连接
psql "postgresql://iflyelf:1q23l@Yc45j@10.0.51.88:6000/consul_mgr"

# 检查网络
ping 10.0.51.88
telnet 10.0.51.88 6000
```

### Session 权限错误
已修复，配置文件已设置 providerConfig="/tmp"

### Docker 镜像拉取慢
使用国内镜像加速器或手动拉取：
```bash
docker pull casbin/casdoor:latest
```

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## 🎯 开发路线图

✅ Phase 0: 准备工作（已完成）
  - 数据库清理
  - Casdoor 部署
  - 文档编写

⏳ Phase 1: Casdoor 配置（1-2小时）
  - 配置组织和应用
  - 创建角色和权限

⏳ Phase 2: 后端集成（2-3天）
  - OAuth2 登录
  - Token 验证
  - 权限检查

⏳ Phase 3: 前端集成（1-2天）
  - 登录页面改造
  - OAuth 回调
  - 权限控制 UI

⏳ Phase 4: 服务组授权（1天）
  - 授权 API
  - 权限管理界面

⏳ Phase 5: 审计日志（1天）
  - 日志增强
  - 查询和导出

⏳ Phase 6: 测试文档（1天）
  - 功能测试
  - 文档完善

总预估：8-10 天

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## 💡 最佳实践

1. **数据库备份**
   定期备份 consul_mgr 数据库

2. **密码安全**
   首次登录后立即修改 admin 密码

3. **环境变量**
   生产环境使用强密码和随机 JWT_SECRET

4. **日志监控**
   定期查看 Casdoor 和 Consul Manager 日志

5. **权限设计**
   遵循最小权限原则

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

## 🔗 有用的链接

Casdoor 官方：
  - 文档：https://casdoor.org/docs/overview
  - GitHub：https://github.com/casdoor/casdoor
  - Demo：https://door.casdoor.com
  - Go SDK：https://github.com/casdoor/casdoor-go-sdk
  - JS SDK：https://github.com/casdoor/casdoor-js-sdk

Go-Zero 官方：
  - 文档：https://go-zero.dev/
  - GitHub：https://github.com/zeromicro/go-zero

Consul：
  - 文档：https://www.consul.io/docs
  - API：https://www.consul.io/api-docs

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

最后更新：2026-09-14 23:00
状态：✅ Casdoor 部署完成，等待配置

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
