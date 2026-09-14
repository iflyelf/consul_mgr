package types

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" validate:"required"`
	Password string `json:"password" validate:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int64  `json:"expires_in"`
	UserInfo     UserInfo `json:"user_info"`
}

// RefreshTokenRequest 刷新令牌请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token" validate:"required"`
}

// RefreshTokenResponse 刷新令牌响应
type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresIn   int64  `json:"expires_in"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID        int64    `json:"id"`
	Username  string   `json:"username"`
	Email     string   `json:"email,omitempty"`
	RealName  string   `json:"real_name,omitempty"`
	Status    int      `json:"status"`
	Roles     []string `json:"roles"`
	CreatedAt string   `json:"created_at,omitempty"`
	UpdatedAt string   `json:"updated_at,omitempty"`
}

// CreateUserRequest 创建用户请求
type CreateUserRequest struct {
	Username string   `json:"username" validate:"required"`
	Password string   `json:"password" validate:"required,min=6"`
	Email    string   `json:"email,omitempty" validate:"omitempty,email"`
	RealName string   `json:"real_name,omitempty"`
	RoleIDs  []int64  `json:"role_ids,omitempty"`
}

// UpdateUserRequest 更新用户请求
type UpdateUserRequest struct {
	Email    string  `json:"email,omitempty" validate:"omitempty,email"`
	RealName string  `json:"real_name,omitempty"`
	Status   *int    `json:"status,omitempty"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password,omitempty"`
	NewPassword string `json:"new_password" validate:"required,min=6"`
}

// AssignRolesRequest 分配角色请求
type AssignRolesRequest struct {
	RoleIDs []int64 `json:"role_ids" validate:"required"`
}

// CreateRoleRequest 创建角色请求
type CreateRoleRequest struct {
	Name        string  `json:"name" validate:"required"`
	Code        string  `json:"code" validate:"required"`
	Description string  `json:"description,omitempty"`
	PermissionIDs []int64 `json:"permission_ids,omitempty"`
}

// UpdateRoleRequest 更新角色请求
type UpdateRoleRequest struct {
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
	Status      *int   `json:"status,omitempty"`
}

// AssignPermissionsRequest 分配权限请求
type AssignPermissionsRequest struct {
	PermissionIDs []int64 `json:"permission_ids" validate:"required"`
}

// AssignGroupsRequest 授权服务组请求
type AssignGroupsRequest struct {
	GroupID     int64    `json:"group_id" validate:"required"`
	Permissions []string `json:"permissions" validate:"required"` // ["read", "write", "delete"]
}

// CreateGroupRequest 创建服务组请求
type CreateGroupRequest struct {
	Name             string `json:"name" validate:"required"`
	Code             string `json:"code" validate:"required"`
	Description      string `json:"description,omitempty"`
	ConsulAddress    string `json:"consul_address" validate:"required,url"`
	ConsulToken      string `json:"consul_token,omitempty"`
	ConsulDatacenter string `json:"consul_datacenter,omitempty"`
}

// UpdateGroupRequest 更新服务组请求
type UpdateGroupRequest struct {
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	ConsulAddress    string `json:"consul_address,omitempty" validate:"omitempty,url"`
	ConsulToken      string `json:"consul_token,omitempty"`
	ConsulDatacenter string `json:"consul_datacenter,omitempty"`
	Status           *int   `json:"status,omitempty"`
}

// QueryRequest 通用查询请求
type QueryRequest struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Keyword  string `form:"keyword,optional"`
	Status   *int   `form:"status,optional"`
}

// PageResponse 分页响应
type PageResponse struct {
	Total    int64       `json:"total"`
	Page     int         `json:"page"`
	PageSize int         `json:"page_size"`
	List     interface{} `json:"list"`
}
