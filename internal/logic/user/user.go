package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
	"golang.org/x/crypto/bcrypt"
)

type ListUsersLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListUsersLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListUsersLogic {
	return &ListUsersLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListUsersLogic) ListUsers(req *types.ListUsersRequest) (*types.ListResponse, error) {
	offset := (req.Page - 1) * req.PageSize
	
	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	
	if req.Keyword != "" {
		whereClause += " AND (username LIKE '%' || $1 || '%' OR email LIKE '%' || $1 || '%')"
		args = append(args, req.Keyword)
	}
	
	if req.Status != nil {
		whereClause += fmt.Sprintf(" AND status = $%d", len(args)+1)
		args = append(args, *req.Status)
	}
	
	// 查询总数
	var total int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM users %s", whereClause)
	err := l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &total, countQuery, args...)
	if err != nil {
		return nil, err
	}
	
	// 查询用户列表
	query := fmt.Sprintf(`
		SELECT id, username, email, status, created_at, updated_at
		FROM users %s 
		ORDER BY created_at DESC 
		LIMIT $%d OFFSET $%d
	`, whereClause, len(args)+1, len(args)+2)
	args = append(args, req.PageSize, offset)
	
	var users []struct {
		ID        int64  `db:"id"`
		Username  string `db:"username"`
		Email     string `db:"email"`
		Status    int    `db:"status"`
		CreatedAt string `db:"created_at"`
		UpdatedAt string `db:"updated_at"`
	}
	
	err = l.svcCtx.DB.QueryRowsPartialCtx(l.ctx, &users, query, args...)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	
	// 转换为 UserInfo 并查询角色
	list := make([]types.UserInfo, 0, len(users))
	for _, u := range users {
		roles, _ := l.getUserRoles(u.ID)
		list = append(list, types.UserInfo{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			Status:    u.Status,
			Roles:     roles,
			CreatedAt: u.CreatedAt,
			UpdatedAt: u.UpdatedAt,
		})
	}
	
	return &types.ListResponse{
		List: list,
		PageInfo: types.PageInfo{
			Page:     req.Page,
			PageSize: req.PageSize,
			Total:    total,
		},
	}, nil
}

func (l *ListUsersLogic) getUserRoles(userID int64) ([]string, error) {
	query := `
		SELECT r.name 
		FROM roles r
		INNER JOIN user_roles ur ON r.id = ur.role_id
		WHERE ur.user_id = $1
	`
	
	var roles []struct {
		Name string `db:"name"`
	}
	
	err := l.svcCtx.DB.QueryRowsPartialCtx(l.ctx, &roles, query, userID)
	if err != nil && err != sql.ErrNoRows {
		return []string{}, nil
	}
	
	result := make([]string, 0, len(roles))
	for _, r := range roles {
		result = append(result, r.Name)
	}
	
	return result, nil
}

type GetUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewGetUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *GetUserLogic {
	return &GetUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *GetUserLogic) GetUser(id int64) (*types.UserInfo, error) {
	query := `
		SELECT id, username, email, status, created_at, updated_at
		FROM users 
		WHERE id = $1
	`
	
	var user struct {
		ID        int64  `db:"id"`
		Username  string `db:"username"`
		Email     string `db:"email"`
		Status    int    `db:"status"`
		CreatedAt string `db:"created_at"`
		UpdatedAt string `db:"updated_at"`
	}
	
	err := l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &user, query, id)
	if err != nil {
		return nil, err
	}
	
	// 查询用户角色
	roles, _ := NewListUsersLogic(l.ctx, l.svcCtx).getUserRoles(id)
	
	return &types.UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Status:    user.Status,
		Roles:     roles,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
	}, nil
}

type CreateUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewCreateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CreateUserLogic {
	return &CreateUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *CreateUserLogic) CreateUser(req *types.CreateUserRequest) (*types.UserInfo, error) {
	// 检查用户名是否存在
	var count int64
	err := l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &count,
		"SELECT COUNT(*) FROM users WHERE username = $1", req.Username)
	if err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("用户名已存在")
	}
	
	// 检查邮箱是否存在（如果提供了）
	if req.Email != "" {
		err = l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &count,
			"SELECT COUNT(*) FROM users WHERE email = $1", req.Email)
		if err != nil {
			return nil, err
		}
		if count > 0 {
			return nil, errors.New("邮箱已存在")
		}
	}
	
	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	
	// 插入用户
	query := `
		INSERT INTO users (username, password, email, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING id, created_at, updated_at
	`
	
	var result struct {
		ID        int64  `db:"id"`
		CreatedAt string `db:"created_at"`
		UpdatedAt string `db:"updated_at"`
	}
	
	email := req.Email
	if email == "" {
		email = req.Username + "@example.com" // 默认邮箱
	}
	
	err = l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &result, query,
		req.Username, string(hashedPassword), email, 1)
	if err != nil {
		return nil, err
	}
	
	return &types.UserInfo{
		ID:        result.ID,
		Username:  req.Username,
		Email:     email,
		Status:    1,
		Roles:     []string{},
		CreatedAt: result.CreatedAt,
		UpdatedAt: result.UpdatedAt,
	}, nil
}

type UpdateUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateUserLogic {
	return &UpdateUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateUserLogic) UpdateUser(id int64, req *types.UpdateUserRequest) error {
	if req.Email == "" && req.Status == nil {
		return nil
	}
	
	if req.Email != "" {
		// 检查邮箱是否被其他用户使用
		var count int64
		err := l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &count,
			"SELECT COUNT(*) FROM users WHERE email = $1 AND id != $2",
			req.Email, id)
		if err != nil {
			return err
		}
		if count > 0 {
			return errors.New("邮箱已被使用")
		}
	}
	
	// 构建更新语句
	query := "UPDATE users SET updated_at = NOW()"
	args := []interface{}{}
	
	if req.Email != "" {
		query += fmt.Sprintf(", email = $%d", len(args)+1)
		args = append(args, req.Email)
	}
	
	if req.Status != nil {
		query += fmt.Sprintf(", status = $%d", len(args)+1)
		args = append(args, *req.Status)
	}
	
	query += fmt.Sprintf(" WHERE id = $%d", len(args)+1)
	args = append(args, id)
	
	_, err := l.svcCtx.DB.ExecCtx(l.ctx, query, args...)
	return err
}

type DeleteUserLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteUserLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteUserLogic {
	return &DeleteUserLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteUserLogic) DeleteUser(id int64) error {
	// 先删除用户角色关联
	_, err := l.svcCtx.DB.ExecCtx(l.ctx, "DELETE FROM user_roles WHERE user_id = $1", id)
	if err != nil {
		return err
	}
	
	// 删除用户
	_, err = l.svcCtx.DB.ExecCtx(l.ctx, "DELETE FROM users WHERE id = $1", id)
	return err
}

type ChangePasswordLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewChangePasswordLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ChangePasswordLogic {
	return &ChangePasswordLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ChangePasswordLogic) ChangePassword(id int64, req *types.ChangePasswordRequest) error {
	// 获取当前密码
	var currentPassword string
	err := l.svcCtx.DB.QueryRowPartialCtx(l.ctx, &currentPassword,
		"SELECT password FROM users WHERE id = $1", id)
	if err != nil {
		return err
	}
	
	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(currentPassword), []byte(req.OldPassword)); err != nil {
		return errors.New("旧密码错误")
	}
	
	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	// 更新密码
	query := "UPDATE users SET password = $1, updated_at = NOW() WHERE id = $2"
	_, err = l.svcCtx.DB.ExecCtx(l.ctx, query, string(hashedPassword), id)
	return err
}

type AssignRolesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewAssignRolesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *AssignRolesLogic {
	return &AssignRolesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *AssignRolesLogic) AssignRoles(id int64, req *types.AssignRolesRequest) error {
	// 删除现有角色
	_, err := l.svcCtx.DB.ExecCtx(l.ctx, "DELETE FROM user_roles WHERE user_id = $1", id)
	if err != nil {
		return err
	}
	
	// 添加新角色
	for _, roleID := range req.RoleIDs {
		query := "INSERT INTO user_roles (user_id, role_id, created_at) VALUES ($1, $2, NOW())"
		_, err := l.svcCtx.DB.ExecCtx(l.ctx, query, id, roleID)
		if err != nil {
			return err
		}
	}
	
	return nil
}
