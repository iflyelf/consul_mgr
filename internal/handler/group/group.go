// Package group 提供服务组管理的 HTTP 处理器
package group

import (
	"net/http"
	"strconv"

	"github.com/zeromicro/go-zero/rest/httpx"
	"github.com/zeromicro/go-zero/rest/pathvar"

	"github.com/iflyelf/consul_mgr/internal/logic/group"
	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/svc"
)

// maskedSecret 敏感字段回显占位符（提交该值表示「保持原值不变」）
const maskedSecret = "******"

// maskGroup 脱敏 Consul Token 后再返回（令牌不应经接口回显）
func maskGroup(g *group.Group) *group.Group {
	if g == nil {
		return nil
	}
	m := *g
	if m.ConsulToken != "" {
		m.ConsulToken = maskedSecret
	}
	return &m
}

// CreateGroupRequest 创建服务组请求
type CreateGroupRequest struct {
	Name             string `json:"name" validate:"required"`
	Code             string `json:"code,optional"`
	ConsulAddress    string `json:"consul_address" validate:"required"`
	ConsulToken      string `json:"consul_token,optional"`
	Datacenter       string `json:"datacenter,optional"`
	ConsulDatacenter string `json:"consul_datacenter,optional"`
	Description      string `json:"description,optional"`
}

// UpdateGroupRequest 更新服务组请求
type UpdateGroupRequest struct {
	Name             string `json:"name,optional"`
	Code             string `json:"code,optional"`
	ConsulAddress    string `json:"consul_address,optional"`
	ConsulToken      string `json:"consul_token,optional"`
	Datacenter       string `json:"datacenter,optional"`
	ConsulDatacenter string `json:"consul_datacenter,optional"`
	Description      string `json:"description,optional"`
	Status           *int   `json:"status,optional"`
}

// DetectDatacenterHandler 探测 Consul 数据中心
//
// 请求方式：POST
// 路径：/api/groups/detect-datacenter
// 说明：根据 Consul 地址与 Token 自动探测数据中心，供前端预览
func DetectDatacenterHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ConsulAddress string `json:"consul_address"`
			ConsulToken   string `json:"consul_token,optional"`
		}
		if err := httpx.Parse(r, &req); err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "参数错误: " + err.Error(),
			})
			return
		}

		if req.ConsulAddress == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "Consul 地址不能为空",
			})
			return
		}

		logic := group.NewGroupLogic(r.Context(), ctx.DB)
		dc, nodeName, err := logic.DetectDatacenter(req.ConsulAddress, req.ConsulToken)
		if err != nil {
			httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
				"code":    500,
				"message": err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data": map[string]interface{}{
				"datacenter": dc,
				"node_name":  nodeName,
			},
		})
	}
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

		masked := make([]*group.Group, 0, len(groups))
		for _, g := range groups {
			masked = append(masked, maskGroup(g))
		}
		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "success",
			"data": map[string]interface{}{
				"list":      masked,
				"total":     total,
				"page":      page,
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

		// 数据中心必须自动探测，不接受前端手填
		dc := ""
		if detected, _, derr := logic.DetectDatacenter(req.ConsulAddress, req.ConsulToken); derr == nil {
			dc = detected
		}
		if dc == "" {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "无法自动获取数据中心，请检查 Consul 地址和 Token 是否正确",
			})
			return
		}

		result, err := logic.CreateGroup(
			req.Name,
			req.Code,
			req.ConsulAddress,
			req.ConsulToken,
			dc,
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
			"data":    maskGroup(result),
		})
	}
}

// UpdateGroupHandler 更新服务组
func UpdateGroupHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取 ID
		idStr := pathvar.Vars(r)["id"]

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
		if req.Code == "" {
			req.Code = original.Code
		}
		if req.ConsulAddress == "" {
			req.ConsulAddress = original.ConsulAddress
		}
		if req.ConsulToken == "" || req.ConsulToken == maskedSecret {
			req.ConsulToken = original.ConsulToken
		}
		if req.Description == "" {
			req.Description = original.Description
		}

		// 数据中心必须自动探测，地址或 Token 变化时重新探测
		reqDatacenter := original.ConsulDatacenter
		if req.ConsulAddress != original.ConsulAddress || req.ConsulToken != original.ConsulToken {
			if detected, _, derr := logic.DetectDatacenter(req.ConsulAddress, req.ConsulToken); derr == nil {
				reqDatacenter = detected
			}
		}

		result, err := logic.UpdateGroup(
			id,
			req.Name,
			req.Code,
			req.ConsulAddress,
			req.ConsulToken,
			reqDatacenter,
			req.Description,
		)

		if err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": "更新失败: " + err.Error(),
			})
			return
		}

		// 地址/Token 可能变化：清理缓存与 Consul 客户端
		ctx.InvalidateGroupCaches(r.Context(), id)
		ctx.ConsulManager.RemoveClient(id)

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "更新成功",
			"data":    maskGroup(result),
		})
	}
}

// DeleteGroupHandler 删除服务组
func DeleteGroupHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// 获取 ID
		idStr := pathvar.Vars(r)["id"]

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

		// 清理该服务组的缓存
		ctx.InvalidateGroupCaches(r.Context(), id)
		ctx.ConsulManager.RemoveClient(id)

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
		idStr := pathvar.Vars(r)["id"]

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
				"group": maskGroup(result),
				"stats": stats,
			},
		})
	}
}

// TestConnectionHandler 测试 Consul 连接
func TestConnectionHandler(ctx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		idStr := pathvar.Vars(r)["id"]

		id, err := strconv.ParseInt(idStr, 10, 64)
		if err != nil {
			httpx.WriteJson(w, http.StatusBadRequest, map[string]interface{}{
				"code":    400,
				"message": "无效的 ID",
			})
			return
		}

		logic := group.NewGroupLogic(r.Context(), ctx.DB)
		if err := logic.TestConnection(id, ctx.ConsulManager); err != nil {
			httpx.WriteJson(w, http.StatusInternalServerError, map[string]interface{}{
				"code":    500,
				"message": err.Error(),
			})
			return
		}

		httpx.WriteJson(w, http.StatusOK, map[string]interface{}{
			"code":    200,
			"message": "连接测试成功",
		})
	}
}
