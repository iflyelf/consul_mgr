# 部署文档 - CI/CD 与 GitHub Secrets

本项目通过 GitHub Actions 自动发布多架构二进制与 Docker 镜像。

## 1. 工作流

`.github/workflows/publish.yml` 在推送到 `main` 时触发：

| Job | 产出 |
|-----|------|
| `二进制文件` | Linux/macOS(amd64,arm64)、Windows(amd64) 二进制 + 校验和，发布到 Release `latest` |
| `Docker镜像` | 多架构镜像推送到 Docker Hub 与华为云 SWR |

## 2. 必需的 Secrets

| Secret | 说明 |
|--------|------|
| `DOCKER_USERNAME` | Docker Hub 用户名 |
| `DOCKER_PASSWORD` | Docker Hub 密码或访问令牌 |
| `SWR_USERNAME` | 华为云 SWR 登录用户名（`docker login -u` 的值，如 `cn-east-3@<AccessKeyId>`） |
| `SWR_PASSWORD` | 华为云 SWR 登录密码（`docker login -p` 的值） |

### 华为云 SWR（国内替代 docker.io）

镜像同时推送到华为云 SWR，Registry 与组织名在 `publish.yml` 顶部 `env` 中配置：

```yaml
env:
  SWR_REGISTRY: swr.cn-east-3.myhuaweicloud.com
  SWR_ORGANIZATION: danxiaonuo
```

配置 Secrets（值取自华为云 SWR 控制台的「登录指令」）：

```bash
gh secret set SWR_USERNAME -b "cn-east-3@<AccessKeyId>" -R iflyelf/consul_mgr
gh secret set SWR_PASSWORD -b "<登录密码>" -R iflyelf/consul_mgr
```

## 3. 配置方法

### 方法一：GitHub 网页

`Settings` → `Secrets and variables` → `Actions` → `New repository secret`

### 方法二：GitHub CLI

```bash
gh secret set DOCKER_USERNAME -b "your-username" -R iflyelf/consul_mgr
gh secret set DOCKER_PASSWORD -b "your-password" -R iflyelf/consul_mgr
gh secret set SWR_USERNAME -b "cn-east-3@<AccessKeyId>" -R iflyelf/consul_mgr
gh secret set SWR_PASSWORD -b "<登录密码>" -R iflyelf/consul_mgr
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
docker pull swr.cn-east-3.myhuaweicloud.com/danxiaonuo/consul-mgr:latest
```

## 6. 发布产物

- Release `latest`：二进制压缩包与 `checksums.txt`
- Docker Hub：`iflyelf/consul-mgr:latest`（amd64 / arm64）
- 华为云 SWR：`swr.cn-east-3.myhuaweicloud.com/danxiaonuo/consul-mgr:latest`（amd64 / arm64）

本地手动推送华为云：

```bash
docker login -u cn-east-3@<AccessKeyId> -p <登录密码> swr.cn-east-3.myhuaweicloud.com
docker tag iflyelf/consul-mgr:latest swr.cn-east-3.myhuaweicloud.com/danxiaonuo/consul-mgr:latest
docker push swr.cn-east-3.myhuaweicloud.com/danxiaonuo/consul-mgr:latest
```

## 7. 相关文档

- [systemd 部署](systemd.md) · [Docker 部署](docker.md)
