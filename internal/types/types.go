package types

// ============ 用户管理（扩展）============

type ListUsersRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Keyword  string `form:"keyword,optional"`
	Status   *int   `form:"status,optional"`
}

// ============ 角色管理（扩展）============

type ListRolesRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Keyword  string `form:"keyword,optional"`
}

type RoleInfo struct {
	ID          int64            `json:"id"`
	Name        string           `json:"name"`
	Code        string           `json:"code"`
	Description string           `json:"description"`
	Status      int              `json:"status"`
	Permissions []PermissionInfo `json:"permissions,omitempty"`
	CreatedAt   string           `json:"created_at"`
	UpdatedAt   string           `json:"updated_at"`
}

// ============ 权限管理 ============

type ListPermissionsRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Keyword  string `form:"keyword,optional"`
}

type PermissionInfo struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Code        string `json:"code"`
	Resource    string `json:"resource"`
	Action      string `json:"action"`
	Description string `json:"description"`
	Status      int    `json:"status"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type CreatePermissionRequest struct {
	Name        string `json:"name" validate:"required"`
	Code        string `json:"code" validate:"required"`
	Resource    string `json:"resource" validate:"required"`
	Action      string `json:"action" validate:"required"`
	Description string `json:"description,optional"`
	Status      int    `json:"status,default=1"`
}

type UpdatePermissionRequest struct {
	Name        string `json:"name,optional"`
	Description string `json:"description,optional"`
	Status      *int   `json:"status,optional"`
}

// ============ 服务组授权 ============

type AssignGroupUsersRequest struct {
	UserIds []int64 `json:"user_ids" validate:"required"`
}

type GroupUserInfo struct {
	ID        int64    `json:"id"`
	GroupID   int64    `json:"group_id"`
	UserID    int64    `json:"user_id"`
	User      UserInfo `json:"user"`
	CreatedAt string   `json:"created_at"`
}

// ============ 通用响应 ============

type PageInfo struct {
	Page     int   `json:"page"`
	PageSize int   `json:"page_size"`
	Total    int64 `json:"total"`
}

type ListResponse struct {
	List     interface{} `json:"list"`
	PageInfo PageInfo    `json:"page_info"`
}

