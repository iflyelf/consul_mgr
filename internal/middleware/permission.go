// Package middleware 提供 HTTP 中间件
package middleware

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"

	"github.com/lib/pq"
	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
)

// PermissionMiddleware 权限检查中间件
//
// 功能：
//   - 检查 Casdoor 全局权限
//   - 检查服务组级别权限（用户直授 + 团队授权 + 角色）
//   - 支持三层权限模型
type PermissionMiddleware struct {
	client *casdoor.Client
	db     *sql.DB
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

			// 3. 获取 group_id（从查询参数或路径参数）
			groupIdStr := r.URL.Query().Get("group_id")
			if groupIdStr == "" {
				// 尝试从路径参数获取（需要在路由中配置）
				groupIdStr = r.URL.Query().Get("id")
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

			// 4. 检查全局 consul_service 权限
			token, _ := GetTokenFromContext(ctx)
			hasGlobalPerm, err := m.client.CheckPermissionByToken(token, "consul_service", action)
			if err == nil && hasGlobalPerm {
				logx.Infof("用户 %s 拥有全局 consul_service:%s 权限", username, action)
				next.ServeHTTP(w, r)
				return
			}

			// 5. 检查服务组级别权限（用户直授 → 团队授权 → 角色）
			granted, gerr := m.checkServiceGroupPermission(userId, groupId, action)
			if gerr != nil {
				logx.Errorf("服务组权限检查失败: %v", gerr)
				httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
					"code":    500,
					"message": "权限检查失败",
				})
				return
			}
			if granted {
				logx.Infof("用户 %s 通过服务组授权获得 %s 权限 (group=%d)", username, action, groupId)
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

// checkServiceGroupPermission 检查用户对某个服务组的权限
//
// 权限来源（任一满足即可）:
//  1. service_group_users 中该用户的直接授权；
//  2. 用户所在团队（team_members）对服务组的授权（team_group_permissions）；
//  3. 团队授权中引用角色的权限（roles.permissions）。
//
// 权限取值: read / write / delete，支持 "*" 通配。
func (m *PermissionMiddleware) checkServiceGroupPermission(userID string, groupID int64, action string) (bool, error) {
	ctx := context.Background()

	// 1. 用户直接授权
	var directPerms []string
	err := m.db.QueryRowContext(ctx,
		`SELECT permissions FROM service_group_users WHERE group_id=$1 AND user_id=$2`,
		groupID, userID).Scan(pq.Array(&directPerms))
	if err == nil {
		for _, p := range directPerms {
			if p == "*" || p == action {
				return true, nil
			}
		}
	} else if err != sql.ErrNoRows {
		return false, err
	}

	// 2. 团队授权（直接权限）
	rows, err := m.db.QueryContext(ctx, `
		SELECT gp.permissions, COALESCE(gp.role_ids, '{}') AS role_ids
		FROM team_group_permissions gp
		JOIN team_members tm ON tm.team_id = gp.team_id
		WHERE tm.user_id = $1 AND gp.group_id = $2
	`, userID, groupID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	var roleIDs []int64
	found := false
	for rows.Next() {
		found = true
		var perms []string
		var rids []int64
		if err := rows.Scan(pq.Array(&perms), pq.Array(&rids)); err != nil {
			return false, err
		}
		for _, p := range perms {
			if p == "*" || p == action {
				return true, nil
			}
		}
		roleIDs = append(roleIDs, rids...)
	}
	if err := rows.Err(); err != nil {
		return false, err
	}
	if !found {
		return false, nil
	}

	// 3. 引用角色的权限
	if len(roleIDs) > 0 {
		rrows, err := m.db.QueryContext(ctx,
			`SELECT DISTINCT unnest(permissions) FROM roles WHERE id = ANY($1)`,
			pq.Array(roleIDs))
		if err != nil {
			logx.Errorf("查询角色权限失败: %v", err)
			return false, nil
		}
		defer rrows.Close()
		for rrows.Next() {
			var p string
			if err := rrows.Scan(&p); err != nil {
				continue
			}
			if p == "*" || p == action {
				return true, nil
			}
		}
	}

	return false, nil
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
