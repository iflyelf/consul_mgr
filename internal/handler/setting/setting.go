// Package setting 应用设置的 HTTP 处理器（页面可配置，DB 优先 / env 兜底）
package setting

import (
	"encoding/json"
	"net/http"

	"github.com/iflyelf/consul_mgr/internal/pkg/response"
	"github.com/iflyelf/consul_mgr/internal/setting"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// ListSettingsHandler 返回全部可配置项
func ListSettingsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		response.Success(w, ctx.Settings.View())
	}
}

// UpdateSettingsHandler 更新设置（写 DB 并即时生效）
func UpdateSettingsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Settings map[string]string `json:"settings"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.BadRequest(w, "参数解析失败: "+err.Error())
			return
		}
		if len(req.Settings) == 0 {
			response.BadRequest(w, "没有需要更新的配置")
			return
		}
		toApply := make(map[string]string, len(req.Settings))
		for k, v := range req.Settings {
			if setting.IsMasked(v) {
				continue
			}
			toApply[k] = v
		}
		if err := ctx.Settings.Apply(r.Context(), toApply); err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		response.Success(w, nil)
	}
}
