package instance

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/consul/api"
	"github.com/iflyelf/consul_mgr/internal/pkg/consul"
	"github.com/iflyelf/consul_mgr/internal/svc"
	"github.com/iflyelf/consul_mgr/internal/types"
	"gopkg.in/yaml.v3"
)

// getConsulClient 获取 Consul 客户端的辅助函数
func getConsulClient(ctx context.Context, svcCtx *svc.ServiceContext, groupID int64) (*api.Client, error) {
	client, _, err := getConsulClientWithAddr(ctx, svcCtx, groupID)
	return client, err
}

// getConsulClientWithAddr 获取 Consul 客户端及其地址
func getConsulClientWithAddr(ctx context.Context, svcCtx *svc.ServiceContext, groupID int64) (*api.Client, string, error) {
	client, err := svcCtx.GetConsulClient(ctx, groupID)
	if err != nil {
		return nil, "", err
	}
	return client.GetAPIClient(), client.Address(), nil
}
type ListInstancesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewListInstancesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ListInstancesLogic {
	return &ListInstancesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ListInstancesLogic) ListInstances(groupID int64, serviceName, status string) ([]types.ConsulInstanceInfo, error) {
	// 缓存键（超大规模实例列表，避免每次全量拉取 Consul）
	cacheKey := l.svcCtx.CacheKeyInstances(groupID, serviceName, status)
	var cached []types.ConsulInstanceInfo
	if l.svcCtx.Cache.Get(l.ctx, cacheKey, &cached) {
		return cached, nil
	}

	client, addr, err := getConsulClientWithAddr(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return nil, err
	}

	// 确定要查询的服务列表：指定服务则只查该服务，否则查全部
	serviceNames := []string{serviceName}
	if serviceName == "" {
		catalogServices, _, cerr := client.Catalog().Services(nil)
		if cerr != nil {
			return nil, fmt.Errorf("查询服务列表失败: %w", consul.FriendlyError(addr, cerr))
		}
		serviceNames = make([]string, 0, len(catalogServices))
		for name := range catalogServices {
			serviceNames = append(serviceNames, name)
		}
	}

	instances := make([]types.ConsulInstanceInfo, 0)
	for _, name := range serviceNames {
		entries, _, herr := client.Health().Service(name, "", false, nil)
		if herr != nil {
			return nil, fmt.Errorf("查询实例失败: %w", consul.FriendlyError(addr, herr))
		}
		for _, entry := range entries {
			service := entry.Service
			node := entry.Node
			checks := entry.Checks
			healthStatus := aggregateStatus(checks)

			// 状态过滤
			if status != "" && healthStatus != status {
				continue
			}

			instances = append(instances, types.ConsulInstanceInfo{
				ID:           service.ID,
				Service:      service.Service,
				Tags:         service.Tags,
				Meta:         service.Meta,
				Address:      service.Address,
				Port:         service.Port,
				Node:         node.Node,
				NodeAddress:  node.Address,
				HealthStatus: healthStatus,
				Checks:       convertChecks(checks),
			})
		}
	}

	// 写入缓存
	l.svcCtx.Cache.Set(l.ctx, cacheKey, instances)

	return instances, nil
}

// invalidateInstanceCache 失效指定服务组的实例/服务列表/服务详情缓存
//
// 实例变更会影响服务列表的实例数与服务详情的实例集，因此一并清理。
func invalidateInstanceCache(ctx context.Context, svcCtx *svc.ServiceContext, groupID int64) {
	svcCtx.InvalidateInstances(ctx, groupID)
	svcCtx.InvalidateServices(ctx, groupID)
	svcCtx.InvalidateServiceDetails(ctx, groupID)
}

// GetDatacenters 获取该服务组 Consul 的数据中心列表
func (l *ListInstancesLogic) GetDatacenters(groupID int64) ([]string, error) {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return nil, err
	}

	dcs, err := client.Catalog().Datacenters()
	if err != nil {
		return nil, fmt.Errorf("查询数据中心失败: %w", err)
	}
	return dcs, nil
}

// GetServiceNames 获取服务名列表
func (l *ListInstancesLogic) GetServiceNames(groupID int64, datacenter string) ([]string, error) {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return nil, err
	}

	services, _, err := client.Catalog().Services(&api.QueryOptions{Datacenter: datacenter})
	if err != nil {
		return nil, fmt.Errorf("查询服务列表失败: %w", err)
	}

	names := make([]string, 0, len(services))
	for name := range services {
		names = append(names, name)
	}
	return names, nil
}

// GetInstance 获取单个实例详情
func (l *ListInstancesLogic) GetInstance(groupID int64, instanceID string) (*types.ConsulInstanceInfo, error) {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return nil, err
	}

	svc, _, err := client.Agent().Service(instanceID, nil)
	if err != nil {
		return nil, fmt.Errorf("获取实例失败: %w", err)
	}
	if svc == nil {
		return nil, fmt.Errorf("实例不存在: %s", instanceID)
	}

	return &types.ConsulInstanceInfo{
		ID:      svc.ID,
		Service: svc.Service,
		Tags:    svc.Tags,
		Meta:    svc.Meta,
		Address: svc.Address,
		Port:    svc.Port,
	}, nil
}

// RegisterInstanceLogic 注册实例逻辑
type RegisterInstanceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewRegisterInstanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *RegisterInstanceLogic {
	return &RegisterInstanceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *RegisterInstanceLogic) RegisterInstance(groupID int64, req *types.RegisterInstanceRequest) error {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return err
	}

	// 集群去重：清理其他节点上的同名实例，避免一个实例被多个节点注册
	dedupSingle(client, req)

	registration := buildRegistration(req)

	if err := client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("注册实例失败: %w", err)
	}

	invalidateInstanceCache(l.ctx, l.svcCtx, groupID)
	return nil
}

// UpdateInstanceLogic 更新实例逻辑
type UpdateInstanceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewUpdateInstanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *UpdateInstanceLogic {
	return &UpdateInstanceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *UpdateInstanceLogic) UpdateInstance(groupID int64, instanceID string, req *types.UpdateInstanceRequest) error {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return err
	}

	// 先获取现有实例信息（instanceID 必须是服务 ID，不能是服务名）
	service, _, err := client.Agent().Service(instanceID, nil)
	if err != nil {
		return fmt.Errorf("获取实例失败: %w", err)
	}
	if service == nil {
		return fmt.Errorf("实例不存在: %s", instanceID)
	}

	// 更新字段
	if req.Tags != nil {
		service.Tags = req.Tags
	}
	if req.Meta != nil {
		service.Meta = req.Meta
	}
	if req.Address != "" {
		service.Address = req.Address
	}
	if req.Port > 0 {
		service.Port = req.Port
	}

	registration := &api.AgentServiceRegistration{
		ID:      service.ID,
		Name:    service.Service,
		Tags:    service.Tags,
		Meta:    service.Meta,
		Address: service.Address,
		Port:    service.Port,
	}

	// 健康检查：请求指定则替换；否则保留该实例已有的自定义检查
	if req.Check != nil && req.Check.Type != "" {
		reg := buildRegistration(&types.RegisterInstanceRequest{
			ID:      service.ID,
			Service: service.Service,
			Address: service.Address,
			Port:    service.Port,
			Check:   req.Check,
		})
		registration.Check = reg.Check
	} else {
		if existing := findCustomCheck(client, service.ID); existing != nil {
			registration.Check = existing
		}
	}

	if err := client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("更新实例失败: %w", err)
	}

	invalidateInstanceCache(l.ctx, l.svcCtx, groupID)
	return nil
}

// findCustomCheck 查找实例已有的自定义健康检查配置
func findCustomCheck(client *api.Client, serviceID string) *api.AgentServiceCheck {
	checks, err := client.Agent().Checks()
	if err != nil {
		return nil
	}
	for _, c := range checks {
		if c.ServiceID != serviceID {
			continue
		}
		if isBuiltinCheck(c.CheckID, c.Name) {
			continue
		}
		check := &api.AgentServiceCheck{
			CheckID: c.CheckID,
			Name:    c.Name,
			Notes:   c.Notes,
		}
		def := c.Definition
		if def.Interval != 0 {
			check.Interval = (&def.Interval).String()
		}
		if def.Timeout != 0 {
			check.Timeout = (&def.Timeout).String()
		}
		check.HTTP = def.HTTP
		check.TCP = def.TCP
		check.GRPC = def.GRPC
		check.GRPCUseTLS = def.GRPCUseTLS
		return check
	}
	return nil
}

// DeleteInstanceLogic 删除实例逻辑
type DeleteInstanceLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewDeleteInstanceLogic(ctx context.Context, svcCtx *svc.ServiceContext) *DeleteInstanceLogic {
	return &DeleteInstanceLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *DeleteInstanceLogic) DeleteInstance(groupID int64, instanceID string) error {
	client, addr, err := getConsulClientWithAddr(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return err
	}

	if err := consul.DeregisterService(client, instanceID); err != nil {
		return fmt.Errorf("删除实例失败: %w", consul.FriendlyError(addr, err))
	}

	invalidateInstanceCache(l.ctx, l.svcCtx, groupID)
	return nil
}

// BatchDeleteInstancesLogic 批量删除实例逻辑
type BatchDeleteInstancesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBatchDeleteInstancesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchDeleteInstancesLogic {
	return &BatchDeleteInstancesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// BatchDeleteInstances 批量删除实例
//
// 返回:
//   int - 成功数量
//   []string - 失败的实例ID
//   error - 致命错误
func (l *BatchDeleteInstancesLogic) BatchDeleteInstances(groupID int64, instanceIDs []string) (int, []string, error) {
	client, addr, err := getConsulClientWithAddr(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return 0, nil, err
	}

	if len(instanceIDs) == 0 {
		return 0, nil, errors.New("未选择任何实例")
	}

	success := 0
	var failed []string
	for _, instanceID := range instanceIDs {
		if err := consul.DeregisterService(client, instanceID); err != nil {
			failed = append(failed, instanceID)
			continue
		}
		success++
	}

	invalidateInstanceCache(l.ctx, l.svcCtx, groupID)

	if len(failed) > 0 {
		return success, failed, fmt.Errorf("部分删除失败（成功 %d，失败 %d）: %v（地址: %s）",
			success, len(failed), failed, addr)
	}

	return success, nil, nil
}

// ExportInstancesLogic 导出实例逻辑
type ExportInstancesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewExportInstancesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ExportInstancesLogic {
	return &ExportInstancesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

func (l *ExportInstancesLogic) ExportInstances(groupID int64, serviceName, format string) ([]byte, error) {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return nil, err
	}

	// 与列表一致：未指定服务时导出全部服务实例
	serviceNames := []string{serviceName}
	if serviceName == "" {
		catalogServices, _, cerr := client.Catalog().Services(nil)
		if cerr != nil {
			return nil, fmt.Errorf("查询服务列表失败: %w", cerr)
		}
		serviceNames = make([]string, 0, len(catalogServices))
		for name := range catalogServices {
			serviceNames = append(serviceNames, name)
		}
	}

	var instances []types.RegisterInstanceRequest
	for _, name := range serviceNames {
		entries, _, herr := client.Health().Service(name, "", false, nil)
		if herr != nil {
			return nil, fmt.Errorf("查询实例失败: %w", herr)
		}
		for _, entry := range entries {
			service := entry.Service
			instances = append(instances, types.RegisterInstanceRequest{
				GroupID: groupID,
				ID:      service.ID,
				Service: service.Service,
				Tags:    service.Tags,
				Meta:    service.Meta,
				Address: service.Address,
				Port:    service.Port,
			})
		}
	}

	switch format {
	case "json":
		return json.MarshalIndent(instances, "", "  ")
	case "yaml":
		return yaml.Marshal(instances)
	case "csv":
		return exportToCSV(instances)
	default:
		return nil, errors.New("不支持的格式")
	}
}

// ImportInstancesLogic 导入实例逻辑
type ImportInstancesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewImportInstancesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *ImportInstancesLogic {
	return &ImportInstancesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// buildRegistration 由请求构建 Consul 服务注册对象
//
// 说明：ID 为空时使用服务名作为 ID（与 Consul 默认行为一致）。
func buildRegistration(instance *types.RegisterInstanceRequest) *api.AgentServiceRegistration {
	id := instance.ID
	if id == "" {
		id = instance.Service
	}

	registration := &api.AgentServiceRegistration{
		ID:      id,
		Name:    instance.Service,
		Tags:    instance.Tags,
		Meta:    instance.Meta,
		Address: instance.Address,
		Port:    instance.Port,
	}

	// 健康检查
	if instance.Check != nil && instance.Check.Type != "" {
		check := &api.AgentServiceCheck{}
		switch instance.Check.Type {
		case "http":
			check.HTTP = instance.Check.HTTP
		case "tcp":
			check.TCP = instance.Check.TCP
		case "ttl":
			check.TTL = instance.Check.TTL
		case "grpc":
			check.GRPC = instance.Check.GRPC
			check.GRPCUseTLS = instance.Check.GRPCUseTLS
		case "script":
			check.Args = []string{instance.Check.Script}
		}
		// TTL 检查无需 interval/timeout；其余检查 Consul 要求 interval
		if instance.Check.Type != "ttl" {
			interval := instance.Check.Interval
			if interval == "" {
				interval = "10s"
			}
			check.Interval = interval
			timeout := instance.Check.Timeout
			if timeout == "" {
				timeout = "3s"
			}
			check.Timeout = timeout
		}
		registration.Check = check
	}

	return registration
}

// ImportInstances 批量导入实例
//
// 参数:
//   groupID   - 服务组 ID
//   format    - 数据格式（json/yaml/csv）
//   data      - 文件内容
//   overwrite - 是否强制覆盖已存在的实例（false 时跳过已存在项）
//
// 返回:
//   int - 成功数量（含覆盖）
//   int - 跳过数量（已存在且未开启覆盖）
//   int - 失败数量
//   error - 致命错误（解析失败等）
func (l *ImportInstancesLogic) ImportInstances(groupID int64, format string, data []byte, overwrite bool) (int, int, int, error) {
	client, err := getConsulClient(l.ctx, l.svcCtx, groupID)
	if err != nil {
		return 0, 0, 0, err
	}

	var instances []types.RegisterInstanceRequest

	switch format {
	case "json":
		// 兼容两种格式：[{...}] 或 {"instances":[{...}]}
		if err := json.Unmarshal(data, &instances); err != nil {
			var wrapper types.InstanceImportData
			if err2 := json.Unmarshal(data, &wrapper); err2 != nil || len(wrapper.Instances) == 0 {
				return 0, 0, 0, fmt.Errorf("解析 JSON 失败: %w", err)
			}
			instances = wrapper.Instances
		}
	case "yaml":
		if err := yaml.Unmarshal(data, &instances); err != nil {
			return 0, 0, 0, fmt.Errorf("解析 YAML 失败: %w", err)
		}
	case "csv":
		instances, err = importFromCSV(data)
		if err != nil {
			return 0, 0, 0, err
		}
	default:
		return 0, 0, 0, errors.New("不支持的格式")
	}

	if len(instances) == 0 {
		return 0, 0, 0, errors.New("没有可导入的实例")
	}

	// 查询现有实例 ID 集合（用于判断是否存在）
	existing := map[string]bool{}
	if all, aerr := client.Agent().Services(); aerr == nil {
		for id := range all {
			existing[id] = true
		}
	}

	// 集群去重：清理其他节点上的同名实例
	_, _ = dedupClusterRegistrations(client, instances)

	success, skipped, failed := 0, 0, 0
	var errs []string
	seen := map[string]bool{}
	for i := range instances {
		ins := &instances[i]
		registration := buildRegistration(ins)

		// 同一批次内重复 ID 检测
		if seen[registration.ID] {
			skipped++
			continue
		}
		seen[registration.ID] = true

		// 已存在且未开启覆盖：跳过
		if existing[registration.ID] && !overwrite {
			skipped++
			continue
		}

		if err := client.Agent().ServiceRegister(registration); err != nil {
			failed++
			errs = append(errs, fmt.Sprintf("%s: %v", registration.ID, err))
			continue
		}
		success++
	}

	invalidateInstanceCache(l.ctx, l.svcCtx, groupID)

	if failed > 0 {
		return success, skipped, failed, fmt.Errorf("部分导入失败(成功 %d, 跳过 %d, 失败 %d): %s",
			success, skipped, failed, strings.Join(errs, "; "))
	}

	return success, skipped, failed, nil
}

// 辅助函数
func aggregateStatus(checks []*api.HealthCheck) string {
	status := "passing"
	for _, check := range checks {
		if check.Status == api.HealthCritical {
			return "critical"
		}
		if check.Status == api.HealthWarning {
			status = "warning"
		}
	}
	return status
}

func convertChecks(checks []*api.HealthCheck) []map[string]interface{} {
	var result []map[string]interface{}
	for _, check := range checks {
		entry := map[string]interface{}{
			"check_id": check.CheckID,
			"name":     check.Name,
			"status":   check.Status,
			"notes":    check.Notes,
			"output":   check.Output,
		}
		// 内置检查（serfHealth 等）标记，便于前端区分
		entry["service_id"] = check.ServiceID
		entry["builtin"] = isBuiltinCheck(check.CheckID, check.Name)

		// 附带检查定义（HTTP/TCP/TTL/GRPC/间隔/超时等）
		def := check.Definition
		if check.Type != "" {
			entry["type"] = check.Type
		}
		if def.HTTP != "" {
			entry["http"] = def.HTTP
		}
		if def.TCP != "" {
			entry["tcp"] = def.TCP
		}
		if def.GRPC != "" {
			entry["grpc"] = def.GRPC
		}
		if def.Interval != 0 {
			entry["interval"] = (&def.Interval).String()
		}
		if def.Timeout != 0 {
			entry["timeout"] = (&def.Timeout).String()
		}
		result = append(result, entry)
	}
	return result
}

// isBuiltinCheck 判断是否为 Consul 内置检查
//
// Consul 会为每个服务自动附加 serfHealth 检查，它不代表用户配置的健康检查。
func isBuiltinCheck(checkID, name string) bool {
	if checkID == "serfHealth" || checkID == "_node_maintenance" || checkID == "_service_maintenance" {
		return true
	}
	if name == "Serf Health Status" || name == "Node Maintenance Mode" {
		return true
	}
	return false
}

// HasCustomCheck 判断检查列表中是否存在用户自定义健康检查
func HasCustomCheck(checks []map[string]interface{}) bool {
	for _, c := range checks {
		if b, ok := c["builtin"].(bool); ok && b {
			continue
		}
		return true
	}
	return false
}

// exportToCSV 将实例列表导出为 CSV
//
// 列：id, service, address, port, tags, meta, check_type, check_target, check_interval, check_timeout
// tags 用 "|" 分隔；meta 用 JSON 字符串。
func exportToCSV(instances []types.RegisterInstanceRequest) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString("id,service,address,port,tags,meta,check_type,check_target,check_interval,check_timeout\n")

	for _, ins := range instances {
		metaJSON, _ := json.Marshal(ins.Meta)

		checkType, checkTarget, interval, timeout := "", "", "", ""
		if ins.Check != nil {
			checkType = ins.Check.Type
			switch ins.Check.Type {
			case "http":
				checkTarget = ins.Check.HTTP
			case "tcp":
				checkTarget = ins.Check.TCP
			case "ttl":
				checkTarget = ins.Check.TTL
			case "grpc":
				checkTarget = ins.Check.GRPC
			case "script":
				checkTarget = ins.Check.Script
			}
			interval = ins.Check.Interval
			timeout = ins.Check.Timeout
		}

		record := []string{
			ins.ID,
			ins.Service,
			ins.Address,
			strconv.Itoa(ins.Port),
			strings.Join(ins.Tags, "|"),
			string(metaJSON),
			checkType,
			checkTarget,
			interval,
			timeout,
		}
		for i, v := range record {
			record[i] = csvEscape(v)
		}
		buf.WriteString(strings.Join(record, ",") + "\n")
	}

	return buf.Bytes(), nil
}

// importFromCSV 从 CSV 解析实例列表
func importFromCSV(data []byte) ([]types.RegisterInstanceRequest, error) {
	reader := csv.NewReader(bytes.NewReader(data))
	reader.FieldsPerRecord = -1
	reader.TrimLeadingSpace = true

	rows, err := reader.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("解析 CSV 失败: %w", err)
	}
	if len(rows) < 2 {
		return nil, errors.New("CSV 内容为空或缺少数据行")
	}

	// 构建表头索引
	header := make(map[string]int)
	for i, h := range rows[0] {
		header[strings.ToLower(strings.TrimSpace(h))] = i
	}
	get := func(row []string, key string) string {
		if idx, ok := header[key]; ok && idx < len(row) {
			return strings.TrimSpace(row[idx])
		}
		return ""
	}

	var instances []types.RegisterInstanceRequest
	for _, row := range rows[1:] {
		// 跳过空行
		joined := strings.TrimSpace(strings.Join(row, ""))
		if joined == "" {
			continue
		}

		service := get(row, "service")
		address := get(row, "address")
		// 跳过无效行（如缺少服务名或地址），不中断整体导入
		if service == "" || address == "" {
			continue
		}

		port, _ := strconv.Atoi(get(row, "port"))
		if port == 0 {
			port = 0
		}

		ins := types.RegisterInstanceRequest{
			ID:      get(row, "id"),
			Service: service,
			Address: address,
			Port:    port,
		}

		// tags：优先 "|" 分隔，兼容 "," 分隔
		if tags := get(row, "tags"); tags != "" {
			sep := "|"
			if !strings.Contains(tags, "|") && strings.Contains(tags, ",") {
				sep = ","
			}
			for _, t := range strings.Split(tags, sep) {
				if t = strings.TrimSpace(t); t != "" {
					ins.Tags = append(ins.Tags, t)
				}
			}
		}

		// meta：JSON 字符串
		if meta := get(row, "meta"); meta != "" {
			m := map[string]string{}
			if jerr := json.Unmarshal([]byte(meta), &m); jerr == nil {
				ins.Meta = m
			}
		}

		// 健康检查
		if ct := get(row, "check_type"); ct != "" {
			check := &types.HealthCheckConfig{
				Type:     ct,
				Interval: get(row, "check_interval"),
				Timeout:  get(row, "check_timeout"),
			}
			target := get(row, "check_target")
			switch ct {
			case "http":
				check.HTTP = target
			case "tcp":
				check.TCP = target
			case "ttl":
				check.TTL = target
			case "grpc":
				check.GRPC = target
			case "script":
				check.Script = target
			}
			ins.Check = check
		}

		instances = append(instances, ins)
	}

	if len(instances) == 0 {
		return nil, errors.New("CSV 中没有有效的实例数据")
	}
	return instances, nil
}

// csvEscape CSV 字段转义
func csvEscape(s string) string {
	if strings.ContainsAny(s, ",\"\n\r") {
		return "\"" + strings.ReplaceAll(s, "\"", "\"\"") + "\""
	}
	return s
}

// BatchRegisterInstancesLogic 批量注册实例逻辑
type BatchRegisterInstancesLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
}

func NewBatchRegisterInstancesLogic(ctx context.Context, svcCtx *svc.ServiceContext) *BatchRegisterInstancesLogic {
	return &BatchRegisterInstancesLogic{
		ctx:    ctx,
		svcCtx: svcCtx,
	}
}

// BatchRegisterInstances 按 IP:端口 表达式批量注册实例
//
// 支持 "10.1.255.24-26:80,10.1.255.38:443,10.1.255.0/24:8080" 形式。
//
// 返回:
//   int - 成功数量
//   int - 跳过数量（已存在且未开启覆盖）
//   int - 失败数量
//   []string - 展开后的实例 ID 列表
//   error - 致命错误
func (l *BatchRegisterInstancesLogic) BatchRegisterInstances(req *types.BatchRegisterRequest) (int, int, int, []string, error) {
	client, err := getConsulClient(l.ctx, l.svcCtx, req.GroupID)
	if err != nil {
		return 0, 0, 0, nil, err
	}

	// 构建目标实例集合
	targets, err := buildBatchTargets(req)
	if err != nil {
		return 0, 0, 0, nil, err
	}

	ids := make([]string, 0, len(targets))
	for i := range targets {
		ids = append(ids, targets[i].ID)
	}

	// 集群去重：清理其他节点上的同名实例
	_, _ = dedupClusterRegistrations(client, targets)

	// 本地已存在实例集合（用于跳过/覆盖判断）
	existing := map[string]bool{}
	if all, aerr := client.Agent().Services(); aerr == nil {
		for id := range all {
			existing[id] = true
		}
	}

	success, skipped, failed := 0, 0, 0
	var errs []string

	for i := range targets {
		ins := &targets[i]

		if existing[ins.ID] && !req.Overwrite {
			skipped++
			continue
		}

		registration := buildRegistration(ins)
		if rerr := client.Agent().ServiceRegister(registration); rerr != nil {
			failed++
			errs = append(errs, fmt.Sprintf("%s: %v", ins.ID, rerr))
			continue
		}
		success++
	}

	invalidateInstanceCache(l.ctx, l.svcCtx, req.GroupID)

	if failed > 0 {
		return success, skipped, failed, ids,
			fmt.Errorf("部分注册失败(成功 %d, 跳过 %d, 失败 %d): %s", success, skipped, failed, strings.Join(errs, "; "))
	}

	return success, skipped, failed, ids, nil
}

// buildBatchTargets 根据批量注册请求构建目标实例列表
func buildBatchTargets(req *types.BatchRegisterRequest) ([]types.RegisterInstanceRequest, error) {
	items, err := ParseIPPorts(req.Instances)
	if err != nil {
		return nil, err
	}

	prefix := req.IDPrefix
	if prefix == "" {
		prefix = req.Service
	}

	targets := make([]types.RegisterInstanceRequest, 0, len(items))
	for _, item := range items {
		address := item
		port := req.DefaultPort
		if idx := strings.LastIndex(item, ":"); idx >= 0 {
			address = item[:idx]
			if p, e := strconv.Atoi(item[idx+1:]); e == nil {
				port = p
			}
		}

		// 生成实例 ID：前缀 + IP + 端口
		id := prefix + "_" + strings.ReplaceAll(address, ".", "_")
		if port > 0 {
			id = id + "_" + strconv.Itoa(port)
		}

		// 复制 Meta，避免多个实例共享同一个 map
		meta := map[string]string{}
		for k, v := range req.Meta {
			meta[k] = v
		}
		if _, ok := meta["instance"]; !ok {
			meta["instance"] = item
		}

		targets = append(targets, types.RegisterInstanceRequest{
			GroupID: req.GroupID,
			ID:      id,
			Service: req.Service,
			Address: address,
			Port:    port,
			Tags:    req.Tags,
			Meta:    meta,
			Check:   req.Check,
		})
	}
	return targets, nil
}

// PreviewBatchRegister 预览批量注册将生成的实例列表（不实际注册）
func (l *BatchRegisterInstancesLogic) PreviewBatchRegister(req *types.BatchRegisterRequest) ([]string, error) {
	targets, err := buildBatchTargets(req)
	if err != nil {
		return nil, err
	}
	result := make([]string, 0, len(targets))
	for i := range targets {
		result = append(result, targets[i].ID)
	}
	return result, nil
}
