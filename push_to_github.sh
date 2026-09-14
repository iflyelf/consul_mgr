#!/bin/bash
# Consul Manager - 快速推送到 GitHub 脚本

set -e

echo "╔══════════════════════════════════════════════════════════════╗"
echo "║         Consul Manager - GitHub 推送脚本                    ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""

# 检查是否在正确的目录
if [ ! -f "go.mod" ] || [ ! -d ".git" ]; then
    echo "❌ 错误: 请在项目根目录运行此脚本"
    exit 1
fi

# 检查 git 状态
echo "📊 检查 Git 状态..."
git status

echo ""
echo "═══════════════════════════════════════════════════════════════"
echo "  请选择推送方式:"
echo "═══════════════════════════════════════════════════════════════"
echo "  1) HTTPS 方式 (推荐，需要 Personal Access Token)"
echo "  2) SSH 方式 (需要配置 SSH Key)"
echo "  3) 查看当前远程仓库"
echo "  4) 退出"
echo "═══════════════════════════════════════════════════════════════"
echo ""

read -p "请选择 [1-4]: " choice

case $choice in
    1)
        echo ""
        echo "📝 使用 HTTPS 方式推送"
        echo ""
        read -p "请输入 GitHub 用户名 (默认: iflyelf): " username
        username=${username:-iflyelf}
        
        REPO_URL="https://github.com/${username}/consul_mgr.git"
        echo "仓库地址: $REPO_URL"
        
        # 检查是否已存在 origin
        if git remote | grep -q "^origin$"; then
            echo "⚠️  远程仓库 origin 已存在，是否更新? [y/N]"
            read -p "> " update
            if [ "$update" = "y" ] || [ "$update" = "Y" ]; then
                git remote set-url origin $REPO_URL
                echo "✅ 远程仓库地址已更新"
            fi
        else
            git remote add origin $REPO_URL
            echo "✅ 远程仓库已添加"
        fi
        
        echo ""
        echo "🚀 开始推送到 GitHub..."
        echo ""
        echo "⚠️  需要输入认证信息:"
        echo "   用户名: $username"
        echo "   密码: 请使用 Personal Access Token (不是密码!)"
        echo ""
        echo "如何获取 Token: https://github.com/settings/tokens"
        echo ""
        
        git push -u origin main
        
        echo ""
        echo "✅ 代码推送成功！"
        echo ""
        read -p "是否创建并推送 v0.3.0 tag? [Y/n]: " create_tag
        create_tag=${create_tag:-y}
        
        if [ "$create_tag" = "y" ] || [ "$create_tag" = "Y" ]; then
            git tag -a v0.3.0 -m "Release v0.3.0: Initial production release

Features:
- Complete backend API (Go + go-zero)
- Modern frontend UI (Vue 3 + Element Plus)
- JWT authentication system
- Service group management
- Three theme switching
- Fully responsive design
- Zero hardcoded configuration

Status: Production ready"
            
            git push origin v0.3.0
            echo "✅ Tag v0.3.0 推送成功！"
            echo ""
            echo "🎉 GitHub Actions 将自动构建并创建 Release"
            echo "查看进度: https://github.com/${username}/consul_mgr/actions"
        fi
        ;;
        
    2)
        echo ""
        echo "📝 使用 SSH 方式推送"
        echo ""
        read -p "请输入 GitHub 用户名 (默认: iflyelf): " username
        username=${username:-iflyelf}
        
        REPO_URL="git@github.com:${username}/consul_mgr.git"
        echo "仓库地址: $REPO_URL"
        
        # 测试 SSH 连接
        echo ""
        echo "🔍 测试 SSH 连接..."
        if ssh -T git@github.com 2>&1 | grep -q "successfully authenticated"; then
            echo "✅ SSH 连接成功"
        else
            echo "⚠️  SSH 连接失败，请先配置 SSH Key"
            echo "配置指南: https://docs.github.com/zh/authentication/connecting-to-github-with-ssh"
            exit 1
        fi
        
        # 检查是否已存在 origin
        if git remote | grep -q "^origin$"; then
            echo "⚠️  远程仓库 origin 已存在，是否更新? [y/N]"
            read -p "> " update
            if [ "$update" = "y" ] || [ "$update" = "Y" ]; then
                git remote set-url origin $REPO_URL
                echo "✅ 远程仓库地址已更新"
            fi
        else
            git remote add origin $REPO_URL
            echo "✅ 远程仓库已添加"
        fi
        
        echo ""
        echo "🚀 开始推送到 GitHub..."
        git push -u origin main
        
        echo ""
        echo "✅ 代码推送成功！"
        echo ""
        read -p "是否创建并推送 v0.3.0 tag? [Y/n]: " create_tag
        create_tag=${create_tag:-y}
        
        if [ "$create_tag" = "y" ] || [ "$create_tag" = "Y" ]; then
            git tag -a v0.3.0 -m "Release v0.3.0: Initial production release"
            git push origin v0.3.0
            echo "✅ Tag v0.3.0 推送成功！"
            echo ""
            echo "🎉 GitHub Actions 将自动构建并创建 Release"
            echo "查看进度: https://github.com/${username}/consul_mgr/actions"
        fi
        ;;
        
    3)
        echo ""
        echo "📋 当前远程仓库配置:"
        echo ""
        git remote -v
        echo ""
        ;;
        
    4)
        echo "退出"
        exit 0
        ;;
        
    *)
        echo "❌ 无效选择"
        exit 1
        ;;
esac

echo ""
echo "╔══════════════════════════════════════════════════════════════╗"
echo "║                   推送完成！                                ║"
echo "╚══════════════════════════════════════════════════════════════╝"
echo ""
echo "下一步:"
echo "  1. 访问仓库: https://github.com/${username}/consul_mgr"
echo "  2. 查看 Actions: https://github.com/${username}/consul_mgr/actions"
echo "  3. 查看 Releases: https://github.com/${username}/consul_mgr/releases"
echo ""
echo "如需帮助，查看 GITHUB_PUSH_GUIDE.md"
echo ""
