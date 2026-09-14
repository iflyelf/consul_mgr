# 推送到 GitHub 远程仓库指南

## 前置准备

1. 确保你有 GitHub 账号并已登录
2. 确保已配置 Git 凭据（SSH 或 HTTPS）

## 方式一: 使用 HTTPS（推荐新手）

### 步骤 1: 在 GitHub 创建新仓库

1. 访问 https://github.com/new
2. 仓库名称: `consul_mgr`
3. 描述: `Consul 服务管理平台 - Go + Vue 3`
4. 选择 Public 或 Private
5. **不要** 勾选任何初始化选项（README, .gitignore, LICENSE）
6. 点击 "Create repository"

### 步骤 2: 推送代码

```bash
cd /xiaonuo/workspace/code/consul_mgr

# 添加远程仓库
git remote add origin https://github.com/iflyelf/consul_mgr.git

# 推送代码
git push -u origin main

# 创建并推送 tag（可选）
git tag -a v0.3.0 -m "Release v0.3.0: Initial production release"
git push origin v0.3.0
```

如果提示需要认证，输入你的 GitHub 用户名和 Personal Access Token（不是密码）。

### 创建 Personal Access Token

1. 访问 https://github.com/settings/tokens
2. 点击 "Generate new token" → "Generate new token (classic)"
3. 勾选权限: `repo` (所有子权限)
4. 点击 "Generate token"
5. 复制 token（只显示一次！）
6. 使用 token 代替密码进行推送

## 方式二: 使用 SSH（推荐熟练用户）

### 步骤 1: 配置 SSH 密钥（如果还没有）

```bash
# 生成 SSH 密钥
ssh-keygen -t ed25519 -C "iflyelf@gmail.com"

# 查看公钥
cat ~/.ssh/id_ed25519.pub

# 复制公钥内容
```

### 步骤 2: 添加公钥到 GitHub

1. 访问 https://github.com/settings/keys
2. 点击 "New SSH key"
3. Title: `Consul Manager Server`
4. 粘贴公钥内容
5. 点击 "Add SSH key"

### 步骤 3: 推送代码

```bash
cd /xiaonuo/workspace/code/consul_mgr

# 添加远程仓库（SSH）
git remote add origin git@github.com:iflyelf/consul_mgr.git

# 推送代码
git push -u origin main

# 创建并推送 tag
git tag -a v0.3.0 -m "Release v0.3.0: Initial production release"
git push origin v0.3.0
```

## 验证推送成功

推送完成后，访问：
- 仓库地址: https://github.com/iflyelf/consul_mgr
- 检查文件是否都已上传
- 查看 Actions 标签页，等待构建完成

## GitHub Actions 构建状态

推送后，GitHub Actions 会自动开始构建：

1. **Build Job**: 构建所有平台的二进制文件
2. **Test Job**: 运行测试套件
3. **Lint Job**: 代码质量检查
4. **Release Job**: (仅 tag 推送) 创建 Release
5. **Docker Job**: (可选) 构建 Docker 镜像

查看构建进度：https://github.com/iflyelf/consul_mgr/actions

## 创建 Release（推送 tag 后）

当你推送 v0.3.0 tag 后，GitHub Actions 会自动：

1. 构建所有平台的二进制文件
2. 打包成 tar.gz 或 zip 格式
3. 创建 GitHub Release
4. 上传所有构建产物

查看 Release：https://github.com/iflyelf/consul_mgr/releases

## 配置 Docker Hub（可选）

如果你想自动构建并推送 Docker 镜像：

### 步骤 1: 在 GitHub 设置 Secrets

1. 访问 https://github.com/iflyelf/consul_mgr/settings/secrets/actions
2. 点击 "New repository secret"
3. 添加以下 secrets:
   - Name: `DOCKER_USERNAME`, Value: 你的 Docker Hub 用户名
   - Name: `DOCKER_PASSWORD`, Value: 你的 Docker Hub 密码或 Token

### 步骤 2: 推送代码

再次推送到 main 分支，Docker 镜像会自动构建并推送到 Docker Hub。

## 常见问题

### Q1: 推送时提示 "Permission denied"

**A**: 使用 HTTPS 方式推送时，确保使用 Personal Access Token 而不是密码。

### Q2: 推送时提示 "remote: Repository not found"

**A**: 检查仓库名称和 URL 是否正确，确保仓库已创建。

### Q3: GitHub Actions 构建失败

**A**: 
1. 检查 Actions 标签页的错误日志
2. 常见原因：Go 版本不匹配、依赖下载失败
3. 如果是测试失败，检查是否需要配置数据库

### Q4: 如何修改远程仓库地址

```bash
# 查看当前远程仓库
git remote -v

# 修改远程仓库 URL
git remote set-url origin <新的URL>
```

## 后续开发流程

### 日常提交和推送

```bash
# 1. 修改代码
# 2. 查看修改
git status
git diff

# 3. 暂存修改
git add .

# 4. 提交
git commit -m "feat: 添加新功能"

# 5. 推送
git push
```

### 创建新版本

```bash
# 1. 更新版本号（修改相关文件）
# 2. 提交修改
git add .
git commit -m "chore: bump version to v0.4.0"

# 3. 创建 tag
git tag -a v0.4.0 -m "Release v0.4.0: 新功能描述"

# 4. 推送代码和 tag
git push
git push origin v0.4.0
```

### 分支管理

```bash
# 创建功能分支
git checkout -b feature/new-feature

# 开发完成后合并到 main
git checkout main
git merge feature/new-feature
git push
```

## 查看构建产物

推送 tag 后，在以下位置查看构建结果：

1. **GitHub Release**: 
   - https://github.com/iflyelf/consul_mgr/releases
   - 包含所有平台的二进制文件

2. **GitHub Actions**: 
   - https://github.com/iflyelf/consul_mgr/actions
   - 查看详细的构建日志

3. **Docker Hub** (如已配置):
   - https://hub.docker.com/r/iflyelf/consul-mgr
   - 拉取镜像: `docker pull iflyelf/consul-mgr:v0.3.0`

## 完成清单

- [ ] 在 GitHub 创建仓库
- [ ] 配置 Git 凭据（HTTPS Token 或 SSH Key）
- [ ] 添加远程仓库
- [ ] 推送代码到 main 分支
- [ ] 创建并推送 v0.3.0 tag
- [ ] 查看 GitHub Actions 构建状态
- [ ] 验证 Release 创建成功
- [ ] (可选) 配置 Docker Hub Secrets
- [ ] (可选) 测试 Docker 镜像构建

## 需要帮助？

- GitHub 文档: https://docs.github.com
- Git 教程: https://git-scm.com/docs
- 项目 Issues: https://github.com/iflyelf/consul_mgr/issues

---

**准备就绪后，运行以下命令开始推送：**

```bash
cd /xiaonuo/workspace/code/consul_mgr
git remote add origin https://github.com/iflyelf/consul_mgr.git
git push -u origin main
```

祝你推送顺利！🚀
