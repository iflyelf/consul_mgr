# API 设计

## 1. 约定

- **Base Path**：`/api`
- **认证**：除 `/health`、`/api/auth/*` 外，均需请求头 `Authorization: Bearer <token>`
- **统一响应**：

```json
{ "code": 200, "message": "success", "data": { } }
```

| code | 含义 |
|------|------|
| 200 | 成功 |
| 400 | 参数错误 |
| 401 | 未认证 / Token 过期 |
| 403 | 无权限 |
| 404 | 资源不存在 |
| 500 | 服务端错误 |

## 2. 认证 `/api/auth`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/auth/config` | 下发前端运行时配置（Casdoor 地址、client_id、回调路径） |
| GET | `/api/auth/login` | 生成 Casdoor 登录地址并 302 跳转（`?format=json` 返回 JSON） |
| GET | `/api/auth/callback` | OAuth 回调，换取 Token 并返回用户信息 |
| GET | `/api/auth/userinfo` | 获取当前登录用户信息 |
| POST | `/api/auth/refresh` | 刷新 Token |
| POST | `/api/auth/logout` | 登出 |

## 3. 服务组 `/api/groups`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/groups` | 列表（支持 `keyword` / `page` / `page_size`） |
| POST | `/api/groups` | 创建（数据中心自动探测，无需传 datacenter） |
| GET | `/api/groups/:id` | 详情（含实例/服务/授权统计） |
| PUT | `/api/groups/:id` | 更新 |
| DELETE | `/api/groups/:id` | 删除 |
| POST | `/api/groups/:id/test` | 测试 Consul 连接 |
| POST | `/api/groups/detect-datacenter` | 根据地址/Token 探测数据中心 |

## 4. 服务 `/api/services`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/services` | 服务列表（`group_id` 必填，`keyword` 可选） |
| GET | `/api/services/detail` | 服务详情（`group_id` + `service`，`keyword` 可过滤实例） |
| DELETE | `/api/services` | 删除整个服务（其下所有实例） |
| POST | `/api/services/batch-delete` | 批量删除服务 |

## 5. 实例 `/api/instances`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/instances` | 列表（`group_id`、`service_name`、`status`、`keyword`、分页） |
| GET | `/api/instances/:id` | 详情 |
| POST | `/api/instances` | 注册实例 |
| PUT | `/api/instances/:id` | 更新实例 |
| DELETE | `/api/instances/:id` | 注销实例（支持集群跨节点） |
| POST | `/api/instances/batch-register` | 批量注册（支持 IP 段/CIDR/范围） |
| POST | `/api/instances/batch-register/preview` | 预览批量注册结果 |
| POST | `/api/instances/batch-delete` | 批量删除 |
| GET | `/api/instances/export` | 导出（`format=json|yaml|csv`） |
| POST | `/api/instances/import` | 导入（`overwrite=true` 强制覆盖） |
| GET | `/api/instances/datacenters` | 数据中心列表 |
| GET | `/api/instances/services` | 服务名列表 |

### 批量注册：IP 表达式

支持以下写法（逗号分隔，可带端口）：

```
10.1.255.24-26:80          # 短范围
10.1.255.38:443            # 单 IP
10.1.255.0/24:8080         # CIDR
10.1.26.5-10.1.26.7        # 完整范围
```

## 6. 权限 `/api/permissions`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET / POST / DELETE | `/api/permissions/users` | 用户授权（查询/授予/撤销） |
| GET / POST / DELETE | `/api/permissions/roles` | 角色授权（查询/授予/撤销） |

## 7. 审计 `/api/audit-logs`

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/api/audit-logs` | 审计日志列表 |
| GET | `/api/audit-logs/export` | 导出审计日志 |

## 8. 健康检查

| 方法 | 路径 | 说明 |
|------|------|------|
| GET | `/health` | 返回 `{"status":"ok",...}`，供容器/K8s 探针使用 |

## 9. 相关文档

- [架构设计](architecture.md) · [数据库设计](database.md)
