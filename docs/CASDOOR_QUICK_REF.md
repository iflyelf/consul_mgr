# Casdoor 快速参考卡片

## 🚀 快速开始

### 1. 启动 Casdoor
```bash
docker-compose -f docker-compose.casdoor-external-db.yml up -d casdoor
```

### 2. 运行配置向导
```bash
cd /xiaonuo/workspace/code/consul_mgr
bash scripts/casdoor_wizard.sh
```

### 3. 加载配置
```bash
source .env
```

---

## 🔗 访问地址

| 服务 | 地址 | 账号 |
|------|------|------|
| Casdoor 管理后台 | http://localhost:8000 | admin / 123 |
| Consul Manager | http://localhost:8080 | 稍后配置 |

---

## 📋 配置信息

### 组织配置
```
Name:         consul_mgr
Display name: Consul Manager
```

### 应用配置
```
Name:         consul_manager
Display name: Consul Manager 应用
Callback URL: http://localhost:8080/api/auth/callback
```

### 角色配置
```
admin     - 管理员（所有权限）
operator  - 运维人员（服务管理）
viewer    - 访客（只读）
```

---

## 🛠️ 常用命令

### 检查 Casdoor 状态
```bash
docker ps | grep casdoor
curl http://localhost:8000/
```

### 查看 Casdoor 日志
```bash
docker logs casdoor
docker logs -f casdoor  # 实时查看
```

### 重启 Casdoor
```bash
docker restart casdoor
```

### 查看配置
```bash
cat .env.casdoor
cat etc/config.yaml | grep -A 10 "Casdoor:"
```

---

## 🔐 环境变量

```bash
CASDOOR_ENDPOINT=http://localhost:8000
CASDOOR_CLIENT_ID=<your_client_id>
CASDOOR_CLIENT_SECRET=<your_client_secret>
CASDOOR_ORGANIZATION=consul_mgr
CASDOOR_APPLICATION=consul_manager
```

---

## 📚 文档索引

| 文档 | 路径 | 用途 |
|------|------|------|
| 配置手册 | `docs/CASDOOR_MANUAL_SETUP.md` | 详细配置步骤 |
| 完成报告 | `docs/PHASE1_COMPLETE.md` | Phase 1 总结 |
| 快速参考 | `docs/CASDOOR_QUICK_REF.md` | 本文档 |

---

## ❓ 常见问题

### Q: Casdoor 无法访问？
```bash
# 检查容器状态
docker ps -a | grep casdoor

# 重启容器
docker restart casdoor

# 查看日志
docker logs casdoor
```

### Q: 忘记 Client ID 和 Secret？
1. 登录 Casdoor 管理后台
2. 进入 Applications
3. 找到 consul_manager
4. 查看或重新生成

### Q: 回调失败？
检查应用配置中的 Redirect URIs：
- 必须包含：`http://localhost:8080/api/auth/callback`
- 端口号要一致

### Q: 权限不生效？
1. 检查角色是否关联到权限
2. 检查用户是否分配了角色
3. 重新登录刷新 Token

---

## ⚡ 快速测试

### 测试 Casdoor API
```bash
# 获取全局配置
curl http://localhost:8000/api/get-global-providers

# 测试访问（需要先登录）
curl http://localhost:8000/api/get-account
```

### 测试数据库
```bash
# 检查 Casdoor 表
psql "postgresql://iflyelf:1q23l@Yc45j@10.0.51.88:6000/consul_mgr" \
  -c "\dt casdoor_*"
```

---

## 🎯 检查清单

配置完成后，确认：

- [ ] Casdoor 运行正常
- [ ] 可以访问管理后台
- [ ] 组织 `consul_mgr` 已创建
- [ ] 应用 `consul_manager` 已创建
- [ ] Client ID 和 Secret 已获取并保存
- [ ] 回调 URL 已配置
- [ ] 配置文件 `.env` 已生成
- [ ] 环境变量已加载
- [ ] (可选) 角色和权限已配置
- [ ] (可选) 测试用户已创建

---

## 🔄 下一步

Phase 1 完成后：

1. ✅ 确认所有配置正确
2. ➡️ 开始 Phase 2：后端 OAuth2 集成
3. 📖 阅读：`docs/PHASE2_PLAN.md`（即将创建）

---

**最后更新：** 2024-01-15  
**版本：** v1.0
