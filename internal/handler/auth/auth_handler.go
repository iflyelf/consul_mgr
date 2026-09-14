package auth

import (
	"net/http"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/logic/auth"
	"github.com/iflyelf/consul_mgr/internal/pkg/response"
	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// LoginHandler 登录处理器
func LoginHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.LoginRequest
		if err := httpx.Parse(r, &req); err != nil {
			response.BadRequest(w, err.Error())
			return
		}
		
		l := auth.NewLoginLogic(r.Context(), svcCtx)
		resp, err := l.Login(&req)
		if err != nil {
			response.Error(w, 401, err.Error())
			return
		}
		
		response.Success(w, resp)
	}
}

// LogoutHandler 登出处理器
func LogoutHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 由于使用 JWT，登出只需要客户端删除 Token
		response.Success(w, map[string]string{"message": "登出成功"})
	}
}

// GetUserInfoHandler 获取用户信息处理器
func GetUserInfoHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := auth.NewGetUserInfoLogic(r.Context(), svcCtx)
		resp, err := l.GetUserInfo()
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		
		response.Success(w, resp)
	}
}
