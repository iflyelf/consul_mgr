// Package userfield 用户字段定义的 HTTP 处理器（与 FlyIAM 对齐）
package userfield

import (
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"

	"github.com/iflyelf/consul_mgr/internal/logic/userfield"
	"github.com/iflyelf/consul_mgr/internal/model"
	"github.com/iflyelf/consul_mgr/internal/pkg/response"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

func fieldID(r *http.Request) (int64, error) {
	return strconv.ParseInt(pathvar.Vars(r)["id"], 10, 64)
}

// ListUserFieldsHandler 字段定义列表
func ListUserFieldsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		l := userfield.NewLogic(ctx.DB)
		list, err := l.List(r.Context())
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.Success(w, list)
	}
}

// CreateUserFieldHandler 新增字段定义
func CreateUserFieldHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var d model.UserFieldDef
		if err := httpx.Parse(r, &d); err != nil {
			response.BadRequest(w, "参数错误: "+err.Error())
			return
		}
		l := userfield.NewLogic(ctx.DB)
		id, err := l.Create(r.Context(), &d)
		if err != nil {
			response.BadRequest(w, err.Error())
			return
		}
		d.ID = id
		response.Success(w, d)
	}
}

// UpdateUserFieldHandler 更新字段定义
func UpdateUserFieldHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := fieldID(r)
		if err != nil {
			response.BadRequest(w, "无效的字段 ID")
			return
		}
		var d model.UserFieldDef
		if err := httpx.Parse(r, &d); err != nil {
			response.BadRequest(w, "参数错误: "+err.Error())
			return
		}
		d.ID = id
		l := userfield.NewLogic(ctx.DB)
		if err := l.Update(r.Context(), &d); err != nil {
			response.BadRequest(w, err.Error())
			return
		}
		response.Success(w, d)
	}
}

// DeleteUserFieldHandler 删除字段定义
func DeleteUserFieldHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := fieldID(r)
		if err != nil {
			response.BadRequest(w, "无效的字段 ID")
			return
		}
		l := userfield.NewLogic(ctx.DB)
		if err := l.Delete(r.Context(), id); err != nil {
			response.BadRequest(w, err.Error())
			return
		}
		response.Success(w, nil)
	}
}

// SyncUserFieldsHandler 从 FlyIAM 同步字段定义（数据源字段变化时无需改代码）
func SyncUserFieldsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg := ctx.Config.FlyIAM
		if cfg.Endpoint == "" || cfg.ServiceToken == "" {
			response.BadRequest(w, "未配置 FlyIAM 地址或服务令牌（FLYIAM_API_ENDPOINT / FLYIAM_SERVICE_TOKEN）")
			return
		}
		l := userfield.NewLogic(ctx.DB)
		added, updated, total, err := l.SyncFromFlyIAM(r.Context(), cfg.Endpoint, cfg.ServiceToken)
		if err != nil {
			response.Error(w, 502, "同步失败: "+err.Error())
			return
		}
		response.Success(w, map[string]interface{}{
			"added":   added,
			"updated": updated,
			"total":   total,
		})
	}
}
