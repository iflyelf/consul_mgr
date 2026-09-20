// Package user 提供「用户管理」逻辑
//
// 说明:
//
//	用户体系由 Casdoor 维护，列表统一走**服务端分页**：
//	单次请求返回「当前页 + 总数」，关键字由 Casdoor 按字段模糊匹配。
//	切勿改为「拉取全量后在内存过滤」——3 万+ 用户时需上百次串行请求（易超时），
//	且无分页接口会因 SDK 泛型中间层导致内存放大（OOM）。
package user

import (
	"context"

	"github.com/iflyelf/consul_mgr/internal/pkg/casdoor"
)

// defaultPageSize 默认每页条数
const defaultPageSize = 20

// maxPageSize 服务端分页允许的最大每页条数
const maxPageSize = 200

// UserLogic 用户管理逻辑
type UserLogic struct {
	ctx    context.Context
	client *casdoor.Client
}

// NewUserLogic 创建用户管理逻辑实例
func NewUserLogic(ctx context.Context, client *casdoor.Client) *UserLogic {
	return &UserLogic{ctx: ctx, client: client}
}

// normalizeField 规范化搜索字段（仅允许 Casdoor 支持的字段，默认 name）
func normalizeField(field string) string {
	switch field {
	case "name", "displayName", "email", "phone":
		return field
	default:
		return "name"
	}
}

// ListUsers 服务端分页查询用户（支持按字段关键字模糊搜索）
//
// 参数:
//
//	field    - 搜索字段：name / displayName / email / phone（默认 name）
//	keyword  - 关键字（为空则返回全部，按 name 升序）
//	page     - 页码（从 1 开始）
//	pageSize - 每页条数（1..200）
//
// 返回当前页数据与过滤后的总数。
func (l *UserLogic) ListUsers(field, keyword string, page, pageSize int) ([]casdoor.CasdoorUser, int, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = defaultPageSize
	}
	if pageSize > maxPageSize {
		pageSize = maxPageSize
	}
	return l.client.GetUsersPage(page, pageSize, normalizeField(field), keyword)
}

// 以下为用户管理（与 FlyIAM 对齐：新增/编辑/删除/重置密码/管理员标记）。
// 用户唯一存储于 Casdoor；写操作直接作用于 Casdoor，无需本地缓存。

// CreateUser 新增用户
func (l *UserLogic) CreateUser(in casdoor.UserUpsert, defaultPassword string) error {
	return l.client.CreateUser(in, defaultPassword)
}

// UpdateUser 更新用户
func (l *UserLogic) UpdateUser(in casdoor.UserUpsert) error {
	return l.client.UpdateUser(in)
}

// DeleteUser 删除用户
func (l *UserLogic) DeleteUser(name string) error {
	return l.client.DeleteUser(name)
}

// ResetPassword 重置用户密码
func (l *UserLogic) ResetPassword(name, password string) error {
	return l.client.ResetPassword(name, password)
}

// SetUserAdmin 设置/取消管理员
func (l *UserLogic) SetUserAdmin(name string, isAdmin bool) error {
	return l.client.SetUserAdmin(name, isAdmin)
}
