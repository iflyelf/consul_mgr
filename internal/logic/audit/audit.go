// Package audit 提供审计日志管理逻辑
package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/zeromicro/go-zero/core/logx"
	"github.com/zeromicro/go-zero/core/stores/sqlx"
)

// AuditLogic 审计日志逻辑
type AuditLogic struct {
	ctx    context.Context
	db     sqlx.SqlConn
	logger logx.Logger
}

// NewAuditLogic 创建审计日志逻辑实例
func NewAuditLogic(ctx context.Context, db sqlx.SqlConn) *AuditLogic {
	return &AuditLogic{
		ctx:    ctx,
		db:     db,
		logger: logx.WithContext(ctx),
	}
}

// AuditLog 审计日志模型
type AuditLog struct {
	ID           int64                  `db:"id" json:"id"`
	UserID       string                 `db:"user_id" json:"user_id"`
	Username     string                 `db:"username" json:"username"`
	Action       string                 `db:"action" json:"action"`
	ResourceType string                 `db:"resource_type" json:"resource_type"`
	ResourceID   string                 `db:"resource_id" json:"resource_id"`
	ResourceName string                 `db:"resource_name" json:"resource_name"`
	GroupID      *int64                 `db:"group_id" json:"group_id,omitempty"`
	Details      map[string]interface{} `db:"details" json:"details"`
	IPAddress    string                 `db:"ip_address" json:"ip_address"`
	UserAgent    string                 `db:"user_agent" json:"user_agent"`
	Status       string                 `db:"status" json:"status"`
	ErrorMessage string                 `db:"error_message" json:"error_message,omitempty"`
	CreatedAt    time.Time              `db:"created_at" json:"created_at"`
}

// LogAction 记录操作日志
func (l *AuditLogic) LogAction(
	userID, username, action, resourceType, resourceID, resourceName string,
	groupID *int64,
	details map[string]interface{},
	ipAddress, userAgent, status, errorMessage string,
) error {
	
	detailsJSON, _ := json.Marshal(details)
	
	query := `
		INSERT INTO audit_logs 
		(user_id, username, action, resource_type, resource_id, resource_name,
		 group_id, details, ip_address, user_agent, status, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
	`
	
	_, err := l.db.ExecCtx(l.ctx, query,
		userID, username, action, resourceType, resourceID, resourceName,
		groupID, detailsJSON, ipAddress, userAgent, status, errorMessage)
	
	if err != nil {
		l.logger.Errorf("记录审计日志失败: %v", err)
		return fmt.Errorf("记录审计日志失败: %w", err)
	}
	
	return nil
}

// ListAuditLogs 查询审计日志列表
func (l *AuditLogic) ListAuditLogs(
	userID, username, action, resourceType string,
	groupID *int64,
	status string,
	startTime, endTime *time.Time,
	page, pageSize int,
) ([]*AuditLog, int64, error) {
	
	// 构建查询条件
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1
	
	if userID != "" {
		whereClause += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	
	if username != "" {
		whereClause += fmt.Sprintf(" AND username LIKE $%d", argIdx)
		args = append(args, "%"+username+"%")
		argIdx++
	}
	
	if action != "" {
		whereClause += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, action)
		argIdx++
	}
	
	if resourceType != "" {
		whereClause += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, resourceType)
		argIdx++
	}
	
	if groupID != nil {
		whereClause += fmt.Sprintf(" AND group_id = $%d", argIdx)
		args = append(args, *groupID)
		argIdx++
	}
	
	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	
	if startTime != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *startTime)
		argIdx++
	}
	
	if endTime != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *endTime)
		argIdx++
	}
	
	// 查询总数
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM audit_logs %s", whereClause)
	var total int64
	err := l.db.QueryRowCtx(l.ctx, &total, countQuery, args...)
	if err != nil {
		l.logger.Errorf("查询审计日志总数失败: %v", err)
		return nil, 0, fmt.Errorf("查询审计日志总数失败: %w", err)
	}
	
	// 查询列表
	offset := (page - 1) * pageSize
	listQuery := fmt.Sprintf(`
		SELECT id, user_id, username, action, resource_type, resource_id, resource_name,
		       group_id, details, ip_address, user_agent, status, error_message, created_at
		FROM audit_logs
		%s
		ORDER BY created_at DESC
		LIMIT $%d OFFSET $%d
	`, whereClause, argIdx, argIdx+1)
	
	args = append(args, pageSize, offset)
	
	var logs []*AuditLog
	err = l.db.QueryRowsCtx(l.ctx, &logs, listQuery, args...)
	if err != nil {
		l.logger.Errorf("查询审计日志列表失败: %v", err)
		return nil, 0, fmt.Errorf("查询审计日志列表失败: %w", err)
	}
	
	return logs, total, nil
}

// ExportAuditLogs 导出审计日志（返回所有数据）
func (l *AuditLogic) ExportAuditLogs(
	userID, username, action, resourceType string,
	groupID *int64,
	status string,
	startTime, endTime *time.Time,
) ([]*AuditLog, error) {
	
	// 构建查询条件（与 ListAuditLogs 相同，但不分页）
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIdx := 1
	
	if userID != "" {
		whereClause += fmt.Sprintf(" AND user_id = $%d", argIdx)
		args = append(args, userID)
		argIdx++
	}
	
	if username != "" {
		whereClause += fmt.Sprintf(" AND username LIKE $%d", argIdx)
		args = append(args, "%"+username+"%")
		argIdx++
	}
	
	if action != "" {
		whereClause += fmt.Sprintf(" AND action = $%d", argIdx)
		args = append(args, action)
		argIdx++
	}
	
	if resourceType != "" {
		whereClause += fmt.Sprintf(" AND resource_type = $%d", argIdx)
		args = append(args, resourceType)
		argIdx++
	}
	
	if groupID != nil {
		whereClause += fmt.Sprintf(" AND group_id = $%d", argIdx)
		args = append(args, *groupID)
		argIdx++
	}
	
	if status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, status)
		argIdx++
	}
	
	if startTime != nil {
		whereClause += fmt.Sprintf(" AND created_at >= $%d", argIdx)
		args = append(args, *startTime)
		argIdx++
	}
	
	if endTime != nil {
		whereClause += fmt.Sprintf(" AND created_at <= $%d", argIdx)
		args = append(args, *endTime)
		argIdx++
	}
	
	// 查询所有记录（限制最多 10000 条）
	query := fmt.Sprintf(`
		SELECT id, user_id, username, action, resource_type, resource_id, resource_name,
		       group_id, details, ip_address, user_agent, status, error_message, created_at
		FROM audit_logs
		%s
		ORDER BY created_at DESC
		LIMIT 10000
	`, whereClause)
	
	var logs []*AuditLog
	err := l.db.QueryRowsCtx(l.ctx, &logs, query, args...)
	if err != nil {
		l.logger.Errorf("导出审计日志失败: %v", err)
		return nil, fmt.Errorf("导出审计日志失败: %w", err)
	}
	
	return logs, nil
}

// CleanupOldLogs 清理过期日志
func (l *AuditLogic) CleanupOldLogs(retentionDays int) (int64, error) {
	cutoffDate := time.Now().AddDate(0, 0, -retentionDays)
	
	query := `DELETE FROM audit_logs WHERE created_at < $1`
	
	result, err := l.db.ExecCtx(l.ctx, query, cutoffDate)
	if err != nil {
		l.logger.Errorf("清理过期日志失败: %v", err)
		return 0, fmt.Errorf("清理过期日志失败: %w", err)
	}
	
	rows, _ := result.RowsAffected()
	l.logger.Infof("清理过期日志成功: %d 条 (保留 %d 天)", rows, retentionDays)
	return rows, nil
}
