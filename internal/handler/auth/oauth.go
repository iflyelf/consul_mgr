// Package auth 提供认证相关的 HTTP 处理器
package auth

import (
	"fmt"
	"net/http"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/config"
	"github.com/iflyelf/consul_mgr/internal/logic/auth"
	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// oauthStateCookie OAuth state 校验用 Cookie 名
const oauthStateCookie = "consul_mgr_oauth_state"

// requestBaseURL 根据请求推导外部可访问的基础地址
//
// 优先使用反向代理透传的 X-Forwarded-Proto / X-Forwarded-Host，
// 否则回退到 Host 头，保证前端在不同域名/IP 下都能拿到正确的回调地址。
func requestBaseURL(r *http.Request) string {
	scheme := r.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		if r.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}
	host := r.Header.Get("X-Forwarded-Host")
	if host == "" {
		host = r.Host
	}
	return fmt.Sprintf("%s://%s", scheme, host)
}

// publicEndpoint 返回浏览器可达的 Casdoor 地址
//
// 未显式配置 PublicEndpoint 时，回退到服务端 Endpoint。
func publicEndpoint(c *config.Config) string {
	if c.Casdoor.PublicEndpoint != "" {
		return c.Casdoor.PublicEndpoint
	}
	return c.Casdoor.Endpoint
}

// ConfigHandler 下发前端所需的 Casdoor 运行时配置
//
// 功能：前端启动时调用，避免把 Casdoor 地址在构建期写死
//
// 请求方式：GET
// 路径：/api/auth/config
func ConfigHandler(c *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		redirectUri := requestBaseURL(r) + "/callback"
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data": map[string]interface{}{
				"endpoint":      publicEndpoint(c),
				"client_id":     c.Casdoor.ClientId,
				"organization":  c.Casdoor.OrganizationName,
				"application":   c.Casdoor.ApplicationName,
				"redirect_path": "/callback",
				"redirect_uri":  redirectUri,
			},
		})
	}
}

// LoginHandler 登录处理器
//
// 功能：生成 Casdoor 登录 URL 并跳转（浏览器访问时默认 302 跳转）
//
// 请求方式：GET
// 路径：/api/auth/login
//
// 参数:
//   format=json  返回 JSON（不跳转），便于接口调试
func LoginHandler(casdoorClient *casdoor.Client, c *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 回调地址基于当前请求动态推导，避免写死
		redirectUri := requestBaseURL(r) + "/callback"

		// 生成随机 state 并写入 Cookie，回调时校验，防止登录 CSRF
		state := casdoor.GenState()
		http.SetCookie(w, c.CookieConfig().NewCookie(oauthStateCookie, state, 600, r))

		// 调用 Logic 层
		logic := auth.NewOAuthLogic(r.Context(), casdoorClient)
		resp, err := logic.GetLoginUrl(redirectUri, state)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "生成登录 URL 失败",
				"error":   err.Error(),
			})
			return
		}

		// 浏览器直接跳转；format=json 时返回 JSON
		if r.URL.Query().Get("format") != "json" {
			http.Redirect(w, r, resp.LoginUrl, http.StatusFound)
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
func CallbackHandler(casdoorClient *casdoor.Client, c *config.Config) http.HandlerFunc {
	cookieCfg := c.CookieConfig()
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

		// 校验 OAuth state，防止登录 CSRF（Cookie 与回调参数必须一致）
		stateCookie, err := r.Cookie(oauthStateCookie)
		if err != nil || stateCookie.Value == "" || req.State != stateCookie.Value {
			logx.Errorf("OAuth state 校验失败（可能存在 CSRF）")
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "登录校验失败（state 不匹配），请重新登录",
			})
			return
		}
		http.SetCookie(w, cookieCfg.ClearCookie(oauthStateCookie))

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

		// 凭证写入 HttpOnly Cookie（JS 不可读），响应不再回传 access_token
		http.SetCookie(w, cookieCfg.NewCookie(middleware.AuthCookieName, resp.AccessToken, int(resp.ExpiresIn), r))

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "登录成功",
			"data": map[string]interface{}{
				"user_info": resp.UserInfo,
			},
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
func LogoutHandler(c *config.Config) http.HandlerFunc {
	cookieCfg := c.CookieConfig()
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取用户名（用于日志）
		username, _ := middleware.GetUsernameFromContext(r.Context())

		// 清除登录 Cookie（HttpOnly）
		http.SetCookie(w, cookieCfg.ClearCookie(middleware.AuthCookieName))

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "登出成功",
			"data": map[string]string{
				"username": username,
			},
		})
	}
}
