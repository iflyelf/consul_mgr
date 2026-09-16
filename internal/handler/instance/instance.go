// Package instance 提供实例管理的 HTTP 处理器
//
// 说明：实例数据实时来自 Consul Catalog，不做本地持久化，
// 保证与 Consul 状态完全一致。
package instance

import (
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"

	"github.com/iflyelf/consul_mgr/internal/handler/access"
	"github.com/iflyelf/consul_mgr/internal/logic/instance"
	"github.com/iflyelf/consul_mgr/internal/middleware"
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

// checkInstanceAccess 解析实例所属服务并校验权限
func checkInstanceAccess(ctx *svc.ServiceContext, w http.ResponseWriter, r *http.Request, groupID int64, instanceID, action string) bool {
	if middleware.IsGlobalAdminFromContext(r.Context()) || middleware.IsAdminFromContext(r.Context()) {
		return true
	}
	l := instance.NewListInstancesLogic(r.Context(), ctx)
	detail, err := l.GetInstance(groupID, instanceID)
	if err != nil || detail == nil {
		// 无法解析时交由后续逻辑处理，避免掩盖真实错误
		return true
	}
	return access.CheckService(ctx, w, r, groupID, detail.Service, action)
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

		// 按授权范围过滤（团队仅被授权部分 Service 时，只返回这些服务）
		if set := access.AllowedSet(ctx, r, groupID); set != nil {
			filtered := make([]types.ConsulInstanceInfo, 0, len(list))
			for _, it := range list {
				if access.ContainsFold(set, it.Service) {
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

		// 服务级权限校验（写操作）
		if !access.CheckService(ctx, w, r, req.GroupID, req.Service, "write") {
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

		// 解析实例所属服务并做服务级权限校验
		if !checkInstanceAccess(ctx, w, r, groupID, id, "write") {
			return
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

		// 解析实例所属服务并做服务级权限校验
		if !checkInstanceAccess(ctx, w, r, groupID, id, "delete") {
			return
		}

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

		// 服务级权限校验：仅保留已授权的实例
		if set := access.AllowedSet(ctx, r, req.GroupID); set != nil {
			ll := instance.NewListInstancesLogic(r.Context(), ctx)
			all, _ := ll.ListInstances(req.GroupID, "", "")
			svcOf := make(map[string]string, len(all))
			for _, it := range all {
				svcOf[it.ID] = it.Service
			}
			kept := make([]string, 0, len(req.IDs))
			denied := false
			for _, id := range req.IDs {
				if svc, ok := svcOf[id]; ok && !access.ContainsFold(set, svc) {
					denied = true
					continue
				}
				kept = append(kept, id)
			}
			if denied && len(kept) == 0 {
				httpx.WriteJson(w, http.StatusForbidden, map[string]interface{}{
					"code":    403,
					"message": "没有权限删除所选实例（需管理员在「人员组织 → 团队管理」中授权）",
				})
				return
			}
			req.IDs = kept
		}

		l := instance.NewBatchDeleteInstancesLogic(r.Context(), ctx)
		success, failed, err := l.BatchDeleteInstances(req.GroupID, req.IDs)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": err.Error(),
				"data":   map[string]interface{}{"success": success, "failed": failed},
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code": 200, "message": fmt.Sprintf("成功删除 %d 个实例", success),
			"data": map[string]interface{}{"success": success, "failed": failed},
		})
	}
}

// BatchRegisterHandler 批量注册实例
//
// 请求方式：POST
// 路径：/api/instances/batch-register
// 说明：支持 "10.1.255.24-26:80,10.1.255.38:443,10.1.255.0/24:8080" 形式
func BatchRegisterHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.BatchRegisterRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 400, "message": "参数错误: " + err.Error(),
			})
			return
		}

		// 服务级权限校验（写操作）
		if !access.CheckService(ctx, w, r, req.GroupID, req.Service, "write") {
			return
		}

		l := instance.NewBatchRegisterInstancesLogic(r.Context(), ctx)
		success, skipped, failed, ids, err := l.BatchRegisterInstances(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": err.Error(),
				"data": map[string]interface{}{
					"success": success, "skipped": skipped, "failed": failed, "ids": ids,
				},
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": fmt.Sprintf("成功注册 %d 个实例（跳过 %d）", success, skipped),
			"data": map[string]interface{}{
				"success": success, "skipped": skipped, "failed": failed, "ids": ids,
			},
		})
	}
}

// PreviewBatchRegisterHandler 预览批量注册结果
//
// 请求方式：POST
// 路径：/api/instances/batch-register/preview
func PreviewBatchRegisterHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.BatchRegisterRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 400, "message": "参数错误: " + err.Error(),
			})
			return
		}

		if !access.CheckService(ctx, w, r, req.GroupID, req.Service, "write") {
			return
		}

		l := instance.NewBatchRegisterInstancesLogic(r.Context(), ctx)
		ids, err := l.PreviewBatchRegister(&req)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": fmt.Sprintf("将注册 %d 个实例", len(ids)),
			"data": map[string]interface{}{
				"total": len(ids),
				"ids":   ids,
			},
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
		// 是否强制覆盖
		overwrite := r.FormValue("overwrite") == "true" || r.FormValue("overwrite") == "1"

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
		success, skipped, failed, err := l.ImportInstances(groupID, format, data, overwrite)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code": 500, "message": err.Error(),
				"data":   map[string]interface{}{"success": success, "skipped": skipped, "failed": failed},
			})
			return
		}

		msg := fmt.Sprintf("成功导入 %d 个实例", success)
		if skipped > 0 {
			msg += fmt.Sprintf("，跳过 %d 个已存在", skipped)
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": msg,
			"data":    map[string]interface{}{"success": success, "skipped": skipped, "failed": failed},
		})
	}
}
