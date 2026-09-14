package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/iflyelf/consul_mgr/internal/pkg/jwt"
	"github.com/iflyelf/consul_mgr/internal/pkg/response"
)

// AuthMiddleware JWT 认证中间件
type AuthMiddleware struct {
	jwtManager *jwt.JWTManager
}

// NewAuthMiddleware 创建认证中间件
func NewAuthMiddleware(jwtManager *jwt.JWTManager) *AuthMiddleware {
	return &AuthMiddleware{
		jwtManager: jwtManager,
	}
}

// Handle 处理认证
func (m *AuthMiddleware) Handle(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 从 Header 中获取 Token
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			response.Unauthorized(w, "未提供认证令牌")
			return
		}
		
		// 检查 Bearer 前缀
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || parts[0] != "Bearer" {
			response.Unauthorized(w, "认证令牌格式错误")
			return
		}
		
		token := parts[1]
		
		// 验证 Token
		claims, err := m.jwtManager.VerifyToken(token)
		if err != nil {
			response.Unauthorized(w, "认证令牌无效或已过期")
			return
		}
		
		// 将用户信息存入 Context
		ctx := context.WithValue(r.Context(), "user_id", claims.UserID)
		ctx = context.WithValue(ctx, "username", claims.Username)
		
		// 继续处理请求
		next(w, r.WithContext(ctx))
	}
}

// GetUserID 从 Context 中获取用户 ID
func GetUserID(ctx context.Context) int64 {
	if userID, ok := ctx.Value("user_id").(int64); ok {
		return userID
	}
	return 0
}

// GetUsername 从 Context 中获取用户名
func GetUsername(ctx context.Context) string {
	if username, ok := ctx.Value("username").(string); ok {
		return username
	}
	return ""
}
