# Consul Manager

> 基于 Go + Vue 3 的 Consul 服务管理平台

[![Go Version](https://img.shields.io/badge/Go-1.26+-blue.svg)](https://golang.org)
[![Vue Version](https://img.shields.io/badge/Vue-3.4+-brightgreen.svg)](https://vuejs.org)
[![License](https://img.shields.io/badge/License-MIT-green.svg)](LICENSE)

## 特性

- 🔐 **零硬编码** - 所有配置通过环境变量
- 🏢 **多集群管理** - 支持管理多个 Consul 集群
- 📊 **服务管理** - Consul Services 增删改查
- 🔧 **实例管理** - 实例注册、更新、删除、批量操作
- 📦 **导入导出** - 支持 JSON/YAML/CSV 格式
- 🎨 **三主题** - 浅色/深色/蓝色主题切换
- 📱 **响应式** - 完全适配移动端

## 快速开始

### 环境要求

- Go 1.26+
- Node.js 18+
- PostgreSQL

### 安装

```bash
# 克隆项目
git clone https://github.com/iflyelf/consul_mgr.git
cd consul_mgr

# 配置环境变量
export DATABASE_URL="postgresql://user:password@localhost:5432/consul_mgr?sslmode=disable"
export ADMIN_PASSWORD="your_password"
export JWT_SECRET="your_jwt_secret_min_32_chars"

# 编译并启动
make build
./consul_mgr -c etc/config.yaml
```

### Docker 部署

```bash
docker-compose up -d
```

## 环境变量

| 变量名 | 必填 | 默认值 | 说明 |
|--------|------|--------|------|
| `DATABASE_URL` | ✅ | - | 数据库连接字符串 |
| `ADMIN_PASSWORD` | ✅ | - | 管理员密码 |
| `JWT_SECRET` | ❌ | - | JWT 密钥(至少32字符) |
| `ADMIN_USERNAME` | ❌ | admin | 管理员用户名 |
| `ADMIN_EMAIL` | ❌ | - | 管理员邮箱 |
| `SERVER_HOST` | ❌ | 0.0.0.0 | 服务监听地址 |
| `SERVER_PORT` | ❌ | 8080 | 服务端口 |
| `WEB_PORT` | ❌ | 5173 | 前端端口 |
| `WEB_HOST` | ❌ | 0.0.0.0 | 前端监听地址 |
| `API_URL` | ❌ | http://localhost:8080 | 后端 API 地址 |

## API 文档

### 认证

```bash
# 登录
POST /api/auth/login
{
  "username": "admin",
  "password": "your_password"
}

# 获取用户信息
GET /api/auth/info
Authorization: Bearer <token>
```

### 服务组管理

```bash
# 获取服务组列表
GET /api/groups?page=1&page_size=20

# 创建服务组
POST /api/groups
{
  "name": "生产环境",
  "code": "prod",
  "consul_address": "http://consul.example.com:8500",
  "consul_token": "optional",
  "consul_datacenter": "dc1"
}

# 测试连接
POST /api/groups/:id/test
```

### Consul Services

```bash
# 获取服务列表
GET /api/services?group_id=1&keyword=api

# 获取服务详情
GET /api/services/detail?group_id=1&service=api-service

# 删除服务
DELETE /api/services?group_id=1&service=api-service

# 批量删除
POST /api/services/batch-delete
{
  "group_id": 1,
  "services": ["service1", "service2"]
}
```

### Consul Instances

```bash
# 获取实例列表
GET /api/instances?group_id=1&service=api-service

# 注册实例
POST /api/instances
{
  "group_id": 1,
  "service": "api-service",
  "id": "api-1",
  "address": "192.168.1.100",
  "port": 8080,
  "tags": ["v1", "prod"],
  "meta": {"version": "1.0.0"}
}

# 更新实例
PUT /api/instances?group_id=1&instance_id=api-1
{
  "tags": ["v2", "prod"],
  "meta": {"version": "2.0.0"}
}

# 删除实例
DELETE /api/instances?group_id=1&instance_id=api-1

# 批量删除
POST /api/instances/batch-delete
{
  "group_id": 1,
  "instance_ids": ["api-1", "api-2"]
}

# 导出实例
GET /api/instances/export?group_id=1&service=api-service&format=json

# 导入实例
POST /api/instances/import
Content-Type: multipart/form-data
- group_id: 1
- format: json
- file: instances.json
```

## 开发

### 后端开发

```bash
# 安装依赖
go mod download

# 运行
go run cmd/api/main.go -c etc/config.yaml

# 或使用 make
make run

# 测试
make test

# 构建
make build
```

### 前端开发

```bash
cd web

# 安装依赖
npm install

# 开发模式
npm run dev

# 构建
npm run build
```

### 跨平台构建

```bash
# Linux
make build-linux

# macOS  
make build-darwin

# Windows
make build-windows

# 所有平台
make build-all
```

## 项目结构

```
consul_mgr/
├── cmd/api/              # 主程序入口
├── internal/
│   ├── config/          # 配置
│   ├── handler/         # HTTP 处理器
│   │   ├── auth/       # 认证
│   │   ├── group/      # 服务组
│   │   ├── service/    # Consul 服务
│   │   └── instance/   # Consul 实例
│   ├── logic/           # 业务逻辑
│   │   ├── service/    # 服务逻辑
│   │   └── instance/   # 实例逻辑
│   ├── model/           # 数据模型
│   ├── middleware/      # 中间件
│   ├── pkg/             # 工具包
│   │   ├── consul/     # Consul 客户端
│   │   ├── jwt/        # JWT
│   │   ├── password/   # 密码加密
│   │   └── response/   # 统一响应
│   ├── svc/             # 服务上下文
│   └── types/           # 类型定义
├── web/                 # 前端项目
│   ├── src/
│   │   ├── api/        # API 接口
│   │   ├── views/      # 页面
│   │   ├── router/     # 路由
│   │   ├── store/      # 状态管理
│   │   └── utils/      # 工具函数
│   └── vite.config.js
├── deploy/              # 部署配置
│   ├── sql/            # 数据库脚本
│   ├── docker/         # Docker 配置
│   └── systemd/        # systemd 服务
├── etc/                 # 配置文件
├── Makefile            # 构建脚本
└── README.md           # 本文件
```

## 数据库

项目使用 PostgreSQL，包含 8 张表：

- `users` - 用户表
- `roles` - 角色表
- `permissions` - 权限表
- `user_roles` - 用户角色关联
- `role_permissions` - 角色权限关联
- `service_groups` - 服务组表
- `service_group_users` - 服务组用户关联
- `audit_logs` - 审计日志

首次启动会自动创建表结构和初始数据。

## 配置示例

### 配置文件 (etc/config.yaml)

```yaml
Name: consul_mgr
Host: ${SERVER_HOST:0.0.0.0}
Port: ${SERVER_PORT:8080}
Mode: ${SERVER_MODE:dev}

Database:
  DSN: ${DATABASE_URL}

JWT:
  Secret: ${JWT_SECRET}
  AccessExpire: 7200
  RefreshExpire: 604800

Admin:
  Username: ${ADMIN_USERNAME:admin}
  Password: ${ADMIN_PASSWORD}
  Email: ${ADMIN_EMAIL}

Web:
  Port: ${WEB_PORT:5173}
  Host: ${WEB_HOST:0.0.0.0}
```



## 故障排查

### 后端启动失败

```bash
# 查看日志
./consul_mgr -c etc/config.yaml

# 检查数据库连接
psql $DATABASE_URL -c "SELECT 1"

# 检查端口占用
lsof -i :8080
```

### 前端无法访问

```bash
# 检查后端是否启动
curl http://localhost:8080/health

# 检查端口占用
lsof -i :5173

# 查看前端日志
cd web && npm run dev
```

## 生产部署

### 方式一：systemd（推荐，直接跑在宿主机）

```bash
# 1. 下载二进制到 /tmp 并解压
wget -q -c --no-check-certificate -O /tmp/consul_mgr.tar.gz \
  "https://github.com/iflyelf/consul_mgr/releases/latest/download/consul_mgr-linux-amd64.tar.gz"
tar -xzf /tmp/consul_mgr.tar.gz -C /tmp

# 2. 替换二进制(mv -f 原子覆盖)
mv -f /tmp/consul_mgr-linux-amd64 /usr/local/bin/consul_mgr
chmod +x /usr/local/bin/consul_mgr

# 3. 创建配置目录并编辑配置
mkdir -p /etc/consul_mgr
vi /etc/consul_mgr/config.yaml

# 4. 配置环境变量(在配置文件中设置或导出)
# 必填:
#   DATABASE_URL="postgresql://user:pass@host:5432/consul_mgr"
#   ADMIN_PASSWORD="your_password"
# 可选:
#   JWT_SECRET="your_jwt_secret_min_32_chars"

# 5. 下载 systemd 单元到 /tmp 再替换
wget -q -c --no-check-certificate -O /tmp/consul_mgr.service \
  "https://raw.githubusercontent.com/iflyelf/consul_mgr/main/deploy/systemd/consul_mgr.service"
mv -f /tmp/consul_mgr.service /etc/systemd/system/consul_mgr.service

# 6. 启动服务
systemctl daemon-reload
systemctl enable --now consul_mgr

# 7. 查看状态
systemctl status consul_mgr
journalctl -u consul_mgr -f
```

### 方式二：Docker（host 网络 + 特权）

```bash
# 1. 创建工作目录
mkdir -p consul_mgr && cd consul_mgr

# 2. 下载 docker-compose.yml 与配置模板到 /tmp 再替换
wget -q -c --no-check-certificate -O /tmp/docker-compose.yml \
  "https://raw.githubusercontent.com/iflyelf/consul_mgr/main/deploy/docker/docker-compose.yml"
mv -f /tmp/docker-compose.yml ./docker-compose.yml

wget -q -c --no-check-certificate -O /tmp/config.yaml \
  "https://raw.githubusercontent.com/iflyelf/consul_mgr/main/etc/config.yaml"
mv -f /tmp/config.yaml ./config.yaml

# 3. 修改配置文件和环境变量
vi ./config.yaml
vi ./docker-compose.yml  # 修改环境变量

# 4. 启动
docker compose up -d

# 5. 查看日志
docker compose logs -f          # 容器 stdout
tail -f ./logs/consul_mgr.log   # 文件日志(如果配置了)
```

### Nginx 反向代理

```nginx
server {
    listen 80;
    server_name consul_mgr.example.com;

    location / {
        proxy_pass http://localhost:5173;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    location /api {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

## 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 许可证

[MIT License](LICENSE)

## 联系方式

- 作者: iflyelf
- 邮箱: iflyelf@gmail.com
- GitHub: https://github.com/iflyelf/consul_mgr

---

⭐ 如果这个项目对你有帮助，请给一个 Star！
