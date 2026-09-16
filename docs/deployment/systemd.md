# 部署文档 - systemd（推荐）

直接跑在宿主机，适合长期稳定的单机部署。

## 前置条件

- Linux（Debian / Ubuntu 推荐）
- PostgreSQL 14+、Redis 6+（可选）、Casdoor（认证服务）

## 部署步骤

```bash
# 1. 下载二进制到 /tmp 并解压
wget -q -c --no-check-certificate -O /tmp/consul_mgr.tar.gz \
  "https://down.xiaonuo.live?url=https://github.com/iflyelf/consul_mgr/releases/latest/download/consul_mgr-linux-amd64.tar.gz"
tar -xzf /tmp/consul_mgr.tar.gz -C /tmp

# 2. 替换二进制（mv -f 原子覆盖）
mv -f /tmp/consul_mgr-linux-amd64 /usr/local/bin/consul_mgr
chmod +x /usr/local/bin/consul_mgr

# 3. 下载配置模板到 /tmp 再替换
wget -q -c --no-check-certificate -O /tmp/config.yaml \
  "https://down.xiaonuo.live?url=https://raw.githubusercontent.com/iflyelf/consul_mgr/main/deploy/config/config.yaml"
mkdir -p /etc/consul_mgr
mv -f /tmp/config.yaml /etc/consul_mgr/config.yaml

# 4. 编辑配置（务必修改数据库、管理员密码与 Casdoor 凭据）
vi /etc/consul_mgr/config.yaml

# 5. 下载 systemd 单元到 /tmp 再替换
wget -q -c --no-check-certificate -O /tmp/consul_mgr.service \
  "https://down.xiaonuo.live?url=https://raw.githubusercontent.com/iflyelf/consul_mgr/main/deploy/systemd/consul_mgr.service"
mv -f /tmp/consul_mgr.service /etc/systemd/system/consul_mgr.service

# 6. 启动服务
systemctl daemon-reload
systemctl enable --now consul_mgr

# 7. 查看状态
systemctl status consul_mgr
journalctl -u consul_mgr -f
```

## 环境变量配置

配置文件中的参数均可用环境变量覆盖（推荐用 `EnvironmentFile` 管理）。

| 变量 | 说明 | 默认 |
|------|------|------|
| `SERVER_HOST` | 监听地址 | `0.0.0.0` |
| `SERVER_PORT` | 监听端口 | `8080` |
| `SERVER_MODE` | 运行模式 `dev\|test\|rt\|pre\|pro` | `release` |
| `DATABASE_URL` | PostgreSQL 连接串（必填） | — |
| `JWT_SECRET` | JWT 密钥（必填，建议 32+ 位） | — |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | 管理员账号（必填） | — |
| `CASDOOR_ENDPOINT` | Casdoor 后端地址（必填） | — |
| `CASDOOR_PUBLIC_ENDPOINT` | 浏览器可达地址（跨域名部署必填） | 同 Endpoint |
| `CASDOOR_CLIENT_ID` / `CASDOOR_CLIENT_SECRET` | Casdoor 凭据（必填） | — |
| `REDIS_ENABLED` / `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` | 缓存 | `true` / `localhost` / `6379` / 空 |
| `CONSUL_ADDRESS` / `CONSUL_TOKEN` / `CONSUL_DATACENTER` | 默认 Consul | — / 空 / `dc1` |
| `CONSUL_MAX_CONCURRENCY` | 批量查询/操作并发上限 | `16` |

### 使用 EnvironmentFile（推荐）

```bash
cat > /etc/consul_mgr/env <<'EOF'
SERVER_PORT=8080
DATABASE_URL=postgresql://user:pass@host:5432/consul_mgr?sslmode=disable
JWT_SECRET=change_me_to_long_random_secret
ADMIN_USERNAME=admin
ADMIN_PASSWORD=change_me
CASDOOR_ENDPOINT=http://127.0.0.1:8000
CASDOOR_PUBLIC_ENDPOINT=http://your-domain:8000
CASDOOR_CLIENT_ID=xxx
CASDOOR_CLIENT_SECRET=xxx
REDIS_ENABLED=true
REDIS_HOST=127.0.0.1
REDIS_PORT=6379
REDIS_PASSWORD=
EOF
chmod 600 /etc/consul_mgr/env
```

然后在 `consul_mgr.service` 的 `[Service]` 段增加：

```ini
EnvironmentFile=-/etc/consul_mgr/env
```

## 升级

```bash
wget -q -c --no-check-certificate -O /tmp/consul_mgr.tar.gz \
  "https://down.xiaonuo.live?url=https://github.com/iflyelf/consul_mgr/releases/latest/download/consul_mgr-linux-amd64.tar.gz"
tar -xzf /tmp/consul_mgr.tar.gz -C /tmp
systemctl stop consul_mgr
mv -f /tmp/consul_mgr-linux-amd64 /usr/local/bin/consul_mgr
systemctl start consul_mgr
systemctl status consul_mgr
```

## 卸载

```bash
systemctl disable --now consul_mgr
rm -f /etc/systemd/system/consul_mgr.service /usr/local/bin/consul_mgr
rm -rf /etc/consul_mgr
systemctl daemon-reload
```

## 常见问题

| 现象 | 排查 |
|------|------|
| 启动即失败 | `journalctl -u consul_mgr -n 50` 查看配置/数据库连接错误 |
| 无法读取配置文件 | 确认 `ReadWritePaths` 已包含 `/etc/consul_mgr`（单元文件已含） |
| 端口被占用 | 修改 `SERVER_PORT` 或 `ss -tlnp \| grep 8080` |
| 登录跳转失败 | 检查 `CASDOOR_PUBLIC_ENDPOINT` 是否为浏览器可达地址 |

## 相关文档

- [Docker 部署](docker.md) · [Kubernetes 部署](kubernetes.md) · [Casdoor 配置](casdoor.md)
