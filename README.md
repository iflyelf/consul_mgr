# Consul Manager

> 统一的 Consul 服务注册与配置管理平台 — Go (go-zero) + Vue 3 + PostgreSQL，单二进制交付。

[![Release](https://img.shields.io/github/v/release/iflyelf/consul_mgr)](https://github.com/iflyelf/consul_mgr/releases/latest)
[![License](https://img.shields.io/github/license/iflyelf/consul_mgr)](LICENSE)

## 简介

Consul Manager 面向多 Consul 集群的日常运维，提供「服务组 → 服务 → 实例」的完整生命周期管理，
认证与权限复用 Casdoor，支持服务组级授权与操作审计。前端产物嵌入二进制，部署只需**一个进程、一个端口**。

## 功能特性

**服务组管理**
- 多 Consul 集群配置（地址 / Token）
- **数据中心自动探测**，无需手填
- 连接测试、启停、删除

**Services 管理**
- 服务列表、关键词搜索
- 健康状态总览（服务数 / 实例数 / 健康度）
- 删除服务、批量删除

**Instances 管理**
- 多条件检索（服务组 / 服务名 / 状态 / 关键字）
- 注册 / 编辑 / 删除，**Tags、Meta、健康检查可视化配置**
- 批量注册（支持 IP 段 / CIDR / 范围，如 `10.1.255.24-26:80,10.1.255.0/24:8080`）
- 导入 / 导出（JSON / YAML / CSV，支持强制覆盖）
- 批量删除

**集群与性能**
- **集群去重**：同一实例不会被多节点重复注册
- **跨节点删除**：自动识别并清理其他节点上的实例
- Redis 缓存（不可用时自动降级）

**认证与安全**
- Casdoor OAuth2 登录，支持跨域名/IP 部署
- 服务组级 RBAC、全局管理员
- 操作审计日志

**界面**
- 三套主题：🌞 暖沙米 / 🌊 冷蓝 / 🌙 暗黑
- Mac 圆角风格，H5 自适应

## 快速开始

### Docker（推荐）

```bash
mkdir -p consul_mgr && cd consul_mgr
wget -q -c --no-check-certificate -O docker-compose.yml \
  "https://down.xiaonuo.live?url=https://raw.githubusercontent.com/iflyelf/consul_mgr/main/deploy/docker/docker-compose.yml"
wget -q -c --no-check-certificate -O config.yaml \
  "https://down.xiaonuo.live?url=https://raw.githubusercontent.com/iflyelf/consul_mgr/main/deploy/config/config.yaml"
# 修改凭据后启动
vi config.yaml && docker compose up -d
```

### systemd

见 [部署文档 - systemd](docs/deployment/systemd.md)。

### 本地开发

```bash
cd web && npm install && npm run build && cd ..
export DATABASE_URL="postgresql://user:pass@host:5432/consul_mgr?sslmode=disable"
export JWT_SECRET="your-secret-at-least-32-chars"
export ADMIN_USERNAME=admin ADMIN_PASSWORD=your-password
export CASDOOR_ENDPOINT=http://localhost:8000 \
       CASDOOR_CLIENT_ID=xxx CASDOOR_CLIENT_SECRET=xxx
go run ./cmd/api -c etc/config.yaml
```

访问 `http://localhost:8080`。

## 配置

**零硬编码**：所有配置通过环境变量注入，端口、地址、凭据均可自定义。

| 变量 | 说明 | 默认 |
|------|------|------|
| `SERVER_PORT` | 监听端口 | `8080` |
| `SERVER_HOST` | 监听地址 | `0.0.0.0` |
| `DATABASE_URL` | PostgreSQL 连接串（必填） | — |
| `JWT_SECRET` | JWT 密钥（必填） | — |
| `ADMIN_USERNAME` / `ADMIN_PASSWORD` | 管理员（必填） | — |
| `CASDOOR_ENDPOINT` / `CASDOOR_PUBLIC_ENDPOINT` | Casdoor 地址 | — |
| `CASDOOR_CLIENT_ID` / `CASDOOR_CLIENT_SECRET` | Casdoor 凭据 | — |
| `REDIS_ENABLED` / `REDIS_HOST` / `REDIS_PORT` / `REDIS_PASSWORD` | 缓存 | `true`/`localhost`/`6379`/空 |
| `CONSUL_ADDRESS` / `CONSUL_TOKEN` / `CONSUL_DATACENTER` | 默认 Consul | — / 空 / `dc1` |

完整清单见 [systemd 部署文档](docs/deployment/systemd.md#环境变量配置)。

## 文档

| 分类 | 文档 |
|------|------|
| **设计** | [架构设计](docs/design/architecture.md) · [数据库设计](docs/design/database.md) · [API 设计](docs/design/api.md) |
| **开发** | [开发文档](docs/development/development.md) |
| **测试** | [测试文档](docs/testing/testing.md) |
| **部署** | [systemd](docs/deployment/systemd.md) · [Docker](docs/deployment/docker.md) · [Kubernetes](docs/deployment/kubernetes.md) · [Casdoor](docs/deployment/casdoor.md) · [CI/CD](docs/deployment/github-secrets.md) |

## 技术栈

| 层 | 技术 |
|----|------|
| 后端 | Go 1.26 · go-zero · hashicorp/consul/api |
| 前端 | Vue 3 · Element Plus · Pinia · Vite |
| 存储 | PostgreSQL（业务） · Redis（缓存） |
| 认证 | Casdoor（OAuth2 / RBAC） |

## 目录结构

```
cmd/api/            程序入口
internal/           后端源码（handler/logic/middleware/pkg/svc/config/types）
web/                Vue 3 前端（构建产物嵌入二进制）
deploy/             部署物料（systemd / docker / k8s / sql / config）
docs/               文档（design / development / testing / deployment）
tools/              运维辅助工具
```

## 许可证

[MIT](LICENSE) © iflyelf
