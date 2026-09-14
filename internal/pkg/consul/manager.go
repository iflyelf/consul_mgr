package consul

import (
	"sync"
	"time"
)

// Manager Consul 集群管理器
type Manager struct {
	clients map[int64]*Client // key: service_group_id
	mu      sync.RWMutex
}

// NewManager 创建 Consul 管理器
func NewManager() *Manager {
	return &Manager{
		clients: make(map[int64]*Client),
	}
}

// GetClient 根据服务组 ID 获取 Consul 客户端
func (m *Manager) GetClient(groupID int64, cfg *Config) (*Client, error) {
	m.mu.RLock()
	client, exists := m.clients[groupID]
	m.mu.RUnlock()
	
	if exists {
		return client, nil
	}
	
	// 创建新客户端
	m.mu.Lock()
	defer m.mu.Unlock()
	
	// 双重检查
	if client, exists := m.clients[groupID]; exists {
		return client, nil
	}
	
	client, err := NewClient(cfg)
	if err != nil {
		return nil, err
	}
	
	m.clients[groupID] = client
	return client, nil
}

// RemoveClient 移除客户端
func (m *Manager) RemoveClient(groupID int64) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.clients, groupID)
}

// TestConnection 测试指定服务组的连接
func (m *Manager) TestConnection(groupID int64, cfg *Config) error {
	client, err := NewClient(cfg)
	if err != nil {
		return err
	}
	
	if err := client.TestConnection(); err != nil {
		return err
	}
	
	// 测试成功后缓存客户端
	m.mu.Lock()
	m.clients[groupID] = client
	m.mu.Unlock()
	
	return nil
}

// GetOrCreateClient 获取或创建客户端
func (m *Manager) GetOrCreateClient(groupID int64, address, token, datacenter string, timeout int) (*Client, error) {
	m.mu.RLock()
	client, exists := m.clients[groupID]
	m.mu.RUnlock()
	
	if exists {
		return client, nil
	}
	
	cfg := &Config{
		Address:    address,
		Token:      token,
		Datacenter: datacenter,
	}
	
	if timeout > 0 {
		cfg.Timeout = time.Duration(timeout) * time.Second
	}
	
	return m.GetClient(groupID, cfg)
}

// ClearCache 清除所有缓存的客户端
func (m *Manager) ClearCache() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients = make(map[int64]*Client)
}

// GetCachedClients 获取所有缓存的客户端数量
func (m *Manager) GetCachedClients() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}
