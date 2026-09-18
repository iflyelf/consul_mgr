# 部署文档 - FlyIAM 认证对接

Consul Manager 的认证（OAuth2 授权码模式）**完全依赖外部 [FlyIAM](https://github.com/iflyelf/flyiam)**
提供的 Casdoor 服务。本项目**不再内置 Casdoor**，也不再保存用户密码。

> 历史版本曾自带 Casdoor（`docker-compose.casdoor*.yml`、`scripts/setup_casdoor.sh`、`tools/add-redirect-uri`），
> 现已全部移除；如从旧版本升级，请参见文末「从内置 Casdoor 迁移」。

## 1. 部署 FlyIAM

FlyIAM 内置 Casdoor，一条命令即可启动（详见 FlyIAM 文档）：

```bash
git clone https://github.com/iflyelf/flyiam.git
cd flyiam
docker compose up -d

# Casdoor：http://localhost:8000
# FlyIAM ：http://localhost:8081
# 默认管理员：admin / ysyh!9Sky
```

Kubernetes 环境请使用 FlyIAM 自带 Helm Chart（`charts/flyiam`），其中包含 Casdoor 的
Deployment / Service / ConfigMap。

## 2. 获取应用凭据

FlyIAM 启动时会在 Casdoor 中自动创建组织 `flyiam`、应用 `flyiam` 并回填凭据。
从 FlyIAM 数据库查询：

```bash
PGPASSWORD='<flyiam-db-password>' psql -h <host> -p 5432 -U flyiam -d flyiam -c \
  "SELECT client_id, client_secret FROM casdoor_application WHERE name='flyiam';"
```

或在 FlyIAM/Casdoor 管理后台的应用详情中查看。

## 3. 配置 Consul Manager

```bash
export CASDOOR_ENDPOINT="http://127.0.0.1:8000"         # 后端访问（容器内/内网）
export CASDOOR_PUBLIC_ENDPOINT="http://10.0.88.88:8000" # 浏览器访问（跨域名必填）
export CASDOOR_CLIENT_ID="<从 FlyIAM 获取>"
export CASDOOR_CLIENT_SECRET="<从 FlyIAM 获取>"
export CASDOOR_ORGANIZATION="flyiam"
export CASDOOR_APPLICATION="flyiam"
```

> **跨域名/多机部署要点**：
> - `CASDOOR_PUBLIC_ENDPOINT` 必须是浏览器能直接访问的地址
> - 前端不写死任何 Casdoor 地址，登录时由后端 `/api/auth/login` 按当前访问域名动态生成并 302
> - 换域名后**无需重新构建前端**，只需更新回调白名单

Kubernetes 部署请对应设置 `CONSUL_MGR_CASDOOR_*`（见 [Kubernetes 部署](kubernetes.md)）。

> **Kubernetes 接入方式**：默认采用**外置域名方式**（`casdoorInCluster=false`）——
> 将 `CONSUL_MGR_CASDOOR_ENDPOINT` / `CONSUL_MGR_CASDOOR_PUBLIC_ENDPOINT`
> 设为 Casdoor 对外域名（如 `https://casdoor.example.com`），后端与浏览器均通过该域名访问，
> 外置域名默认走标准端口 **443 / 80**。
> 若 Casdoor 仅在集群内可达，设置 `CONSUL_MGR_CASDOOR_IN_CLUSTER=true`，并使用
> 跨命名空间 FQDN `http://casdoor.flyiam.svc.cluster.local:8000`，
> 或启用 Chart 的 ExternalName 别名使用短名。
> NetworkPolicy 会随形态自动调整：外置按端口（443/80，可配）/CIDR 放行，集群内按
> `namespaceSelector` + `podSelector` 精确放行。

## 4. 回调白名单维护

新增域名时，需要把回调地址加入 Casdoor 应用白名单。推荐在 FlyIAM 侧维护
（FlyIAM 提供「回调地址自动追加」能力），或直接在 Casdoor 管理后台编辑应用
的 `Redirect URIs`：

```
http(s)://<你的域名>/callback
http(s)://<你的域名>/api/auth/callback
```

## 5. 验证

```bash
# 1. 后端下发的运行时配置
curl -s http://localhost:8080/api/auth/config

# 2. 登录跳转（应 302 到 Casdoor）
curl -s -o /dev/null -w "%{redirect_url}\n" http://localhost:8080/api/auth/login

# 3. 浏览器完整走一遍登录
```

## 5.1 登录安全与默认密码

- **OAuth state 校验**：登录时生成随机 state 写入 HttpOnly Cookie，回调时比对，
  防止登录 CSRF（无需配置）。
- **默认密码**：`CASDOOR_DEFAULT_PASSWORD` 不再提供代码默认值，必须显式注入
  （环境变量 / Secret），否则启动即失败；新增用户与重置密码使用该值。
- **用户列表新鲜度**：缓存 TTL 默认 30s（`CASDOOR_USER_CACHE_TTL` 可调），
  并以 SingleFlight 合并并发回源；列表接口支持 `?refresh=1` 强制绕过缓存。

### 跨域部署（前后端不同源）

默认前后端同源（前端嵌入后端），`SameSite=Lax` 即可。仅当前端与后端部署到
不同域名时，需配置：

```bash
# 前端构建时指定后端地址
export VITE_API_BASE_URL="https://consul-api.example.com"

# 后端 Cookie（跨域必须 None + HTTPS）
export CONSUL_MGR_AUTH_COOKIE_SAMESITE="none"
# 跨子域共享时：export CONSUL_MGR_AUTH_COOKIE_DOMAIN=".example.com"

# 后端 CORS（显式列出来源，不能用 *）
export CONSUL_MGR_CORS_ALLOWED_ORIGINS="https://consul.example.com"
```

> ⚠️ `SameSite=None` 会削弱 CSRF 防护，非跨域部署请保持默认 `lax`。
- **令牌校验**：访问令牌经 Casdoor 权威校验（查库存在、未过期、用户未禁用/删除），
  校验结果缓存 60s，不再信任未验签的 JWT 载荷。

## 6. 常见问题

| 现象 | 原因与处理 |
|------|-----------|
| `Redirect URI ... doesn't exist in the allowed Redirect URI list` | 回调地址未加入 Casdoor 白名单，见第 4 节 |
| 浏览器跳到 `localhost:8000` 打不开 | `CASDOOR_PUBLIC_ENDPOINT` 未设置为浏览器可达地址 |
| `Casdoor client_id 不能为空` | 未配置 `CASDOOR_CLIENT_ID/SECRET`，见第 2、3 节 |
| 登录后回到登录页 | 检查 `/api/auth/callback` 返回；查看后端日志 |
| `invalid key: Key must be a PEM...` | 后端已改为不验证签名直接解析载荷，若仍出现请升级镜像 |
| 启动报 `invalid character '<' looking for beginning of value` | `CASDOOR_ENDPOINT` 指向的不是 Casdoor API（返回了 HTML，如前端页面/Ingress 首页）。用 `curl -sS -i "$CASDOOR_ENDPOINT/api/health"` 确认应返回 JSON `{"status":"ok"}`；集群内正确值通常为 `http://casdoor.flyiam.svc.cluster.local:8000` |

## 6.1 用户字段定义同步（可选）

Consul Manager 的用户列表 / 表单由「用户字段定义」驱动（存本地库），
内置字段与 FlyIAM 一致。若 FlyIAM 从数据源同步的用户字段发生变化，
无需改代码，配置后一键同步即可：

**FlyIAM 侧**：登录后在「API 令牌」页点击 **生成令牌**（可设有效期或永久），
复制令牌。

**Consul Manager 侧**：在「人员组织 → 系统设置 → FlyIAM 集成」中填入
FlyIAM 地址与上一步的令牌，保存即生效（无需环境变量 / Chart 配置）。

然后在「人员组织 → 用户字段」点击 **立即同步**（按字段键幂等更新：
新增缺失字段、更新显示名/类型/可见性/排序）。

**自动同步**：同一页面可配置「定时同步 / 启动时同步 / 同步间隔」，
保存后由后台调度器执行（启动时同步一次，之后按间隔检查），
页面展示实时进度与上次执行结果。配置存于数据库，改完即时生效，无需重启。

> 未配置 FlyIAM 对接时页面会提示，其余功能不受影响。
> 同步只更新字段定义，不修改任何用户数据。
> K8s 部署建议把令牌放到 Secret（Chart 已内置该 key）。

## 7. 从内置 Casdoor 迁移

1. 部署 FlyIAM，拿到 Casdoor 地址与 `flyiam` 应用凭据；
2. 将 Consul Manager 的 `CASDOOR_ORGANIZATION` / `CASDOOR_APPLICATION` 改为 `flyiam`；
3. 将 `CASDOOR_CLIENT_ID` / `CASDOOR_CLIENT_SECRET` 替换为 FlyIAM 的值；
4. 在 FlyIAM/Casdoor 中创建用户，并把回调地址加入白名单；
5. 无需迁移 Consul Manager 业务表（服务组 / 实例 / 审计数据与认证无关）。

## 8. 相关文档

- [systemd 部署](systemd.md) · [Docker 部署](docker.md) · [Kubernetes 部署](kubernetes.md)
- FlyIAM 项目：https://github.com/iflyelf/flyiam
