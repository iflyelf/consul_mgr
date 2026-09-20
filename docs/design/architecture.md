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
        ┌────────────────┐
        │ FlyIAM         │
        │ (内置 Casdoor)  │
        │ 认证 / RBAC     │
        └────────────────┘
```

## 2. 分层结构

| 目录 | 职责 |
|------|------|
| `cmd/api/` | 程序入口：加载配置、注册路由、启动服务、SPA 静态资源处理 |
| `internal/handler/` | HTTP 层：参数解析、响应封装 |
| `internal/logic/` | 业务逻辑层：服务组、实例、服务、权限、审计 |
| `internal/middleware/` | 中间件：Casdoor 认证（FlyIAM）、权限校验、审计、CORS |
| `internal/pkg/` | 基础组件：Consul 客户端、Casdoor 客户端（对接 FlyIAM）、Redis 缓存、统一响应 |
| `internal/svc/` | 服务上下文：DB 连接、自动建表（幂等）、组件初始化 |
| `internal/config/` | 配置结构体与环境变量覆盖 |
| `internal/types/` | 请求/响应数据结构 |
| `web/` | Vue 3 前端（构建产物嵌入二进制） |

## 3. 关键设计

### 3.1 认证与授权
- **认证**：完全委托外部 [FlyIAM](https://github.com/iflyelf/flyiam)（内置 Casdoor，OAuth2 授权码模式）。
  本项目不内置 Casdoor，也不保存用户密码。
  - 登录：前端跳转 `/api/auth/login` → 后端按当前访问域名动态生成 Casdoor 地址并 302。
  - 回调：`/api/auth/callback` 换取 Token，解析 JWT 载荷得到用户信息。
- **授权**：中间件 `casdoor_auth` 校验 Token；`permission` 校验服务组级权限
  （`service_group_users` / `service_group_roles`）；全局管理员放行。

### 3.2 多 Consul 集群
- 每个「服务组」对应一套 Consul 连接配置（地址 / Token / 数据中心）。
- `internal/pkg/consul/manager.go` 按服务组 ID 缓存并复用客户端。
- **数据中心自动探测**：仅在**创建/修改服务组**时调用 Consul Agent 探测 DC 并落库，
  也可通过 `POST /api/groups/detect-datacenter` 手动触发。
  > ⚠️ 查询路径（`GetConsulClient`）**不再探测数据中心**：`Agent().Self()` 属于 Agent API，
  > 在只读网关/负载均衡后可能不可达并卡满超时；且每次列表都多一次往返会导致查询极慢
  > （实测单请求被拖到 10s，移除后降至 0.05s）。

### 3.3 集群去重
- Consul 的 `Agent.ServiceDeregister` 只能注销本节点注册的服务。
- 集群中若同一 `ServiceID` 归属其他节点，注册前先扫描 Catalog 清理远端同名项，
  删除时回退到 `Catalog.Deregister` 实现跨节点操作（见 `logic/instance/dedup.go`）。

### 3.4 缓存
- Redis 缓存实例列表 / 服务列表 / 服务详情，默认 TTL 300s。
- **优雅降级**：Redis 不可用时自动跳过，不影响主流程。
- 写操作（注册/更新/删除/导入/批量）后按服务组前缀失效相关缓存。

### 3.5 性能：全面并行化
- **问题**：实例/服务列表需要按服务逐个查询 Consul，串行执行会因网络往返累积而极慢（N+1）。
- **方案**（`internal/pkg/parallel`，统一的有界并发原语）：
  - 查询并行：实例列表 / 服务列表 / 服务详情 / 导出 / 集群去重全部并行；
  - 写入并行：批量删除实例、删除服务、批量删除服务、集群去重清理全部并行；
  - 并发上限由 `Consul.MaxConcurrency`（`CONSUL_MAX_CONCURRENCY`，默认 16）统一控制，
    既提速又避免打爆 Consul；
  - 单服务失败仅记录日志并跳过，不阻断整体列表。
- **按服务缓存**（`consul_mgr:instances:{groupID}:svc:{service}`）：
  - 各服务的实例单独缓存；全量列表由各服务缓存合并而来；
  - 因此「先看服务 A → 再看服务 B → 再看全量」不会重复请求 Consul；
  - 服务列表 / 服务详情复用同一份按服务缓存，跨模块共享。
- **分页**：
  - 实例列表（Instances 页）与服务详情（实例列表）均由后端/前端分页，
    避免一次渲染上千行。
- **实测**（300 服务 / 900 实例，模拟 50ms 往返；以及真实远端 7 服务 / 4125 实例）：

  | 操作 | 优化前 | 优化后 |
  |------|--------|--------|
  | 服务列表（冷） | 19.5s | 0.73s |
  | 实例列表（冷） | 19.7s | 0.99s |
  | 实例列表（复用服务列表缓存） | — | 0.14s |
  | 切换服务 / 翻页 / 热缓存 | 每次全量重拉 | < 0.13s |
  | 批量删除 201 实例 | 串行逐个 | 0.20s |

### 3.6 前端
- Vue 3 + Element Plus + Pinia + Vue Router，Vite 构建。
- **三套主题**：暖沙米（默认）/ 冷蓝 / 暗黑，Mac 圆角风格，H5 自适应。
- 主题选择本地记忆，首次访问按系统深色偏好选择。

### 3.7 多副本与并发安全

**多副本（默认 2 副本）下的定时任务去重**

每个副本都有独立调度器，进程内标记无法跨 Pod 互斥。用户字段自动同步执行前，
通过 `internal/pkg/distlock`（PostgreSQL 会话级 advisory lock）获取**全局锁**：

- 抢到锁的副本执行，其余直接跳过（`ErrSyncRunning`，视为正常）；
- 锁绑定专用连接，副本异常退出时连接断开、锁自动释放，不会死锁；
- 获取连接带 10s 超时；锁服务异常时降级为进程内互斥（仅告警）。

**配置的并发安全（原子快照）**

`Config` 由 `internal/config.Store`（`atomic.Pointer[Config]`）承载，采用**写时复制**：
页面保存设置时先 `Clone()` 副本、在副本上修改，再原子替换；读者经
`ServiceContext.Config()` 获取不可变快照，消除「页面保存」与「请求读取」的数据竞争。

**并发原语与 panic 兜底**

- `internal/pkg/parallel` 提供统一的有界并发原语：信号量在**派发前**获取，
  同时在跑的 goroutine 数严格受限（不预创建大量 goroutine）；单项 panic 被
  捕获为该下标的错误，不终止进程。
- 后台 goroutine（调度器、同步、令牌校验等）均加 `recover`：Go 的 `recover`
  仅对**同 goroutine** 有效，worker / 后台循环内 panic 会终止整个进程。

### 3.8 自动获取 Casdoor 凭据

本系统与 FlyIAM 复用同一 Casdoor，但无法自行获取应用凭据。配置 FlyIAM 地址
（`CONSUL_MGR_FLYIAM_API_ENDPOINT`）与服务令牌（`CONSUL_MGR_FLYIAM_SERVICE_TOKEN`）后，
启动时（及页面保存 FlyIAM 配置后）会自动调用 FlyIAM
`GET /api/casdoor/app-credentials` 获取 `ClientID/Secret` 并落库（DB 优先），
**免手工填写**。`CASDOOR_ENDPOINT` 仍须显式提供（非敏感；FlyIAM 的地址可能是
其命名空间内短名，跨命名空间不可达）。

## 4. 配置原则

**零硬编码**：所有地址、端口、凭据均通过环境变量注入，详见 [部署文档](../deployment/systemd.md)。
端口（`SERVER_PORT`）等运行时参数由 `internal/config/config.go: ApplyEnvOverrides()` 显式覆盖。

## 5. 相关文档

- [数据库设计](database.md)
- [API 设计](api.md)
- [开发文档](../development/development.md)
- [测试文档](../testing/testing.md)
- [Kubernetes 部署](../deployment/kubernetes.md) · [FlyIAM 认证对接](../deployment/flyiam.md)
