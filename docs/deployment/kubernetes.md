# 部署文档 - Kubernetes

本项目提供 Helm Chart（`charts/consul_mgr`），采用 Helmfile 多环境管理。

> 认证**复用外部 FlyIAM** 提供的 Casdoor，本 Chart **不部署 Casdoor**，也**不部署** PostgreSQL / Redis。

## 1. 架构

```
Helmfile
  └── consul_mgr Chart（namespace: consul-mgr）
       ├── Deployment      consul-mgr      # 应用（含嵌入前端）
       ├── Service         consul-mgr
       ├── Secret          consul-mgr-secret
       ├── ServiceAccount
       ├── NetworkPolicy   consul-mgr      # 出站精确放行（含 Casdoor）
       ├── Service         casdoor         # 仅集群内方式：ExternalName 别名（可选）
       └── HPA（可选）

外部依赖（不由本 Chart 部署）：
  PostgreSQL（业务库 consul_mgr）
  Redis（缓存，可选）
  FlyIAM（内置 Casdoor，提供 OAuth2 认证；默认以**外置域名**方式访问）
```

Chart 仅暴露 ClusterIP Service，不包含 Ingress；域名访问请在集群入口层（Ingress Controller / Gateway）统一配置。

## 2. 前置条件

- Kubernetes 1.24+
- Helm 3.x、Helmfile 0.150+
- 外置 PostgreSQL（已创建空库 `consul_mgr`）
- 外置 Redis（可选，不可用时自动降级）
- **目标节点已打上 `consul_mgr=true` 标签**（硬性节点亲和性要求）
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

# 认证：默认「外置域名方式」（Casdoor 已通过域名对外暴露）
export CONSUL_MGR_CASDOOR_ENDPOINT="https://casdoor.example.com"
export CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT="https://casdoor.example.com"
export CONSUL_MGR_CASDOOR_CLIENT_ID="<从 FlyIAM 获取>"
export CONSUL_MGR_CASDOOR_CLIENT_SECRET="<从 FlyIAM 获取>"
export CONSUL_MGR_CASDOOR_ORGANIZATION="flyiam"
export CONSUL_MGR_CASDOOR_APPLICATION="flyiam"

# 可选：Redis
export CONSUL_MGR_REDIS_HOST="redis.default.svc.cluster.local"
export CONSUL_MGR_REDIS_PASSWORD="your-redis-password"

# 为目标节点打标签（硬性节点亲和性要求）
kubectl get nodes
kubectl label nodes <node-1> consul_mgr=true
kubectl label nodes <node-2> consul_mgr=true

# 部署
helmfile sync

# 指定环境
helmfile -e prod sync
```

首次启动会自动建表，无需手工执行 SQL。

### 3.1 节点亲和性

应用配置了**硬性节点亲和性**，必须调度到带 `consul_mgr=true` 标签的 Linux 节点：

```bash
kubectl get nodes                     # 查看节点
kubectl label nodes <node-1> consul_mgr=true
kubectl get nodes -l consul_mgr=true  # 确认标签
```

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: consul_mgr      # nodeLabel
              operator: In
              values:
                - "true"            # nodeLabelValue
            - key: kubernetes.io/os
              operator: In
              values:
                - linux
  podAntiAffinity:                 # 硬性打散：多副本不共节点
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - consul_mgr
        topologyKey: kubernetes.io/hostname
```

标签可通过环境变量覆盖：

```bash
export CONSUL_MGR_NODE_LABEL="consul_mgr"
export CONSUL_MGR_NODE_LABEL_VALUE="true"
```

> **硬性调度**：不满足时 Pod 会一直 `Pending`，用 `kubectl describe pod` 查看调度事件。
> 打散为硬性（`required`），**带 `consul_mgr=true` 标签的节点数需 ≥ 副本数**（默认 2）。

## 4. 对接 FlyIAM（认证）说明

| 项 | 说明 | 环境变量 |
|----|------|---------|
| Casdoor 地址 | 后端访问（默认外置域名） | `CONSUL_MGR_CASDOOR_ENDPOINT` |
| Casdoor 浏览器地址 | 浏览器可达，用于登录跳转 | `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` |
| 部署形态 | 外置域名 / 集群内 | `CONSUL_MGR_CASDOOR_IN_CLUSTER` |
| 应用凭据 | 从 FlyIAM 获取 | `CONSUL_MGR_CASDOOR_CLIENT_ID` / `..._SECRET` |
| 组织 / 应用 | 与 FlyIAM 一致 | `CONSUL_MGR_CASDOOR_ORGANIZATION` / `..._APPLICATION` |

**回调白名单**：需在 FlyIAM/Casdoor 应用中把本服务的回调地址加入白名单：

```
http(s)://<你的域名>/callback
http(s)://<你的域名>/api/auth/callback
```

> `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` 必须是**浏览器可达**地址，否则登录跳转会失败。
> 详见 [FlyIAM 认证对接](flyiam.md)。

### 4.1 认证接入方式

**默认：外置域名方式（`casdoorInCluster=false`）**

Casdoor 通过 Ingress / 网关以域名对外暴露，后端直接以域名访问：

```bash
export CONSUL_MGR_CASDOOR_ENDPOINT="https://casdoor.example.com"
export CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT="https://casdoor.example.com"
```

> 外置域名默认走标准端口 **443（https）/ 80（http）**，NetworkPolicy 已同时放行两者；
> 非标准端口请设置 `CONSUL_MGR_NETWORK_POLICY_CASDOOR_PORTS`（如 `8443`）。

**可选：集群内方式（`casdoorInCluster=true`）**

Casdoor 仅在集群内可达时使用，二选一：

方式 A（FQDN）:

```bash
export CONSUL_MGR_CASDOOR_IN_CLUSTER="true"
export CONSUL_MGR_CASDOOR_NAMESPACE="flyiam"
export CONSUL_MGR_CASDOOR_ENDPOINT="http://casdoor.flyiam.svc.cluster.local:8000"
```

方式 B（ExternalName 别名，`casdoorExternalServiceEnabled=true`）：

```yaml
apiVersion: v1
kind: Service
metadata:
  name: casdoor
  namespace: consul-mgr
spec:
  type: ExternalName
  externalName: casdoor.flyiam.svc.cluster.local
  ports:
    - name: http
      port: 8000
```

| 环境变量 | 说明 | 默认 |
|---------|------|------|
| `CONSUL_MGR_CASDOOR_IN_CLUSTER` | Casdoor 是否在集群内 | `false` |
| `CONSUL_MGR_CASDOOR_NAMESPACE` | FlyIAM 命名空间 | `flyiam` |
| `CONSUL_MGR_CASDOOR_SERVICE_NAME` | Casdoor Service 名 | `casdoor` |
| `CONSUL_MGR_CASDOOR_SERVICE_PORT` | Casdoor 端口（集群内方式） | `8000` |
| `CONSUL_MGR_CASDOOR_EXTERNAL_SERVICE_ENABLED` | 是否创建别名 | `false` |
| `CONSUL_MGR_CASDOOR_EXTERNAL_SERVICE_NAME` | 别名名称 | `casdoor` |

### 4.2 NetworkPolicy

Chart 默认创建 NetworkPolicy。外置域名方式下按端口/CIDR 放行 Casdoor；
集群内方式下使用 `namespaceSelector` + `podSelector` 精确放行：

```yaml
egress:
  - to:                                   # DNS
      - namespaceSelector:
          matchLabels:
            kubernetes.io/metadata.name: kube-system
    ports: [{ protocol: UDP, port: 53 }, { protocol: TCP, port: 53 }]

  # 外置域名方式（默认）：放行标准端口 443/80；填 casdoorCidrs 后仅放行对应网段
  - ports:
      - protocol: TCP
        port: 443
      - protocol: TCP
        port: 80
  # 集群内方式（casdoorInCluster=true）：
  # - to:
  #     - namespaceSelector:
  #         matchLabels:
  #           kubernetes.io/metadata.name: flyiam
  #       podSelector:
  #         matchLabels:
  #           app.kubernetes.io/component: casdoor
  #   ports:
  #     - protocol: TCP
  #       port: 8000

  - to:                                   # Consul 集群（consulCidrs 为空时不限制）
      - ipBlock:
          cidr: 10.0.56.0/24
      # 全部端口

  - ports:                                # 数据库（可按 dbCidrs 收紧）
      - protocol: TCP
        port: 5432
```

| 环境变量 | 说明 | 默认 |
|---------|------|------|
| `CONSUL_MGR_NETWORK_POLICY_ENABLED` | 是否创建 | `true` |
| `CONSUL_MGR_NETWORK_POLICY_ALLOW_ALL_INGRESS` | 入站放开所有来源 | `true` |
| `CONSUL_MGR_NETWORK_POLICY_INGRESS_NAMESPACES` | 收紧入站时允许的命名空间 | 空 |
| `CONSUL_MGR_NETWORK_POLICY_CASDOOR_CIDRS` | 外置 Casdoor 域名对应网段 | 空 |
| `CONSUL_MGR_NETWORK_POLICY_CASDOOR_PORTS` | 外置 Casdoor 端口 | `443,80` |
| `CONSUL_MGR_NETWORK_POLICY_CONSUL_CIDRS` | Consul 集群网段（全端口） | 空 |
| `CONSUL_MGR_NETWORK_POLICY_DB_CIDRS` | 数据库/Redis 目标网段 | 空 |
| `CONSUL_MGR_NETWORK_POLICY_ALLOW_ALL_EGRESS` | 放行全部出站（调试） | `false` |
| `CONSUL_MGR_CASDOOR_POD_LABEL_KEY` / `_VALUE` | Casdoor Pod 标签（集群内方式） | `app.kubernetes.io/component` / `casdoor` |

> 前置条件：集群 CNI 需支持 NetworkPolicy（Calico / Cilium / Antrea 等）。
> 若使用不支持 NetworkPolicy 的 CNI，该对象会被忽略，不影响部署。
> 集群内方式下 FlyIAM Chart 的 Casdoor Pod 默认带
> `app.kubernetes.io/component=casdoor` 标签，与放行规则匹配。
>
> ⚠️ Consul 集群地址/端口在运行时按「服务组」配置，无法预先穷举。
> `CONSUL_MGR_NETWORK_POLICY_CONSUL_CIDRS` 为空时出站对 Consul 不限制（`- {}`）；
> 填写后可真正收紧出站，例如 `10.0.56.0/24,10.1.0.0/16`。

## 5. 多环境

| 环境 | 命名空间 | 副本数 | HPA |
|------|---------|--------|-----|
| default | consul-mgr | 2 | 否 |
| dev | consul-mgr-dev | 1 | 否 |
| staging | consul-mgr-staging | 2 | 否 |
| prod | consul-mgr | 3 | 是（3~10） |

> ⚠️ Pod 反亲和为**硬性打散**（`required`），带 `consul_mgr=true` 标签的节点数需
> ≥ 副本数；启用 HPA 时上限同样受节点数限制，`maxReplicas` 过大将出现 Pending。

## 6. 配置项（节选）

`charts/consul_mgr/values/_base.yaml.gotmpl` 集中管理，全部支持环境变量覆盖：

| 变量 | 说明 | 默认 |
|------|------|------|
| `CONSUL_MGR_NAMESPACE` | 命名空间 | `consul-mgr` |
| `CONSUL_MGR_REPLICAS` | 应用副本数 | `2` |
| `CONSUL_MGR_IMAGE_TAG` | 应用镜像标签 | `latest` |
| `CONSUL_MGR_IMAGE_PULL_POLICY` | 应用镜像拉取策略 | `Always` |
| `CONSUL_MGR_NODE_LABEL` / `CONSUL_MGR_NODE_LABEL_VALUE` | 硬性节点亲和性标签 | `consul_mgr` / `true` |
| `CONSUL_MGR_DB_HOST` / `CONSUL_MGR_DB_PASSWORD` | 数据库 | - |
| `CONSUL_MGR_DATABASE_URL` | 完整连接串（优先于分项） | - |
| `CONSUL_MGR_REDIS_HOST` / `CONSUL_MGR_REDIS_PASSWORD` | 缓存 | - |
| `CONSUL_MGR_JWT_SECRET` | JWT 密钥（≥32 位） | - |
| `CONSUL_MGR_ADMIN_PASSWORD` | 管理员密码 | - |
| `CONSUL_MGR_CASDOOR_ENDPOINT` | Casdoor 地址（默认外置域名） | `https://casdoor.example.com` |
| `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` | Casdoor 浏览器地址（留空回退 Endpoint） | - |
| `CONSUL_MGR_CASDOOR_IN_CLUSTER` | Casdoor 是否在集群内 | `false` |
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

## 11. 更新与卸载

```bash
# 更新（修改配置/镜像后）
helmfile -f helmfile.yaml.gotmpl diff    # 查看变更
helmfile -f helmfile.yaml.gotmpl sync    # 应用变更

# 卸载
helmfile -e default destroy
# 或
helm uninstall consul-mgr -n consul-mgr
```

## 相关文档

- [systemd 部署](systemd.md) · [Docker 部署](docker.md) · [FlyIAM 认证对接](flyiam.md)
