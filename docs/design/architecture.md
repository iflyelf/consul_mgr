# 架构设计

## 1. 总览

Consul Manager 是一个 Consul 服务与实例管理平台，采用 **单二进制 + 嵌入式前端** 的交付方式：
后端（Go / go-zero）编译时通过 `go:embed` 将 Vue 3 前端产物打包进可执行文件，运行时只需一个进程、一个端口。

```
┌──────────────────────────────────────────────────────────────┐
│                        浏览器 (Vue 3 SPA)                     │
│   服务组管理 / Services 管理 / Instances 管理 / 审计日志         │
└───────────────┬──────────────────────────────────────────────┘
                │ HTTP (REST, Bearer Token)
┌───────────────▼──────────────────────────────────────────────┐
│                    consul_mgr (单二进制)                      │
│  ┌────────────┐  ┌──────────────┐  ┌──────────────────────┐  │
│  │ REST 路由   │→ │ 中间件链      │→ │ Handler → Logic      │  │
│  │ (go-zero)  │  │ 认证/权限/审计│  │ (业务逻辑)            │  │
│  └────────────┘  └──────────────┘  └──────────┬───────────┘  │
│                                               │              │
│  ┌───────────────┐  ┌──────────────┐  ┌───────▼───────────┐  │
│  │ 嵌入式前端      │  │ Redis 缓存    │  │ Consul SDK (api)  │  │
│  │ (go:embed)    │  │ (可降级)      │  │ 客户端池 + 去重    │  │
│  └───────────────┘  └──────────────┘  └───────┬───────────┘  │
└───────────────┬───────────────────────────────┼──────────────┘
                │                               │
        ┌───────▼────────┐              ┌───────▼────────┐
        │  PostgreSQL    │              │ Consul 集群     │
        │  服务组/审计    │              │ (多集群/多 DC)  │
        └────────────────┘              └────────────────┘
                │
        ┌───────▼────────┐
        │    Casdoor     │
        │ 认证 / RBAC     │
        └────────────────┘
```

## 2. 分层结构

| 目录 | 职责 |
|------|------|
| `cmd/api/` | 程序入口：加载配置、注册路由、启动服务、SPA 静态资源处理 |
| `internal/handler/` | HTTP 层：参数解析、响应封装 |
| `internal/logic/` | 业务逻辑层：服务组、实例、服务、权限、审计 |
| `internal/middleware/` | 中间件：Casdoor 认证、权限校验、审计、CORS |
| `internal/pkg/` | 基础组件：Consul 客户端、Casdoor 客户端、Redis 缓存、统一响应 |
| `internal/svc/` | 服务上下文：DB 连接、建表迁移、组件初始化 |
| `internal/config/` | 配置结构体与环境变量覆盖 |
| `internal/types/` | 请求/响应数据结构 |
| `web/` | Vue 3 前端（构建产物嵌入二进制） |

## 3. 关键设计

### 3.1 认证与授权
- **认证**：完全委托 Casdoor（OAuth2 授权码模式）。后端不保存用户密码。
  - 登录：前端跳转 `/api/auth/login` → 后端按当前访问域名动态生成 Casdoor 地址并 302。
  - 回调：`/api/auth/callback` 换取 Token，解析 JWT 载荷得到用户信息。
- **授权**：中间件 `casdoor_auth` 校验 Token；`permission` 校验服务组级权限
  （`service_group_users` / `service_group_roles`）；全局管理员放行。

### 3.2 多 Consul 集群
- 每个「服务组」对应一套 Consul 连接配置（地址 / Token / 数据中心）。
- `internal/pkg/consul/manager.go` 按服务组 ID 缓存并复用客户端。
- **数据中心自动探测**：创建/修改服务组时调用 Consul Agent 自动获取 DC，无需手填。

### 3.3 集群去重
- Consul 的 `Agent.ServiceDeregister` 只能注销本节点注册的服务。
- 集群中若同一 `ServiceID` 归属其他节点，注册前先扫描 Catalog 清理远端同名项，
  删除时回退到 `Catalog.Deregister` 实现跨节点操作（见 `logic/instance/dedup.go`）。

### 3.4 缓存
- Redis 缓存实例列表 / 服务列表 / 服务详情，默认 TTL 30s。
- **优雅降级**：Redis 不可用时自动跳过，不影响主流程。
- 写操作（注册/更新/删除/导入/批量）后按服务组前缀失效相关缓存。

### 3.5 前端
- Vue 3 + Element Plus + Pinia + Vue Router，Vite 构建。
- **三套主题**：暖沙米（默认）/ 冷蓝 / 暗黑，Mac 圆角风格，H5 自适应。
- 主题选择本地记忆，首次访问按系统深色偏好选择。

## 4. 配置原则

**零硬编码**：所有地址、端口、凭据均通过环境变量注入，详见 [部署文档](../deployment/systemd.md)。
端口（`SERVER_PORT`）等运行时参数由 `internal/config/config.go: ApplyEnvOverrides()` 显式覆盖。

## 5. 相关文档

- [数据库设计](database.md)
- [API 设计](api.md)
- [开发文档](../development/development.md)
- [测试文档](../testing/testing.md)
- [部署文档](../deployment/systemd.md)
