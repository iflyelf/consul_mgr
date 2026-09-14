package instance

import (
	"io"
	"net/http"
	"strconv"

	"github.com/iflyelf/consul_mgr/internal/logic/instance"
	"github.com/iflyelf/consul_mgr/internal/pkg/response"
	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
	"github.com/zeromicro/go-zero/rest/httpx"
)

// ListInstancesHandler 实例列表
func ListInstancesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupIDStr := r.URL.Query().Get("group_id")
		serviceName := r.URL.Query().Get("service")
		status := r.URL.Query().Get("status")

		groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "group_id 参数无效")
			return
		}

		if serviceName == "" {
			response.BadRequest(w, "service 参数不能为空")
			return
		}

		l := instance.NewListInstancesLogic(r.Context(), svcCtx)
		instances, err := l.ListInstances(groupID, serviceName, status)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}

		response.Success(w, map[string]interface{}{
			"instances": instances,
			"total":     len(instances),
		})
	}
}

// RegisterInstanceHandler 注册实例
func RegisterInstanceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.RegisterInstanceRequest
		if err := httpx.Parse(r, &req); err != nil {
			response.BadRequest(w, err.Error())
			return
		}

		l := instance.NewRegisterInstanceLogic(r.Context(), svcCtx)
		err := l.RegisterInstance(req.GroupID, &req)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}

		response.Success(w, map[string]string{
			"message": "注册成功",
		})
	}
}

// UpdateInstanceHandler 更新实例
func UpdateInstanceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupIDStr := r.URL.Query().Get("group_id")
		instanceID := r.URL.Query().Get("instance_id")

		groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "group_id 参数无效")
			return
		}

		if instanceID == "" {
			response.BadRequest(w, "instance_id 参数不能为空")
			return
		}

		var req types.UpdateInstanceRequest
		if err := httpx.Parse(r, &req); err != nil {
			response.BadRequest(w, err.Error())
			return
		}

		l := instance.NewUpdateInstanceLogic(r.Context(), svcCtx)
		err = l.UpdateInstance(groupID, instanceID, &req)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}

		response.Success(w, map[string]string{
			"message": "更新成功",
		})
	}
}

// DeleteInstanceHandler 删除实例
func DeleteInstanceHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupIDStr := r.URL.Query().Get("group_id")
		instanceID := r.URL.Query().Get("instance_id")

		groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "group_id 参数无效")
			return
		}

		if instanceID == "" {
			response.BadRequest(w, "instance_id 参数不能为空")
			return
		}

		l := instance.NewDeleteInstanceLogic(r.Context(), svcCtx)
		err = l.DeleteInstance(groupID, instanceID)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}

		response.Success(w, map[string]string{
			"message": "删除成功",
		})
	}
}

// BatchDeleteInstancesHandler 批量删除实例
func BatchDeleteInstancesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			GroupID     int64    `json:"group_id"`
			InstanceIDs []string `json:"instance_ids"`
		}

		if err := httpx.Parse(r, &req); err != nil {
			response.BadRequest(w, err.Error())
			return
		}

		l := instance.NewBatchDeleteInstancesLogic(r.Context(), svcCtx)
		err := l.BatchDeleteInstances(req.GroupID, req.InstanceIDs)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}

		response.Success(w, map[string]string{
			"message": "批量删除成功",
		})
	}
}

// ExportInstancesHandler 导出实例
func ExportInstancesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupIDStr := r.URL.Query().Get("group_id")
		serviceName := r.URL.Query().Get("service")
		format := r.URL.Query().Get("format")
		if format == "" {
			format = "json"
		}

		groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "group_id 参数无效")
			return
		}

		if serviceName == "" {
			response.BadRequest(w, "service 参数不能为空")
			return
		}

		l := instance.NewExportInstancesLogic(r.Context(), svcCtx)
		data, err := l.ExportInstances(groupID, serviceName, format)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}

		// 设置响应头
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", "attachment; filename=instances."+format)
		w.Write(data)
	}
}

// ImportInstancesHandler 导入实例
func ImportInstancesHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		groupIDStr := r.FormValue("group_id")
		format := r.FormValue("format")
		if format == "" {
			format = "json"
		}

		groupID, err := strconv.ParseInt(groupIDStr, 10, 64)
		if err != nil {
			response.BadRequest(w, "group_id 参数无效")
			return
		}

		// 读取上传的文件
		file, _, err := r.FormFile("file")
		if err != nil {
			response.BadRequest(w, "文件上传失败: "+err.Error())
			return
		}
		defer file.Close()

		data, err := io.ReadAll(file)
		if err != nil {
			response.BadRequest(w, "读取文件失败: "+err.Error())
			return
		}

		l := instance.NewImportInstancesLogic(r.Context(), svcCtx)
		err = l.ImportInstances(groupID, format, data)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}

		response.Success(w, map[string]string{
			"message": "导入成功",
		})
	}
}
