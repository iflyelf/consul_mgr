# 部署文档 - Casdoor 认证配置

Consul Manager 的认证（OAuth2 授权码模式）完全依赖 Casdoor，本文说明如何配置。

## 1. 部署 Casdoor

Casdoor 可独立部署（Docker / 二进制）。最小示例：

```bash
docker run -d --name casdoor --network host \
  -e driverName=postgres \
  -e dataSourceName="user=xxx password=xxx host=127.0.0.1 port=5432 sslmode=disable dbname=consul_mgr" \
  casbin/casdoor:latest
```

> 生产环境建议为 Casdoor 使用**独立数据库**，并配置持久化与 TLS。

## 2. 创建应用

登录 Casdoor 管理后台（默认 `admin/123`，首次登录请立即修改）：

1. **组织**：创建或使用已有组织（如 `consul_mgr`）
2. **应用**：
   - 名称：如 `consul_manager`
   - 客户端 ID / 密钥：记录下来，填入 Consul Manager 的 `CASDOOR_CLIENT_ID` / `CASDOOR_CLIENT_SECRET`
   - **重定向 URL**（关键，必须包含）：
     ```
     http://<你的域名>:8080/callback
     http://<你的域名>:8080/api/auth/callback
     ```

3. **用户**：为用户设置账号密码（或在组织下创建）。

## 3. 配置 Consul Manager

```bash
export CASDOOR_ENDPOINT="http://127.0.0.1:8000"        # 后端访问（容器内/内网）
export CASDOOR_PUBLIC_ENDPOINT="http://10.0.88.88:8000" # 浏览器访问（跨域名必填）
export CASDOOR_CLIENT_ID="007c84a681153f2f6c48"
export CASDOOR_CLIENT_SECRET="xxxxxxxx"
export CASDOOR_ORGANIZATION="consul_mgr"
export CASDOOR_APPLICATION="consul_manager"
```

> **跨域名/多机部署要点**：
> - `CASDOOR_PUBLIC_ENDPOINT` 必须是浏览器能直接访问的地址
> - 前端不再写死任何 Casdoor 地址，登录时由后端 `/api/auth/login` 按当前访问域名动态生成并 302
> - 因此换域名后**无需重新构建前端**，只需更新回调白名单

## 4. 回调白名单维护

新增域名时，使用内置工具批量追加回调地址：

```bash
export DATABASE_URL="postgresql://user:pass@host:5432/consul_mgr?sslmode=disable"
export CASDOOR_APPLICATION="app-built-in"   # 你的应用名
go run tools/add_redirect_uri.go http://10.0.88.88:8080 https://consul.example.com
```

参数可以是「源」（工具自动补 `/callback`），也可以是完整回调地址。

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
| `Redirect URI ... doesn't exist in the allowed Redirect URI list` | 回调地址未加入 Casdoor 白名单，执行第 4 步 |
| 浏览器跳到 `localhost:8000` 打不开 | `CASDOOR_PUBLIC_ENDPOINT` 未设置为浏览器可达地址 |
| 登录后回到登录页 | 检查 `/api/auth/callback` 返回；查看后端日志 |
| `invalid key: Key must be a PEM...` | 后端已改为不验证签名直接解析载荷，若仍出现请升级镜像 |

## 7. 相关文档

- [systemd 部署](systemd.md) · [Docker 部署](docker.md) · [Kubernetes 部署](kubernetes.md)
