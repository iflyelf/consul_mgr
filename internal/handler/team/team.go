// Package team 提供团队管理的 HTTP 处理器
package team

import (
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"

	"github.com/iflyelf/consul_mgr/internal/logic/team"
	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// CreateTeamRequest 创建团队请求
type CreateTeamRequest struct {
	Name        string `json:"name" validate:"required"`
	Code        string `json:"code,optional"`
	Description string `json:"description,optional"`
}

// UpdateTeamRequest 更新团队请求
type UpdateTeamRequest struct {
	Name        string `json:"name,optional"`
	Description string `json:"description,optional"`
	Status      *int   `json:"status,optional"`
}

// AddMemberRequest 添加成员请求
type AddMemberRequest struct {
	UserID      string `json:"user_id" validate:"required"`
	Username    string `json:"username,optional"`
	DisplayName string `json:"display_name,optional"`
}

// GrantPermissionRequest 授予服务组权限请求
type GrantPermissionRequest struct {
	GroupID     int64    `json:"group_id" validate:"required"`
	Permissions []string `json:"permissions,optional"`
	RoleIDs     []int64  `json:"role_ids,optional"`
}

func teamID(r *http.Request) (int64, error) {
	return strconv.ParseInt(pathvar.Vars(r)["id"], 10, 64)
}

// ListTeamsHandler 团队列表
func ListTeamsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := team.NewTeamLogic(r.Context(), ctx.RawDB)
		list, err := l.ListTeams(r.URL.Query().Get("keyword"))
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

// CreateTeamHandler 创建团队
func CreateTeamHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateTeamRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "参数错误: " + err.Error()})
			return
		}
		username, _ := middleware.GetUsernameFromContext(r.Context())
		l := team.NewTeamLogic(r.Context(), ctx.RawDB)
		res, err := l.CreateTeam(req.Name, req.Code, req.Description, username)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "创建成功", "data": res})
	}
}

// UpdateTeamHandler 更新团队
func UpdateTeamHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := teamID(r)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "无效的 ID"})
			return
		}
		var req UpdateTeamRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "参数错误: " + err.Error()})
			return
		}
		l := team.NewTeamLogic(r.Context(), ctx.RawDB)
		res, err := l.UpdateTeam(id, req.Name, req.Description, req.Status)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "更新成功", "data": res})
	}
}

// DeleteTeamHandler 删除团队
func DeleteTeamHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := teamID(r)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "无效的 ID"})
			return
		}
		l := team.NewTeamLogic(r.Context(), ctx.RawDB)
		if err := l.DeleteTeam(id); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "删除成功"})
	}
}

// ListMembersHandler 团队成员列表
func ListMembersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := teamID(r)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "无效的 ID"})
			return
		}
		l := team.NewTeamLogic(r.Context(), ctx.RawDB)
		list, err := l.ListMembers(id)
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

// AddMemberHandler 添加团队成员
func AddMemberHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := teamID(r)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "无效的 ID"})
			return
		}
		var req AddMemberRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "参数错误: " + err.Error()})
			return
		}
		l := team.NewTeamLogic(r.Context(), ctx.RawDB)
		if err := l.AddMember(id, req.UserID, req.Username, req.DisplayName); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "添加成功"})
	}
}

// RemoveMemberHandler 移除团队成员
func RemoveMemberHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := teamID(r)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "无效的 ID"})
			return
		}
		userID := r.URL.Query().Get("user_id")
		if userID == "" {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "user_id 不能为空"})
			return
		}
		l := team.NewTeamLogic(r.Context(), ctx.RawDB)
		if err := l.RemoveMember(id, userID); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "移除成功"})
	}
}

// ListGroupPermissionsHandler 团队的服务组授权列表
func ListGroupPermissionsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := teamID(r)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "无效的 ID"})
			return
		}
		l := team.NewTeamLogic(r.Context(), ctx.RawDB)
		list, err := l.ListGroupPermissions(id)
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

// GrantGroupPermissionHandler 授予团队服务组权限
func GrantGroupPermissionHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := teamID(r)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "无效的 ID"})
			return
		}
		var req GrantPermissionRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "参数错误: " + err.Error()})
			return
		}
		l := team.NewTeamLogic(r.Context(), ctx.RawDB)
		if err := l.GrantGroupPermission(id, req.GroupID, req.Permissions, req.RoleIDs); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "授权成功"})
	}
}

// RevokeGroupPermissionHandler 撤销团队服务组权限
func RevokeGroupPermissionHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := teamID(r)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "无效的 ID"})
			return
		}
		groupID, err := strconv.ParseInt(r.URL.Query().Get("group_id"), 10, 64)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "group_id 无效"})
			return
		}
		l := team.NewTeamLogic(r.Context(), ctx.RawDB)
		if err := l.RevokeGroupPermission(id, groupID); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "撤销成功"})
	}
}
