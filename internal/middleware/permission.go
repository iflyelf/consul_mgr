// Package middleware 提供 HTTP 中间件
package middleware

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"

	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
	"github.com/iflyelf/consul_mgr/internal/pkg/perm"
)

// peekBodyGroupID 读取请求体中的 group_id（读取后还原 Body，供后续 handler 使用）。
//
// 仅用于权限中间件在查询/路径参数缺失时兜底判断组归属；请求体过大或非 JSON
// 时返回 0（此时依赖 handler 层校验）。
func peekBodyGroupID(r *http.Request) int64 {
	if r.Body == nil {
		return 0
	}
	const maxPeek = 1 << 20 // 1MB
	buf, err := io.ReadAll(io.LimitReader(r.Body, maxPeek))
	if err != nil {
		return 0
	}
	// 还原 Body，避免影响后续 handler
	r.Body = io.NopCloser(io.MultiReader(bytes.NewReader(buf), r.Body))

	var payload struct {
		GroupID int64 `json:"group_id"`
	}
	if err := json.Unmarshal(buf, &payload); err != nil {
		return 0
	}
	return payload.GroupID
}

// PermissionMiddleware 权限检查中间件
//
// 功能：
//   - 检查 Casdoor 全局权限
//   - 检查服务组级别权限（用户直授 + 团队授权 + 角色）
//   - 支持三层权限模型
type PermissionMiddleware struct {
	client *casdoor.Client
	db     *sql.DB
	perm   *perm.Checker
}

// NewPermissionMiddleware 创建权限检查中间件
//
// 参数:
//   client - Casdoor 客户端实例
//   db     - 原生数据库连接（PostgreSQL TEXT[] 需 pq.Array 扫描）
//
// 返回:
//   *PermissionMiddleware - 中间件实例
func NewPermissionMiddleware(client *casdoor.Client, db *sql.DB) *PermissionMiddleware {
	return &PermissionMiddleware{
		client: client,
		db:     db,
		perm:   perm.New(db),
	}
}

// RequirePermission 要求指定权限（中间件生成器）
//
// 参数:
//   resource - 资源类型（如: consul_group, consul_service, consul_instance）
//   action - 操作类型（如: read, write, delete, admin）
//
// 返回:
//   func(http.HandlerFunc) http.HandlerFunc - 中间件函数
//
// 使用示例：
//   router.Use(permissionMw.RequirePermission("consul_group", "write"))
func (m *PermissionMiddleware) RequirePermission(resource, action string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 1. 获取用户信息
			userId, ok := GetUserIdFromContext(ctx)
			if !ok {
				logx.Error("无法获取用户 ID")
				httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
					"code":    401,
					"message": "未认证，请先登录",
				})
				return
			}

			username, _ := GetUsernameFromContext(ctx)

			// 2. 检查是否为全局管理员（拥有所有权限）
			if IsGlobalAdminFromContext(ctx) {
				logx.Infof("全局管理员 %s 访问资源 %s:%s", username, resource, action)
				next.ServeHTTP(w, r)
				return
			}

			// 3. 检查 Casdoor 权限
			token, _ := GetTokenFromContext(ctx)
			hasPermission, err := m.client.CheckPermissionByToken(token, resource, action)
			if err != nil {
				logx.Errorf("权限检查失败: %v", err)
				httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
					"code":    500,
					"message": "权限检查失败",
				})
				return
			}

			if !hasPermission {
				logx.Errorf("用户 %s (%s) 没有权限: %s:%s", username, userId, resource, action)
				httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
					"code":    403,
					"message": fmt.Sprintf("没有权限执行此操作（需要 %s:%s 权限）", resource, action),
				})
				return
			}

			// 4. 权限验证通过
			logx.Infof("用户 %s 权限验证通过: %s:%s", username, resource, action)
			next.ServeHTTP(w, r)
		}
	}
}

// RequireServiceGroupAccess 要求服务组访问权限（中间件生成器）
//
// 参数:
//   action - 操作类型（read, write, delete, admin）
//
// 返回:
//   func(http.HandlerFunc) http.HandlerFunc - 中间件函数
//
// 权限检查逻辑：
//   1. 检查全局权限（consul_service:action）
//   2. 检查服务组级别权限（从数据库查询）
//   3. 支持用户级权限和角色级权限
//
// 使用示例：
//   router.Use(permissionMw.RequireServiceGroupAccess("write"))
func (m *PermissionMiddleware) RequireServiceGroupAccess(action string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 1. 获取用户信息
			userId, ok := GetUserIdFromContext(ctx)
			if !ok {
				httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
					"code":    401,
					"message": "未认证，请先登录",
				})
				return
			}

			username, _ := GetUsernameFromContext(ctx)

			// 2. 检查是否为全局管理员
			if IsGlobalAdminFromContext(ctx) {
				logx.Infof("全局管理员 %s 访问服务组", username)
				next.ServeHTTP(w, r)
				return
			}

			// 3. 获取 group_id（查询参数 → 路径参数 → 请求体）
			groupIdStr := r.URL.Query().Get("group_id")
			if groupIdStr == "" {
				groupIdStr = r.URL.Query().Get("id")
			}
			if groupIdStr == "" {
				// 路径参数（如 /api/groups/:id、/api/instances/:id）
				if v := pathvar.Vars(r)["id"]; v != "" {
					groupIdStr = v
				}
			}
			if groupIdStr == "" {
				// 请求体中的 group_id（如批量删除）；仅在查询/路径均无时兜底，
				// 避免「查询参数与请求体不一致」造成的越权（见 BatchDeleteHandler）。
				if body := peekBodyGroupID(r); body > 0 {
					groupIdStr = strconv.FormatInt(body, 10)
				}
			}

			// 如果没有 group_id，检查全局权限
			if groupIdStr == "" {
				token, _ := GetTokenFromContext(ctx)
				hasPermission, err := m.client.CheckPermissionByToken(token, "consul_service", action)
				if err != nil || !hasPermission {
					httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
						"code":    403,
						"message": "没有权限访问服务组",
					})
					return
				}
				next.ServeHTTP(w, r)
				return
			}

			groupId, err := strconv.ParseInt(groupIdStr, 10, 64)
			if err != nil {
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "无效的服务组 ID",
				})
				return
			}

			// 一致性校验：请求体中的 group_id 若与查询/路径参数不一致，直接拒绝。
			// 防止「用有权限的组通过中间件，再在请求体操作另一组」的越权。
			if bodyGroup := peekBodyGroupID(r); bodyGroup > 0 && bodyGroup != groupId {
				logx.Errorf("用户 %s 请求体 group_id=%d 与参数 group_id=%d 不一致，拒绝", username, bodyGroup, groupId)
				httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
					"code":    400,
					"message": "请求参数不一致（group_id 与请求体不符）",
				})
				return
			}

			// 4. 检查全局 consul_service 权限
			token, _ := GetTokenFromContext(ctx)
			hasGlobalPerm, err := m.client.CheckPermissionByToken(token, "consul_service", action)
			if err == nil && hasGlobalPerm {
				logx.Infof("用户 %s 拥有全局 consul_service:%s 权限", username, action)
				next.ServeHTTP(w, r)
				return
			}

			// 5. 检查服务组/服务级权限（用户直授 → 团队授权 → 角色）
			//    service_name 存在时按服务维度判定（未授权服务将被拒绝）
			serviceName := r.URL.Query().Get("service_name")
			if serviceName == "" {
				serviceName = r.URL.Query().Get("service")
			}
			granted, gerr := m.perm.Check(ctx, userId, groupId, serviceName, action)
			if gerr != nil {
				logx.Errorf("服务组权限检查失败: %v", gerr)
				httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
					"code":    500,
					"message": "权限检查失败",
				})
				return
			}
			if granted {
				logx.Infof("用户 %s 通过服务组授权获得 %s 权限 (group=%d service=%s)", username, action, groupId, serviceName)
				next.ServeHTTP(w, r)
				return
			}

			logx.Errorf("用户 %s 没有访问服务组 %d 的权限（需要 %s）", username, groupId, action)
			httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
				"code":    403,
				"message": fmt.Sprintf("没有权限访问该服务组（需要 %s 权限）", action),
			})
		}
	}
}

// RequireAdmin 要求管理员权限（中间件生成器）
//
// 返回:
//   func(http.HandlerFunc) http.HandlerFunc - 中间件函数
//
// 使用示例：
//   router.Use(permissionMw.RequireAdmin())
func (m *PermissionMiddleware) RequireAdmin() func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 检查是否为管理员或全局管理员
			if IsAdminFromContext(ctx) || IsGlobalAdminFromContext(ctx) {
				next.ServeHTTP(w, r)
				return
			}

			username, _ := GetUsernameFromContext(ctx)
			logx.Errorf("用户 %s 尝试访问管理员接口", username)

			httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
				"code":    403,
				"message": "需要管理员权限",
			})
		}
	}
}

// RequireRole 要求指定角色（中间件生成器）
//
// 参数:
//   roles - 允许的角色列表
//
// 返回:
//   func(http.HandlerFunc) http.HandlerFunc - 中间件函数
//
// 使用示例：
//   router.Use(permissionMw.RequireRole("admin", "operator"))
func (m *PermissionMiddleware) RequireRole(roles ...string) func(http.HandlerFunc) http.HandlerFunc {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			ctx := r.Context()

			// 全局管理员拥有所有角色
			if IsGlobalAdminFromContext(ctx) {
				next.ServeHTTP(w, r)
				return
			}

			// 获取用户角色
			userRoles, ok := GetUserRolesFromContext(ctx)
			if !ok || len(userRoles) == 0 {
				httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
					"code":    403,
					"message": "没有分配任何角色",
				})
				return
			}

			// 检查角色匹配
			for _, userRole := range userRoles {
				for _, requiredRole := range roles {
					if userRole == requiredRole {
						next.ServeHTTP(w, r)
						return
					}
				}
			}

			username, _ := GetUsernameFromContext(ctx)
			logx.Errorf("用户 %s 角色不匹配，要求: %v，拥有: %v", username, roles, userRoles)

			httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
				"code":    403,
				"message": fmt.Sprintf("需要以下角色之一: %v", roles),
			})
		}
	}
}

// OptionalAuth 可选认证中间件（不强制要求登录）
//
// 参数:
//   next - 下一个处理函数
//
// 返回:
//   http.HandlerFunc - 包装后的处理函数
//
// 功能：
//   如果提供了 Token，则验证并提取用户信息
//   如果没有提供 Token，则继续执行，但 Context 中没有用户信息
//
// 使用场景：
//   某些接口对登录用户和未登录用户有不同的行为
func (m *PermissionMiddleware) OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	authMiddleware := NewCasdoorAuthMiddleware(m.client)
	
	return func(w http.ResponseWriter, r *http.Request) {
		// 尝试提取 Token
		token := extractToken(r)
		if token == "" {
			// 没有 Token，直接继续
			next.ServeHTTP(w, r)
			return
		}

		// 有 Token，使用认证中间件
		authMiddleware.Handle(next).ServeHTTP(w, r)
	}
}
