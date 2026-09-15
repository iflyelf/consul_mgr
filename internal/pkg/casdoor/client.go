// Package casdoor 提供 Casdoor 客户端封装和认证功能
package casdoor

import (
	"fmt"

	"github.com/casdoor/casdoor-go-sdk/casdoorsdk"
)

// Client Casdoor 客户端封装
type Client struct {
	config *Config
	sdk    *casdoorsdk.Client
}

// NewClient 创建 Casdoor 客户端
//
// 参数:
//   config - Casdoor 配置信息
//
// 返回:
//   *Client - Casdoor 客户端实例
//   error - 错误信息
func NewClient(config *Config) (*Client, error) {
	// 验证配置
	if config.Endpoint == "" {
		return nil, fmt.Errorf("Casdoor endpoint 不能为空")
	}
	if config.ClientId == "" {
		return nil, fmt.Errorf("Casdoor client_id 不能为空")
	}
	if config.ClientSecret == "" {
		return nil, fmt.Errorf("Casdoor client_secret 不能为空")
	}
	if config.OrganizationName == "" {
		return nil, fmt.Errorf("Casdoor organization_name 不能为空")
	}
	if config.ApplicationName == "" {
		return nil, fmt.Errorf("Casdoor application_name 不能为空")
	}

	// 创建 Casdoor SDK 客户端
	sdk := casdoorsdk.NewClient(
		config.Endpoint,
		config.ClientId,
		config.ClientSecret,
		config.Certificate,
		config.OrganizationName,
		config.ApplicationName,
	)

	return &Client{
		config: config,
		sdk:    sdk,
	}, nil
}

// GetSigninUrl 获取登录 URL
//
// 参数:
//   redirectUri - 回调地址
//
// 返回:
//   string - 登录 URL
func (c *Client) GetSigninUrl(redirectUri string) string {
	return c.sdk.GetSigninUrl(redirectUri)
}

// GetSignupUrl 获取注册 URL
//
// 参数:
//   redirectUri - 回调地址
//
// 返回:
//   string - 注册 URL
func (c *Client) GetSignupUrl(redirectUri string) string {
	return c.sdk.GetSignupUrl(true, redirectUri)
}

// GetToken 通过授权码获取 Token
//
// 参数:
//   code - 授权码
//
// 返回:
//   string - Access Token
//   error - 错误信息
func (c *Client) GetToken(code string) (string, error) {
	token, err := c.sdk.GetOAuthToken(code, "")
	if err != nil {
		return "", fmt.Errorf("获取 Token 失败: %w", err)
	}
	return token.AccessToken, nil
}

// ParseToken 解析并验证 Token
//
// 参数:
//   token - JWT Token
//
// 返回:
//   *Claims - Token 声明信息
//   error - 错误信息
func (c *Client) ParseToken(token string) (*Claims, error) {
	// 解析 JWT Token
	claims, err := c.sdk.ParseJwtToken(token)
	if err != nil {
		return nil, fmt.Errorf("Token 解析失败: %w", err)
	}

	// 转换为自定义 Claims 结构
	result := &Claims{
		TokenType: claims.TokenType,
		Scope:     claims.Scope,
		Iss:       claims.Issuer,
		Sub:       claims.Subject,
		Aud:       claims.Audience,
		Exp:       claims.ExpiresAt.Unix(),
		Nbf:       claims.NotBefore.Unix(),
		Iat:       claims.IssuedAt.Unix(),
		Jti:       claims.Id,
	}

	// 提取用户信息
	if claims.User.Name != "" {
		result.User = &UserInfo{
			Owner:       claims.User.Owner,
			Name:        claims.User.Name,
			Id:          claims.User.Id,
			DisplayName: claims.User.DisplayName,
			Email:       claims.User.Email,
			Phone:       claims.User.Phone,
			Avatar:      claims.User.Avatar,
			IsAdmin:     claims.User.IsAdmin,
		}

		// 提取角色信息
		if len(claims.User.Roles) > 0 {
			result.User.Roles = make([]*Role, 0, len(claims.User.Roles))
			for _, role := range claims.User.Roles {
				result.User.Roles = append(result.User.Roles, &Role{
					Owner:       role.Owner,
					Name:        role.Name,
					DisplayName: role.DisplayName,
					Description: role.Description,
					IsEnabled:   role.IsEnabled,
				})
			}
		}

		// 提取权限信息
		if len(claims.User.Permissions) > 0 {
			result.User.Permissions = make([]*Permission, 0, len(claims.User.Permissions))
			for _, perm := range claims.User.Permissions {
				result.User.Permissions = append(result.User.Permissions, &Permission{
					Owner:        perm.Owner,
					Name:         perm.Name,
					DisplayName:  perm.DisplayName,
					Description:  perm.Description,
					ResourceType: perm.ResourceType,
					Resources:    perm.Resources,
					Actions:      perm.Actions,
					Effect:       perm.Effect,
					IsEnabled:    perm.IsEnabled,
				})
			}
		}
	}

	return result, nil
}

// GetUserInfo 获取用户详细信息
//
// 参数:
//   username - 用户名
//
// 返回:
//   *UserInfo - 用户信息
//   error - 错误信息
func (c *Client) GetUserInfo(username string) (*UserInfo, error) {
	user, err := c.sdk.GetUser(username)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("用户不存在: %s", username)
	}

	return &UserInfo{
		Owner:         user.Owner,
		Name:          user.Name,
		Id:            user.Id,
		DisplayName:   user.DisplayName,
		Email:         user.Email,
		Phone:         user.Phone,
		Avatar:        user.Avatar,
		IsAdmin:       user.IsAdmin,
		IsGlobalAdmin: user.IsAdmin, // Casdoor SDK 中使用 IsAdmin 表示全局管理员
	}, nil
}

// GetUserRoles 获取用户的角色列表
//
// 参数:
//   username - 用户名
//
// 返回:
//   []string - 角色名称列表
//   error - 错误信息
func (c *Client) GetUserRoles(username string) ([]string, error) {
	user, err := c.sdk.GetUser(username)
	if err != nil {
		return nil, fmt.Errorf("获取用户信息失败: %w", err)
	}

	if user == nil {
		return nil, fmt.Errorf("用户不存在: %s", username)
	}

	roles := make([]string, 0, len(user.Roles))
	for _, role := range user.Roles {
		roles = append(roles, role.Name)
	}

	return roles, nil
}

// CheckPermission 检查用户是否有指定权限
//
// 参数:
//   username - 用户名
//   resource - 资源类型（如: consul_group, consul_service）
//   action - 操作类型（如: read, write, delete）
//
// 返回:
//   bool - 是否有权限
//   error - 错误信息
//
// 权限检查逻辑：
//   1. 获取用户的所有权限
//   2. 遍历权限列表，匹配资源类型和操作
//   3. 支持通配符匹配（* 表示所有）
func (c *Client) CheckPermission(username, resource, action string) (bool, error) {
	user, err := c.sdk.GetUser(username)
	if err != nil {
		return false, fmt.Errorf("获取用户信息失败: %w", err)
	}

	if user == nil {
		return false, fmt.Errorf("用户不存在: %s", username)
	}

	// 全局管理员拥有所有权限
	if user.IsAdmin {
		return true, nil
	}

	// 遍历用户的所有权限
	for _, perm := range user.Permissions {
		// 权限未启用，跳过
		if !perm.IsEnabled {
			continue
		}

		// 效果不是 Allow，跳过
		if perm.Effect != "Allow" {
			continue
		}

		// 检查资源类型匹配
		resourceMatch := false
		if perm.ResourceType == "*" || perm.ResourceType == resource {
			resourceMatch = true
		}

		if !resourceMatch {
			continue
		}

		// 检查操作匹配
		for _, act := range perm.Actions {
			if act == "*" || act == action {
				return true, nil
			}
		}
	}

	return false, nil
}

// CheckPermissionByToken 通过 Token 检查权限
//
// 参数:
//   token - JWT Token
//   resource - 资源类型
//   action - 操作类型
//
// 返回:
//   bool - 是否有权限
//   error - 错误信息
func (c *Client) CheckPermissionByToken(token, resource, action string) (bool, error) {
	// 解析 Token 获取用户信息
	claims, err := c.ParseToken(token)
	if err != nil {
		return false, err
	}

	if claims.User == nil {
		return false, fmt.Errorf("Token 中没有用户信息")
	}

	// 全局管理员拥有所有权限
	if claims.User.IsAdmin {
		return true, nil
	}

	// 遍历权限列表
	for _, perm := range claims.User.Permissions {
		// 权限未启用，跳过
		if !perm.IsEnabled {
			continue
		}

		// 效果不是 Allow，跳过
		if perm.Effect != "Allow" {
			continue
		}

		// 检查资源类型匹配
		resourceMatch := false
		if perm.ResourceType == "*" || perm.ResourceType == resource {
			resourceMatch = true
		}

		if !resourceMatch {
			continue
		}

		// 检查操作匹配
		for _, act := range perm.Actions {
			if act == "*" || act == action {
				return true, nil
			}
		}
	}

	return false, nil
}

// RefreshToken 刷新 Token
//
// 参数:
//   refreshToken - 刷新令牌
//
// 返回:
//   string - 新的 Access Token
//   error - 错误信息
func (c *Client) RefreshToken(refreshToken string) (string, error) {
	token, err := c.sdk.RefreshOAuthToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("刷新 Token 失败: %w", err)
	}
	return token.AccessToken, nil
}

// GetConfig 获取客户端配置
//
// 返回:
//   *Config - 配置信息
func (c *Client) GetConfig() *Config {
	return c.config
}
