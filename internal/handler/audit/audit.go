// Package audit 提供审计日志的 HTTP 处理器
package audit

import (
	"encoding/csv"
	"net/http"
	"strconv"
	"time"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/logic/audit"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// ListAuditLogsHandler 查询审计日志列表
func ListAuditLogsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		username := r.URL.Query().Get("username")
		action := r.URL.Query().Get("action")
		resourceType := r.URL.Query().Get("resource_type")
		status := r.URL.Query().Get("status")

		var groupID *int64
		if gid := r.URL.Query().Get("group_id"); gid != "" {
			id, _ := strconv.ParseInt(gid, 10, 64)
			groupID = &id
		}

		var startTime, endTime *time.Time
		if st := r.URL.Query().Get("start_time"); st != "" {
			t, _ := time.Parse(time.RFC3339, st)
			startTime = &t
		}
		if et := r.URL.Query().Get("end_time"); et != "" {
			t, _ := time.Parse(time.RFC3339, et)
			endTime = &t
		}

		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}

		logic := audit.NewAuditLogic(r.Context(), ctx.DB)
		logs, total, err := logic.ListAuditLogs(
			userID, username, action, resourceType, groupID, status,
			startTime, endTime, page, pageSize)

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
				"list":      logs,
				"total":     total,
				"page":      page,
				"page_size": pageSize,
			},
		})
	}
}

// ExportAuditLogsHandler 导出审计日志（CSV 格式）
func ExportAuditLogsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		userID := r.URL.Query().Get("user_id")
		username := r.URL.Query().Get("username")
		action := r.URL.Query().Get("action")
		resourceType := r.URL.Query().Get("resource_type")
		status := r.URL.Query().Get("status")

		var groupID *int64
		if gid := r.URL.Query().Get("group_id"); gid != "" {
			id, _ := strconv.ParseInt(gid, 10, 64)
			groupID = &id
		}

		var startTime, endTime *time.Time
		if st := r.URL.Query().Get("start_time"); st != "" {
			t, _ := time.Parse(time.RFC3339, st)
			startTime = &t
		}
		if et := r.URL.Query().Get("end_time"); et != "" {
			t, _ := time.Parse(time.RFC3339, et)
			endTime = &t
		}

		logic := audit.NewAuditLogic(r.Context(), ctx.DB)
		logs, err := logic.ExportAuditLogs(
			userID, username, action, resourceType, groupID, status,
			startTime, endTime)

		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": err.Error(),
			})
			return
		}

		// 设置 CSV 响应头
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", "attachment; filename=audit_logs.csv")

		// 写入 UTF-8 BOM（让 Excel 正确识别编码）
		w.Write([]byte{0xEF, 0xBB, 0xBF})

		// 创建 CSV Writer
		writer := csv.NewWriter(w)
		defer writer.Flush()

		// 写入表头
		writer.Write([]string{
			"ID", "用户ID", "用户名", "操作", "资源类型", "资源ID",
			"资源名称", "服务组ID", "状态", "IP地址", "创建时间",
		})

		// 写入数据
		for _, log := range logs {
			groupIDStr := ""
			if log.GroupID != nil {
				groupIDStr = strconv.FormatInt(*log.GroupID, 10)
			}

			writer.Write([]string{
				strconv.FormatInt(log.ID, 10),
				log.UserID,
				log.Username,
				log.Action,
				log.ResourceType,
				log.ResourceID,
				log.ResourceName,
				groupIDStr,
				log.Status,
				log.IPAddress,
				log.CreatedAt.Format("2006-01-02 15:04:05"),
			})
		}
	}
}
