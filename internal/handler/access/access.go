// Package access 提供 handler 层的服务级权限校验与列表过滤辅助
//
// 说明:
//   - 中间件已对「查询参数中带 service_name/service」的请求做校验；
//   - 对于服务名在请求体/路径中的写操作，以及需要按授权范围过滤的列表，
//     由 handler 调用本包辅助函数完成。
package access

import (
	"net/http"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// CheckService 校验当前用户对指定服务的权限
//
// 管理员 / 全局权限由中间件放行，此处仅针对服务组授权。
//
// 返回:
//   true  - 通过（或无需校验）
//   false - 已写出 403 响应
func CheckService(ctx *svc.ServiceContext, w http.ResponseWriter, r *http.Request, groupID int64, serviceName, action string) bool {
	if groupID == 0 {
		return true // 无服务组上下文，交由其他校验
	}
	if middleware.IsGlobalAdminFromContext(r.Context()) || middleware.IsAdminFromContext(r.Context()) {
		return true
	}
	userID, ok := middleware.GetUserIdFromContext(r.Context())
	if !ok {
		httpx.WriteJson(w, http.StatusUnauthorized, map[string]interface{}{"code": 401, "message": "未认证，请先登录"})
		return false
	}
	allowed, err := ctx.Perm.Check(r.Context(), userID, groupID, serviceName, action)
	if err != nil {
		httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{"code": 500, "message": "权限检查失败"})
		return false
	}
	if !allowed {
		httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
			"code":    403,
			"message": "没有权限操作该 Service（需管理员在「人员组织 → 团队管理」中授权）",
		})
		return false
	}
	return true
}

// AllowedSet 便捷方法：返回当前请求用户对服务组的服务白名单
//
// 返回 nil 表示不限制（管理员或已授权全部服务）。
func AllowedSet(ctx *svc.ServiceContext, r *http.Request, groupID int64) map[string]bool {
	if middleware.IsGlobalAdminFromContext(r.Context()) || middleware.IsAdminFromContext(r.Context()) {
		return nil
	}
	userID, ok := middleware.GetUserIdFromContext(r.Context())
	if !ok {
		return map[string]bool{}
	}
	all, list, err := ctx.Perm.AllowedServices(r.Context(), userID, groupID)
	if err != nil {
		return map[string]bool{}
	}
	if all {
		return nil
	}
	set := make(map[string]bool, len(list))
	for _, s := range list {
		set[s] = true
	}
	return set
}

// ContainsFold 判断服务名是否在白名单内（大小写敏感，与 Consul 一致）
func ContainsFold(set map[string]bool, name string) bool {
	if set == nil {
		return true
	}
	if set[name] {
		return true
	}
	for k := range set {
		if strings.EqualFold(k, name) {
			return true
		}
	}
	return false
}
