package group

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/zeromicro/go-zero/rest/httpx"

	"github.com/iflyelf/consul_mgr/internal/logic/group"
	"github.com/iflyelf/consul_mgr/internal/pkg/response"
	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// getIDFromPath 从路径中提取ID
func getIDFromPath(r *http.Request) (int64, error) {
	// 从 URL 路径中提取 ID
	path := r.URL.Path
	parts := strings.Split(strings.Trim(path, "/"), "/")
	
	// 查找 groups 后面的数字
	for i, part := range parts {
		if part == "groups" && i+1 < len(parts) {
			idStr := parts[i+1]
			// 去除可能的 /test 后缀
			idStr = strings.Split(idStr, "/")[0]
			return strconv.ParseInt(idStr, 10, 64)
		}
	}
	
	return 0, strconv.ErrSyntax
}

// ListGroupsHandler 服务组列表
func ListGroupsHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.QueryRequest
		if err := httpx.Parse(r, &req); err != nil {
			response.BadRequest(w, err.Error())
			return
		}
		
		l := group.NewListGroupsLogic(r.Context(), svcCtx)
		resp, err := l.ListGroups(&req)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		
		response.Success(w, resp)
	}
}

// CreateGroupHandler 创建服务组
func CreateGroupHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req types.CreateGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.BadRequest(w, "请求参数错误: "+err.Error())
			return
		}
		
		// 手动验证必填字段
		if req.Name == "" {
			response.BadRequest(w, "名称不能为空")
			return
		}
		if req.Code == "" {
			response.BadRequest(w, "代码不能为空")
			return
		}
		if req.ConsulAddress == "" {
			response.BadRequest(w, "Consul 地址不能为空")
			return
		}
		
		l := group.NewCreateGroupLogic(r.Context(), svcCtx)
		resp, err := l.CreateGroup(&req)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		
		response.Success(w, resp)
	}
}

// GetGroupHandler 获取服务组详情
func GetGroupHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := getIDFromPath(r)
		if err != nil {
			response.BadRequest(w, "服务组ID格式错误")
			return
		}
		
		l := group.NewGetGroupLogic(r.Context(), svcCtx)
		resp, err := l.GetGroup(id)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		
		response.Success(w, resp)
	}
}

// UpdateGroupHandler 更新服务组
func UpdateGroupHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := getIDFromPath(r)
		if err != nil {
			response.BadRequest(w, "服务组ID格式错误")
			return
		}
		
		// 解析请求体
		var req types.UpdateGroupRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			response.BadRequest(w, "请求参数错误: "+err.Error())
			return
		}
		
		l := group.NewUpdateGroupLogic(r.Context(), svcCtx)
		err = l.UpdateGroup(id, &req)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		
		response.Success(w, map[string]string{"message": "更新成功"})
	}
}

// DeleteGroupHandler 删除服务组
func DeleteGroupHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := getIDFromPath(r)
		if err != nil {
			response.BadRequest(w, "服务组ID格式错误")
			return
		}
		
		l := group.NewDeleteGroupLogic(r.Context(), svcCtx)
		err = l.DeleteGroup(id)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		
		response.Success(w, map[string]string{"message": "删除成功"})
	}
}

// TestConnectionHandler 测试 Consul 连接
func TestConnectionHandler(svcCtx *svc.ServiceContext) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id, err := getIDFromPath(r)
		if err != nil {
			response.BadRequest(w, "服务组ID格式错误")
			return
		}
		
		l := group.NewTestConnectionLogic(r.Context(), svcCtx)
		err = l.TestConnection(id)
		if err != nil {
			response.Error(w, 500, err.Error())
			return
		}
		
		response.Success(w, map[string]string{"message": "连接测试成功"})
	}
}
