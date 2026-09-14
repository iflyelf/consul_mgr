// Package auth 提供认证逻辑
package auth

import (
	"context"
	"fmt"

	"github.com/zeromicro/go-zero/core/logx"

	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// OAuthLogic OAuth 认证逻辑
type OAuthLogic struct {
	ctx    context.Context
	client *casdoor.Client
	logx.Logger
}

// NewOAuthLogic 创建 OAuth 认证逻辑实例
func NewOAuthLogic(ctx context.Context, client *casdoor.Client) *OAuthLogic {
	return &OAuthLogic{
		ctx:    ctx,
		client: client,
		Logger: logx.WithContext(ctx),
	}
}

// GetLoginUrl 获取登录 URL
//
// 参数:
//   redirectUri - 回调地址
//
// 返回:
//   *types.LoginResponse - 登录响应
//   error - 错误信息
func (l *OAuthLogic) GetLoginUrl(redirectUri string) (*types.LoginResponse, error) {
	// 获取 Casdoor 登录 URL
	loginUrl := l.client.GetSigninUrl(redirectUri)

	l.Infof("生成登录 URL: %s", loginUrl)

	return &types.LoginResponse{
		LoginUrl: loginUrl,
	}, nil
}

// HandleCallback 处理 OAuth 回调
//
// 参数:
//   req - 回调请求
//
// 返回:
//   *types.CallbackResponse - 回调响应
//   error - 错误信息
func (l *OAuthLogic) HandleCallback(req *types.CallbackRequest) (*types.CallbackResponse, error) {
	// 1. 验证参数
	if req.Code == "" {
		return nil, fmt.Errorf("授权码不能为空")
	}

	// 2. 使用授权码换取 Token
	accessToken, err := l.client.GetToken(req.Code)
	if err != nil {
		l.Errorf("获取 Token 失败: %v", err)
		return nil, fmt.Errorf("获取 Token 失败: %w", err)
	}

	// 3. 解析 Token 获取用户信息
	claims, err := l.client.ParseToken(accessToken)
	if err != nil {
		l.Errorf("解析 Token 失败: %v", err)
		return nil, fmt.Errorf("解析 Token 失败: %w", err)
	}

	if claims.User == nil {
		return nil, fmt.Errorf("Token 中没有用户信息")
	}

	// 4. 提取角色列表
	roles := make([]string, 0)
	if len(claims.User.Roles) > 0 {
		for _, role := range claims.User.Roles {
			roles = append(roles, role.Name)
		}
	}

	// 5. 构造响应
	response := &types.CallbackResponse{
		AccessToken:  accessToken,
		RefreshToken: claims.RefreshToken,
		ExpiresIn:    claims.Exp - claims.Iat,
		TokenType:    "Bearer",
		UserInfo: &types.UserInfo{
			Id:          claims.User.Id,
			Name:        claims.User.Name,
			DisplayName: claims.User.DisplayName,
			Email:       claims.User.Email,
			Phone:       claims.User.Phone,
			Avatar:      claims.User.Avatar,
			IsAdmin:     claims.User.IsAdmin,
			Roles:       roles,
		},
	}

	l.Infof("用户登录成功: %s (%s)", claims.User.Name, claims.User.Id)

	return response, nil
}

// RefreshToken 刷新 Token
//
// 参数:
//   req - 刷新请求
//
// 返回:
//   *types.RefreshTokenResponse - 刷新响应
//   error - 错误信息
func (l *OAuthLogic) RefreshToken(req *types.RefreshTokenRequest) (*types.RefreshTokenResponse, error) {
	// 1. 验证参数
	if req.RefreshToken == "" {
		return nil, fmt.Errorf("刷新令牌不能为空")
	}

	// 2. 刷新 Token
	newAccessToken, err := l.client.RefreshToken(req.RefreshToken)
	if err != nil {
		l.Errorf("刷新 Token 失败: %v", err)
		return nil, fmt.Errorf("刷新 Token 失败: %w", err)
	}

	// 3. 解析新 Token 获取过期时间
	claims, err := l.client.ParseToken(newAccessToken)
	if err != nil {
		l.Errorf("解析新 Token 失败: %v", err)
		return nil, fmt.Errorf("解析新 Token 失败: %w", err)
	}

	l.Infof("Token 刷新成功")

	return &types.RefreshTokenResponse{
		AccessToken: newAccessToken,
		ExpiresIn:   claims.Exp - claims.Iat,
	}, nil
}

// GetCurrentUser 获取当前用户信息
//
// 参数:
//   token - 访问令牌
//
// 返回:
//   *types.UserInfo - 用户信息
//   error - 错误信息
func (l *OAuthLogic) GetCurrentUser(token string) (*types.UserInfo, error) {
	// 1. 解析 Token
	claims, err := l.client.ParseToken(token)
	if err != nil {
		l.Errorf("解析 Token 失败: %v", err)
		return nil, fmt.Errorf("解析 Token 失败: %w", err)
	}

	if claims.User == nil {
		return nil, fmt.Errorf("Token 中没有用户信息")
	}

	// 2. 提取角色列表
	roles := make([]string, 0)
	if len(claims.User.Roles) > 0 {
		for _, role := range claims.User.Roles {
			roles = append(roles, role.Name)
		}
	}

	// 3. 构造用户信息
	userInfo := &types.UserInfo{
		Id:          claims.User.Id,
		Name:        claims.User.Name,
		DisplayName: claims.User.DisplayName,
		Email:       claims.User.Email,
		Phone:       claims.User.Phone,
		Avatar:      claims.User.Avatar,
		IsAdmin:     claims.User.IsAdmin || claims.User.IsGlobalAdmin,
		Roles:       roles,
	}

	return userInfo, nil
}
