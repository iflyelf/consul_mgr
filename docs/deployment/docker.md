# 部署文档 - Docker

使用已构建的多架构镜像（amd64 / arm64），无需本地编译。

## 方式一：docker compose（推荐）

```bash
# 1. 创建工作目录
mkdir -p consul_mgr && cd consul_mgr

# 2. 下载 docker-compose.yml 与配置模板到 /tmp 再替换
wget -q -c --no-check-certificate -O /tmp/docker-compose.yml \
  "https://down.xiaonuo.live?url=https://raw.githubusercontent.com/iflyelf/consul_mgr/main/deploy/docker/docker-compose.yml"
mv -f /tmp/docker-compose.yml ./docker-compose.yml

wget -q -c --no-check-certificate -O /tmp/config.yaml \
  "https://down.xiaonuo.live?url=https://raw.githubusercontent.com/iflyelf/consul_mgr/main/deploy/config/config.yaml"
mv -f /tmp/config.yaml ./config.yaml

# 3. 修改配置与凭据（端口、数据库、Casdoor 等）
vi ./config.yaml
vi ./docker-compose.yml

# 4. 启动
docker compose up -d

# 5. 查看日志
docker compose logs -f          # 容器 stdout
```

访问 `http://<主机IP>:8080`（端口由 `SERVER_PORT` 控制）。

## 方式二：docker run

```bash
docker run -d \
  --name consul_mgr \
  --network host \
  --restart always \
  -e SERVER_PORT=8080 \
  -e DATABASE_URL="postgresql://user:pass@host:5432/consul_mgr?sslmode=disable" \
  -e JWT_SECRET="change_me_to_long_random_secret" \
  -e ADMIN_USERNAME="admin" \
  -e ADMIN_PASSWORD="change_me" \
  -e CASDOOR_ENDPOINT="http://127.0.0.1:8000" \
  -e CASDOOR_PUBLIC_ENDPOINT="http://your-domain:8000" \
  -e CASDOOR_CLIENT_ID="xxx" \
  -e CASDOOR_CLIENT_SECRET="xxx" \
  -e REDIS_ENABLED="true" \
  -e REDIS_HOST="127.0.0.1" \
  -e REDIS_PORT="6379" \
  -v /etc/consul_mgr/config.yaml:/etc/consul_mgr/config.yaml:ro \
  iflyelf/consul-mgr:latest
```

## 环境变量

与 systemd 部署一致，见 [systemd 部署的环境变量表](systemd.md#环境变量配置)。

关键变量：

| 变量 | 说明 | 默认 |
|------|------|------|
| `SERVER_PORT` | 监听端口（自定义） | `8080` |
| `SERVER_HOST` | 监听地址 | `0.0.0.0` |
| `DATABASE_URL` | PostgreSQL 连接串（必填） | — |
| `JWT_SECRET` | JWT 密钥（必填） | — |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | 管理员（必填） | — |
| `CASDOOR_ENDPOINT` / `CASDOOR_PUBLIC_ENDPOINT` | Casdoor 地址 | — |
| `CASDOOR_CLIENT_ID` / `CASDOOR_CLIENT_SECRET` | Casdoor 凭据 | — |
| `REDIS_*` | 缓存配置 | `true` / `localhost` / `6379` |

## 网络模式说明

compose 示例使用 `network_mode: host`，便于直连宿主机的 PostgreSQL / Redis / Casdoor。
若需使用容器网络，请：
1. 删除 `network_mode: host`
2. 增加 `ports: ["8080:8080"]`
3. 将配置中的 `localhost` 改为对应服务名或主机 IP

## 升级

```bash
docker compose pull
docker compose up -d
```

## 卸载

```bash
docker compose down
# 如需清理挂载数据
# rm -rf ./logs ./data
```

## 相关文档

- [systemd 部署](systemd.md) · [Kubernetes 部署](kubernetes.md) · [Casdoor 配置](casdoor.md)
