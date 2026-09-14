// Package auth 提供认证相关的类型定义
package auth

// LoginResponse 登录响应
type LoginResponse struct {
	LoginUrl string `json:"login_url"` // Casdoor 登录 URL
}

// CallbackRequest OAuth 回调请求
type CallbackRequest struct {
	Code  string `json:"code" form:"code"`   // 授权码
	State string `json:"state" form:"state"` // 状态码
}

// CallbackResponse OAuth 回调响应
type CallbackResponse struct {
	AccessToken  string    `json:"access_token"`  // 访问令牌
	RefreshToken string    `json:"refresh_token"` // 刷新令牌
	ExpiresIn    int64     `json:"expires_in"`    // 过期时间（秒）
	TokenType    string    `json:"token_type"`    // Token 类型
	UserInfo     *UserInfo `json:"user_info"`     // 用户信息
}

// UserInfo 用户信息
type UserInfo struct {
	Id          string   `json:"id"`           // 用户 ID
	Name        string   `json:"name"`         // 用户名
	DisplayName string   `json:"display_name"` // 显示名称
	Email       string   `json:"email"`        // 邮箱
	Phone       string   `json:"phone"`        // 电话
	Avatar      string   `json:"avatar"`       // 头像
	IsAdmin     bool     `json:"is_admin"`     // 是否管理员
	Roles       []string `json:"roles"`        // 角色列表
}

// RefreshTokenRequest 刷新 Token 请求
type RefreshTokenRequest struct {
	RefreshToken string `json:"refresh_token"` // 刷新令牌
}

// RefreshTokenResponse 刷新 Token 响应
type RefreshTokenResponse struct {
	AccessToken string `json:"access_token"` // 新的访问令牌
	ExpiresIn   int64  `json:"expires_in"`   // 过期时间（秒）
}

// GetUserInfoResponse 获取用户信息响应
type GetUserInfoResponse struct {
	UserInfo *UserInfo `json:"user_info"` // 用户信息
}
