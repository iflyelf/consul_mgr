# 部署文档 - Kubernetes

本项目提供 Helm Chart（`charts/consul_mgr`），采用 Helmfile 多环境管理。

> 认证**复用外部 FlyIAM** 提供的 Casdoor，本 Chart **不部署 Casdoor**，也**不部署** PostgreSQL / Redis。

## 1. 架构

```
Helmfile
  └── consul_mgr Chart
       ├── Deployment  consul-mgr    # 应用（含嵌入前端）
       ├── Service     consul-mgr
       ├── Secret      consul-mgr-secret
       ├── ServiceAccount
       └── HPA（可选）

外部依赖（不由本 Chart 部署）：
  PostgreSQL（业务库 consul_mgr）
  Redis（缓存，可选）
  FlyIAM（内置 Casdoor，提供 OAuth2 认证）
```

Chart 仅暴露 ClusterIP Service，不包含 Ingress；域名访问请在集群入口层（Ingress Controller / Gateway）统一配置。

## 2. 前置条件

- Kubernetes 1.24+
- Helm 3.x、Helmfile 0.150+
- 外置 PostgreSQL（已创建空库 `consul_mgr`）
- 外置 Redis（可选，不可用时自动降级）
- **已部署 FlyIAM**（提供 Casdoor 认证中心），并准备好：
  - Casdoor 集群内可达地址（如 `http://casdoor.flyiam.svc.cluster.local:8000`）
  - 从 FlyIAM 获取的 `ClientID` / `ClientSecret`

## 3. 快速部署

```bash
cd charts/consul_mgr

# 必填环境变量
export CONSUL_MGR_DB_HOST="postgres.default.svc.cluster.local"
export CONSUL_MGR_DB_PORT="5432"
export CONSUL_MGR_DB_NAME="consul_mgr"
export CONSUL_MGR_DB_USER="consul_mgr"
export CONSUL_MGR_DB_PASSWORD="your-db-password"
export CONSUL_MGR_JWT_SECRET="your-jwt-secret-at-least-32-chars"
export CONSUL_MGR_ADMIN_PASSWORD="your-admin-password"

# 认证：指向 FlyIAM 的 Casdoor
export CONSUL_MGR_CASDOOR_ENDPOINT="http://casdoor.flyiam.svc.cluster.local:8000"
export CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT="http://casdoor.example.com:8000"
export CONSUL_MGR_CASDOOR_CLIENT_ID="<从 FlyIAM 获取>"
export CONSUL_MGR_CASDOOR_CLIENT_SECRET="<从 FlyIAM 获取>"
export CONSUL_MGR_CASDOOR_ORGANIZATION="flyiam"
export CONSUL_MGR_CASDOOR_APPLICATION="flyiam"

# 可选：Redis
export CONSUL_MGR_REDIS_HOST="redis.default.svc.cluster.local"
export CONSUL_MGR_REDIS_PASSWORD="your-redis-password"

# 部署
helmfile sync

# 指定环境
helmfile -e prod sync
```

首次启动会自动建表，无需手工执行 SQL。

## 4. 对接 FlyIAM（认证）说明

| 项 | 说明 | 环境变量 |
|----|------|---------|
| Casdoor 内网地址 | 集群内可达，供后端换取 Token | `CONSUL_MGR_CASDOOR_ENDPOINT` |
| Casdoor 浏览器地址 | 浏览器可达，用于登录跳转 | `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` |
| 应用凭据 | 从 FlyIAM 获取 | `CONSUL_MGR_CASDOOR_CLIENT_ID` / `..._SECRET` |
| 组织 / 应用 | 与 FlyIAM 一致 | `CONSUL_MGR_CASDOOR_ORGANIZATION` / `..._APPLICATION` |

**回调白名单**：需在 FlyIAM/Casdoor 应用中把本服务的回调地址加入白名单：

```
http(s)://<你的域名>/callback
http(s)://<你的域名>/api/auth/callback
```

> `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` 必须是**浏览器可达**地址，否则登录跳转会失败。
> 详见 [FlyIAM 认证对接](flyiam.md)。

## 5. 多环境

| 环境 | 命名空间 | 副本数 | HPA |
|------|---------|--------|-----|
| default | consul-mgr | 2 | 否 |
| dev | consul-mgr-dev | 1 | 否 |
| staging | consul-mgr-staging | 2 | 否 |
| prod | consul-mgr | 3 | 是（3~10） |

## 6. 配置项（节选）

`charts/consul_mgr/values/_base.yaml.gotmpl` 集中管理，全部支持环境变量覆盖：

| 变量 | 说明 | 默认 |
|------|------|------|
| `CONSUL_MGR_NAMESPACE` | 命名空间 | `consul-mgr` |
| `CONSUL_MGR_REPLICAS` | 应用副本数 | `2` |
| `CONSUL_MGR_IMAGE_TAG` | 应用镜像标签 | `latest` |
| `CONSUL_MGR_DB_HOST` / `CONSUL_MGR_DB_PASSWORD` | 数据库 | - |
| `CONSUL_MGR_DATABASE_URL` | 完整连接串（优先于分项） | - |
| `CONSUL_MGR_REDIS_HOST` / `CONSUL_MGR_REDIS_PASSWORD` | 缓存 | - |
| `CONSUL_MGR_JWT_SECRET` | JWT 密钥（≥32 位） | - |
| `CONSUL_MGR_ADMIN_PASSWORD` | 管理员密码 | - |
| `CONSUL_MGR_CASDOOR_ENDPOINT` | Casdoor 内网地址 | `http://casdoor:8000` |
| `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` | Casdoor 浏览器地址 | - |
| `CONSUL_MGR_CASDOOR_ORGANIZATION` | 组织 | `flyiam` |
| `CONSUL_MGR_CASDOOR_APPLICATION` | 应用 | `flyiam` |
| `CONSUL_MGR_HPA_ENABLED` | 是否启用 HPA | `false` |

## 7. 访问

```bash
# 应用
kubectl port-forward -n consul-mgr svc/consul-mgr 8080:8080
# 浏览器访问 http://localhost:8080
```

> 通过集群入口（Ingress Controller / Gateway）暴露时，需同时配置回调白名单，
> 并将 `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` 设为浏览器可达的 Casdoor 地址。

## 8. 验证

```bash
helmfile -e default lint      # 语法检查
helmfile -e default template  # 渲染清单
helmfile -e default diff      # 查看变更
helmfile -e default sync      # 部署

kubectl get pods -n consul-mgr
kubectl logs -n consul-mgr deploy/consul-mgr | head -30
```

预期日志：

```
✅ 数据库连接成功
✅ 数据库初始化完成
✅ Casdoor 客户端初始化成功
🚀 Starting Consul Manager Server
```

## 9. 使用 existingSecret（推荐生产环境）

```bash
kubectl create secret generic consul-mgr-secret \
  --from-literal=DB_PASSWORD='...' \
  --from-literal=REDIS_PASSWORD='...' \
  --from-literal=JWT_SECRET='...' \
  --from-literal=ADMIN_PASSWORD='...' \
  --from-literal=CASDOOR_CLIENT_ID='...' \
  --from-literal=CASDOOR_CLIENT_SECRET='...' \
  -n consul-mgr

export CONSUL_MGR_EXISTING_SECRET="consul-mgr-secret"
helmfile sync
```

> Secret 需包含 key：`DB_PASSWORD`、`REDIS_PASSWORD`、`JWT_SECRET`、`ADMIN_PASSWORD`、`CASDOOR_CLIENT_ID`、`CASDOOR_CLIENT_SECRET`（可选 `CONSUL_TOKEN`）。

## 10. 扩缩容

应用为无状态服务（数据都在 PostgreSQL / Redis），可直接调副本数：

```bash
kubectl -n consul-mgr scale deploy/consul-mgr --replicas=3
```

或启用 HPA（`CONSUL_MGR_HPA_ENABLED=true`）。

## 11. 卸载

```bash
helmfile -e default destroy
# 或
helm uninstall consul-mgr -n consul-mgr
```

## 相关文档

- [systemd 部署](systemd.md) · [Docker 部署](docker.md) · [FlyIAM 认证对接](flyiam.md)
