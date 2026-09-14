// Package auth 提供认证相关的 HTTP 处理器
package auth

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/logic/auth"
	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// LoginHandler 登录处理器
//
// 功能：返回 Casdoor 登录 URL
//
// 请求方式：GET
// 路径：/api/auth/login
func LoginHandler(casdoorClient *casdoor.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取回调地址（从配置或环境变量）
		redirectUri := "http://localhost:8080/api/auth/callback"

		// 调用 Logic 层
		logic := auth.NewOAuthLogic(r.Context(), casdoorClient)
		resp, err := logic.GetLoginUrl(redirectUri)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "生成登录 URL 失败",
				"error":   err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data":    resp,
		})
	}
}

// CallbackHandler OAuth 回调处理器
//
// 功能：处理 Casdoor 回调，换取 Token
//
// 请求方式：GET
// 路径：/api/auth/callback
func CallbackHandler(casdoorClient *casdoor.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 解析请求参数
		var req types.CallbackRequest
		req.Code = r.URL.Query().Get("code")
		req.State = r.URL.Query().Get("state")

		// 验证参数
		if req.Code == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "缺少授权码",
			})
			return
		}

		// 调用 Logic 层
		logic := auth.NewOAuthLogic(r.Context(), casdoorClient)
		resp, err := logic.HandleCallback(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "登录失败",
				"error":   err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "登录成功",
			"data":    resp,
		})
	}
}

// RefreshTokenHandler 刷新 Token 处理器
//
// 功能：使用 Refresh Token 刷新 Access Token
//
// 请求方式：POST
// 路径：/api/auth/refresh
func RefreshTokenHandler(casdoorClient *casdoor.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 解析请求体
		var req types.RefreshTokenRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "请求参数错误",
				"error":   err.Error(),
			})
			return
		}

		// 调用 Logic 层
		logic := auth.NewOAuthLogic(r.Context(), casdoorClient)
		resp, err := logic.RefreshToken(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "刷新 Token 失败",
				"error":   err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data":    resp,
		})
	}
}

// GetUserInfoHandler 获取当前用户信息处理器
//
// 功能：获取当前登录用户的详细信息
//
// 请求方式：GET
// 路径：/api/auth/userinfo
// 需要认证：是
func GetUserInfoHandler(casdoorClient *casdoor.Client) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// 从 Context 获取 Token
		token, ok := middleware.GetTokenFromContext(ctx)
		if !ok {
			httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{
				"code":    401,
				"message": "未认证",
			})
			return
		}

		// 调用 Logic 层
		logic := auth.NewOAuthLogic(ctx, casdoorClient)
		userInfo, err := logic.GetCurrentUser(token)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "获取用户信息失败",
				"error":   err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data":    userInfo,
		})
	}
}

// LogoutHandler 登出处理器
//
// 功能：登出（客户端需清除 Token）
//
// 请求方式：POST
// 路径：/api/auth/logout
// 需要认证：是
func LogoutHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取用户名（用于日志）
		username, _ := middleware.GetUsernameFromContext(r.Context())

		// 注意：JWT Token 是无状态的，服务端无法主动失效
		// 需要客户端清除 Token
		// 如果需要服务端主动失效，需要维护 Token 黑名单

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "登出成功，请清除本地 Token",
			"data": map[string]string{
				"username": username,
			},
		})
	}
}
