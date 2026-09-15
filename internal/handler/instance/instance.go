// Package instance 提供实例管理的 HTTP 处理器
//
// 说明：实例数据实时来自 Consul Catalog，不做本地持久化，
// 保证与 Consul 状态完全一致。
package instance

import (
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"

	"github.com/iflyelf/consul_mgr/internal/logic/instance"
	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// instanceID 从路径或查询参数中提取实例 ID
func instanceID(r *http.Request) string {
	if v := pathvar.Vars(r)["id"]; v != "" {
		return v
	}
	if v := r.URL.Query().Get("instance_id"); v != "" {
		return v
	}
	return r.URL.Query().Get("id")
}

// groupIDFrom 解析 group_id
func groupIDFrom(r *http.Request) int64 {
	if v := r.URL.Query().Get("group_id"); v != "" {
		id, _ := strconv.ParseInt(v, 10, 64)
		return id
	}
	return 0
}

// ListInstancesHandler 查询实例列表
//
// 支持条件：group_id、service_name、status、keyword、分页
func ListInstancesHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID := groupIDFrom(r)
		if groupID == 0 {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 400, "message": "group_id 不能为空",
			})
			return
		}
		serviceName := r.URL.Query().Get("service_name")
		if serviceName == "" {
			serviceName = r.URL.Query().Get("service")
		}
		status := r.URL.Query().Get("status")
		keyword := r.URL.Query().Get("keyword")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		l := instance.NewListInstancesLogic(r.Context(), ctx)
		list, err := l.ListInstances(groupID, serviceName, status)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": err.Error(),
			})
			return
		}

		// 关键字过滤（实例ID/服务名/地址/节点）
		if keyword != "" {
			kw := strings.ToLower(keyword)
			filtered := make([]types.ConsulInstanceInfo, 0, len(list))
			for _, it := range list {
				if strings.Contains(strings.ToLower(it.ID), kw) ||
					strings.Contains(strings.ToLower(it.Service), kw) ||
					strings.Contains(strings.ToLower(it.Address), kw) ||
					strings.Contains(strings.ToLower(it.Node), kw) {
					filtered = append(filtered, it)
				}
			}
			list = filtered
		}

		total := len(list)
		start := (page - 1) * pageSize
		if start > total {
			start = total
		}
		end := start + pageSize
		if end > total {
			end = total
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data": map[string]interface{}{
				"list":      list[start:end],
				"total":     total,
				"page":      page,
				"page_size": pageSize,
			},
		})
	}
}

// GetInstanceHandler 获取实例详情
func GetInstanceHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := instanceID(r)
		if id == "" {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 400, "message": "实例ID不能为空",
			})
			return
		}

		groupID := groupIDFrom(r)
		l := instance.NewListInstancesLogic(r.Context(), ctx)
		detail, err := l.GetInstance(groupID, id)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "success", "data": detail,
		})
	}
}

// RegisterInstanceHandler 注册实例
func RegisterInstanceHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RegisterInstanceRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 400, "message": "参数错误: " + err.Error(),
			})
			return
		}

		l := instance.NewRegisterInstanceLogic(r.Context(), ctx)
		if err := l.RegisterInstance(req.GroupID, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": "注册失败: " + err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "注册成功", "data": req,
		})
	}
}

// UpdateInstanceHandler 更新实例
func UpdateInstanceHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := instanceID(r)
		if id == "" {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 400, "message": "实例ID不能为空",
			})
			return
		}

		var req types.UpdateInstanceRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 400, "message": "参数错误: " + err.Error(),
			})
			return
		}

		groupID := groupIDFrom(r)
		if groupID == 0 {
			groupID, _ = strconv.ParseInt(r.Header.Get("X-Group-Id"), 10, 64)
		}

		l := instance.NewUpdateInstanceLogic(r.Context(), ctx)
		if err := l.UpdateInstance(groupID, id, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": "更新失败: " + err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "更新成功",
		})
	}
}

// DeregisterInstanceHandler 注销实例
func DeregisterInstanceHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := instanceID(r)
		if id == "" {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 400, "message": "实例ID不能为空",
			})
			return
		}

		groupID := groupIDFrom(r)
		l := instance.NewDeleteInstanceLogic(r.Context(), ctx)
		if err := l.DeleteInstance(groupID, id); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": "删除失败: " + err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "删除成功",
		})
	}
}

// BatchDeleteHandler 批量删除实例
func BatchDeleteHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.BatchDeleteRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 400, "message": "参数错误: " + err.Error(),
			})
			return
		}

		l := instance.NewBatchDeleteInstancesLogic(r.Context(), ctx)
		if err := l.BatchDeleteInstances(req.GroupID, req.IDs); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "批量删除成功",
		})
	}
}

// GetDatacentersHandler 获取数据中心列表
func GetDatacentersHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID := groupIDFrom(r)
		l := instance.NewListInstancesLogic(r.Context(), ctx)
		dcs, err := l.GetDatacenters(groupID)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": err.Error(),
			})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "success", "data": dcs,
		})
	}
}

// GetServicesHandler 获取服务名列表
func GetServicesHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID := groupIDFrom(r)
		l := instance.NewListInstancesLogic(r.Context(), ctx)
		svcs, err := l.GetServiceNames(groupID, r.URL.Query().Get("datacenter"))
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": err.Error(),
			})
			return
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "success", "data": svcs,
		})
	}
}

// ExportInstancesHandler 导出实例
func ExportInstancesHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID := groupIDFrom(r)
		serviceName := r.URL.Query().Get("service")
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "json"
		}

		l := instance.NewExportInstancesLogic(r.Context(), ctx)
		data, err := l.ExportInstances(groupID, serviceName, format)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": err.Error(),
			})
			return
		}

		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=instances."+format)
		w.Write(data)
	}
}

// ImportInstancesHandler 导入实例
func ImportInstancesHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupID, _ := strconv.ParseInt(r.FormValue("group_id"), 10, 64)
		format := r.FormValue("format")
		if format == "" {
			format = "json"
		}

		data := []byte(r.FormValue("data"))
		if len(data) == 0 {
			if file, _, err := r.FormFile("file"); err == nil {
				defer file.Close()
				data, _ = io.ReadAll(file)
			}
		}

		if len(data) == 0 {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 400, "message": "导入数据不能为空",
			})
			return
		}

		l := instance.NewImportInstancesLogic(r.Context(), ctx)
		if err := l.ImportInstances(groupID, format, data); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": "导入成功",
		})
	}
}
