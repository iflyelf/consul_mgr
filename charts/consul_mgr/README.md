# Consul Manager Helm Chart

Consul Manager 的 Helm Chart，采用 Helmfile 结构，支持多环境与全量环境变量覆盖。

> 认证**复用外部 FlyIAM** 提供的 Casdoor，本 Chart **不部署 Casdoor**。

## 特性

- ✅ **无状态应用**：多副本 + Pod 反亲和性 + 可选 HPA
- ✅ **零手工初始化**：业务表由程序启动时自动创建
- ✅ **Helmfile 结构** + 多环境（default / dev / staging / prod）
- ✅ **全量环境变量覆盖**（`{{ env "VAR" | default "值" }}`）
- ✅ **部署前置条件检查**（hooks）
- ✅ **外置 PostgreSQL / Redis**（密码集中在 `values/_base.yaml.gotmpl` 或 `existingSecret`）
- ✅ **认证对接 FlyIAM**：仅需 Casdoor 地址与从 FlyIAM 获取的应用凭据

## 前置要求

- Kubernetes 1.24+、Helm 3.x、Helmfile 0.150+
- 外置 PostgreSQL（已创建空库 `consul_mgr`）
- 外置 Redis（可选，不可用时自动降级）
- **已部署 FlyIAM**（提供 Casdoor 认证中心），并记录：
  - Casdoor 内网地址（集群内可达）
  - 从 FlyIAM 获取的组织 / 应用 / ClientID / ClientSecret

## 快速开始

```bash
cd charts/consul_mgr

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

helmfile sync
```

首次启动会自动建表，无需手工执行 SQL。

## 项目结构

```
charts/consul_mgr/
├── Chart.yaml
├── helmfile.yaml.gotmpl          # Helmfile 主配置（多环境）
├── values/
│   ├── _base.yaml.gotmpl         # 公共默认值（唯一需要编辑的配置文件）
│   └── consul_mgr.yaml.gotmpl    # Chart 值模板
└── templates/
    ├── deployment.yaml           # 应用
    ├── service.yaml
    ├── secret.yaml
    ├── serviceaccount.yaml
    ├── hpa.yaml
    └── NOTES.txt
```

## 多环境

| 环境 | 命名空间 | 副本数 | HPA |
|------|---------|--------|-----|
| default | consul-mgr | 2 | 否 |
| dev | consul-mgr-dev | 1 | 否 |
| staging | consul-mgr-staging | 2 | 否 |
| prod | consul-mgr | 3 | 是（3~10） |

## 关键配置

`values/_base.yaml.gotmpl` 集中管理（全部支持环境变量覆盖）：

| 变量 | 说明 | 默认 |
|------|------|------|
| `CONSUL_MGR_NAMESPACE` | 命名空间 | `consul-mgr` |
| `CONSUL_MGR_REPLICAS` | 应用副本数 | `2` |
| `CONSUL_MGR_IMAGE_TAG` | 应用镜像标签 | `latest` |
| `CONSUL_MGR_DB_HOST` / `CONSUL_MGR_DB_PASSWORD` | 数据库 | - |
| `CONSUL_MGR_REDIS_HOST` / `CONSUL_MGR_REDIS_PASSWORD` | 缓存 | - |
| `CONSUL_MGR_JWT_SECRET` | JWT 密钥 | - |
| `CONSUL_MGR_ADMIN_PASSWORD` | 管理员密码 | - |
| `CONSUL_MGR_CASDOOR_ENDPOINT` | Casdoor 内网地址（FlyIAM） | `http://casdoor:8000` |
| `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` | Casdoor 浏览器地址 | - |
| `CONSUL_MGR_CASDOOR_CLIENT_ID` / `..._SECRET` | 应用凭据（FlyIAM 获取） | - |
| `CONSUL_MGR_CASDOOR_ORGANIZATION` | 组织 | `flyiam` |
| `CONSUL_MGR_CASDOOR_APPLICATION` | 应用 | `flyiam` |

> Chart 仅暴露 ClusterIP Service，不包含 Ingress；域名/HTTPS 请在集群入口层（Ingress Controller / Gateway）统一配置。

## 安装后验证

```bash
kubectl get pods -n consul-mgr
kubectl logs -n consul-mgr deploy/consul-mgr | grep -E "数据库|Casdoor"
```

预期日志：

```
✅ 数据库连接成功
✅ 数据库初始化完成
✅ Casdoor 客户端初始化成功
```

## 使用 existingSecret（推荐生产环境）

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

## 对接 FlyIAM 的注意事项

1. `CONSUL_MGR_CASDOOR_ENDPOINT` 必须是**集群内可达**的 Casdoor 地址。
2. `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` 必须是**浏览器可达**地址，否则登录跳转会失败。
3. 需在 FlyIAM/Casdoor 应用中把回调地址加入白名单：`http(s)://<你的域名>/callback`。
4. 组织与应用默认 `flyiam` / `flyiam`，与 FlyIAM 配置保持一致。

## 故障排查

| 现象 | 处理 |
|------|------|
| Pod CrashLoopBackOff | `kubectl logs` 查看；确认数据库可达、`JWT_SECRET` 与 `ADMIN_PASSWORD` 已设置 |
| 启动报 Casdoor 客户端初始化失败 | 检查 `CONSUL_MGR_CASDOOR_CLIENT_ID/SECRET` 是否从 FlyIAM 正确获取 |
| 登录报 Redirect URI 错误 | 在 FlyIAM/Casdoor 应用白名单中加入回调地址 |
| 浏览器跳转 localhost | 设置 `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` 为浏览器可达地址 |

## 更多文档

- [Kubernetes 部署](../../docs/deployment/kubernetes.md)
- [FlyIAM 认证对接](../../docs/deployment/flyiam.md)
- [架构设计](../../docs/design/architecture.md)
