# Consul Manager + Casdoor 集成方案

## 🎯 目标

将 Casdoor (开源身份认证平台) 集成到 Consul Manager 中，实现：
1. 统一的用户认证和授权
2. 完善的 RBAC 权限管理
3. 服务组级别的访问控制
4. 完整的审计日志
5. SSO 单点登录支持

## 📋 Casdoor 简介

Casdoor 是一个 UI-first 的身份和访问管理 (IAM) / 单点登录 (SSO) 平台：
- 🔐 OAuth 2.0, OIDC, SAML 支持
- 👥 用户、组织、角色、权限管理
- 📝 完整的审计日志
- 🌐 多语言支持
- 🎨 可自定义 UI
- 🔌 易于集成

GitHub: https://github.com/casdoor/casdoor
官网: https://casdoor.org/

## 🏗️ 架构设计

```
┌─────────────────────────────────────────────────────────────┐
│                    Consul Manager 前端                       │
│  (Vue 3 + Element Plus)                                     │
└────────────────────┬────────────────────────────────────────┘
                     │
                     │ HTTP/HTTPS
                     ▼
┌─────────────────────────────────────────────────────────────┐
│              Consul Manager 后端 (Go-Zero)                  │
│                                                              │
│  ┌────────────────────────────────────────────────────┐    │
│  │  Casdoor 中间件                                     │    │
│  │  - Token 验证                                       │    │
│  │  - 权限检查                                         │    │
│  │  - 审计日志                                         │    │
│  └────────────────┬───────────────────────────────────┘    │
│                   │                                          │
│  ┌────────────────▼───────────────────────────────────┐    │
│  │  业务逻辑层                                         │    │
│  │  - Consul 服务管理                                 │    │
│  │  - 服务组管理                                       │    │
│  │  - 实例管理                                         │    │
│  └────────────────┬───────────────────────────────────┘    │
│                   │                                          │
│  ┌────────────────▼───────────────────────────────────┐    │
│  │  服务组权限检查                                     │    │
│  │  - 用户 → 服务组映射                               │    │
│  │  - 操作权限验证                                     │    │
│  └────────────────────────────────────────────────────┘    │
└───────────────┬──────────────────────────────┬──────────────┘
                │                              │
                │                              │
                ▼                              ▼
┌───────────────────────────┐   ┌────────────────────────────┐
│      Casdoor 服务          │   │      PostgreSQL            │
│  - 用户认证                │   │  - 服务组配置               │
│  - 角色管理                │   │  - 服务组权限映射           │
│  - 权限管理                │   │  - 审计日志                 │
│  - 审计日志                │   └────────────────────────────┘
└────────────┬──────────────┘
             │
             ▼
┌────────────────────────────┐
│    Casdoor 数据库          │
│  (MySQL/PostgreSQL)        │
└────────────────────────────┘
```

## 🔧 集成步骤

### 阶段 1: Casdoor 部署和配置

#### 1.1 部署 Casdoor
```bash
# 使用 Docker Compose 部署
version: '3.8'
services:
  casdoor:
    image: casbin/casdoor:latest
    ports:
      - "8000:8000"
    environment:
      - DATABASE_TYPE=postgres
      - DATABASE_HOST=postgres
      - DATABASE_PORT=5432
      - DATABASE_NAME=casdoor
      - DATABASE_USER=casdoor
      - DATABASE_PASSWORD=casdoor_password
    depends_on:
      - postgres
    volumes:
      - ./conf:/conf
  
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: casdoor
      POSTGRES_USER: casdoor
      POSTGRES_PASSWORD: casdoor_password
    volumes:
      - casdoor_data:/var/lib/postgresql/data

volumes:
  casdoor_data:
```

#### 1.2 Casdoor 初始化配置
- 创建组织 (Organization): consul_mgr
- 创建应用 (Application): consul_manager
- 配置回调 URL: http://localhost:8080/callback
- 配置权限模型

#### 1.3 创建资源和权限
```yaml
# Casdoor 权限配置
Resources:
  - consul_group:read    # 查看服务组
  - consul_group:write   # 修改服务组
  - consul_service:read  # 查看服务
  - consul_service:write # 修改服务
  - consul_instance:read # 查看实例
  - consul_instance:write# 修改实例
  - audit:read           # 查看审计日志
  - user:manage          # 用户管理

Roles:
  - name: admin
    permissions: [*]
  
  - name: operator
    permissions:
      - consul_group:read
      - consul_group:write
      - consul_service:*
      - consul_instance:*
  
  - name: viewer
    permissions:
      - consul_group:read
      - consul_service:read
      - consul_instance:read
```

### 阶段 2: Go-Zero 后端集成

#### 2.1 安装 Casdoor Go SDK
```bash
go get github.com/casdoor/casdoor-go-sdk
```

#### 2.2 配置文件更新
```yaml
# etc/config.yaml
Casdoor:
  Endpoint: "http://localhost:8000"
  ClientId: "your-client-id"
  ClientSecret: "your-client-secret"
  OrganizationName: "consul_mgr"
  ApplicationName: "consul_manager"
  Certificate: ""
  
ServiceGroupAuth:
  Enabled: true
  # 服务组权限映射表
  Table: "service_group_permissions"
```

#### 2.3 Casdoor 客户端初始化
```go
// internal/pkg/casdoor/client.go
package casdoor

import (
    "github.com/casdoor/casdoor-go-sdk/auth"
)

type Client struct {
    *auth.Client
}

func NewClient(config CasdoorConfig) *Client {
    authConfig := &auth.AuthConfig{
        Endpoint:         config.Endpoint,
        ClientId:         config.ClientId,
        ClientSecret:     config.ClientSecret,
        Certificate:      config.Certificate,
        OrganizationName: config.OrganizationName,
        ApplicationName:  config.ApplicationName,
    }
    
    return &Client{
        Client: auth.NewClient(authConfig),
    }
}

// 验证 Token
func (c *Client) ValidateToken(token string) (*auth.Claims, error) {
    return c.ParseJwtToken(token)
}

// 检查权限
func (c *Client) CheckPermission(userId, resource, action string) (bool, error) {
    // 调用 Casdoor API 检查权限
    return c.Enforce(userId, resource, action)
}

// 获取用户角色
func (c *Client) GetUserRoles(userId string) ([]string, error) {
    return c.GetRolesByUser(userId)
}
```

#### 2.4 Casdoor 中间件
```go
// internal/middleware/casdoor_middleware.go
package middleware

import (
    "net/http"
    "strings"
    
    "github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
    "github.com/iflyelf/consul_mgr/internal/pkg/response"
)

type CasdoorMiddleware struct {
    client *casdoor.Client
}

func NewCasdoorMiddleware(client *casdoor.Client) *CasdoorMiddleware {
    return &CasdoorMiddleware{
        client: client,
    }
}

// 认证中间件
func (m *CasdoorMiddleware) Auth(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        // 从 Header 获取 Token
        token := extractToken(r)
        if token == "" {
            response.Unauthorized(w, "未提供认证令牌")
            return
        }
        
        // 验证 Token
        claims, err := m.client.ValidateToken(token)
        if err != nil {
            response.Unauthorized(w, "认证令牌无效")
            return
        }
        
        // 将用户信息存入 Context
        ctx := context.WithValue(r.Context(), "userId", claims.User.Id)
        ctx = context.WithValue(ctx, "username", claims.User.Name)
        ctx = context.WithValue(ctx, "roles", claims.User.Roles)
        
        next.ServeHTTP(w, r.WithContext(ctx))
    }
}

// 权限检查中间件
func (m *CasdoorMiddleware) RequirePermission(resource, action string) func(http.HandlerFunc) http.HandlerFunc {
    return func(next http.HandlerFunc) http.HandlerFunc {
        return func(w http.ResponseWriter, r *http.Request) {
            userId := r.Context().Value("userId").(string)
            
            // 检查权限
            hasPermission, err := m.client.CheckPermission(userId, resource, action)
            if err != nil || !hasPermission {
                response.Forbidden(w, "没有权限执行此操作")
                return
            }
            
            next.ServeHTTP(w, r)
        }
    }
}

// 服务组权限检查
func (m *CasdoorMiddleware) CheckServiceGroupAccess(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        userId := r.Context().Value("userId").(string)
        groupId := r.URL.Query().Get("group_id")
        
        // 检查用户是否有访问该服务组的权限
        hasAccess, err := m.checkServiceGroupPermission(userId, groupId)
        if err != nil || !hasAccess {
            response.Forbidden(w, "没有权限访问该服务组")
            return
        }
        
        next.ServeHTTP(w, r)
    }
}

func extractToken(r *http.Request) string {
    bearerToken := r.Header.Get("Authorization")
    if len(strings.Split(bearerToken, " ")) == 2 {
        return strings.Split(bearerToken, " ")[1]
    }
    return ""
}
```

#### 2.5 服务组权限表设计
```sql
-- 服务组用户权限映射表
CREATE TABLE IF NOT EXISTS service_group_users (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    user_id VARCHAR(100) NOT NULL, -- Casdoor 用户 ID
    username VARCHAR(100) NOT NULL,
    permissions VARCHAR(50)[] DEFAULT ARRAY['read'], -- read, write, admin
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    created_by VARCHAR(100),
    UNIQUE(group_id, user_id)
);

CREATE INDEX idx_service_group_users_group ON service_group_users(group_id);
CREATE INDEX idx_service_group_users_user ON service_group_users(user_id);

-- 服务组角色权限映射表
CREATE TABLE IF NOT EXISTS service_group_roles (
    id BIGSERIAL PRIMARY KEY,
    group_id BIGINT NOT NULL,
    role_name VARCHAR(100) NOT NULL, -- Casdoor 角色名
    permissions VARCHAR(50)[] DEFAULT ARRAY['read'],
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW(),
    UNIQUE(group_id, role_name)
);

CREATE INDEX idx_service_group_roles_group ON service_group_roles(group_id);
```

#### 2.6 路由注册（带权限控制）
```go
// cmd/api/main.go
func registerHandlers(server *rest.Server, ctx *svc.ServiceContext) {
    // Casdoor 中间件
    casdoorMw := middleware.NewCasdoorMiddleware(ctx.CasdoorClient)
    
    // 认证路由（使用 Casdoor OAuth2）
    server.AddRoute(rest.Route{
        Method:  http.MethodGet,
        Path:    "/api/auth/login",
        Handler: auth.LoginHandler(ctx), // 重定向到 Casdoor
    })
    
    server.AddRoute(rest.Route{
        Method:  http.MethodGet,
        Path:    "/api/auth/callback",
        Handler: auth.CallbackHandler(ctx), // 处理 Casdoor 回调
    })
    
    // 服务组管理（需要认证 + 权限检查）
    server.AddRoutes(
        []rest.Route{
            {
                Method:  http.MethodGet,
                Path:    "/api/groups",
                Handler: casdoorMw.Auth(
                    casdoorMw.RequirePermission("consul_group", "read")(
                        group.ListGroupsHandler(ctx),
                    ),
                ),
            },
            {
                Method:  http.MethodPost,
                Path:    "/api/groups",
                Handler: casdoorMw.Auth(
                    casdoorMw.RequirePermission("consul_group", "write")(
                        group.CreateGroupHandler(ctx),
                    ),
                ),
            },
            // ... 其他路由
        },
    )
    
    // Consul 服务管理（需要认证 + 服务组权限）
    server.AddRoutes(
        []rest.Route{
            {
                Method:  http.MethodGet,
                Path:    "/api/services",
                Handler: casdoorMw.Auth(
                    casdoorMw.CheckServiceGroupAccess(
                        service.ListServicesHandler(ctx),
                    ),
                ),
            },
            // ... 其他路由
        },
    )
}
```

### 阶段 3: 前端集成

#### 3.1 安装 Casdoor JS SDK
```bash
cd web
npm install casdoor-js-sdk
```

#### 3.2 Casdoor 配置
```javascript
// web/src/config/casdoor.js
import Sdk from 'casdoor-js-sdk'

export const CasdoorConfig = {
  serverUrl: 'http://localhost:8000',
  clientId: 'your-client-id',
  appName: 'consul_manager',
  organizationName: 'consul_mgr',
  redirectPath: '/callback'
}

export const casdoorSdk = new Sdk(CasdoorConfig)
```

#### 3.3 登录流程
```javascript
// web/src/views/Login.vue
<template>
  <div class="login-container">
    <el-button @click="loginWithCasdoor" type="primary">
      使用 Casdoor 登录
    </el-button>
  </div>
</template>

<script setup>
import { casdoorSdk } from '@/config/casdoor'

const loginWithCasdoor = () => {
  // 跳转到 Casdoor 登录页面
  window.location.href = casdoorSdk.getSigninUrl()
}
</script>
```

#### 3.4 回调处理
```javascript
// web/src/views/Callback.vue
<script setup>
import { onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { casdoorSdk } from '@/config/casdoor'
import { useUserStore } from '@/stores/user'

const router = useRouter()
const userStore = useUserStore()

onMounted(async () => {
  try {
    // 获取 code 参数
    const params = new URLSearchParams(window.location.search)
    const code = params.get('code')
    const state = params.get('state')
    
    if (code && state) {
      // 交换 token
      const token = await casdoorSdk.signin(code, state)
      
      // 保存 token
      localStorage.setItem('token', token)
      
      // 获取用户信息
      const userInfo = await casdoorSdk.getAccount()
      userStore.setUser(userInfo)
      
      // 跳转到首页
      router.push('/')
    }
  } catch (error) {
    console.error('登录失败:', error)
    router.push('/login')
  }
})
</script>
```

### 阶段 4: 审计日志集成

#### 4.1 审计日志表（使用 Consul Manager 数据库）
```sql
CREATE TABLE IF NOT EXISTS audit_logs (
    id BIGSERIAL PRIMARY KEY,
    user_id VARCHAR(100) NOT NULL,
    username VARCHAR(100) NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(50) NOT NULL,
    resource_id VARCHAR(100),
    group_id BIGINT,
    ip_address VARCHAR(50),
    user_agent TEXT,
    request_method VARCHAR(10),
    request_path VARCHAR(255),
    request_body TEXT,
    response_status INT,
    error_message TEXT,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE INDEX idx_audit_logs_user ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_created ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_group ON audit_logs(group_id);
```

#### 4.2 审计中间件
```go
// internal/middleware/audit_middleware.go
package middleware

import (
    "bytes"
    "io"
    "net/http"
    "time"
)

type AuditMiddleware struct {
    db sqlx.SqlConn
}

func (m *AuditMiddleware) Log(next http.HandlerFunc) http.HandlerFunc {
    return func(w http.ResponseWriter, r *http.Request) {
        start := time.Now()
        
        // 读取请求体
        body, _ := io.ReadAll(r.Body)
        r.Body = io.NopCloser(bytes.NewBuffer(body))
        
        // 包装 ResponseWriter 以捕获状态码
        ww := &responseWriter{ResponseWriter: w, statusCode: 200}
        
        // 执行请求
        next.ServeHTTP(ww, r)
        
        // 记录审计日志
        go m.saveAuditLog(r, ww.statusCode, body, time.Since(start))
    }
}

func (m *AuditMiddleware) saveAuditLog(r *http.Request, status int, body []byte, duration time.Duration) {
    userId := r.Context().Value("userId")
    username := r.Context().Value("username")
    groupId := r.URL.Query().Get("group_id")
    
    query := `
        INSERT INTO audit_logs 
        (user_id, username, action, resource_type, resource_id, group_id, 
         ip_address, user_agent, request_method, request_path, request_body, response_status)
        VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
    `
    
    m.db.ExecCtx(context.Background(), query,
        userId, username,
        extractAction(r.Method, r.URL.Path),
        extractResourceType(r.URL.Path),
        extractResourceId(r.URL.Path),
        groupId,
        getClientIP(r),
        r.UserAgent(),
        r.Method,
        r.URL.Path,
        string(body),
        status,
    )
}
```

## 📊 数据流程

### 1. 用户登录流程
```
用户 → 前端 → Casdoor 登录页 → Casdoor 认证 
→ 回调到前端 → 前端获取 Token → 保存 Token → 访问后端
```

### 2. API 请求流程
```
前端请求 → 携带 Casdoor Token 
→ 后端 Casdoor 中间件验证 Token 
→ 检查资源权限 
→ 检查服务组权限 
→ 执行业务逻辑 
→ 记录审计日志 
→ 返回响应
```

### 3. 权限检查流程
```
用户请求 → 提取用户 ID 
→ Casdoor 检查用户角色 
→ 检查角色权限 
→ 检查服务组权限映射 
→ 允许/拒绝访问
```

## 🔐 权限模型

### 三层权限控制

1. **Casdoor 层 - 全局权限**
   - 控制用户能访问哪些功能模块
   - 例如：console_group:read, console_service:write

2. **服务组层 - 服务组权限**
   - 控制用户能访问哪些服务组
   - 存储在 service_group_users 表

3. **操作层 - 细粒度权限**
   - 控制用户在服务组内能执行的操作
   - 例如：read, write, admin

### 权限判断逻辑
```go
func CheckAccess(userId, groupId, operation string) bool {
    // 1. 检查 Casdoor 全局权限
    if !casdoor.HasPermission(userId, "consul_service", operation) {
        return false
    }
    
    // 2. 检查服务组权限
    groupPermissions := getServiceGroupPermissions(userId, groupId)
    if !contains(groupPermissions, operation) {
        return false
    }
    
    return true
}
```

## 📈 实施优先级

### Phase 1 - 基础集成 (1-2 周)
- [ ] 部署 Casdoor
- [ ] 实现 OAuth2 登录
- [ ] 实现 Token 验证中间件
- [ ] 前端登录流程

### Phase 2 - 权限系统 (1-2 周)
- [ ] 实现权限检查中间件
- [ ] 服务组权限表和 API
- [ ] 权限管理前端界面

### Phase 3 - 审计日志 (1 周)
- [ ] 实现审计日志中间件
- [ ] 审计日志查询 API
- [ ] 审计日志前端展示

### Phase 4 - 优化完善 (1 周)
- [ ] 性能优化
- [ ] 文档完善
- [ ] 测试覆盖

## 🎁 预期收益

1. **开箱即用的用户管理**
   - 无需自己实现用户 CRUD
   - Casdoor 提供完整的用户界面

2. **强大的权限管理**
   - 支持 RBAC, ABAC 等多种模型
   - 灵活的权限配置

3. **完整的审计追踪**
   - 所有操作可追溯
   - 满足合规要求

4. **易于扩展**
   - 支持 SSO 单点登录
   - 支持多因素认证
   - 支持第三方登录

5. **降低维护成本**
   - Casdoor 独立部署和维护
   - 专业团队持续更新

## 📚 参考资源

- Casdoor 官方文档: https://casdoor.org/docs/overview
- Casdoor Go SDK: https://github.com/casdoor/casdoor-go-sdk
- Casdoor JS SDK: https://github.com/casdoor/casdoor-js-sdk
- Go-Zero 官方文档: https://go-zero.dev/

---

**下一步行动**: 
1. 评审此方案
2. 确定实施时间表
3. 准备 Casdoor 部署环境
4. 开始 Phase 1 开发

