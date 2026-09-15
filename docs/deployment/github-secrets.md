# 部署文档 - CI/CD 与 GitHub Secrets

本项目通过 GitHub Actions 自动发布多架构二进制与 Docker 镜像。

## 1. 工作流

`.github/workflows/publish.yml` 在推送到 `main` 时触发：

| Job | 产出 |
|-----|------|
| `二进制文件` | Linux/macOS(amd64,arm64)、Windows(amd64) 二进制 + 校验和，发布到 Release `latest` |
| `Docker镜像` | 多架构镜像 `iflyelf/consul-mgr:latest` 推送到 Docker Hub |

## 2. 必需的 Secrets

| Secret | 说明 |
|--------|------|
| `DOCKER_USERNAME` | Docker Hub 用户名 |
| `DOCKER_PASSWORD` | Docker Hub 密码或访问令牌 |

## 3. 配置方法

### 方法一：GitHub 网页

`Settings` → `Secrets and variables` → `Actions` → `New repository secret`

### 方法二：GitHub CLI

```bash
gh secret set DOCKER_USERNAME -b "your-username" -R iflyelf/consul_mgr
gh secret set DOCKER_PASSWORD -b "your-password" -R iflyelf/consul_mgr
```

## 4. 手动触发

`Actions` → `构建并发布(latest)` → `Run workflow`。

## 5. 验证

推送后：

```bash
# 查看运行
gh run list -R iflyelf/consul_mgr
gh run watch -R iflyelf/consul_mgr

# 验证镜像
docker pull iflyelf/consul-mgr:latest
```

## 6. 发布产物

- Release `latest`：二进制压缩包与 `checksums.txt`
- Docker Hub：`iflyelf/consul-mgr:latest`（amd64 / arm64）

## 7. 相关文档

- [systemd 部署](systemd.md) · [Docker 部署](docker.md)
