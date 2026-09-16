package service

import (
	"net/http"
	"strconv"

	"github.com/iflyelf/consul_mgr/internal/handler/access"
	"github.com/iflyelf/consul_mgr/internal/logic/service"
	"github.com/iflyelf/consul_mgr/internal/pkg/response"
	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// ListServicesHandler 服务列表
func ListServicesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupIDStr := r.URL.Query().Get("group_id")
		keyword := r.URL.Query().Get("keyword")

		groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "group_id 参数无效")
			return
		}

		l := service.NewListServicesLogic(r.Context(), svcCtx)
		services, err := l.ListServices(groupID, keyword)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}

		// 按授权范围过滤（团队仅被授权部分 Service 时只返回这些服务）
		if set := access.AllowedSet(svcCtx, r, groupID); set != nil {
			filtered := make([]types.ConsulServiceInfo, 0, len(services))
			for _, sv := range services {
				if access.ContainsFold(set, sv.Service) {
					filtered = append(filtered, sv)
				}
			}
			services = filtered
		}

		response.Success(w, map[string]interface{}{
			"list":  services,
			"total": len(services),
		})
	}
}

// GetServiceDetailHandler 获取服务详情
func GetServiceDetailHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupIDStr := r.URL.Query().Get("group_id")
		serviceName := r.URL.Query().Get("service")

		groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "group_id 参数无效")
			return
		}

		if serviceName == "" {
			response.BadRequest(w, "service 参数不能为空")
			return
		}

		l := service.NewGetServiceDetailLogic(r.Context(), svcCtx)
		detail, err := l.GetServiceDetail(groupID, serviceName, r.URL.Query().Get("keyword"))
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}

		response.Success(w, detail)
	}
}

// DeleteServiceHandler 删除服务
func DeleteServiceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupIDStr := r.URL.Query().Get("group_id")
		serviceName := r.URL.Query().Get("service")

		groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "group_id 参数无效")
			return
		}

		if serviceName == "" {
			response.BadRequest(w, "service 参数不能为空")
			return
		}

		l := service.NewDeleteServiceLogic(r.Context(), svcCtx)
		err = l.DeleteService(groupID, serviceName)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}

		response.Success(w, map[string]string{
			"message": "删除成功",
		})
	}
}

// BatchDeleteServicesHandler 批量删除服务
func BatchDeleteServicesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			GroupID  int64    `json:"group_id"`
			Services []string `json:"services"`
		}

		if err := httpx.Parse(r, &req); err != nil {
			response.BadRequest(w, err.Error())
			return
		}

		// 服务级权限校验：全部服务都需有 delete 权限
		for _, name := range req.Services {
			if !access.CheckService(svcCtx, w, r, req.GroupID, name, "delete") {
				return
			}
		}

		l := service.NewBatchDeleteServicesLogic(r.Context(), svcCtx)
		err := l.BatchDeleteServices(req.GroupID, req.Services)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}

		response.Success(w, map[string]string{
			"message": "批量删除成功",
		})
	}
}
