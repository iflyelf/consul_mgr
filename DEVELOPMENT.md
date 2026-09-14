# Consul_Mgr 开发指南

## 🎯 项目架构

### 技术栈
- **后端**: Go 1.26+ + go-zero
- **前端**: Vue 3 + Element Plus（待实现）
- **数据库**: PostgreSQL
- **认证**: JWT + bcrypt

### 目录结构说明
```
consul_mgr/
├── cmd/api/                    # 主程序入口
├── internal/                   # 内部代码（不对外暴露）
│   ├── config/                 # 配置结构
│   ├── handler/                # HTTP 处理器（路由处理）
│   ├── logic/                  # 业务逻辑（核心逻辑）
│   ├── model/                  # 数据模型
│   ├── middleware/             # 中间件
│   ├── pkg/                    # 内部工具包
│   ├── svc/                    # 服务上下文
│   └── types/                  # 类型定义
├── deploy/                     # 部署相关
│   ├── sql/                    # 数据库脚本
│   ├── config/                 # 配置模板
│   ├── systemd/                # systemd 服务
│   └── docker/                 # Docker 配置
└── web/                        # 前端代码（待实现）
```

---

## 🔧 开发环境设置

### 1. 安装依赖

```bash
# Go 1.26+
go version

# PostgreSQL 客户端（可选，用于测试）
sudo apt install postgresql-client

# Make
sudo apt install make
```

### 2. 克隆项目

```bash
cd /xiaonuo/workspace/code
git clone https://github.com/iflyelf/consul_mgr.git
cd consul_mgr
```

### 3. 安装 Go 依赖

```bash
go mod tidy
go mod download
```

### 4. 配置环境变量

```bash
# 创建 .env 文件（不要提交到 Git）
cat > .env << 'EOF'
export DATABASE_URL="postgresql://iflyelf:1q23l@Yc45j@10.0.51.88:6000/consul_mgr?sslmode=disable"
export JWT_SECRET="consul_mgr_jwt_secret_2024_min_32_chars"
export ADMIN_PASSWORD="ysyh!9Sky"
export ADMIN_USERNAME="iflyelf"
export ADMIN_EMAIL="iflyelf@gmail.com"
EOF

# 加载环境变量
source .env
```

### 5. 编译项目

```bash
make build
```

### 6. 启动服务

```bash
./consul_mgr -c etc/config.yaml

# 或使用快速启动脚本
./scripts/start.sh
```

---

## 📝 开发流程

### 添加新的 API 端点

#### 1. 定义类型（internal/types/）

```go
// internal/types/your_feature.go
type YourRequest struct {
    Field1 string `json:"field1" validate:"required"`
    Field2 int    `json:"field2"`
}

type YourResponse struct {
    Result string `json:"result"`
}
```

#### 2. 实现业务逻辑（internal/logic/）

```go
// internal/logic/your_feature/your_logic.go
package yourfeature

import (
    "context"
    "github.com/iflyelf/consul_mgr/internal/svc"
    "github.com/iflyelf/consul_mgr/internal/types"
)

type YourLogic struct {
    ctx    context.Context
    svcCtx *svc.ServiceContext
}

func NewYourLogic(ctx context.Context, svcCtx *svc.ServiceContext) *YourLogic {
    return &YourLogic{
        ctx:    ctx,
        svcCtx: svcCtx,
    }
}

func (l *YourLogic) Handle(req *types.YourRequest) (*types.YourResponse, error) {
    // 实现业务逻辑
    return &types.YourResponse{
        Result: "success",
    }, nil
}
```

#### 3. 创建 HTTP 处理器（internal/handler/）

```go
// internal/handler/your_feature/your_handler.go
package yourfeature

import (
    "net/http"
    "github.com/zeromicro/go-zero/rest/httpx"
    "github.com/iflyelf/consul_mgr/internal/logic/yourfeature"
    "github.com/iflyelf/consul_mgr/internal/pkg/response"
    "github.com/iflyelf/consul_mgr/internal/svc"
    "github.com/iflyelf/consul_mgr/internal/types"
)

func YourHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        var req types.YourRequest
        if err := httpx.Parse(r, &req); err != nil {
            response.BadRequest(w, err.Error())
            return
        }
        
        l := yourfeature.NewYourLogic(r.Context(), svcCtx)
        resp, err := l.Handle(&req)
        if err != nil {
            response.Error(w, 500, err.Error())
            return
        }
        
        response.Success(w, resp)
    }
}
```

#### 4. 注册路由（cmd/api/main.go）

```go
// 在 registerHandlers 函数中添加
server.AddRoutes(
    []rest.Route{
        {
            Method:  http.MethodPost,
            Path:    "/api/your/endpoint",
            Handler: yourfeature.YourHandler(ctx),
        },
    },
    rest.WithJwt(ctx.Config.JWT.Secret), // 需要认证
)
```

---

## 🗄️ 数据库操作

### 查询示例

```go
// 查询单行
var user struct {
    ID       int64  `db:"id"`
    Username string `db:"username"`
}
query := `SELECT id, username FROM users WHERE id = $1`
err := l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &user, query, userID)

// 查询多行
var users []struct {
    ID       int64  `db:"id"`
    Username string `db:"username"`
}
query := `SELECT id, username FROM users WHERE status = $1`
err := l.svcCtx.DB.QueryRowsPartialCtx(l.ctx, &users, query, 1)

// 执行更新/删除
query := `UPDATE users SET status = $1 WHERE id = $2`
_, err := l.svcCtx.DB.ExecCtx(l.ctx, query, 0, userID)
```

---

## 🔐 认证与权限

### 获取当前用户信息

```go
import "github.com/iflyelf/consul_mgr/internal/middleware"

func (l *YourLogic) Handle(req *types.YourRequest) (*types.YourResponse, error) {
    // 从 Context 获取用户 ID
    userID := middleware.GetUserID(l.ctx)
    username := middleware.GetUsername(l.ctx)
    
    if userID == 0 {
        return nil, errors.New("未授权")
    }
    
    // 继续处理...
}
```

### 检查权限（TODO: 待实现）

```go
// 检查用户是否有特定权限
hasPermission := checkPermission(userID, "consul:service:create")

// 检查用户是否有访问特定服务组的权限
hasAccess := checkGroupAccess(userID, groupID, "write")
```

---

## 🧪 测试

### 单元测试

```bash
# 运行所有测试
go test ./...

# 运行特定包的测试
go test ./internal/logic/auth/...

# 带覆盖率
go test -cover ./...
```

### API 测试

```bash
# 健康检查
curl http://localhost:8080/health

# 登录
curl -X POST http://localhost:8080/api/auth/login \
  -H "Content-Type: application/json" \
  -d '{"username":"iflyelf","password":"ysyh!9Sky"}'

# 获取用户信息（需要 Token）
TOKEN="eyJhbGciOiJIUzI1NiIs..."
curl -X GET http://localhost:8080/api/auth/info \
  -H "Authorization: Bearer $TOKEN"
```

---

## 📦 构建与发布

### 本地构建

```bash
# 编译当前平台
make build

# 交叉编译 Linux AMD64
make build-linux-amd64

# 交叉编译 Linux ARM64
make build-linux-arm64

# 编译所有架构
make build-all
```

### Docker 构建

```bash
# 构建镜像
docker build -t consul_mgr:dev .

# 运行容器
docker run -d \
  --name consul_mgr \
  -p 8080:8080 \
  -e DATABASE_URL="..." \
  -e JWT_SECRET="..." \
  -e ADMIN_PASSWORD="..." \
  consul_mgr:dev
```

### 发布到 GitHub

```bash
# 提交代码
git add .
git commit -m "feat: add new feature"
git push origin main

# GitHub Actions 会自动构建并发布
# - 编译多架构二进制
# - 构建 Docker 镜像
# - 发布到 GitHub Release 和 Docker Hub
```

---

## 🐛 调试技巧

### 1. 查看日志

```bash
# 运行时查看日志
./consul_mgr -c etc/config.yaml

# systemd 查看日志
journalctl -u consul_mgr -f

# Docker 查看日志
docker compose logs -f
```

### 2. 数据库调试

```bash
# 连接数据库
psql postgresql://iflyelf:1q23l@Yc45j@10.0.51.88:6000/consul_mgr

# 查看表结构
\dt

# 查看用户
SELECT * FROM users;

# 查看权限
SELECT * FROM permissions;
```

### 3. 使用 Delve 调试器

```bash
# 安装 Delve
go install github.com/go-delve/delve/cmd/dlv@latest

# 调试运行
dlv debug ./cmd/api -- -c etc/config.yaml
```

---

## 📚 参考资料

### go-zero 文档
- 官方文档: https://go-zero.dev/
- API 示例: https://go-zero.dev/docs/tutorials

### JWT 认证
- jwt-go: https://github.com/golang-jwt/jwt

### PostgreSQL
- 官方文档: https://www.postgresql.org/docs/

### Element Plus
- 官方文档: https://element-plus.org/

---

## 🤝 贡献指南

### 代码风格

- 使用 `gofmt` 格式化代码
- 使用 `golangci-lint` 进行代码检查
- 变量命名使用驼峰命名
- 导出的函数和类型添加注释

### 提交规范

```
feat: 新功能
fix: 修复 bug
docs: 文档更新
style: 代码格式调整
refactor: 重构
test: 测试相关
chore: 构建、工具等
```

### Pull Request 流程

1. Fork 项目
2. 创建特性分支 (`git checkout -b feature/xxx`)
3. 提交更改 (`git commit -m 'feat: add xxx'`)
4. 推送到分支 (`git push origin feature/xxx`)
5. 创建 Pull Request

---

## 💬 获取帮助

- GitHub Issues: https://github.com/iflyelf/consul_mgr/issues
- Email: iflyelf@gmail.com

---

**更新时间**: 2026年9月14日
