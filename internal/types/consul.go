package types

// ConsulServiceInfo Consul 服务信息
type ConsulServiceInfo struct {
	ID             string            `json:"id"`
	Service        string            `json:"service"`
	Tags           []string          `json:"tags"`
	Meta           map[string]string `json:"meta"`
	Address        string            `json:"address"`
	Port           int               `json:"port"`
	Datacenter     string            `json:"datacenter"`
	HealthStatus   string            `json:"health_status"` // passing, warning, critical
	InstanceCount  int               `json:"instance_count"`
	HealthyCount   int               `json:"healthy_count"`
	UnhealthyCount int               `json:"unhealthy_count"`
}

// ConsulInstanceInfo Consul 实例信息
type ConsulInstanceInfo struct {
	ID           string                   `json:"id"`
	Service      string                   `json:"service"`
	Tags         []string                 `json:"tags"`
	Meta         map[string]string        `json:"meta"`
	Address      string                   `json:"address"`
	Port         int                      `json:"port"`
	Node         string                   `json:"node"`
	NodeAddress  string                   `json:"node_address"`
	Datacenter   string                   `json:"datacenter"`
	HealthStatus string                   `json:"health_status"`
	Checks       []map[string]interface{} `json:"checks"`
}

// ServiceDetail 服务详情
type ServiceDetail struct {
	ServiceName string               `json:"service_name"`
	Instances   []ConsulInstanceInfo `json:"instances"`
}

// ConsulHealthCheck 健康检查信息
type ConsulHealthCheck struct {
	CheckID     string `json:"check_id"`
	Name        string `json:"name"`
	Status      string `json:"status"`
	Notes       string `json:"notes"`
	Output      string `json:"output"`
	ServiceID   string `json:"service_id"`
	ServiceName string `json:"service_name"`
}

// RegisterInstanceRequest 注册实例请求
type RegisterInstanceRequest struct {
	GroupID int64              `json:"group_id" validate:"required"`
	Service string             `json:"service" validate:"required"`
	ID      string             `json:"id,optional"`
	Tags    []string           `json:"tags,optional"`
	Meta    map[string]string  `json:"meta,optional"`
	Address string             `json:"address" validate:"required"`
	Port    int                `json:"port" validate:"required,min=1,max=65535"`
	Check   *HealthCheckConfig `json:"check,optional"`
}

// UpdateInstanceRequest 更新实例请求
type UpdateInstanceRequest struct {
	Tags    []string           `json:"tags,optional"`
	Meta    map[string]string  `json:"meta,optional"`
	Address string             `json:"address,optional"`
	Port    int                `json:"port,optional" validate:"omitempty,min=1,max=65535"`
	Check   *HealthCheckConfig `json:"check,optional"`
}

// HealthCheckConfig 健康检查配置
type HealthCheckConfig struct {
	Type       string              `json:"type" validate:"required,oneof=http tcp ttl script grpc"` // http, tcp, ttl, script, grpc
	HTTP       string              `json:"http,optional"`
	TCP        string              `json:"tcp,optional"`
	Interval   string              `json:"interval,optional"`
	Timeout    string              `json:"timeout,optional"`
	TTL        string              `json:"ttl,optional"`
	Script     string              `json:"script,optional"`
	GRPC       string              `json:"grpc,optional"`
	GRPCUseTLS bool                `json:"grpc_use_tls,optional"`
	Method     string              `json:"method,optional"`
	Header     map[string][]string `json:"header,optional"`
	Body       string              `json:"body,optional"`
}

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	GroupID int64    `json:"group_id" validate:"required"`
	IDs     []string `json:"ids" validate:"required,min=1"`
}

// BatchRegisterRequest 批量注册请求
//
// 支持 IP:端口 表达式批量展开注册（IP段/CIDR/范围/IP:端口）
type BatchRegisterRequest struct {
	GroupID int64 `json:"group_id" validate:"required"`
	// 服务名称
	Service string `json:"service" validate:"required"`
	// IP:端口 表达式，如 "10.1.255.24-26:80,10.1.255.38:443,10.1.255.0/24:8080"
	Instances string `json:"instances" validate:"required"`
	// 服务 ID 前缀（为空时使用服务名）
	IDPrefix string `json:"id_prefix,optional"`
	// 实例默认端口（表达式未指定端口时使用）
	DefaultPort int `json:"default_port,optional"`
	// 端口覆盖（表达式未指定端口时优先使用）
	Tags      []string           `json:"tags,optional"`
	Meta      map[string]string  `json:"meta,optional"`
	Check     *HealthCheckConfig `json:"check,optional"`
	Overwrite bool               `json:"overwrite,optional"`
}

// ImportInstancesRequest 批量导入请求
type ImportInstancesRequest struct {
	GroupID   int64  `json:"group_id" validate:"required"`
	Format    string `json:"format" validate:"required,oneof=json yaml csv"`
	Data      string `json:"data" validate:"required"`
	OverWrite bool   `json:"overwrite,omitempty"` // 是否覆盖已存在的实例
}

// ExportInstancesRequest 批量导出请求
type ExportInstancesRequest struct {
	GroupID int64  `form:"group_id" validate:"required"`
	Service string `form:"service,optional"`
	Format  string `form:"format,default=json" validate:"oneof=json yaml csv"`
}

// InstanceImportData 导入数据结构
type InstanceImportData struct {
	Instances []RegisterInstanceRequest `json:"instances"`
}

// ServiceHealthSummary 服务健康状态汇总
type ServiceHealthSummary struct {
	Service        string `json:"service"`
	TotalInstances int    `json:"total_instances"`
	Passing        int    `json:"passing"`
	Warning        int    `json:"warning"`
	Critical       int    `json:"critical"`
}

// QueryServicesRequest 查询服务请求
type QueryServicesRequest struct {
	GroupID int64  `form:"group_id,optional"`
	Keyword string `form:"keyword,optional"`
}

// QueryInstancesRequest 查询实例请求
type QueryInstancesRequest struct {
	GroupID int64  `form:"group_id,optional"`
	Service string `form:"service,optional"`
	Status  string `form:"status,optional"` // passing, warning, critical
}
