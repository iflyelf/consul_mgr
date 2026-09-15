package consul

import (
	"fmt"
	"net/http"
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

// DeregisterService 注销服务实例
func (c *Client) DeregisterService(serviceID string) error {
	err := c.client.Agent().ServiceDeregister(serviceID)
	if err != nil {
		return fmt.Errorf("注销服务失败: %w", err)
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
