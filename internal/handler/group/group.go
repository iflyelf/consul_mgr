// Package group 提供服务组管理的 HTTP 处理器
package group

import (
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/logic/group"
	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// CreateGroupRequest 创建服务组请求
type CreateGroupRequest struct {
	Name          string `json:"name" validate:"required"`
	ConsulAddress string `json:"consul_address" validate:"required"`
	ConsulToken   string `json:"consul_token,omitempty"`
	Datacenter    string `json:"datacenter,default=dc1"`
	Description   string `json:"description"`
}

// UpdateGroupRequest 更新服务组请求
type UpdateGroupRequest struct {
	Name          string `json:"name,omitempty"`
	ConsulAddress string `json:"consul_address,omitempty"`
	ConsulToken   string `json:"consul_token,omitempty"`
	Datacenter    string `json:"datacenter,omitempty"`
	Description   string `json:"description,omitempty"`
}

// ListGroupsHandler 查询服务组列表
func ListGroupsHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		keyword := r.URL.Query().Get("keyword")
		page, _ := strconv.Atoi(r.URL.Query().Get("page"))
		pageSize, _ := strconv.Atoi(r.URL.Query().Get("page_size"))
		
		if page <= 0 {
			page = 1
		}
		if pageSize <= 0 {
			pageSize = 20
		}
		
		logic := group.NewGroupLogic(r.Context(), ctx.DB)
		groups, total, err := logic.ListGroups(keyword, page, pageSize)
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
				"list":  groups,
				"total": total,
				"page":  page,
				"page_size": pageSize,
			},
		})
	}
}

// CreateGroupHandler 创建服务组
func CreateGroupHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req CreateGroupRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "参数错误: " + err.Error(),
			})
			return
		}
		
		// 获取当前用户（用于审计）
		username, _ := middleware.GetUsernameFromContext(r.Context())
		
		logic := group.NewGroupLogic(r.Context(), ctx.DB)
		result, err := logic.CreateGroup(
			req.Name,
			req.ConsulAddress,
			req.ConsulToken,
			req.Datacenter,
			req.Description,
		)
		
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "创建失败: " + err.Error(),
			})
			return
		}
		
		// 记录操作日志
		_ = username
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "创建成功",
			"data":    result,
		})
	}
}

// UpdateGroupHandler 更新服务组
func UpdateGroupHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取 ID
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
		
		var req UpdateGroupRequest
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "参数错误: " + err.Error(),
			})
			return
		}
		
		logic := group.NewGroupLogic(r.Context(), ctx.DB)
		
		// 先获取原数据
		original, err := logic.GetGroup(id)
		if err != nil {
			httpx.WriteJson(w, http.StatusNotFound, map[string]interface{}{
				"code":    404,
				"message": err.Error(),
			})
			return
		}
		
		// 合并更新
		if req.Name == "" {
			req.Name = original.Name
		}
		if req.ConsulAddress == "" {
			req.ConsulAddress = original.ConsulAddress
		}
		if req.Datacenter == "" {
			req.Datacenter = original.Datacenter
		}
		if req.Description == "" {
			req.Description = original.Description
		}
		
		result, err := logic.UpdateGroup(
			id,
			req.Name,
			req.ConsulAddress,
			req.ConsulToken,
			req.Datacenter,
			req.Description,
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

// DeleteGroupHandler 删除服务组
func DeleteGroupHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取 ID
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
		
		logic := group.NewGroupLogic(r.Context(), ctx.DB)
		err = logic.DeleteGroup(id)
		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "删除失败: " + err.Error(),
			})
			return
		}
		
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "删除成功",
		})
	}
}

// GetGroupHandler 获取服务组详情
func GetGroupHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取 ID
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
		
		logic := group.NewGroupLogic(r.Context(), ctx.DB)
		result, stats, err := logic.GetGroupDetail(id)
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
			"data": map[string]interface{}{
				"group": result,
				"stats": stats,
			},
		})
	}
}
