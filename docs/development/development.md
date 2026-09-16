# 开发文档

## 1. 环境要求

| 组件 | 版本 |
|------|------|
| Go | 1.26+ |
| Node.js | 18+（构建前端） |
| PostgreSQL | 14+ |
| Redis | 6+（可选，不可用时自动降级） |
| FlyIAM | 最新（提供 Casdoor 认证服务，本地开发需先启动） |

## 2. 本地开发

### 2.1 克隆与依赖

```bash
git clone https://github.com/iflyelf/consul_mgr.git
cd consul_mgr
go mod download
cd web && npm install && cd ..
```

### 2.2 配置

复制并修改配置（**所有参数均可用环境变量覆盖**）：

```bash
cp etc/config.yaml etc/config.local.yaml
```

最小必填环境变量：

```bash
export DATABASE_URL="postgresql://user:pass@host:5432/consul_mgr?sslmode=disable"
export JWT_SECRET="your-secret-at-least-32-chars"
export ADMIN_USERNAME="admin"
export ADMIN_PASSWORD="your-password"
export CASDOOR_ENDPOINT="http://localhost:8000"
export CASDOOR_ORGANIZATION="flyiam"
export CASDOOR_APPLICATION="flyiam"
export CASDOOR_CLIENT_ID="xxx"      # 从 FlyIAM 获取
export CASDOOR_CLIENT_SECRET="xxx"  # 从 FlyIAM 获取
```

> 认证依赖 FlyIAM：本地开发请先启动 FlyIAM（内置 Casdoor），并确保组织的回调白名单包含
> `http://localhost:8080/callback`。详见 [FlyIAM 认证对接](../deployment/flyiam.md)。

### 2.3 启动

```bash
# 终端 1：前端热更新（可选，开发期）
cd web && npm run dev

# 终端 2：后端
go run ./cmd/api -c etc/config.yaml
```

访问 `http://localhost:8080`。

## 3. 构建

```bash
# 1) 构建前端（产物输出到 web/dist，会被 embed 进二进制）
cd web && npm run build && cd ..

# 2) 构建后端（含嵌入式前端）
CGO_ENABLED=0 go build -trimpath \
  -ldflags "-s -w -X main.version=$(git rev-parse --short HEAD)" \
  -o consul_mgr ./cmd/api

# 3) 验证
./consul_mgr --version
```

## 4. 代码规范

- **分层**：`handler`（HTTP）→ `logic`（业务）→ `pkg`（基础组件）
- **注释**：导出的函数/类型必须有注释；注释使用中文
- **错误处理**：错误需携带上下文（`fmt.Errorf("xxx失败: %w", err)`）
- **零硬编码**：地址/端口/凭据一律通过配置注入，不得写死在代码中
- **格式化**：`gofmt` / `goimports`；前端遵循项目既有风格

## 5. 目录约定

```
cmd/api/            程序入口
internal/handler/   HTTP 处理器（按资源分包）
internal/logic/     业务逻辑（按资源分包）
internal/middleware/ 中间件
internal/pkg/       基础组件（consul/casdoor/cache/response）
internal/svc/       服务上下文与自动建表
internal/config/    配置与环境变量覆盖
internal/types/     请求响应结构
web/                Vue 3 前端
charts/consul_mgr/  Helm Chart（Helmfile 多环境）
deploy/             部署物料（systemd/docker/sql/config）
docs/               文档（design/development/testing/deployment）
```

## 6. 新增功能指引

1. 在 `internal/types/` 定义请求/响应结构（可选字段用 `optional` 标签，go-zero 要求）
2. 在 `internal/logic/<resource>/` 实现业务逻辑
3. 在 `internal/handler/<resource>/` 实现 HTTP 层
4. 在 `cmd/api/main.go` 注册路由（按需套认证/权限/审计中间件）
5. 在 `web/src/api/` 与 `web/src/views/` 实现前端
6. 补充测试与文档

> **注意**：go-zero 的可选字段必须使用 `json:"xxx,optional"`，`omitempty` 不被识别。

## 7. 相关文档

- [架构设计](../design/architecture.md) · [API 设计](../design/api.md) · [测试文档](../testing/testing.md)
