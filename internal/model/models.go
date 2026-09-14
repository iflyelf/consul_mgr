package model

import (
	"database/sql"
	"time"
)

// User 用户模型
type User struct {
	ID          int64          `db:"id" json:"id"`
	Username    string         `db:"username" json:"username"`
	Password    string         `db:"password" json:"-"`
	Email       sql.NullString `db:"email" json:"email,omitempty"`
	RealName    sql.NullString `db:"real_name" json:"real_name,omitempty"`
	Status      int            `db:"status" json:"status"`
	CreatedAt   time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time      `db:"updated_at" json:"updated_at"`
	LastLoginAt sql.NullTime   `db:"last_login_at" json:"last_login_at,omitempty"`
	LastLoginIP sql.NullString `db:"last_login_ip" json:"last_login_ip,omitempty"`
}

// Role 角色模型
type Role struct {
	ID          int64     `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Code        string    `db:"code" json:"code"`
	Description string    `db:"description" json:"description,omitempty"`
	Status      int       `db:"status" json:"status"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// Permission 权限模型
type Permission struct {
	ID          int64     `db:"id" json:"id"`
	Name        string    `db:"name" json:"name"`
	Code        string    `db:"code" json:"code"`
	Resource    string    `db:"resource" json:"resource"`
	Action      string    `db:"action" json:"action"`
	Description string    `db:"description" json:"description,omitempty"`
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
}

// UserRole 用户角色关联
type UserRole struct {
	ID        int64     `db:"id" json:"id"`
	UserID    int64     `db:"user_id" json:"user_id"`
	RoleID    int64     `db:"role_id" json:"role_id"`
	CreatedAt time.Time `db:"created_at" json:"created_at"`
}

// RolePermission 角色权限关联
type RolePermission struct {
	ID           int64     `db:"id" json:"id"`
	RoleID       int64     `db:"role_id" json:"role_id"`
	PermissionID int64     `db:"permission_id" json:"permission_id"`
	CreatedAt    time.Time `db:"created_at" json:"created_at"`
}

// ServiceGroup 服务组模型
type ServiceGroup struct {
	ID                int64          `db:"id" json:"id"`
	Name              string         `db:"name" json:"name"`
	Code              string         `db:"code" json:"code"`
	Description       string         `db:"description" json:"description,omitempty"`
	ConsulAddress     string         `db:"consul_address" json:"consul_address"`
	ConsulToken       string         `db:"consul_token" json:"consul_token,omitempty"`
	ConsulDatacenter  string         `db:"consul_datacenter" json:"consul_datacenter"`
	Status            int            `db:"status" json:"status"`
	CreatedBy         sql.NullInt64  `db:"created_by" json:"created_by,omitempty"`
	CreatedAt         time.Time      `db:"created_at" json:"created_at"`
	UpdatedAt         time.Time      `db:"updated_at" json:"updated_at"`
}

// RoleGroupPermission 角色服务组授权
type RoleGroupPermission struct {
	ID          int64     `db:"id" json:"id"`
	RoleID      int64     `db:"role_id" json:"role_id"`
	GroupID     int64     `db:"group_id" json:"group_id"`
	Permissions string    `db:"permissions" json:"permissions"` // JSONB
	CreatedAt   time.Time `db:"created_at" json:"created_at"`
	UpdatedAt   time.Time `db:"updated_at" json:"updated_at"`
}

// AuditLog 审计日志模型
type AuditLog struct {
	ID           int64          `db:"id" json:"id"`
	UserID       sql.NullInt64  `db:"user_id" json:"user_id,omitempty"`
	Username     sql.NullString `db:"username" json:"username,omitempty"`
	Action       string         `db:"action" json:"action"`
	ResourceType sql.NullString `db:"resource_type" json:"resource_type,omitempty"`
	ResourceID   sql.NullString `db:"resource_id" json:"resource_id,omitempty"`
	ResourceName sql.NullString `db:"resource_name" json:"resource_name,omitempty"`
	GroupID      sql.NullInt64  `db:"group_id" json:"group_id,omitempty"`
	Details      sql.NullString `db:"details" json:"details,omitempty"` // JSONB
	IPAddress    sql.NullString `db:"ip_address" json:"ip_address,omitempty"`
	UserAgent    sql.NullString `db:"user_agent" json:"user_agent,omitempty"`
	Status       sql.NullString `db:"status" json:"status,omitempty"`
	ErrorMessage sql.NullString `db:"error_message" json:"error_message,omitempty"`
	CreatedAt    time.Time      `db:"created_at" json:"created_at"`
}
