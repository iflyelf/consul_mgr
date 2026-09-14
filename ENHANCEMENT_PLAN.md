# Consul Manager 增强改造计划

## 1. 完善 RBAC 权限系统

### 1.1 用户管理 API
- [ ] POST /api/users - 创建用户
- [ ] GET /api/users - 获取用户列表
- [ ] GET /api/users/:id - 获取用户详情
- [ ] PUT /api/users/:id - 更新用户
- [ ] DELETE /api/users/:id - 删除用户
- [ ] PUT /api/users/:id/password - 修改密码
- [ ] POST /api/users/:id/roles - 分配角色

### 1.2 角色管理 API
- [ ] POST /api/roles - 创建角色
- [ ] GET /api/roles - 获取角色列表
- [ ] GET /api/roles/:id - 获取角色详情
- [ ] PUT /api/roles/:id - 更新角色
- [ ] DELETE /api/roles/:id - 删除角色
- [ ] POST /api/roles/:id/permissions - 分配权限

### 1.3 权限管理 API
- [ ] GET /api/permissions - 获取权限列表
- [ ] POST /api/permissions - 创建权限
- [ ] PUT /api/permissions/:id - 更新权限
- [ ] DELETE /api/permissions/:id - 删除权限

### 1.4 权限中间件增强
- [ ] 实现基于资源的权限检查
- [ ] 支持服务组级别权限隔离
- [ ] 添加操作日志记录

## 2. 服务组授权管理

### 2.1 服务组授权 API
- [ ] POST /api/groups/:id/users - 授权用户访问服务组
- [ ] GET /api/groups/:id/users - 获取服务组授权用户列表
- [ ] DELETE /api/groups/:id/users/:user_id - 取消授权
- [ ] GET /api/users/:id/groups - 获取用户可访问的服务组

### 2.2 权限过滤
- [ ] 用户只能看到已授权的服务组
- [ ] 服务/实例操作前检查服务组权限
- [ ] 超级管理员可访问所有服务组

## 3. 前后端集成

### 3.1 前端构建
- [ ] 添加前端构建脚本
- [ ] 配置生产环境 API 地址
- [ ] 优化前端打包大小

### 3.2 Go embed 集成
- [ ] 使用 embed 嵌入 web/dist
- [ ] 添加静态文件服务路由
- [ ] 实现 SPA 路由支持（处理刷新）

### 3.3 Makefile 优化
- [ ] 添加 build-web 任务
- [ ] 添加 build-all 任务（前端+后端）
- [ ] 添加 release 任务

## 4. Consul 混合模式支持

### 4.1 自动模式识别
- [ ] 检测 Consul 是否为集群模式
- [ ] 获取集群节点列表
- [ ] 自动发现数据中心

### 4.2 数据中心管理
- [ ] 自动获取所有数据中心列表
- [ ] 支持跨数据中心查询
- [ ] 数据中心切换功能

### 4.3 集群状态监控
- [ ] 显示集群节点状态
- [ ] Leader 节点识别
- [ ] 节点健康检查

## 5. 数据自动关联

### 5.1 服务组增强
- [ ] 自动获取并保存数据中心列表
- [ ] 定期同步数据中心信息
- [ ] 数据中心变更通知

### 5.2 服务/实例关联
- [ ] 自动关联数据中心信息
- [ ] 显示服务所属数据中心
- [ ] 支持按数据中心筛选

## 实施顺序

Phase 1: 权限系统增强 (优先级: 高)
Phase 2: 服务组授权 (优先级: 高)
Phase 3: 前后端集成 (优先级: 高)
Phase 4: Consul 混合模式 (优先级: 中)
Phase 5: 数据自动关联 (优先级: 中)

