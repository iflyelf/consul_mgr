// Package role 提供角色管理的 HTTP 处理器
package role

import (
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"

	"github.com/iflyelf/consul_mgr/internal/logic/role"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name        string   `json:"name" validate:"required"`
	Code        string   `json:"code,optional"`
	Description string   `json:"description,optional"`
	Permissions []string `json:"permissions,optional"`
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Name        string   `json:"name,optional"`
	Description string   `json:"description,optional"`
	Permissions []string `json:"permissions,optional"`
}

func roleID(r *http.Request) (int64, error) {
	return strconv.ParseInt(pathvar.Vars(r)["id"], 10, 64)
}

// ListRolesHandler 角色列表
func ListRolesHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := role.NewRoleLogic(r.Context(), ctx.RawDB)
		list, err := l.ListRoles(r.URL.Query().Get("keyword"))
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "success",
			"data": map[string]interface{}{"list": list, "total": len(list)},
		})
	}
}

// CreateRoleHandler 创建角色
func CreateRoleHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateRoleRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "参数错误: " + err.Error()})
			return
		}
		l := role.NewRoleLogic(r.Context(), ctx.RawDB)
		res, err := l.CreateRole(req.Name, req.Code, req.Description, req.Permissions)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "创建成功", "data": res})
	}
}

// UpdateRoleHandler 更新角色
func UpdateRoleHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := roleID(r)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "无效的 ID"})
			return
		}
		var req UpdateRoleRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "参数错误: " + err.Error()})
			return
		}
		l := role.NewRoleLogic(r.Context(), ctx.RawDB)
		res, err := l.UpdateRole(id, req.Name, req.Description, req.Permissions)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "更新成功", "data": res})
	}
}

// DeleteRoleHandler 删除角色
func DeleteRoleHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := roleID(r)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "无效的 ID"})
			return
		}
		l := role.NewRoleLogic(r.Context(), ctx.RawDB)
		if err := l.DeleteRole(id); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "删除成功"})
	}
}

// ListCasdoorRolesHandler 展示 Casdoor 角色（只读）
func ListCasdoorRolesHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		list, err := ctx.Casdoor().ListCasdoorRoles()
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "success",
			"data": map[string]interface{}{"list": list, "total": len(list)},
		})
	}
}
