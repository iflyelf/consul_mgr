package user

import (
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/logic/user"
	"github.com/iflyelf/consul_mgr/internal/pkg/response"
	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// ListUsersHandler 获取用户列表
func ListUsersHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.ListUsersRequest
		if err := httpx.Parse(r, &req); err != nil {
			response.BadRequest(w, err.Error())
			return
		}

		l := user.NewListUsersLogic(r.Context(), svcCtx)
		resp, err := l.ListUsers(&req)
		if err != nil {
			response.InternalError(w, err.Error())
		} else {
			response.Success(w, resp)
		}
	}
}

// GetUserHandler 获取用户详情
func GetUserHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			response.BadRequest(w, "id 参数不能为空")
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "id 参数格式错误")
			return
		}

		l := user.NewGetUserLogic(r.Context(), svcCtx)
		resp, err := l.GetUser(id)
		if err != nil {
			response.InternalError(w, err.Error())
		} else {
			response.Success(w, resp)
		}
	}
}

// CreateUserHandler 创建用户
func CreateUserHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateUserRequest
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.BadRequest(w, err.Error())
			return
		}

		l := user.NewCreateUserLogic(r.Context(), svcCtx)
		resp, err := l.CreateUser(&req)
		if err != nil {
			response.InternalError(w, err.Error())
		} else {
			response.Success(w, resp)
		}
	}
}

// UpdateUserHandler 更新用户
func UpdateUserHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			response.BadRequest(w, "id 参数不能为空")
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "id 参数格式错误")
			return
		}

		var req types.UpdateUserRequest
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.BadRequest(w, err.Error())
			return
		}

		l := user.NewUpdateUserLogic(r.Context(), svcCtx)
		err = l.UpdateUser(id, &req)
		if err != nil {
			response.InternalError(w, err.Error())
		} else {
			response.Success(w, nil)
		}
	}
}

// DeleteUserHandler 删除用户
func DeleteUserHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			response.BadRequest(w, "id 参数不能为空")
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "id 参数格式错误")
			return
		}

		l := user.NewDeleteUserLogic(r.Context(), svcCtx)
		err = l.DeleteUser(id)
		if err != nil {
			response.InternalError(w, err.Error())
		} else {
			response.Success(w, nil)
		}
	}
}

// ChangePasswordHandler 修改密码
func ChangePasswordHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			response.BadRequest(w, "id 参数不能为空")
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "id 参数格式错误")
			return
		}

		var req types.ChangePasswordRequest
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.BadRequest(w, err.Error())
			return
		}

		l := user.NewChangePasswordLogic(r.Context(), svcCtx)
		err = l.ChangePassword(id, &req)
		if err != nil {
			response.InternalError(w, err.Error())
		} else {
			response.Success(w, nil)
		}
	}
}

// AssignRolesHandler 分配角色
func AssignRolesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		if idStr == "" {
			response.BadRequest(w, "id 参数不能为空")
			return
		}

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "id 参数格式错误")
			return
		}

		var req types.AssignRolesRequest
		if err := httpx.ParseJsonBody(r, &req); err != nil {
			response.BadRequest(w, err.Error())
			return
		}

		l := user.NewAssignRolesLogic(r.Context(), svcCtx)
		err = l.AssignRoles(id, &req)
		if err != nil {
			response.InternalError(w, err.Error())
		} else {
			response.Success(w, nil)
		}
	}
}
