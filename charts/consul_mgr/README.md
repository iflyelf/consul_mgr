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
- ✅ **跨命名空间引用 Casdoor**：ExternalName Service 别名 + 跨命名空间 FQDN 两种方式
- ✅ **NetworkPolicy**：精确放行 DNS / Casdoor（跨命名空间）/ 数据库 / Redis

## 前置要求

- Kubernetes 1.24+、Helm 3.x、Helmfile 0.150+
- 外置 PostgreSQL（已创建空库 `consul_mgr`）
- 外置 Redis（可选，不可用时自动降级）
- **目标节点已打上 `consul_mgr=true` 标签**（硬性节点亲和性要求，见[节点亲和性配置](#节点亲和性配置)）
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

# 认证：默认「外置域名方式」，后端通过域名访问 Casdoor
export CONSUL_MGR_CASDOOR_ENDPOINT="https://casdoor.example.com"
export CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT="https://casdoor.example.com"
export CONSUL_MGR_CASDOOR_CLIENT_ID="<从 FlyIAM 获取>"
export CONSUL_MGR_CASDOOR_CLIENT_SECRET="<从 FlyIAM 获取>"
export CONSUL_MGR_CASDOOR_ORGANIZATION="flyiam"
export CONSUL_MGR_CASDOOR_APPLICATION="flyiam"

# 为目标节点打标签（硬性节点亲和性要求）
kubectl get nodes
kubectl label nodes <node-1> consul_mgr=true
kubectl label nodes <node-2> consul_mgr=true

helmfile sync
```

> 默认即外置域名方式（`casdoorInCluster=false`），无需配置命名空间。
> 若 Casdoor 仅在集群内可达，见下文「认证接入方式」的集群内方式。

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
    ├── deployment.yaml               # 应用
    ├── service.yaml
    ├── secret.yaml
    ├── serviceaccount.yaml
    ├── hpa.yaml
    ├── networkpolicy.yaml            # 网络策略（DNS/Casdoor/DB/Redis）
    ├── casdoor-external-service.yaml # 集群内 Casdoor 别名（ExternalName，可选）
    └── NOTES.txt
```

## 多环境

| 环境 | 命名空间 | 副本数 | HPA |
|------|---------|--------|-----|
| default | consul-mgr | 2 | 否 |
| dev | consul-mgr-dev | 1 | 否 |
| staging | consul-mgr-staging | 2 | 否 |
| prod | consul-mgr | 3 | 是（3~10） |

> ⚠️ Pod 反亲和为**硬性打散**（`required`），带 `consul_mgr=true` 标签的节点数需
> **≥ 应用副本数**；启用 HPA 时上限同样受节点数限制
> （`CONSUL_MGR_HPA_MAX_REPLICAS` 过大将出现 Pending）。

## 关键配置

`values/_base.yaml.gotmpl` 集中管理（全部支持环境变量覆盖）：

| 变量 | 说明 | 默认 |
|------|------|------|
| `CONSUL_MGR_NAMESPACE` | 命名空间 | `consul-mgr` |
| `CONSUL_MGR_REPLICAS` | 应用副本数 | `2` |
| `CONSUL_MGR_IMAGE_TAG` | 应用镜像标签 | `latest` |
| `CONSUL_MGR_IMAGE_PULL_POLICY` | 应用镜像拉取策略 | `Always` |
| `CONSUL_MGR_NODE_LABEL` / `CONSUL_MGR_NODE_LABEL_VALUE` | 硬性节点亲和性标签 | `consul_mgr` / `true` |
| `CONSUL_MGR_DB_HOST` / `CONSUL_MGR_DB_PASSWORD` | 数据库 | - |
| `CONSUL_MGR_REDIS_HOST` / `CONSUL_MGR_REDIS_PASSWORD` | 缓存 | - |
| `CONSUL_MGR_JWT_SECRET` | JWT 密钥 | - |
| `CONSUL_MGR_ADMIN_PASSWORD` | 管理员密码 | - |
| `CONSUL_MGR_CASDOOR_ENDPOINT` | Casdoor 地址（默认外置域名） | `https://casdoor.example.com` |
| `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` | Casdoor 浏览器地址（留空回退 Endpoint） | - |
| `CONSUL_MGR_CASDOOR_IN_CLUSTER` | Casdoor 是否在集群内 | `false` |
| `CONSUL_MGR_CASDOOR_CLIENT_ID` / `..._SECRET` | 应用凭据（FlyIAM 获取） | - |
| `CONSUL_MGR_CASDOOR_ORGANIZATION` | 组织 | `flyiam` |
| `CONSUL_MGR_CASDOOR_APPLICATION` | 应用 | `flyiam` |

> Chart 仅暴露 ClusterIP Service，不包含 Ingress；域名/HTTPS 请在集群入口层（Ingress Controller / Gateway）统一配置。

## 节点亲和性配置

Consul Manager 配置了**硬性节点亲和性**，必须调度到带 `consul_mgr=true` 标签的
Linux 节点。部署前需为目标节点打标签：

```bash
# 查看节点
kubectl get nodes

# 为节点打标签
kubectl label nodes <node-1> consul_mgr=true
kubectl label nodes <node-2> consul_mgr=true

# 确认标签
kubectl get nodes -l consul_mgr=true
```

渲染后的亲和性规则：

```yaml
affinity:
  nodeAffinity:
    requiredDuringSchedulingIgnoredDuringExecution:
      nodeSelectorTerms:
        - matchExpressions:
            - key: consul_mgr    # nodeLabel
              operator: In
              values:
                - "true"          # nodeLabelValue
            - key: kubernetes.io/os
              operator: In
              values:
                - linux
  podAntiAffinity:            # 硬性打散：多副本不共节点
    requiredDuringSchedulingIgnoredDuringExecution:
      - labelSelector:
          matchExpressions:
            - key: app.kubernetes.io/name
              operator: In
              values:
                - consul_mgr
        topologyKey: kubernetes.io/hostname
```

标签由 `CONSUL_MGR_NODE_LABEL` / `CONSUL_MGR_NODE_LABEL_VALUE` 控制（默认
`consul_mgr` / `true`）：

```bash
export CONSUL_MGR_NODE_LABEL="consul_mgr"
export CONSUL_MGR_NODE_LABEL_VALUE="true"
```

> ⚠️ **硬性调度**：节点数不满足时 Pod 会一直 `Pending`，可用
> `kubectl describe pod -n consul-mgr <pod>` 查看调度事件。
>
> 打散为硬性（`required`），**带 `consul_mgr=true` 标签的节点数需 ≥ 副本数**
> （默认 2，可用 `CONSUL_MGR_REPLICAS` 调整）。

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

## 认证接入方式

**默认：外置域名方式（推荐）**

Casdoor 已通过 Ingress / 网关以域名对外暴露，后端直接通过域名访问：

```bash
export CONSUL_MGR_CASDOOR_ENDPOINT="https://casdoor.example.com"
export CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT="https://casdoor.example.com"
# casdoorInCluster 默认 false
```

> 外置域名默认走标准端口 **443（https）/ 80（http）**，NetworkPolicy 已同时放行两者；
> 如使用非标准端口，请设置 `CONSUL_MGR_NETWORK_POLICY_CASDOOR_PORTS`（如 `8443`）。

**可选：集群内方式（Casdoor 仅集群内可达）**

```bash
export CONSUL_MGR_CASDOOR_IN_CLUSTER="true"
export CONSUL_MGR_CASDOOR_NAMESPACE="flyiam"
# 二选一：
export CONSUL_MGR_CASDOOR_ENDPOINT="http://casdoor.flyiam.svc.cluster.local:8000"   # 方式 A：FQDN
# 或 enable 别名后继续用短名 http://casdoor:8000                                     # 方式 B
export CONSUL_MGR_CASDOOR_EXTERNAL_SERVICE_ENABLED="true"
```

方式 B 会创建 ExternalName Service 别名：

```yaml
kind: Service
metadata:
  name: casdoor                    # 别名（位于 consul-mgr 命名空间）
spec:
  type: ExternalName
  externalName: casdoor.flyiam.svc.cluster.local
```

相关变量：

| 变量 | 说明 | 默认 |
|------|------|------|
| `CONSUL_MGR_CASDOOR_IN_CLUSTER` | Casdoor 是否在集群内 | `false` |
| `CONSUL_MGR_CASDOOR_NAMESPACE` | FlyIAM 所在命名空间（集群内方式） | `flyiam` |
| `CONSUL_MGR_CASDOOR_SERVICE_NAME` | FlyIAM 中 Casdoor Service 名 | `casdoor` |
| `CONSUL_MGR_CASDOOR_SERVICE_PORT` | Casdoor 端口（集群内方式） | `8000` |
| `CONSUL_MGR_CASDOOR_EXTERNAL_SERVICE_ENABLED` | 是否创建别名 | `false` |
| `CONSUL_MGR_CASDOOR_EXTERNAL_SERVICE_NAME` | 别名名称 | `casdoor` |

## NetworkPolicy 网络策略

Chart 默认创建 NetworkPolicy（`CONSUL_MGR_NETWORK_POLICY_ENABLED=true`），
仅放行必要流量（需 CNI 支持，如 Calico / Cilium / Antrea）：

- **入站**：应用端口（默认放开所有来源，可用 `allowAllIngress=false` 收紧到指定命名空间）
- **出站**：
  - DNS（kube-system:53）
  - **Casdoor**：
    - 外置域名方式（默认）：放行标准端口 443/80 到任意目标；填 `networkPolicyCasdoorCidrs` 后仅放行对应网段
    - 集群内方式：`namespaceSelector=flyiam` + `podSelector=component=casdoor`
  - **Consul 集群**：地址/端口运行时可配，`consulCidrs` 为空时不限制（`- {}`）
  - 数据库（PostgreSQL 端口，可选按 `dbCidrs` 收紧）
  - Redis（启用缓存时）

| 变量 | 说明 | 默认 |
|------|------|------|
| `CONSUL_MGR_NETWORK_POLICY_ENABLED` | 是否创建 NetworkPolicy | `true` |
| `CONSUL_MGR_NETWORK_POLICY_ALLOW_ALL_INGRESS` | 入站放开所有来源 | `true` |
| `CONSUL_MGR_NETWORK_POLICY_INGRESS_NAMESPACES` | 收紧入站时允许的命名空间（逗号分隔） | 空 |
| `CONSUL_MGR_NETWORK_POLICY_CASDOOR_CIDRS` | 外置 Casdoor 域名对应网段（逗号分隔） | 空 |
| `CONSUL_MGR_NETWORK_POLICY_CASDOOR_PORTS` | 外置 Casdoor 端口（逗号分隔） | `443,80` |
| `CONSUL_MGR_NETWORK_POLICY_CONSUL_CIDRS` | Consul 集群网段（逗号分隔，全端口） | 空 |
| `CONSUL_MGR_NETWORK_POLICY_DB_CIDRS` | 数据库/Redis 目标网段（逗号分隔） | 空 |
| `CONSUL_MGR_NETWORK_POLICY_ALLOW_ALL_EGRESS` | 放行全部出站（调试） | `false` |
| `CONSUL_MGR_CASDOOR_POD_LABEL_KEY` / `_VALUE` | Casdoor Pod 标签（集群内方式） | `app.kubernetes.io/component` / `casdoor` |

> ⚠️ Consul 集群地址/端口在运行时按「服务组」配置，无法预先穷举。
> `CONSUL_MGR_NETWORK_POLICY_CONSUL_CIDRS` 为空时出站对 Consul 不限制；
> 填写后可真正收紧出站。

> 收紧示例（入站仅入口控制器命名空间，出站限定 Casdoor / Consul / DB 网段）：
>
> ```bash
> export CONSUL_MGR_NETWORK_POLICY_ALLOW_ALL_INGRESS=false
> export CONSUL_MGR_NETWORK_POLICY_INGRESS_NAMESPACES="ingress-nginx"
> export CONSUL_MGR_NETWORK_POLICY_CASDOOR_CIDRS="203.0.113.0/24"
> export CONSUL_MGR_NETWORK_POLICY_CASDOOR_PORTS="443"
> export CONSUL_MGR_NETWORK_POLICY_CONSUL_CIDRS="10.0.56.0/24,10.1.0.0/16"
> export CONSUL_MGR_NETWORK_POLICY_DB_CIDRS="10.0.51.0/24,10.1.230.0/24"
> ```

## 对接 FlyIAM 的注意事项

1. 默认外置域名方式：`CONSUL_MGR_CASDOOR_ENDPOINT` 应为集群可访问的域名（通常 HTTPS）。
2. `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` 必须是**浏览器可达**地址，否则登录跳转会失败。
3. 需在 FlyIAM/Casdoor 应用中把回调地址加入白名单：`http(s)://<你的域名>/callback`。
4. 组织与应用默认 `flyiam` / `flyiam`，与 FlyIAM 配置保持一致。
5. 集群内方式（`CONSUL_MGR_CASDOOR_IN_CLUSTER=true`）下若启用 NetworkPolicy，
   需确保 FlyIAM 的 Casdoor Pod 带有 `app.kubernetes.io/component=casdoor`
   标签（FlyIAM Chart 默认已带）。

## 运维操作

统一在 `charts/consul_mgr` 目录执行：

```bash
# 安装部署
helmfile -f helmfile.yaml.gotmpl sync

# 更新（修改配置/镜像后重新同步）
helmfile -f helmfile.yaml.gotmpl diff     # 查看变更
helmfile -f helmfile.yaml.gotmpl sync     # 应用变更

# 指定环境
helmfile -f helmfile.yaml.gotmpl -e prod sync

# 卸载
helmfile -f helmfile.yaml.gotmpl destroy
```

## 故障排查

| 现象 | 处理 |
|------|------|
| Pod Pending（节点亲和性不满足） | `kubectl get nodes -l consul_mgr=true` 确认节点已打标签，或调整 `CONSUL_MGR_NODE_LABEL` |
| Pod CrashLoopBackOff | `kubectl logs` 查看；确认数据库可达、`JWT_SECRET` 与 `ADMIN_PASSWORD` 已设置 |
| 启动报 Casdoor 客户端初始化失败 | 检查 `CONSUL_MGR_CASDOOR_CLIENT_ID/SECRET` 是否从 FlyIAM 正确获取 |
| 登录报 Redirect URI 错误 | 在 FlyIAM/Casdoor 应用白名单中加入回调地址 |
| 浏览器跳转 localhost | 设置 `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT` 为浏览器可达地址 |

## 更多文档

- [Kubernetes 部署](../../docs/deployment/kubernetes.md)
- [FlyIAM 认证对接](../../docs/deployment/flyiam.md)
- [架构设计](../../docs/design/architecture.md)
