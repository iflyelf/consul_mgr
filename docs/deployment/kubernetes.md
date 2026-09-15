# 部署文档 - Kubernetes

提供开箱即用的清单（`deploy/k8s/`），端口通过 ConfigMap 自定义。

## 清单说明

| 文件 | 内容 |
|------|------|
| `00-namespace.yaml` | 命名空间 `consul-mgr` |
| `01-config.yaml` | ConfigMap（非敏感配置）+ Secret（凭据） |
| `02-dependencies.yaml` | PostgreSQL + Redis（**示例**，生产请用托管服务） |
| `03-app.yaml` | Deployment + Service + Ingress |

## 部署步骤

```bash
# 1. 克隆（或只下载 deploy/k8s 目录）
git clone https://github.com/iflyelf/consul_mgr.git
cd consul_mgr

# 2. 修改配置与凭据（务必修改 Secret 中所有 change_me）
vi deploy/k8s/01-config.yaml

# 3. 按需修改镜像、副本数、端口、域名
vi deploy/k8s/03-app.yaml

# 4. 应用
kubectl apply -f deploy/k8s/00-namespace.yaml
kubectl apply -f deploy/k8s/01-config.yaml
kubectl apply -f deploy/k8s/02-dependencies.yaml   # 使用外部数据库/Redis 时跳过
kubectl apply -f deploy/k8s/03-app.yaml

# 5. 查看状态
kubectl -n consul-mgr get pods,svc,ingress
kubectl -n consul-mgr logs -f deploy/consul-mgr
```

## 自定义端口

端口通过 ConfigMap 的 `SERVER_PORT` 控制：

```yaml
data:
  SERVER_PORT: "8080"
```

同时需同步修改 `03-app.yaml` 中容器的 `containerPort` 与 Service 的 `targetPort`/`port`。

## 使用外部数据库 / Redis

1. 跳过 `02-dependencies.yaml`
2. 修改 `01-config.yaml` 的 Secret：
   - `DATABASE_URL` 指向外部 PostgreSQL
3. 修改 ConfigMap：
   - `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` 指向外部 Redis
   - `REDIS_ENABLED: "false"` 可完全关闭缓存

## 使用外部 Casdoor

```yaml
data:
  CASDOOR_ENDPOINT: "http://casdoor.casdoor.svc.cluster.local:8000"  # 后端访问
  CASDOOR_PUBLIC_ENDPOINT: "https://casdoor.example.com"             # 浏览器访问
```

> `CASDOOR_PUBLIC_ENDPOINT` 必须是**浏览器可达**的地址，否则登录跳转会失败。
> 同时需在 Casdoor 应用的回调白名单中加入 `http(s)://<你的域名>/callback`。

## TLS / Ingress

`03-app.yaml` 中的 Ingress 为最小示例。启用 HTTPS：

```yaml
spec:
  tls:
    - hosts: ["consul-mgr.example.com"]
      secretName: consul-mgr-tls
  rules:
    - host: consul-mgr.example.com
      # ...
```

并使用 cert-manager 或预先创建 `consul-mgr-tls` Secret。

## 探针

已配置：
- `readinessProbe` / `livenessProbe`：`GET /health`

Pod 就绪后才会接入流量；异常时自动重启。

## 扩缩容

应用为无状态服务（数据都在 PostgreSQL / Redis），可直接调副本数：

```bash
kubectl -n consul-mgr scale deploy/consul-mgr --replicas=3
```

## 卸载

```bash
kubectl delete namespace consul-mgr
```

> 注意：若使用内置 PostgreSQL，PVC 默认不会随命名空间删除而回收，请按需手动清理。

## 相关文档

- [systemd 部署](systemd.md) · [Docker 部署](docker.md) · [Casdoor 配置](casdoor.md)
