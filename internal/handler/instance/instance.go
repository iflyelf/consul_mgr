// Package instance 提供实例管理的 HTTP 处理器
package instance

import (
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/logic/instance"
	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// RegisterInstanceRequest 注册实例请求
type RegisterInstanceRequest struct {
	InstanceID  string                 `json:"instance_id" validate:"required"`
	ServiceName string                 `json:"service_name" validate:"required"`
	GroupID     int64                  `json:"group_id" validate:"required"`
	Address     string                 `json:"address" validate:"required"`
	Port        int                    `json:"port" validate:"required"`
	Tags        []string               `json:"tags,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	HealthCheck map[string]interface{} `json:"health_check,omitempty"`
	Datacenter  string                 `json:"datacenter,default=dc1"`
	NodeName    string                 `json:"node_name,omitempty"`
}

// UpdateInstanceRequest 更新实例请求
type UpdateInstanceRequest struct {
	Address     string                 `json:"address,omitempty"`
	Port        int                    `json:"port,omitempty"`
	Tags        []string               `json:"tags,omitempty"`
	Meta        map[string]interface{} `json:"meta,omitempty"`
	HealthCheck map[string]interface{} `json:"health_check,omitempty"`
	Status      string                 `json:"status,omitempty"`
	Datacenter  string                 `json:"datacenter,omitempty"`
	NodeName    string                 `json:"node_name,omitempty"`
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	IDs []int64 `json:"ids" validate:"required"`
}

// ListInstancesHandler 查询实例列表
func ListInstancesHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID, _ := strconv.ParseInt(r.URL.Query().Get("group_id"), 10, 64)
		serviceName := r.URL.Query().Get("service_name")
		status := r.URL.Query().Get("status")
		datacenter := r.URL.Query().Get("datacenter")
		keyword := r.URL.Query().Get("keyword")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}
		
		logic := instance.NewInstanceLogic(r.Context(), ctx.DB)
		instances, total, err := logic.ListInstances(
			groupID, serviceName, status, datacenter, keyword, page, pageSize)
		
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
			"data": map[string]interface{}{
				"list":      instances,
				"total":     total,
				"page":      page,
				"page_size": pageSize,
			},
		})
	}
}

// RegisterInstanceHandler 注册实例
func RegisterInstanceHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterInstanceRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "参数错误: " + err.Error(),
			})
			return
		}
		
		// 获取当前用户
		username, _ := middleware.GetUsernameFromContext(r.Context())
		
		logic := instance.NewInstanceLogic(r.Context(), ctx.DB)
		result, err := logic.RegisterInstance(
			req.InstanceID,
			req.ServiceName,
			req.GroupID,
			req.Address,
			req.Port,
			req.Tags,
			req.Meta,
			req.HealthCheck,
			req.Datacenter,
			req.NodeName,
			username,
		)
		
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "注册失败: " + err.Error(),
			})
			return
		}
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "注册成功",
			"data":    result,
		})
	}
}

// UpdateInstanceHandler 更新实例
func UpdateInstanceHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get(":id")
		if idStr == "" {
			idStr = r.URL.Query().Get("id")
		}
		
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "无效的 ID",
			})
			return
		}
		
		var req UpdateInstanceRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "参数错误: " + err.Error(),
			})
			return
		}
		
		logic := instance.NewInstanceLogic(r.Context(), ctx.DB)
		
		// 获取原数据
		original, err := logic.GetInstance(id)
		if err != nil {
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": err.Error(),
			})
			return
		}
		
		// 合并更新
		if req.Address == "" {
			req.Address = original.Address
		}
		if req.Port == 0 {
			req.Port = original.Port
		}
		if req.Tags == nil {
			req.Tags = original.Tags
		}
		if req.Meta == nil {
			req.Meta = original.Meta
		}
		if req.Status == "" {
			req.Status = original.Status
		}
		if req.Datacenter == "" {
			req.Datacenter = original.Datacenter
		}
		if req.NodeName == "" {
			req.NodeName = original.NodeName
		}
		
		result, err := logic.UpdateInstance(
			id, req.Address, req.Port, req.Tags, req.Meta,
			req.HealthCheck, req.Status, req.Datacenter, req.NodeName,
		)
		
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "更新失败: " + err.Error(),
			})
			return
		}
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "更新成功",
			"data":    result,
		})
	}
}

// DeregisterInstanceHandler 注销实例
func DeregisterInstanceHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get(":id")
		if idStr == "" {
			idStr = r.URL.Query().Get("id")
		}
		
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "无效的 ID",
			})
			return
		}
		
		logic := instance.NewInstanceLogic(r.Context(), ctx.DB)
		err = logic.DeregisterInstance(id)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "注销失败: " + err.Error(),
			})
			return
		}
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "注销成功",
		})
	}
}

// GetInstanceHandler 获取实例详情
func GetInstanceHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get(":id")
		if idStr == "" {
			idStr = r.URL.Query().Get("id")
		}
		
		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "无效的 ID",
			})
			return
		}
		
		logic := instance.NewInstanceLogic(r.Context(), ctx.DB)
		result, err := logic.GetInstance(id)
		if err != nil {
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": err.Error(),
			})
			return
		}
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data":    result,
		})
	}
}

// BatchDeleteHandler 批量删除实例
func BatchDeleteHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req BatchDeleteRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "参数错误: " + err.Error(),
			})
			return
		}
		
		logic := instance.NewInstanceLogic(r.Context(), ctx.DB)
		deleted, err := logic.BatchDelete(req.IDs)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "批量删除失败: " + err.Error(),
			})
			return
		}
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "批量删除成功",
			"data": map[string]interface{}{
				"deleted": deleted,
			},
		})
	}
}

// GetDatacentersHandler 获取数据中心列表
func GetDatacentersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID, _ := strconv.ParseInt(r.URL.Query().Get("group_id"), 10, 64)
		
		logic := instance.NewInstanceLogic(r.Context(), ctx.DB)
		datacenters, err := logic.GetDatacenters(groupID)
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
			"data":    datacenters,
		})
	}
}

// GetServicesHandler 获取服务列表
func GetServicesHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID, _ := strconv.ParseInt(r.URL.Query().Get("group_id"), 10, 64)
		datacenter := r.URL.Query().Get("datacenter")
		
		logic := instance.NewInstanceLogic(r.Context(), ctx.DB)
		services, err := logic.GetServices(groupID, datacenter)
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
			"data":    services,
		})
	}
}
