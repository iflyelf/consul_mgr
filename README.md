# Consul_Mgr - Consul 服务管理平台

基于 Go (go-zero) + Vue 3 开发的 Consul 服务管理平台，提供直观的 Web 界面管理 Consul 服务和实例。

## ✨ 特性

- 🚀 **开箱即用**: 静态编译，单一二进制文件，无需依赖
- 🔐 **权限管理**: 完整的 RBAC 权限系统，支持用户、角色、权限管理
- 🏢 **多集群支持**: 支持管理多个 Consul 集群
- 📊 **服务组管理**: 服务组概念，灵活组织和授权
- 🔧 **实例管理**: 可视化配置 Tags、Meta、健康检查
- 📝 **审计日志**: 完整记录所有操作，可追溯
- 🎨 **三主题切换**: 浅色主题☀️ / 深色主题🌙 / 蓝色主题🔵
- 📱 **H5 自适应**: 完全支持移动端访问
- ⚡ **零硬编码**: 所有配置通过环境变量或配置文件

## 🏗️ 架构

- **后端**: Go 1.26+ + go-zero
- **前端**: Vue 3 + Element Plus + Vite
- **数据库**: PostgreSQL
- **认证**: JWT + bcrypt

## 🚀 快速开始

### 环境要求

- PostgreSQL 数据库
- Go 1.26+ (开发)
- Node.js 18+ (前端开发)

### 一键启动（开发模式）

```bash
# 1. 克隆仓库
git clone https://github.com/iflyelf/consul_mgr.git
cd consul_mgr

# 2. 设置环境变量（必填）
export DATABASE_URL="postgresql://user:password@host:port/dbname?sslmode=disable"
export ADMIN_PASSWORD="your_strong_password"

# 3. 设置环境变量（可选）
export JWT_SECRET="your_jwt_secret_min_32_chars"  # 不设置则使用默认值
export ADMIN_USERNAME="admin"                     # 不设置则使用 admin
export ADMIN_EMAIL="admin@example.com"            # 不设置则自动生成

# 4. 编译
make build

# 5. 一键启动（后端 + 前端）
./start.sh
```

启动后访问: http://localhost:5173

### 生产部署

#### 方式一: systemd（推荐）

```bash
# 1. 编译
make build

# 2. 复制文件
sudo cp consul_mgr /usr/local/bin/
sudo mkdir -p /etc/consul_mgr
sudo cp etc/config.yaml /etc/consul_mgr/

# 3. 编辑配置
sudo vi /etc/consul_mgr/config.yaml

# 4. 创建 systemd 服务
sudo cp deploy/consul_mgr.service /etc/systemd/system/
sudo systemctl daemon-reload
sudo systemctl enable consul_mgr
sudo systemctl start consul_mgr

# 5. 查看状态
sudo systemctl status consul_mgr
```

#### 方式二: Docker Compose

```bash
# 1. 编辑 docker-compose.yml 设置环境变量
vi docker-compose.yml

# 2. 启动
docker-compose up -d

# 3. 查看日志
docker-compose logs -f
```

## 🔧 配置说明

### 环境变量（优先级最高）

| 变量名 | 必填 | 说明 | 示例 |
|--------|------|------|------|
| `DATABASE_URL` | ✅ | 数据库连接字符串 | `postgresql://user:pass@host:5432/db` |
| `ADMIN_PASSWORD` | ✅ | 管理员密码 | `StrongP@ssw0rd` |
| `JWT_SECRET` | ❌ | JWT 密钥（至少32字符） | `your_jwt_secret_key_min_32_chars` |
| `ADMIN_USERNAME` | ❌ | 管理员用户名 | `admin` |
| `ADMIN_EMAIL` | ❌ | 管理员邮箱 | `admin@example.com` |

### 配置文件 (etc/config.yaml)

如果不使用环境变量，可以在配置文件中设置：

```yaml
Server:
  Host: 0.0.0.0
  Port: 8080

Database:
  Driver: postgres
  Source: postgresql://user:password@localhost:5432/consul_mgr?sslmode=disable
  MaxOpenConns: 100
  MaxIdleConns: 10

JWT:
  Secret: consul_mgr_jwt_secret_2024_min_32_chars
  AccessExpire: 7200  # 2小时
  RefreshExpire: 604800  # 7天

Admin:
  Username: admin
  Password: your_password
  Email: admin@example.com
```

**注意**: 环境变量优先级高于配置文件！

## 📖 使用指南

### 1. 登录系统

访问 http://localhost:5173，使用配置的管理员账号登录。

### 2. 管理服务组

1. 点击"添加服务组"
2. 填写表单：
   - **名称**: 服务组名称（如：生产环境）
   - **代码**: 唯一标识符（如：prod）
   - **Consul 地址**: http://consul.example.com:8500
   - **Consul Token**: 可选，如果 Consul 启用了 ACL
   - **数据中心**: 可选，默认 dc1

3. 支持的操作：
   - **编辑**: 修改服务组配置
   - **测试**: 测试 Consul 连接
   - **删除**: 删除服务组

### 3. 主题切换

点击右上角太阳图标，选择主题：
- ☀️ **浅色主题**: 适合白天使用
- 🌙 **深色主题**: 适合夜间使用，护眼
- 🔵 **蓝色主题**: 清新蓝色风格

主题选择会自动保存到本地。

### 4. 移动端访问

系统完全支持手机访问，布局会自动适配小屏幕。

## 🛠️ 开发

### 后端开发

```bash
# 1. 安装依赖
go mod download

# 2. 设置环境变量
export DATABASE_URL="postgresql://user:pass@localhost:5432/consul_mgr?sslmode=disable"
export ADMIN_PASSWORD="your_password"

# 3. 运行
go run cmd/api/main.go -c etc/config.yaml

# 或使用 Makefile
make run
```

### 前端开发

```bash
cd web

# 1. 安装依赖
npm install

# 2. 启动开发服务器
npm run dev

# 访问 http://localhost:5173
```

### 构建

```bash
# 构建后端
make build

# 构建前端
cd web && npm run build

# 跨平台构建
make build-linux
make build-darwin
make build-windows
```

## 📁 项目结构

```
consul_mgr/
├── cmd/api/              # 主程序入口
├── internal/
│   ├── config/          # 配置
│   ├── handler/         # HTTP 处理器
│   ├── logic/           # 业务逻辑
│   ├── middleware/      # 中间件
│   ├── model/           # 数据模型
│   ├── pkg/             # 工具包
│   │   ├── consul/      # Consul 客户端
│   │   ├── jwt/         # JWT 工具
│   │   ├── password/    # 密码加密
│   │   └── response/    # 统一响应
│   ├── svc/             # 服务上下文
│   └── types/           # 类型定义
├── deploy/              # 部署配置
│   ├── sql/            # 数据库脚本
│   ├── config/         # 配置模板
│   └── docker/         # Docker 配置
├── web/                 # 前端项目
│   ├── src/
│   │   ├── api/        # API 接口
│   │   ├── assets/     # 静态资源
│   │   ├── components/ # 组件
│   │   ├── router/     # 路由
│   │   ├── store/      # 状态管理
│   │   ├── utils/      # 工具函数
│   │   └── views/      # 页面
│   ├── index.html
│   ├── package.json
│   └── vite.config.js
├── etc/                 # 配置文件
├── Makefile            # 构建脚本
├── start.sh            # 启动脚本
└── README.md
```

## 🔌 API 文档

### 认证相关

```bash
# 登录
POST /api/auth/login
Content-Type: application/json
{
  "username": "admin",
  "password": "your_password"
}

# 登出
POST /api/auth/logout
Authorization: Bearer <token>

# 获取用户信息
GET /api/auth/info
Authorization: Bearer <token>
```

### 服务组管理

```bash
# 获取服务组列表
GET /api/groups?page=1&page_size=20
Authorization: Bearer <token>

# 创建服务组
POST /api/groups
Authorization: Bearer <token>
Content-Type: application/json
{
  "name": "生产环境",
  "code": "prod",
  "description": "生产环境 Consul",
  "consul_address": "http://consul.prod.com:8500",
  "consul_token": "optional-token",
  "consul_datacenter": "dc1"
}

# 获取服务组详情
GET /api/groups/:id
Authorization: Bearer <token>

# 更新服务组
PUT /api/groups/:id
Authorization: Bearer <token>
Content-Type: application/json
{
  "description": "更新后的描述"
}

# 删除服务组
DELETE /api/groups/:id
Authorization: Bearer <token>

# 测试连接
POST /api/groups/:id/test
Authorization: Bearer <token>
```

## 🎯 核心功能

### ✅ 已实现

- ✅ 用户认证（JWT）
- ✅ 服务组 CRUD
- ✅ Consul 客户端封装
- ✅ 多集群管理
- ✅ 三主题切换
- ✅ 响应式布局
- ✅ 数据库自动初始化
- ✅ 零硬编码配置

### 🚧 规划中

- ⏳ Consul 服务管理
- ⏳ 实例注册/注销
- ⏳ 健康检查配置
- ⏳ 批量导入导出
- ⏳ 审计日志查询
- ⏳ 用户权限管理

## 🐛 故障排查

### 后端启动失败

```bash
# 查看日志
tail -f /tmp/consul_mgr.log

# 常见问题：
# 1. 数据库连接失败 -> 检查 DATABASE_URL
# 2. 端口被占用 -> 修改配置文件中的端口
# 3. 权限不足 -> 使用 sudo 或修改文件权限
```

### 前端无法访问

```bash
# 查看日志
tail -f /tmp/vite.log

# 常见问题：
# 1. 端口 3000 被占用 -> 修改 vite.config.js
# 2. API 代理失败 -> 检查后端是否启动
# 3. 依赖安装失败 -> 删除 node_modules 重新安装
```

### 数据库问题

```bash
# 手动初始化数据库
psql -h host -U user -d dbname -f deploy/sql/schema.sql
psql -h host -U user -d dbname -f deploy/sql/init_data.sql

# 重置数据库（危险！会删除所有数据）
psql -h host -U user -d dbname -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
```

## 📝 变更日志

### v0.3.0 (2026-09-14)

- ✨ 完整的前端界面
- ✨ 三主题切换支持
- ✨ 响应式布局（H5 自适应）
- ✨ 服务组完整 CRUD
- ✨ 零硬编码配置
- ✨ 一键启动脚本
- 🐛 修复路径参数解析问题
- 🐛 修复 SQL 更新语句错误

### v0.2.0

- ✨ Consul 客户端封装
- ✨ 多集群管理
- ✨ JWT 认证系统
- ✨ 数据库自动初始化

### v0.1.0

- ✨ 项目初始化
- ✨ 基础架构搭建

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/AmazingFeature`)
3. 提交更改 (`git commit -m 'Add some AmazingFeature'`)
4. 推送到分支 (`git push origin feature/AmazingFeature`)
5. 开启 Pull Request

## 📄 许可证

MIT License

Copyright (c) 2026 iflyelf

## 📧 联系方式

- 作者: iflyelf
- 邮箱: iflyelf@gmail.com
- GitHub: https://github.com/iflyelf/consul_mgr

## 🙏 鸣谢

- [go-zero](https://github.com/zeromicro/go-zero) - 优秀的 Go 微服务框架
- [Vue 3](https://vuejs.org/) - 渐进式 JavaScript 框架
- [Element Plus](https://element-plus.org/) - Vue 3 组件库
- [Consul](https://www.consul.io/) - 服务网格解决方案

---

⭐ 如果这个项目对你有帮助，请给一个 Star！

mv -f /tmp/config.yaml ./config.yaml

# 3. 修改配置
vi ./config.yaml

# 4. 启动
docker compose up -d

# 5. 查看日志
docker compose logs -f
```

## ⚙️ 配置说明

### 环境变量

```bash
# 数据库配置
export DATABASE_URL="postgresql://iflyelf:password@10.0.51.88:6000/consul_mgr?sslmode=disable"

# JWT 密钥（必须设置，最少 32 字符）
export JWT_SECRET="your-super-secret-jwt-key-min-32-chars"

# 管理员密码（首次启动必须设置）
export ADMIN_PASSWORD="your-admin-password"
export ADMIN_USERNAME="iflyelf"  # 可选，默认 iflyelf
export ADMIN_EMAIL="iflyelf@gmail.com"  # 可选
```

### 配置文件 (config.yaml)

详见 `deploy/config/config.yaml`

## 🗄️ 数据库初始化

应用启动时会**自动检查并创建**数据库表结构、初始权限、角色和管理员账号。

如需手动初始化：

```bash
# 连接到 PostgreSQL
psql postgresql://iflyelf:password@10.0.51.88:6000/consul_mgr

# 执行初始化脚本
\i deploy/sql/schema.sql
\i deploy/sql/init_data.sql
```

## 📖 使用指南

### 首次登录

访问 `http://your-server:8080`，使用管理员账号登录：

- 用户名: `iflyelf`（或您配置的 ADMIN_USERNAME）
- 密码: 您设置的 ADMIN_PASSWORD

### 创建服务组

1. 登录后进入「服务组管理」
2. 点击「创建服务组」
3. 填写服务组信息（名称、Consul 地址、Token）
4. 点击「测试连接」验证配置
5. 保存

### 管理 Consul 服务

1. 进入「Consul 服务」页面
2. 选择服务组查看服务列表
3. 点击服务查看实例详情
4. 支持批量删除服务

### 管理 Consul 实例

1. 进入「Consul 实例」页面
2. 点击「注册实例」添加新实例
3. 配置 Tags、Meta、健康检查
4. 支持批量导入/导出/删除

### 用户与权限管理

1. 进入「用户管理」创建用户
2. 进入「角色管理」创建角色并分配权限
3. 为角色授权服务组访问权限
4. 为用户分配角色

## 🛠️ 从源码构建

```bash
# 克隆代码
git clone https://github.com/iflyelf/consul_mgr.git
cd consul_mgr

# 安装依赖
go mod tidy

# 本地构建
make build

# 交叉编译
make build-all

# 运行
./consul_mgr -c deploy/config/config.yaml
```

## 📝 开发

```bash
# 运行开发服务器
make run

# 运行测试
make test

# 代码格式化
make fmt
```

## 🐳 Docker 构建

```bash
# 构建镜像
docker build -t consul_mgr:latest .

# 运行容器
docker run -d \
  --name consul_mgr \
  -p 8080:8080 \
  -e DATABASE_URL="postgresql://..." \
  -e JWT_SECRET="..." \
  -e ADMIN_PASSWORD="..." \
  consul_mgr:latest
```

## 📄 License

MIT © iflyelf

## 🤝 贡献

欢迎提交 Issue 和 Pull Request！

## 📧 联系

- Email: iflyelf@gmail.com
- GitHub: [@iflyelf](https://github.com/iflyelf)
