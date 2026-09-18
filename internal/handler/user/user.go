// Package user 提供「用户管理」的 HTTP 处理器
//
// 说明：用户体系由 Casdoor 维护（认证、密码、角色）。
// 本模块提供用户列表与增删改（与 FlyIAM 对齐），用户唯一存储于 Casdoor。
package user

import (
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"

	"github.com/iflyelf/consul_mgr/internal/logic/user"
	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// userUpsertRequest 用户新增/更新请求
//
// Name 标记为 optional：更新时来自 URL 路径，不能在解析阶段强制校验。
// 新增时由处理器显式校验非空。
type userUpsertRequest struct {
	Name        string            `json:"name,optional"`
	DisplayName string            `json:"displayName,optional"`
	Avatar      string            `json:"avatar,optional"`
	Email       string            `json:"email,optional"`
	Phone       string            `json:"phone,optional"`
	Password    string            `json:"password,optional"`
	Properties  map[string]string `json:"properties,optional"`
}

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
		// refresh=1 强制绕过缓存，立即从 Casdoor 拉取最新列表
		refresh := r.URL.Query().Get("refresh") == "1" || r.URL.Query().Get("refresh") == "true"

		l := user.NewUserLogic(r.Context(), ctx.Casdoor(), ctx.Cache, ctx.Config.Casdoor.UserCacheTTL)
		list, total, err := l.ListUsers(keyword, page, pageSize, refresh)
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

// CreateUserHandler 新增用户
func CreateUserHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req userUpsertRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "参数错误: " + err.Error()})
			return
		}
		if req.Name == "" {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "域账号不能为空"})
			return
		}
		l := user.NewUserLogic(r.Context(), ctx.Casdoor(), ctx.Cache, ctx.Config.Casdoor.UserCacheTTL)
		if err := l.CreateUser(casdoor.UserUpsert{
			Name: req.Name, DisplayName: req.DisplayName, Avatar: req.Avatar, Email: req.Email,
			Phone: req.Phone, Password: req.Password, Properties: req.Properties,
		}, ctx.Config.Casdoor.DefaultPassword); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "创建成功"})
	}
}

// UpdateUserHandler 更新用户
func UpdateUserHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := pathvar.Vars(r)["name"]
		if name == "" {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "缺少用户标识"})
			return
		}
		var req userUpsertRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "参数错误: " + err.Error()})
			return
		}
		req.Name = name
		l := user.NewUserLogic(r.Context(), ctx.Casdoor(), ctx.Cache, ctx.Config.Casdoor.UserCacheTTL)
		if err := l.UpdateUser(casdoor.UserUpsert{
			Name: req.Name, DisplayName: req.DisplayName, Avatar: req.Avatar, Email: req.Email,
			Phone: req.Phone, Properties: req.Properties,
		}); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "更新成功"})
	}
}

// DeleteUserHandler 删除用户
func DeleteUserHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := pathvar.Vars(r)["name"]
		if name == "" {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "缺少用户标识"})
			return
		}
		l := user.NewUserLogic(r.Context(), ctx.Casdoor(), ctx.Cache, ctx.Config.Casdoor.UserCacheTTL)
		if err := l.DeleteUser(name); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		// 级联清理本地关联（服务组授权、团队成员），避免悬挂授权
		cleanupUserAssociations(r, ctx, name)
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "已删除"})
	}
}

// ResetPasswordHandler 重置用户密码（留空使用系统默认密码）
func ResetPasswordHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := pathvar.Vars(r)["name"]
		if name == "" {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "缺少用户标识"})
			return
		}
		var req struct {
			Password string `json:"password,optional"`
		}
		_ = httpx.Parse(r, &req)
		password := req.Password
		if password == "" {
			password = ctx.Config.Casdoor.DefaultPassword
		}
		l := user.NewUserLogic(r.Context(), ctx.Casdoor(), ctx.Cache, ctx.Config.Casdoor.UserCacheTTL)
		if err := l.ResetPassword(name, password); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "密码已重置", "data": map[string]string{"password": password},
		})
	}
}

// cleanupUserAssociations 级联清理用户在本地的关联数据。
//
// 用户体系由 Casdoor 维护，但本地 service_group_users / team_members 以
// user_id（及 username）关联且无外键约束，删除用户后若不清理会残留悬挂授权；
// 同名用户重建后可能继承旧权限。
func cleanupUserAssociations(r *http.Request, ctx *svc.ServiceContext, name string) {
	if ctx.RawDB == nil {
		return
	}
	stmts := []string{
		`DELETE FROM service_group_users WHERE user_id = $1 OR username = $1`,
		`DELETE FROM team_members WHERE user_id = $1 OR username = $1`,
	}
	for _, q := range stmts {
		if _, err := ctx.RawDB.ExecContext(r.Context(), q, name); err != nil {
			logx.Errorf("清理用户关联失败（%s）: %v", name, err)
		}
	}
}

// SetUserAdminHandler 设置/取消管理员
func SetUserAdminHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		name := pathvar.Vars(r)["name"]
		if name == "" {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "缺少用户标识"})
			return
		}
		var req struct {
			IsAdmin bool `json:"isAdmin"`
		}
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 400, "message": "参数错误: " + err.Error()})
			return
		}
		l := user.NewUserLogic(r.Context(), ctx.Casdoor(), ctx.Cache, ctx.Config.Casdoor.UserCacheTTL)
		if err := l.SetUserAdmin(name, req.IsAdmin); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 500, "message": err.Error()})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{"code": 200, "message": "已更新"})
	}
}
