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
	// casdoorFn 运行时获取当前 Casdoor 客户端（支持热重载后自动用新客户端）
	casdoorFn func() *casdoor.Client
}

// NewCasdoorAuthMiddleware 创建认证中间件
//
// 参数:
//
//	casdoorFn - 返回当前 Casdoor 客户端的函数（每次请求时调用，支持热重载）
//
// 返回:
//
//	*CasdoorAuthMiddleware - 中间件实例
func NewCasdoorAuthMiddleware(casdoorFn func() *casdoor.Client) *CasdoorAuthMiddleware {
	return &CasdoorAuthMiddleware{
		casdoorFn: casdoorFn,
	}
}

// Handle 处理认证逻辑
//
// 参数:
//
//	next - 下一个处理函数
//
// 返回:
//
//	http.HandlerFunc - 包装后的处理函数
//
// 功能流程：
//  1. 提取 Authorization Header
//  2. 解析 Bearer Token
//  3. 验证 Token 有效性
//  4. 提取用户信息
//  5. 存入 Context
//  6. 调用下一个处理器
func (m *CasdoorAuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 1. 提取 Token
		token := ExtractToken(r)
		if token == "" {
			logx.Error("未提供认证令牌")
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "未提供认证令牌，请先登录",
			})
			return
		}

		// 2. 权威校验 Token（Casdoor get-account：查库 + 过期 + 用户状态）
		//
		// 安全说明：绝不直接信任令牌载荷（可被伪造）。此处回源 Casdoor 校验令牌
		// 有效性并获取最新用户信息（含 isAdmin / isForbidden），结果带短 TTL 缓存。
		user, err := m.casdoorFn().ValidateToken(token)
		if err != nil {
			logx.Errorf("Token 校验失败: %v", err)
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "认证令牌无效或已过期，请重新登录",
			})
			return
		}
		if user == nil {
			logx.Error("Token 校验后无用户信息")
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "用户信息无效",
			})
			return
		}

		// 3. 将权威校验后的用户信息存入 Context
		ctx := r.Context()
		ctx = context.WithValue(ctx, "userId", user.Id)
		ctx = context.WithValue(ctx, "username", user.Name)
		ctx = context.WithValue(ctx, "userEmail", user.Email)
		ctx = context.WithValue(ctx, "userDisplayName", user.DisplayName)
		ctx = context.WithValue(ctx, "userAvatar", user.Avatar)
		ctx = context.WithValue(ctx, "isAdmin", user.IsAdmin)
		ctx = context.WithValue(ctx, "isGlobalAdmin", user.IsGlobalAdmin)
		ctx = context.WithValue(ctx, "token", token)

		// 存储角色列表（供角色校验使用）
		if len(user.Roles) > 0 {
			roleNames := make([]string, 0, len(user.Roles))
			for _, role := range user.Roles {
				roleNames = append(roleNames, role.Name)
			}
			ctx = context.WithValue(ctx, "userRoles", roleNames)
		}

		// 兼容：存储重建的 Claims（仅供内部辅助函数读取）
		ctx = context.WithValue(ctx, "claims", &casdoor.Claims{User: user})

		// 4. 记录日志
		logx.Infof("用户认证成功: %s (%s)", user.Name, user.Id)

		// 5. 调用下一个处理器
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

// AuthCookieName 登录凭证 Cookie 名（HttpOnly，JS 不可读，降低 XSS 窃取风险）
const AuthCookieName = "consul_mgr_token"

// ExtractToken 从请求中提取 Token。
//
// 优先级：
//  1. HttpOnly Cookie（推荐，浏览器自动携带，JS 不可读）
//  2. Authorization: Bearer <token>（API 调用/旧会话兼容）
//  3. token 查询参数（不推荐）
//
// 说明：这是全项目**唯一**的取 token 实现。登录凭证已改为 HttpOnly Cookie，
// handler 一律从 Context 取 token（GetTokenFromContext），不要另写只读
// Authorization 的实现，否则会出现「Cookie 登录态在部分接口失效」的问题。
func ExtractToken(r *http.Request) string {
	if c, err := r.Cookie(AuthCookieName); err == nil && c.Value != "" {
		return c.Value
	}

	bearerToken := r.Header.Get("Authorization")
	if bearerToken != "" {
		parts := strings.Split(bearerToken, " ")
		if len(parts) == 2 && strings.ToLower(parts[0]) == "bearer" {
			return parts[1]
		}
	}

	return r.URL.Query().Get("token")
}

// GetUserIdFromContext 从 Context 获取用户 ID
//
// 参数:
//
//	ctx - 上下文
//
// 返回:
//
//	string - 用户 ID
//	bool - 是否存在
func GetUserIdFromContext(ctx context.Context) (string, bool) {
	userId, ok := ctx.Value("userId").(string)
	return userId, ok
}

// GetUsernameFromContext 从 Context 获取用户名
//
// 参数:
//
//	ctx - 上下文
//
// 返回:
//
//	string - 用户名
//	bool - 是否存在
func GetUsernameFromContext(ctx context.Context) (string, bool) {
	username, ok := ctx.Value("username").(string)
	return username, ok
}

// GetUserEmailFromContext 从 Context 获取用户邮箱
//
// 参数:
//
//	ctx - 上下文
//
// 返回:
//
//	string - 用户邮箱
//	bool - 是否存在
func GetUserEmailFromContext(ctx context.Context) (string, bool) {
	email, ok := ctx.Value("userEmail").(string)
	return email, ok
}

// GetUserRolesFromContext 从 Context 获取用户角色列表
//
// 参数:
//
//	ctx - 上下文
//
// 返回:
//
//	[]string - 角色列表
//	bool - 是否存在
func GetUserRolesFromContext(ctx context.Context) ([]string, bool) {
	roles, ok := ctx.Value("userRoles").([]string)
	return roles, ok
}

// IsAdminFromContext 从 Context 判断是否为管理员
//
// 参数:
//
//	ctx - 上下文
//
// 返回:
//
//	bool - 是否为管理员
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
//
//	ctx - 上下文
//
// 返回:
//
//	bool - 是否为全局管理员
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
//
//	ctx - 上下文
//
// 返回:
//
//	*casdoor.Claims - Claims 信息
//	bool - 是否存在
func GetClaimsFromContext(ctx context.Context) (*casdoor.Claims, bool) {
	claims, ok := ctx.Value("claims").(*casdoor.Claims)
	return claims, ok
}

// GetTokenFromContext 从 Context 获取 Token
//
// 参数:
//
//	ctx - 上下文
//
// 返回:
//
//	string - Token
//	bool - 是否存在
func GetTokenFromContext(ctx context.Context) (string, bool) {
	token, ok := ctx.Value("token").(string)
	return token, ok
}
