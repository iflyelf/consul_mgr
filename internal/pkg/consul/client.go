package consul

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/consul/api"
)

// Client Consul 客户端封装
type Client struct {
	client     *api.Client
	config     *Config
	address    string
	token      string
	datacenter string
}

// Config Consul 配置
type Config struct {
	Address    string
	Token      string
	Datacenter string
	Timeout    time.Duration
}

// NewClient 创建 Consul 客户端
func NewClient(cfg *Config) (*Client, error) {
	config := api.DefaultConfig()
	config.Address = cfg.Address
	config.Token = cfg.Token
	config.Datacenter = cfg.Datacenter

	// 关键：Consul 通常部署在内网，必须禁用 HTTP 代理，
	// 否则 HTTP_PROXY 环境变量会导致请求被代理拦截而超时。
	timeout := 10 * time.Second
	if cfg.Timeout > 0 {
		timeout = cfg.Timeout
	}
	config.HttpClient = &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			Proxy: nil, // 不使用代理
			// 连接池：并发批量查询各服务时，默认 MaxIdleConnsPerHost=2
			// 会导致大量连接反复新建，显著拖慢。这里放宽以提升复用率。
			MaxIdleConns:        200,
			MaxIdleConnsPerHost: 64,
			MaxConnsPerHost:     0, // 不限制总连接数
			IdleConnTimeout:     90 * time.Second,
		},
	}

	client, err := api.NewClient(config)
	if err != nil {
		return nil, fmt.Errorf("创建 Consul 客户端失败: %w", err)
	}

	return &Client{
		client:     client,
		config:     cfg,
		address:    cfg.Address,
		token:      cfg.Token,
		datacenter: cfg.Datacenter,
	}, nil
}

// GetAPIClient 获取原生 API Client
func (c *Client) GetAPIClient() *api.Client {
	return c.client
}

// Address 返回客户端配置的 Consul 地址
func (c *Client) Address() string {
	return c.address
}

// DetectDatacenter 自动检测 Consul 的数据中心
//
// 返回:
//   string - 数据中心名称
//   string - 节点名称
//   error - 错误信息
func (c *Client) DetectDatacenter() (string, string, error) {
	self, err := c.client.Agent().Self()
	if err != nil {
		return "", "", fmt.Errorf("获取 Consul 信息失败: %w", err)
	}

	datacenter := ""
	nodeName := ""
	if cfg := self["Config"]; cfg != nil {
		if v, ok := cfg["Datacenter"].(string); ok {
			datacenter = v
		}
		if v, ok := cfg["NodeName"].(string); ok {
			nodeName = v
		}
	}

	if datacenter == "" {
		return "", "", fmt.Errorf("无法检测到数据中心")
	}

	return datacenter, nodeName, nil
}

// TestConnection 测试连接
func (c *Client) TestConnection() error {
	_, err := c.client.Agent().Self()
	if err != nil {
		return fmt.Errorf("连接测试失败: %w", err)
	}
	return nil
}

// GetServices 获取所有服务列表
func (c *Client) GetServices() (map[string][]string, error) {
	services, _, err := c.client.Catalog().Services(nil)
	if err != nil {
		return nil, fmt.Errorf("获取服务列表失败: %w", err)
	}
	return services, nil
}

// GetServiceInstances 获取服务的所有实例
func (c *Client) GetServiceInstances(serviceName string) ([]*api.CatalogService, error) {
	services, _, err := c.client.Catalog().Service(serviceName, "", nil)
	if err != nil {
		return nil, fmt.Errorf("获取服务实例失败: %w", err)
	}
	return services, nil
}

// GetServiceHealth 获取服务健康状态
func (c *Client) GetServiceHealth(serviceName string) ([]*api.ServiceEntry, error) {
	entries, _, err := c.client.Health().Service(serviceName, "", false, nil)
	if err != nil {
		return nil, fmt.Errorf("获取服务健康状态失败: %w", err)
	}
	return entries, nil
}

// RegisterService 注册服务实例
func (c *Client) RegisterService(registration *api.AgentServiceRegistration) error {
	err := c.client.Agent().ServiceRegister(registration)
	if err != nil {
		return fmt.Errorf("注册服务失败: %w", err)
	}
	return nil
}

// DeregisterService 注销服务实例（支持集群）
//
// 说明:
//   Consul 的 Agent.ServiceDeregister 只能注销「本 Agent 注册」的服务。
//   集群中若服务注册在其他节点，本节点 Agent 会返回 404 Unknown service ID。
//   此时回退到 Catalog.Deregister（可按节点注销集群中任意实例）。
func (c *Client) DeregisterService(serviceID string) error {
	return DeregisterService(c.client, serviceID)
}

// DeregisterService 注销服务实例（包级函数，支持集群回退）
func DeregisterService(client *api.Client, serviceID string) error {
	err := client.Agent().ServiceDeregister(serviceID)
	if err == nil {
		return nil
	}

	// 仅当「本 Agent 不认识该服务」时才走 Catalog 回退
	if !isUnknownServiceErr(err) {
		return fmt.Errorf("注销服务失败: %w", err)
	}

	if cerr := DeregisterFromCatalog(client, serviceID); cerr != nil {
		return fmt.Errorf("注销服务失败: %w", cerr)
	}
	return nil
}

// isUnknownServiceErr 判断是否为「未知服务ID」错误
func isUnknownServiceErr(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "Unknown service ID") ||
		strings.Contains(msg, "Unexpected response code: 404")
}

// DeregisterFromCatalog 通过 Catalog 在集群范围内注销实例
//
// 扫描所有节点，删除匹配 ServiceID 的条目。
func DeregisterFromCatalog(client *api.Client, serviceID string) error {
	// 先按服务名定位（ServiceID 可能与服务名不同，需回退全量扫描）
	entries, _, err := client.Catalog().Service(serviceID, "", nil)
	if err != nil || len(entries) == 0 {
		all, _, aerr := client.Catalog().Services(nil)
		if aerr != nil {
			if err != nil {
				return err
			}
			return aerr
		}
		entries = nil
		for name := range all {
			es, _, eerr := client.Catalog().Service(name, "", nil)
			if eerr != nil {
				continue
			}
			for _, e := range es {
				if e.ServiceID == serviceID {
					entries = append(entries, e)
				}
			}
		}
	}

	if len(entries) == 0 {
		return fmt.Errorf("集群中未找到实例: %s", serviceID)
	}

	var lastErr error
	for _, e := range entries {
		if _, derr := client.Catalog().Deregister(&api.CatalogDeregistration{
			Node:      e.Node,
			ServiceID: e.ServiceID,
		}, nil); derr != nil {
			lastErr = derr
		}
	}
	return lastErr
}

// DeregisterRemote 按节点注销实例（供集群去重等场景使用）
func DeregisterRemote(client *api.Client, node, serviceID string) error {
	_, err := client.Catalog().Deregister(&api.CatalogDeregistration{
		Node:      node,
		ServiceID: serviceID,
	}, nil)
	if err != nil {
		return fmt.Errorf("按节点注销失败 %s@%s: %w", serviceID, node, err)
	}
	return nil
}

// GetServiceDetail 获取服务详情（包含健康检查信息）
func (c *Client) GetServiceDetail(serviceID string) (*api.AgentService, error) {
	service, _, err := c.client.Agent().Service(serviceID, nil)
	if err != nil {
		return nil, fmt.Errorf("获取服务详情失败: %w", err)
	}
	return service, nil
}

// GetChecks 获取健康检查列表
func (c *Client) GetChecks() (map[string]*api.AgentCheck, error) {
	checks, err := c.client.Agent().Checks()
	if err != nil {
		return nil, fmt.Errorf("获取健康检查列表失败: %w", err)
	}
	return checks, nil
}

// GetAgentServices 获取代理上的所有服务
func (c *Client) GetAgentServices() (map[string]*api.AgentService, error) {
	services, err := c.client.Agent().Services()
	if err != nil {
		return nil, fmt.Errorf("获取代理服务列表失败: %w", err)
	}
	return services, nil
}

// UpdateServiceTags 更新服务标签
func (c *Client) UpdateServiceTags(serviceID string, tags []string) error {
	// Consul 不支持直接更新标签，需要重新注册
	service, err := c.GetServiceDetail(serviceID)
	if err != nil {
		return err
	}
	
	registration := &api.AgentServiceRegistration{
		ID:      service.ID,
		Name:    service.Service,
		Tags:    tags,
		Port:    service.Port,
		Address: service.Address,
		Meta:    service.Meta,
	}
	
	return c.RegisterService(registration)
}

// UpdateServiceMeta 更新服务元数据
func (c *Client) UpdateServiceMeta(serviceID string, meta map[string]string) error {
	service, err := c.GetServiceDetail(serviceID)
	if err != nil {
		return err
	}
	
	registration := &api.AgentServiceRegistration{
		ID:      service.ID,
		Name:    service.Service,
		Tags:    service.Tags,
		Port:    service.Port,
		Address: service.Address,
		Meta:    meta,
	}
	
	return c.RegisterService(registration)
}

// FriendlyError 将底层网络/Consul 错误转换为面向用户的友好提示
//
// 参数:
//   address - Consul 地址
//   err     - 原始错误
//
// 返回:
//   error - 友好错误（含原始信息）
func FriendlyError(address string, err error) error {
	if err == nil {
		return nil
	}
	msg := err.Error()

	switch {
	case strings.Contains(msg, "context deadline exceeded"),
		strings.Contains(msg, "Client.Timeout exceeded"),
		strings.Contains(msg, "i/o timeout"):
		return fmt.Errorf("连接 Consul 超时（地址: %s），请检查该地址是否可达、网络是否通、Token 是否正确", address)
	case strings.Contains(msg, "connection refused"):
		return fmt.Errorf("无法连接 Consul（地址: %s），连接被拒绝，请确认 Consul 是否已启动、端口是否正确", address)
	case strings.Contains(msg, "no such host"):
		return fmt.Errorf("Consul 域名无法解析（地址: %s），请检查地址是否正确", address)
	case strings.Contains(msg, "no route to host"):
		return fmt.Errorf("无法路由到 Consul（地址: %s），请检查网络连通性", address)
	case strings.Contains(msg, "permission denied"),
		strings.Contains(msg, "403"):
		return fmt.Errorf("访问 Consul 被拒绝（地址: %s），Token 权限不足或无效", address)
	case strings.Contains(msg, "Unknown service ID"):
		return fmt.Errorf("Consul 中不存在该实例（地址: %s），可能已被注销或不属于该 Consul 节点", address)
	}

	return fmt.Errorf("%s（地址: %s）", msg, address)
}
