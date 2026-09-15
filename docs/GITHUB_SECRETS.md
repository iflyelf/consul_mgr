# GitHub Secrets 配置说明

本项目需要以下 GitHub Secrets 用于 CI/CD：

## 必需的 Secrets

### Docker Hub 凭证

用于构建和推送 Docker 镜像到 Docker Hub。

- `DOCKER_USERNAME`: Docker Hub 用户名
- `DOCKER_PASSWORD`: Docker Hub 密码或访问令牌

## 配置方法

### 方法 1: 通过 GitHub 网页界面

1. 访问仓库的 Settings 页面
2. 选择 "Secrets and variables" -> "Actions"
3. 点击 "New repository secret"
4. 添加所需的 secrets

### 方法 2: 通过 GitHub CLI

```bash
gh secret set DOCKER_USERNAME -b "your-username" -R iflyelf/consul_mgr
gh secret set DOCKER_PASSWORD -b "your-password" -R iflyelf/consul_mgr
```

## 验证

配置完成后，推送代码到 main 分支将自动触发构建：

- ✅ 编译多架构二进制文件
- ✅ 构建 Docker 镜像
- ✅ 推送到 Docker Hub
- ✅ 创建 GitHub Release

## 相关链接

- [GitHub Secrets 文档](https://docs.github.com/en/actions/security-guides/encrypted-secrets)
- [Docker Hub](https://hub.docker.com/)
