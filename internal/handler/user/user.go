// Package user 提供「用户管理」的 HTTP 处理器
//
// 说明：用户体系由 Casdoor 维护（认证、密码、角色），
// 本模块仅提供只读列表，供「人员组织 → 用户管理」展示与团队选人。
package user

import (
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/svc"
)

// ListUsersHandler 用户列表（来自 Casdoor，支持关键字/分页）
func ListUsersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		users, err := ctx.CasdoorClient.ListUsers()
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}

		// 关键字过滤（用户名/显示名/邮箱）
		keyword := strings.ToLower(r.URL.Query().Get("keyword"))
		if keyword != "" {
			filtered := make([]interface{}, 0, len(users))
			for _, u := range users {
				if strings.Contains(strings.ToLower(u.Name), keyword) ||
					strings.Contains(strings.ToLower(u.DisplayName), keyword) ||
					strings.Contains(strings.ToLower(u.Email), keyword) {
					filtered = append(filtered, u)
				}
			}
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 200, "message": "success",
				"data": map[string]interface{}{"list": filtered, "total": len(filtered)},
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "success",
			"data": map[string]interface{}{"list": users, "total": len(users)},
		})
	}
}
