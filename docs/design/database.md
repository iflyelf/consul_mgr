# 数据库设计

## 1. 概述

- **数据库**：PostgreSQL
- **建表方式**：应用启动时自动执行 `CREATE TABLE IF NOT EXISTS`（幂等，见 `internal/svc/service_context.go`），无需手工执行 SQL
- **认证**：用户体系由外部 [FlyIAM](https://github.com/iflyelf/flyiam)（内置 Casdoor）管理，本项目不创建用户密码表
- **参考快照**：`deploy/sql/schema.sql`（人工建库/审计参考，运行时以内嵌 SQL 为准）

## 2. 表结构

### 2.1 service_groups（服务组）

一个服务组 = 一个 Consul 集群连接配置。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 主键 |
| name | VARCHAR(100) UNIQUE | 服务组名称 |
| code | VARCHAR(100) | 服务组代码 |
| consul_address | VARCHAR(255) | Consul 地址 |
| consul_token | VARCHAR(255) | Consul Token（可选） |
| consul_datacenter | VARCHAR(50) | 数据中心（**自动探测**） |
| description | TEXT | 描述 |
| status | SMALLINT | 1 启用 / 0 停用 |
| created_by | VARCHAR(100) | 创建人 |
| created_at / updated_at | TIMESTAMP | 时间戳 |

### 2.2 service_group_users（服务组用户授权）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 主键 |
| group_id | BIGINT FK | 关联 service_groups，级联删除 |
| user_id | VARCHAR(100) | FlyIAM(Casdoor) 用户 ID |
| permissions | TEXT[] | 权限列表（read/write/delete） |
| created_at | TIMESTAMP | 创建时间 |

唯一约束：`(group_id, user_id)`

### 2.3 service_group_roles（服务组角色授权）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 主键 |
| group_id | BIGINT FK | 关联 service_groups，级联删除 |
| role_name | VARCHAR(100) | FlyIAM(Casdoor) 角色名 |
| permissions | TEXT[] | 权限列表 |
| created_at | TIMESTAMP | 创建时间 |

唯一约束：`(group_id, role_name)`

### 2.4 roles（角色）

角色 = 可复用的权限集合，用于团队授权时引用。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 主键 |
| name | VARCHAR(100) UNIQUE | 角色名称 |
| code | VARCHAR(100) | 角色代码 |
| description | TEXT | 描述 |
| permissions | TEXT[] | 权限集合（read/write/delete） |
| created_at / updated_at | TIMESTAMP | 时间戳 |

### 2.5 teams（团队）

团队是权限分配的主体。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 主键 |
| name | VARCHAR(100) UNIQUE | 团队名称 |
| code | VARCHAR(100) | 团队代码 |
| description | TEXT | 描述 |
| status | SMALLINT | 1 启用 / 0 停用 |
| created_by | VARCHAR(100) | 创建人 |
| created_at / updated_at | TIMESTAMP | 时间戳 |

### 2.6 team_members（团队成员）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 主键 |
| team_id | BIGINT FK | 关联 teams，级联删除 |
| user_id | VARCHAR(100) | FlyIAM(Casdoor) 用户 ID |
| username / display_name | VARCHAR | 用户名 / 显示名 |
| created_at | TIMESTAMP | 创建时间 |

唯一约束：`(team_id, user_id)`

### 2.7 team_group_permissions（团队-服务组授权）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 主键 |
| team_id | BIGINT FK | 关联 teams，级联删除 |
| group_id | BIGINT FK | 关联 service_groups，级联删除 |
| permissions | TEXT[] | 直接权限 |
| role_ids | BIGINT[] | 引用角色的权限（取并集） |
| services | TEXT[] | 授权的服务名（空=无权限；包含 `*`=全部） |
| created_at / updated_at | TIMESTAMP | 时间戳 |

唯一约束：`(team_id, group_id)`

### 2.8 consul_instances（实例持久化）

用于批量导入/导出与本地参考，**列表展示以 Consul 实时数据为准**。

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 主键 |
| instance_id | VARCHAR(255) | Consul 实例 ID |
| service_name | VARCHAR(100) | 服务名 |
| group_id | BIGINT FK | 关联 service_groups |
| address / port | VARCHAR(100) / INT | 地址与端口 |
| tags | TEXT[] | 标签 |
| meta | JSONB | 元数据 |
| health_check | JSONB | 健康检查配置 |
| status | VARCHAR(20) | passing/warning/critical |
| datacenter | VARCHAR(50) | 数据中心 |
| node_name | VARCHAR(100) | 节点名 |
| created_by | VARCHAR(100) | 创建人 |
| created_at / updated_at | TIMESTAMP | 时间戳 |

唯一约束：`(group_id, instance_id)`

### 2.9 audit_logs（审计日志）

| 字段 | 类型 | 说明 |
|------|------|------|
| id | BIGSERIAL PK | 主键 |
| user_id / username | VARCHAR | 操作人 |
| action | VARCHAR(50) | 操作类型（create/update/delete） |
| resource_type / resource_id / resource_name | VARCHAR | 资源信息 |
| group_id | BIGINT | 关联服务组 |
| details | JSONB | 详情 |
| ip_address / user_agent | VARCHAR / TEXT | 来源信息 |
| status / error_message | VARCHAR / TEXT | 结果 |
| created_at | TIMESTAMP | 时间 |

## 3. 索引

见 `deploy/sql/schema.sql` 末尾，主要覆盖：
- 服务组 code、授权表的 group_id/user_id
- 实例的 group_id / service_name / status / datacenter
- 实例 meta 的 GIN 索引
- 审计日志的 user_id / created_at / group_id

## 4. 迁移策略

应用每次启动都会执行 `CREATE TABLE IF NOT EXISTS` 与 `ALTER TABLE ... ADD COLUMN IF NOT EXISTS`，
因此**新增表或列对已有库是兼容的**，无需手工迁移。删除列/表请谨慎评估。

## 5. 相关文档

- [架构设计](architecture.md) · [Kubernetes 部署](../deployment/kubernetes.md)
