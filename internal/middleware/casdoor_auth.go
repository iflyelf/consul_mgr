// Package middleware 提供 HTTP 中间件
package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
)

// CasdoorAuthMiddleware Casdoor 认证中间件
//
// 功能：
//   - 从请求头提取 Token
//   - 验证 Token 有效性
//   - 提取用户信息到 Context
//   - 处理认证错误
type CasdoorAuthMiddleware struct {
	client *casdoor.Client
}

// NewCasdoorAuthMiddleware 创建认证中间件
//
// 参数:
//   client - Casdoor 客户端实例
//
// 返回:
//   *CasdoorAuthMiddleware - 中间件实例
func NewCasdoorAuthMiddleware(client *casdoor.Client) *CasdoorAuthMiddleware {
	return &CasdoorAuthMiddleware{
		client: client,
	}
}

// Handle 处理认证逻辑
//
// 参数:
//   next - 下一个处理函数
//
// 返回:
//   http.HandlerFunc - 包装后的处理函数
//
// 功能流程：
//   1. 提取 Authorization Header
//   2. 解析 Bearer Token
//   3. 验证 Token 有效性
//   4. 提取用户信息
//   5. 存入 Context
//   6. 调用下一个处理器
func (m *CasdoorAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 提取 Token
		token := extractToken(r)
		if token == "" {
			logx.Error("未提供认证令牌")
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "未提供认证令牌，请先登录",
			})
			return
		}

		// 2. 解析和验证 Token
		claims, err := m.client.ParseToken(token)
		if err != nil {
			logx.Errorf("Token 验证失败: %v", err)
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "认证令牌无效或已过期，请重新登录",
			})
			return
		}

		// 3. 检查用户信息
		if claims.User == nil {
			logx.Error("Token 中没有用户信息")
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "用户信息无效",
			})
			return
		}

		// 4. 检查用户是否被禁用
		if claims.User.IsForbidden {
			logx.Warnf("用户已被禁用: %s", claims.User.Name)
			httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
				"code":    403,
				"message": "您的账号已被禁用，请联系管理员",
			})
			return
		}

		// 5. 将用户信息存入 Context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "userId", claims.User.Id)
		ctx = context.WithValue(ctx, "username", claims.User.Name)
		ctx = context.WithValue(ctx, "userEmail", claims.User.Email)
		ctx = context.WithValue(ctx, "userDisplayName", claims.User.DisplayName)
		ctx = context.WithValue(ctx, "userAvatar", claims.User.Avatar)
		ctx = context.WithValue(ctx, "isAdmin", claims.User.IsAdmin)
		ctx = context.WithValue(ctx, "isGlobalAdmin", claims.User.IsGlobalAdmin)
		ctx = context.WithValue(ctx, "token", token)
		
		// 存储角色列表
		if len(claims.User.Roles) > 0 {
			roleNames := make([]string, 0, len(claims.User.Roles))
			for _, role := range claims.User.Roles {
				roleNames = append(roleNames, role.Name)
			}
			ctx = context.WithValue(ctx, "userRoles", roleNames)
		}

		// 存储完整的 Claims 信息（供后续使用）
		ctx = context.WithValue(ctx, "claims", claims)

		// 6. 记录日志
		logx.Infof("用户认证成功: %s (%s)", claims.User.Name, claims.User.Id)

		// 7. 调用下一个处理器
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// extractToken 从请求中提取 Token
//
// 参数:
//   r - HTTP 请求
//
// 返回:
//   string - Token 字符串，如果没有则返回空字符串
//
// 支持的格式：
//   - Authorization: Bearer <token>
//   - token 查询参数（不推荐，仅用于某些特殊场景）
func extractToken(r *http.Request) string {
	// 1. 从 Authorization Header 提取
	bearerToken := r.Header.Get("Authorization")
	if bearerToken != "" {
		// 格式：Bearer <token>
		parts := strings.Split(bearerToken, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	// 2. 从查询参数提取（备选方案，不推荐）
	token := r.URL.Query().Get("token")
	if token != "" {
		return token
	}

	return ""
}

// GetUserIdFromContext 从 Context 获取用户 ID
//
// 参数:
//   ctx - 上下文
//
// 返回:
//   string - 用户 ID
//   bool - 是否存在
func GetUserIdFromContext(ctx context.Context) (string, bool) {
	userId, ok := ctx.Value("userId").(string)
	return userId, ok
}

// GetUsernameFromContext 从 Context 获取用户名
//
// 参数:
//   ctx - 上下文
//
// 返回:
//   string - 用户名
//   bool - 是否存在
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value("username").(string)
	return username, ok
}

// GetUserEmailFromContext 从 Context 获取用户邮箱
//
// 参数:
//   ctx - 上下文
//
// 返回:
//   string - 用户邮箱
//   bool - 是否存在
func GetUserEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value("userEmail").(string)
	return email, ok
}

// GetUserRolesFromContext 从 Context 获取用户角色列表
//
// 参数:
//   ctx - 上下文
//
// 返回:
//   []string - 角色列表
//   bool - 是否存在
func GetUserRolesFromContext(ctx context.Context) ([]string, bool) {
	roles, ok := ctx.Value("userRoles").([]string)
	return roles, ok
}

// IsAdminFromContext 从 Context 判断是否为管理员
//
// 参数:
//   ctx - 上下文
//
// 返回:
//   bool - 是否为管理员
func IsAdminFromContext(ctx context.Context) bool {
	isAdmin, ok := ctx.Value("isAdmin").(bool)
	if !ok {
		return false
	}
	return isAdmin
}

// IsGlobalAdminFromContext 从 Context 判断是否为全局管理员
//
// 参数:
//   ctx - 上下文
//
// 返回:
//   bool - 是否为全局管理员
func IsGlobalAdminFromContext(ctx context.Context) bool {
	isGlobalAdmin, ok := ctx.Value("isGlobalAdmin").(bool)
	if !ok {
		return false
	}
	return isGlobalAdmin
}

// GetClaimsFromContext 从 Context 获取完整的 Claims 信息
//
// 参数:
//   ctx - 上下文
//
// 返回:
//   *casdoor.Claims - Claims 信息
//   bool - 是否存在
func GetClaimsFromContext(ctx context.Context) (*casdoor.Claims, bool) {
	claims, ok := ctx.Value("claims").(*casdoor.Claims)
	return claims, ok
}

// GetTokenFromContext 从 Context 获取 Token
//
// 参数:
//   ctx - 上下文
//
// 返回:
//   string - Token
//   bool - 是否存在
func GetTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value("token").(string)
	return token, ok
}
