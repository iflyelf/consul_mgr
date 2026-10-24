package instance

import (
	"net/http"
	"strconv"

	"github.com/iflyelf/consul_mgr/internal/logic/instance"
	"github.com/iflyelf/consul_mgr/internal/pkg/response"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// ExportHostsHandler 导出按 IP 聚合的主机清单（服务间只读接口）。
//
// 路径：GET /api/service/hosts?group_id=
// 鉴权：X-Service-Token（见 middleware.ServiceTokenMiddleware），不走用户会话。
//
// 用途：upgrade_mgr 的「从 consul_mgr 同步主机」。相比直连单个 Consul，
// 这里能一次拿到所有服务组（各自独立的 Consul 地址/令牌）的主机，
// 并带上服务组归属与完整 ServiceMeta，由调用方按自己的映射规则落库。
func ExportHostsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var groupID int64
		if v := r.URL.Query().Get("group_id"); v != "" {
			id, err := strconv.ParseInt(v, 10, 64)
			if err != nil || id <= 0 {
				response.BadRequest(w, "group_id 无效")
				return
			}
			groupID = id
		}

		res, err := instance.ExportHosts(r.Context(), ctx, groupID)
		if err != nil {
			response.InternalError(w, err.Error())
			return
		}
		response.Success(w, res)
	}
}
