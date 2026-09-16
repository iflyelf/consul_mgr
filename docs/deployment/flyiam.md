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

## 6. 常见问题

| 现象 | 原因与处理 |
|------|-----------|
| `Redirect URI ... doesn't exist in the allowed Redirect URI list` | 回调地址未加入 Casdoor 白名单，见第 4 节 |
| 浏览器跳到 `localhost:8000` 打不开 | `CASDOOR_PUBLIC_ENDPOINT` 未设置为浏览器可达地址 |
| `Casdoor client_id 不能为空` | 未配置 `CASDOOR_CLIENT_ID/SECRET`，见第 2、3 节 |
| 登录后回到登录页 | 检查 `/api/auth/callback` 返回；查看后端日志 |
| `invalid key: Key must be a PEM...` | 后端已改为不验证签名直接解析载荷，若仍出现请升级镜像 |

## 7. 从内置 Casdoor 迁移

1. 部署 FlyIAM，拿到 Casdoor 地址与 `flyiam` 应用凭据；
2. 将 Consul Manager 的 `CASDOOR_ORGANIZATION` / `CASDOOR_APPLICATION` 改为 `flyiam`；
3. 将 `CASDOOR_CLIENT_ID` / `CASDOOR_CLIENT_SECRET` 替换为 FlyIAM 的值；
4. 在 FlyIAM/Casdoor 中创建用户，并把回调地址加入白名单；
5. 无需迁移 Consul Manager 业务表（服务组 / 实例 / 审计数据与认证无关）。

## 8. 相关文档

- [systemd 部署](systemd.md) · [Docker 部署](docker.md) · [Kubernetes 部署](kubernetes.md)
- FlyIAM 项目：https://github.com/iflyelf/flyiam
