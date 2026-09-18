package model

import "time"

// UserFieldSyncConfig 用户字段自动同步配置（单例，页面可配置）
type UserFieldSyncConfig struct {
	ID uint8 `db:"id" json:"id"`
	// Enabled 是否启用定时自动同步
	Enabled bool `db:"enabled" json:"enabled"`
	// Interval 同步间隔（如 6h、30m、1d）
	Interval string `db:"interval" json:"interval"`
	// SyncOnStartup 启动时是否同步一次
	SyncOnStartup bool `db:"sync_on_startup" json:"syncOnStartup"`
	// LastRunAt 上次执行时间
	LastRunAt *time.Time `db:"last_run_at" json:"lastRunAt,omitempty"`
	// LastStatus 上次执行状态：success / failed
	LastStatus string `db:"last_status" json:"lastStatus"`
	// LastMessage 上次执行结果摘要
	LastMessage string    `db:"last_message" json:"lastMessage"`
	UpdatedAt   time.Time `db:"updated_at" json:"updatedAt"`
}

// UserFieldSyncLog 用户字段同步日志
type UserFieldSyncLog struct {
	ID          int64      `db:"id" json:"id"`
	Status      string     `db:"status" json:"status"` // running / success / failed
	TriggerType string     `db:"trigger_type" json:"triggerType"`
	Total       int        `db:"total" json:"total"`
	Added       int        `db:"added" json:"added"`
	Updated     int        `db:"updated" json:"updated"`
	Message     string     `db:"message" json:"message"`
	StartedAt   time.Time  `db:"started_at" json:"startedAt"`
	CompletedAt *time.Time `db:"completed_at" json:"completedAt,omitempty"`
}

// UserFieldSyncProgress 同步进度（内存态，用于前端轮询展示）
type UserFieldSyncProgress struct {
	Running   bool   `json:"running"`
	Status    string `json:"status"`
	Total     int    `json:"total"`
	Added     int    `json:"added"`
	Updated   int    `json:"updated"`
	Message   string `json:"message"`
	StartedAt string `json:"startedAt"`
	Trigger   string `json:"trigger"`
}
