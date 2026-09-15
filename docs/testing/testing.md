# 测试文档

## 1. 测试策略

| 层级 | 范围 | 工具 |
|------|------|------|
| 单元测试 | IP 解析、工具函数 | `go test` |
| 接口测试 | 全部 REST API | Playwright / curl |
| 端到端测试 | 真实浏览器操作 + 真实 Consul | Playwright |

## 2. 单元测试

```bash
# 全部单元测试
go test ./...

# 带覆盖率
go test -cover ./...

# 单个包
go test ./internal/logic/instance/ -v
```

已覆盖的关键逻辑：
- `internal/logic/instance/ip_parser_test.go` — IP 段 / CIDR / 范围 / 端口解析

## 3. 编译检查

```bash
go vet ./internal/pkg/consul/ ./internal/logic/... ./internal/handler/...
go build ./...
```

## 4. 端到端测试（推荐）

使用真实 Consul 与浏览器验证完整业务闭环。

### 4.1 准备

```bash
# 启动测试用 Consul
docker run -d --name consul-test -p 8500:8500 \
  hashicorp/consul:latest agent -dev -client 0.0.0.0

# 启动被测服务（指向测试 Consul）
export CONSUL_ADDRESS="http://localhost:8500"
./consul_mgr -c etc/config.yaml
```

### 4.2 测试清单

**服务组**
- [x] 新增服务组（数据中心自动探测，无需手填）
- [x] 编辑 / 删除服务组
- [x] 连接测试

**Services**
- [x] 列表加载、关键词搜索
- [x] 健康状态总览（总数 / 健康 / 异常 / 健康度）
- [x] 删除服务、批量删除

**Instances**
- [x] 列表（服务组 / 服务名 / 状态 / 关键字）
- [x] 注册 / 编辑（Tags、Meta、健康检查）/ 删除
- [x] 批量注册（IP 段 / CIDR / 范围 / IP:端口）
- [x] 导出（JSON / YAML / CSV）
- [x] 导入（含强制覆盖）
- [x] 批量删除

**集群场景**
- [x] 跨节点实例删除（其他节点注册的实例）
- [x] 集群注册去重（同一 ServiceID 不重复出现在多节点）

**认证**
- [x] 未登录跳转登录页
- [x] Casdoor 登录 / 回调 / 登出
- [x] Token 过期处理

**UI**
- [x] 三套主题切换与持久化
- [x] H5 自适应（390px 无横向溢出）
- [x] Mac 圆角风格一致性

## 5. 多节点集群测试

```bash
# 启动两节点集群
docker network create cnet
docker run -d --name cn1 --network cnet -p 8611:8500 \
  hashicorp/consul:latest agent -server -bootstrap-expect=1 -node=n1 -client=0.0.0.0 -datacenter=dc1
N1IP=$(docker inspect -f '{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}' cn1)
docker run -d --name cn2 --network cnet -p 8612:8500 \
  hashicorp/consul:latest agent -node=n2 -client=0.0.0.0 -datacenter=dc1 -retry-join=$N1IP
```

验证要点：
1. 在 n2 注册实例 `X`，通过平台（指向 n1）删除 `X` → 集群中应消失
2. 在 n2 注册实例 `X`，通过平台注册同名 `X` → 集群中应只剩 n1 一条

## 6. 回归验证脚本

前端构建后建议执行一次完整回归：

```bash
cd web && npm run build && cd ..
go build -o /tmp/consul_mgr ./cmd/api
# 启动后运行端到端脚本（Playwright）
```

## 7. 相关文档

- [开发文档](../development/development.md) · [部署文档](../deployment/systemd.md)
