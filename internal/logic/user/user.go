package user

import (
	"context"
	"errors"

	"github.com/iflyelf/consul_mgr/internal/model"
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
	
	query := l.svcCtx.DB.Model(&model.User{}).Preload("Roles")
	
	// 关键词搜索
	if req.Keyword != "" {
		query = query.Where("username LIKE ? OR email LIKE ? OR nickname LIKE ?",
			"%"+req.Keyword+"%", "%"+req.Keyword+"%", "%"+req.Keyword+"%")
	}
	
	// 状态筛选
	if req.Status != nil {
		query = query.Where("status = ?", *req.Status)
	}
	
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	
	var users []model.User
	if err := query.Offset(offset).Limit(req.PageSize).Find(&users).Error; err != nil {
		return nil, err
	}
	
	// 转换为响应格式
	list := make([]types.UserInfo, 0, len(users))
	for _, u := range users {
		roles := make([]types.RoleInfo, 0, len(u.Roles))
		for _, r := range u.Roles {
			roles = append(roles, types.RoleInfo{
				ID:          r.ID,
				Name:        r.Name,
				Code:        r.Code,
				Description: r.Description,
				Status:      r.Status,
				CreatedAt:   r.CreatedAt,
				UpdatedAt:   r.UpdatedAt,
			})
		}
		
		list = append(list, types.UserInfo{
			ID:        u.ID,
			Username:  u.Username,
			Email:     u.Email,
			Nickname:  u.Nickname,
			Avatar:    u.Avatar,
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
	var user model.User
	if err := l.svcCtx.DB.Preload("Roles").First(&user, id).Error; err != nil {
		return nil, err
	}
	
	roles := make([]types.RoleInfo, 0, len(user.Roles))
	for _, r := range user.Roles {
		roles = append(roles, types.RoleInfo{
			ID:          r.ID,
			Name:        r.Name,
			Code:        r.Code,
			Description: r.Description,
			Status:      r.Status,
			CreatedAt:   r.CreatedAt,
			UpdatedAt:   r.UpdatedAt,
		})
	}
	
	return &types.UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
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
	if err := l.svcCtx.DB.Model(&model.User{}).Where("username = ?", req.Username).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("用户名已存在")
	}
	
	// 检查邮箱是否存在
	if err := l.svcCtx.DB.Model(&model.User{}).Where("email = ?", req.Email).Count(&count).Error; err != nil {
		return nil, err
	}
	if count > 0 {
		return nil, errors.New("邮箱已存在")
	}
	
	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	
	user := &model.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Email:    req.Email,
		Nickname: req.Nickname,
		Avatar:   req.Avatar,
		Status:   req.Status,
	}
	
	if err := l.svcCtx.DB.Create(user).Error; err != nil {
		return nil, err
	}
	
	return &types.UserInfo{
		ID:        user.ID,
		Username:  user.Username,
		Email:     user.Email,
		Nickname:  user.Nickname,
		Avatar:    user.Avatar,
		Status:    user.Status,
		Roles:     []types.RoleInfo{},
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
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
	updates := make(map[string]interface{})
	
	if req.Email != "" {
		// 检查邮箱是否被其他用户使用
		var count int64
		if err := l.svcCtx.DB.Model(&model.User{}).
			Where("email = ? AND id != ?", req.Email, id).
			Count(&count).Error; err != nil {
			return err
		}
		if count > 0 {
			return errors.New("邮箱已被使用")
		}
		updates["email"] = req.Email
	}
	
	if req.Nickname != "" {
		updates["nickname"] = req.Nickname
	}
	
	if req.Avatar != "" {
		updates["avatar"] = req.Avatar
	}
	
	if req.Status != nil {
		updates["status"] = *req.Status
	}
	
	if len(updates) == 0 {
		return nil
	}
	
	return l.svcCtx.DB.Model(&model.User{}).Where("id = ?", id).Updates(updates).Error
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
	// 软删除
	return l.svcCtx.DB.Delete(&model.User{}, id).Error
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
	var user model.User
	if err := l.svcCtx.DB.First(&user, id).Error; err != nil {
		return err
	}
	
	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return errors.New("旧密码错误")
	}
	
	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	
	return l.svcCtx.DB.Model(&user).Update("password", string(hashedPassword)).Error
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
	var user model.User
	if err := l.svcCtx.DB.First(&user, id).Error; err != nil {
		return err
	}
	
	// 获取角色
	var roles []model.Role
	if err := l.svcCtx.DB.Where("id IN ?", req.RoleIds).Find(&roles).Error; err != nil {
		return err
	}
	
	// 替换用户角色
	return l.svcCtx.DB.Model(&user).Association("Roles").Replace(roles)
}
