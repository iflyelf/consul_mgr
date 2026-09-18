package types

import "time"

// AuditLogQuery 审计日志查询请求
type AuditLogQuery struct {
	Page         int       `form:"page,default=1"`
	PageSize     int       `form:"page_size,default=20"`
	UserID       *int64    `form:"user_id,optional"`
	Action       string    `form:"action,optional"`
	ResourceType string    `form:"resource_type,optional"`
	GroupID      *int64    `form:"group_id,optional"`
	Status       string    `form:"status,optional"`
	StartTime    time.Time `form:"start_time,optional"`
	EndTime      time.Time `form:"end_time,optional"`
}

// AuditLogStats 审计日志统计
type AuditLogStats struct {
	TotalLogs      int64            `json:"total_logs"`
	ByAction       map[string]int64 `json:"by_action"`
	ByUser         map[string]int64 `json:"by_user"`
	ByResourceType map[string]int64 `json:"by_resource_type"`
	ByStatus       map[string]int64 `json:"by_status"`
	RecentActions  []RecentAction   `json:"recent_actions"`
}

// RecentAction 最近操作
type RecentAction struct {
	Action       string    `json:"action"`
	Username     string    `json:"username"`
	ResourceType string    `json:"resource_type"`
	ResourceName string    `json:"resource_name"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
}

// DashboardOverview 仪表盘概览
type DashboardOverview struct {
	TotalGroups     int64       `json:"total_groups"`
	TotalServices   int64       `json:"total_services"`
	TotalInstances  int64       `json:"total_instances"`
	HealthyServices int64       `json:"healthy_services"`
	TotalUsers      int64       `json:"total_users"`
	ActiveUsers     int64       `json:"active_users"`
	TodayLogs       int64       `json:"today_logs"`
	GroupStats      []GroupStat `json:"group_stats"`
}

// GroupStat 服务组统计
type GroupStat struct {
	GroupID        int64  `json:"group_id"`
	GroupName      string `json:"group_name"`
	ServiceCount   int    `json:"service_count"`
	InstanceCount  int    `json:"instance_count"`
	HealthyCount   int    `json:"healthy_count"`
	UnhealthyCount int    `json:"unhealthy_count"`
}
