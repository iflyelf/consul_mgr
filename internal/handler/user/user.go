// Package user 提供「用户管理」的 HTTP 处理器
//
// 说明：用户体系由 Casdoor 维护（认证、密码、角色），
// 本模块仅提供只读列表，供「人员组织 → 用户管理」展示与团队选人。
package user

import (
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/logic/user"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// ListUsersHandler 用户列表（来自 Casdoor，支持关键字/分页）
//
// 查询参数:
//   keyword   - 用户名 / 显示名 / 邮箱 模糊搜索
//   page      - 页码（默认 1）
//   page_size - 每页条数（默认 20）
//
// 说明:
//   全量用户经 Redis 缓存后在内存过滤与分页，避免人员多时每次请求都打 Casdoor。
func ListUsersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		keyword := r.URL.Query().Get("keyword")

		l := user.NewUserLogic(r.Context(), ctx.CasdoorClient, ctx.Cache)
		list, total, err := l.ListUsers(keyword, page, pageSize)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "success",
			"data": map[string]interface{}{
				"list":      list,
				"total":     total,
				"page":      page,
				"page_size": pageSize,
			},
		})
	}
}
