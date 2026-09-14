package auth

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/iflyelf/consul_mgr/internal/middleware"
	"github.com/iflyelf/consul_mgr/internal/pkg/password"
	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
)

// LoginLogic 登录逻辑
type LoginLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewLoginLogic 创建登录逻辑
func NewLoginLogic(ctx context.Context, svcCtx *svc.ServiceContext) *LoginLogic {
	return &LoginLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// Login 用户登录
func (l *LoginLogic) Login(req *types.LoginRequest) (*types.LoginResponse, error) {
	// 查询用户
	var user struct {
		ID       int64  `db:"id"`
		Username string `db:"username"`
		Password string `db:"password"`
		Status   int    `db:"status"`
	}
	
	query := `SELECT id, username, password, status FROM users WHERE username = $1`
	err := l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &user, query, req.Username)
	if err != nil {
		return nil, errors.New("用户名或密码错误")
	}
	
	// 检查用户状态
	if user.Status != 1 {
		return nil, errors.New("用户已被禁用")
	}
	
	// 验证密码
	if !password.Verify(req.Password, user.Password) {
		return nil, errors.New("用户名或密码错误")
	}
	
	// 生成 Token
	accessToken, err := l.svcCtx.JWTManager.GenerateAccessToken(user.ID, user.Username)
	if err != nil {
		return nil, errors.New("生成令牌失败")
	}
	
	refreshToken, err := l.svcCtx.JWTManager.GenerateRefreshToken(user.ID, user.Username)
	if err != nil {
		return nil, errors.New("生成刷新令牌失败")
	}
	
	// 更新最后登录时间
	updateQuery := `UPDATE users SET last_login_at = $1, last_login_ip = $2, updated_at = $1 WHERE id = $3`
	_, _ = l.svcCtx.DB.ExecCtx(l.ctx, updateQuery, time.Now(), "", user.ID)
	
	// 查询用户角色
	roles, _ := l.getUserRoles(user.ID)
	
	return &types.LoginResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    l.svcCtx.Config.JWT.AccessExpire,
		UserInfo: types.UserInfo{
			ID:       user.ID,
			Username: user.Username,
			Roles:    roles,
		},
	}, nil
}

// getUserRoles 获取用户角色
func (l *LoginLogic) getUserRoles(userID int64) ([]string, error) {
	query := `
		SELECT r.code 
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.status = 1
	`
	
	var roles []string
	err := l.svcCtx.DB.QueryRowsPartialCtx(l.ctx, &roles, query, userID)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	
	return roles, nil
}

// GetUserInfoLogic 获取用户信息逻辑
type GetUserInfoLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

// NewGetUserInfoLogic 创建获取用户信息逻辑
func NewGetUserInfoLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserInfoLogic {
	return &GetUserInfoLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// GetUserInfo 获取用户信息
func (l *GetUserInfoLogic) GetUserInfo() (*types.UserInfo, error) {
	userID := middleware.GetUserID(l.ctx)
	username := middleware.GetUsername(l.ctx)
	
	if userID == 0 {
		return nil, errors.New("未授权")
	}
	
	// 查询用户详细信息
	var user struct {
		Email    sql.NullString `db:"email"`
		RealName sql.NullString `db:"real_name"`
	}
	
	query := `SELECT email, real_name FROM users WHERE id = $1`
	_ = l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &user, query, userID)
	
	// 查询用户角色
	roles, _ := l.getUserRoles(userID)
	
	email := ""
	if user.Email.Valid {
		email = user.Email.String
	}
	
	realName := ""
	if user.RealName.Valid {
		realName = user.RealName.String
	}
	
	return &types.UserInfo{
		ID:       userID,
		Username: username,
		Email:    email,
		RealName: realName,
		Roles:    roles,
	}, nil
}

// getUserRoles 获取用户角色
func (l *GetUserInfoLogic) getUserRoles(userID int64) ([]string, error) {
	query := `
		SELECT r.code 
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1 AND r.status = 1
	`
	
	var roles []string
	err := l.svcCtx.DB.QueryRowsPartialCtx(l.ctx, &roles, query, fmt.Sprint(userID))
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	
	return roles, nil
}
