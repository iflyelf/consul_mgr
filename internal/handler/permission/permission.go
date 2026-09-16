// Package permission 提供权限管理的 HTTP 处理器
package permission

import (
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/logic/permission"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// GrantUserPermissionRequest 授予用户权限请求
type GrantUserPermissionRequest struct {
	GroupID     int64    `json:"group_id" validate:"required"`
	UserID      string   `json:"user_id" validate:"required"`
	Permissions []string `json:"permissions" validate:"required"`
}

// GrantRolePermissionRequest 授予角色权限请求
type GrantRolePermissionRequest struct {
	GroupID     int64    `json:"group_id" validate:"required"`
	RoleName    string   `json:"role_name" validate:"required"`
	Permissions []string `json:"permissions" validate:"required"`
}

// GrantUserPermissionHandler 授予用户权限
func GrantUserPermissionHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req GrantUserPermissionRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "参数错误: " + err.Error(),
			})
			return
		}
		
		logic := permission.NewPermissionLogic(r.Context(), ctx.RawDB)
		result, err := logic.GrantUserPermission(req.GroupID, req.UserID, req.Permissions)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "授予权限失败: " + err.Error(),
			})
			return
		}
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "授予权限成功",
			"data":    result,
		})
	}
}

// RevokeUserPermissionHandler 撤销用户权限
func RevokeUserPermissionHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID, _ := strconv.ParseInt(r.URL.Query().Get("group_id"), 10, 64)
		userID := r.URL.Query().Get("user_id")
		
		if groupID == 0 || userID == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "缺少必要参数",
			})
			return
		}
		
		logic := permission.NewPermissionLogic(r.Context(), ctx.RawDB)
		err := logic.RevokeUserPermission(groupID, userID)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "撤销权限失败: " + err.Error(),
			})
			return
		}
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "撤销权限成功",
		})
	}
}

// ListUserPermissionsHandler 查询用户权限列表
func ListUserPermissionsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID, _ := strconv.ParseInt(r.URL.Query().Get("group_id"), 10, 64)
		
		if groupID == 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "缺少 group_id 参数",
			})
			return
		}
		
		logic := permission.NewPermissionLogic(r.Context(), ctx.RawDB)
		perms, err := logic.ListUserPermissions(groupID)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": err.Error(),
			})
			return
		}
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data":    perms,
		})
	}
}

// GrantRolePermissionHandler 授予角色权限
func GrantRolePermissionHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req GrantRolePermissionRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "参数错误: " + err.Error(),
			})
			return
		}
		
		logic := permission.NewPermissionLogic(r.Context(), ctx.RawDB)
		result, err := logic.GrantRolePermission(req.GroupID, req.RoleName, req.Permissions)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "授予权限失败: " + err.Error(),
			})
			return
		}
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "授予权限成功",
			"data":    result,
		})
	}
}

// RevokeRolePermissionHandler 撤销角色权限
func RevokeRolePermissionHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID, _ := strconv.ParseInt(r.URL.Query().Get("group_id"), 10, 64)
		roleName := r.URL.Query().Get("role_name")
		
		if groupID == 0 || roleName == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "缺少必要参数",
			})
			return
		}
		
		logic := permission.NewPermissionLogic(r.Context(), ctx.RawDB)
		err := logic.RevokeRolePermission(groupID, roleName)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "撤销权限失败: " + err.Error(),
			})
			return
		}
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "撤销权限成功",
		})
	}
}

// ListRolePermissionsHandler 查询角色权限列表
func ListRolePermissionsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID, _ := strconv.ParseInt(r.URL.Query().Get("group_id"), 10, 64)
		
		if groupID == 0 {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "缺少 group_id 参数",
			})
			return
		}
		
		logic := permission.NewPermissionLogic(r.Context(), ctx.RawDB)
		perms, err := logic.ListRolePermissions(groupID)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": err.Error(),
			})
			return
		}
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data":    perms,
		})
	}
}
