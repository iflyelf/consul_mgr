#!/bin/bash

# Casdoor 自动配置脚本
# 功能：检查环境、执行配置、更新配置文件
# 作者: iflyelf
# 创建时间: 2024-01-15

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 Casdoor 自动配置脚本"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查 Casdoor 是否运行
check_casdoor() {
    echo ""
    echo "📡 检查 Casdoor 服务状态..."
    
    CASDOOR_URL="${CASDOOR_ENDPOINT:-http://localhost:8000}"
    
    if curl -s -f "$CASDOOR_URL/api/get-global-providers" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Casdoor 服务运行正常${NC}"
        return 0
    else
        echo -e "${RED}❌ Casdoor 服务未运行或无法访问${NC}"
        echo ""
        echo "💡 解决方法："
        echo "   1. 启动 Casdoor:"
        echo "      docker-compose -f docker-compose.casdoor-external-db.yml up -d casdoor"
        echo ""
        echo "   2. 等待 10 秒后再次运行此脚本"
        echo ""
        return 1
    fi
}

# 执行配置
run_setup() {
    echo ""
    echo "🔧 开始配置 Casdoor..."
    echo ""
    
    cd "$(dirname "$0")/.."
    
    # 编译并运行配置工具
    go run tools/casdoor_setup.go
    
    if [ $? -eq 0 ]; then
        echo ""
        echo -e "${GREEN}✅ Casdoor 配置成功！${NC}"
        return 0
    else
        echo ""
        echo -e "${RED}❌ Casdoor 配置失败${NC}"
        return 1
    fi
}

# 更新项目配置文件
update_project_config() {
    echo ""
    echo "📝 更新项目配置文件..."
    
    if [ ! -f ".env.casdoor" ]; then
        echo -e "${RED}❌ 配置文件 .env.casdoor 不存在${NC}"
        return 1
    fi
    
    # 读取配置
    source .env.casdoor
    
    # 更新 etc/config.yaml
    if [ -f "etc/config.yaml" ]; then
        echo "   更新 etc/config.yaml..."
        
        # 检查是否已有 Casdoor 配置段
        if ! grep -q "^Casdoor:" etc/config.yaml; then
            cat >> etc/config.yaml << EOF

# Casdoor 配置（自动添加）
Casdoor:
  Endpoint: \${CASDOOR_ENDPOINT:http://localhost:8000}
  ClientId: \${CASDOOR_CLIENT_ID}
  ClientSecret: \${CASDOOR_CLIENT_SECRET}
  Certificate: \${CASDOOR_CERTIFICATE:}
  OrganizationName: \${CASDOOR_ORGANIZATION:consul_mgr}
  ApplicationName: \${CASDOOR_APPLICATION:consul_manager}

# 权限配置
Permission:
  EnableServiceGroupAuth: true
  DefaultPermissions: ["read"]
EOF
            echo -e "   ${GREEN}✅ 配置已添加${NC}"
        else
            echo -e "   ${YELLOW}⚠️  Casdoor 配置段已存在，跳过${NC}"
        fi
    fi
    
    # 创建或更新 .env 文件
    echo "   更新 .env 文件..."
    
    if [ -f ".env" ]; then
        # 备份原文件
        cp .env .env.backup
        echo -e "   ${GREEN}✅ 已备份原 .env 文件到 .env.backup${NC}"
    fi
    
    # 合并配置
    cat .env.casdoor >> .env
    echo -e "   ${GREEN}✅ Casdoor 配置已添加到 .env${NC}"
    
    return 0
}

# 显示下一步操作
show_next_steps() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📋 配置文件位置："
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "   casdoor_config.json    - Casdoor 完整配置（JSON 格式）"
    echo "   .env.casdoor           - 环境变量配置"
    echo "   .env                   - 已更新（包含 Casdoor 配置）"
    echo "   etc/config.yaml        - 已更新（包含 Casdoor 配置）"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🔗 访问地址："
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "   Casdoor 管理后台:  http://localhost:8000"
    echo "   默认账号:          admin / 123"
    echo ""
    echo "   Consul Manager:    http://localhost:8080"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📝 下一步操作："
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "   1. 加载环境变量:"
    echo "      source .env"
    echo ""
    echo "   2. 重新编译项目:"
    echo "      go build -o consul_mgr ./cmd/api"
    echo ""
    echo "   3. 启动服务:"
    echo "      ./consul_mgr -c etc/config.yaml"
    echo ""
    echo "   或使用 Docker Compose:"
    echo "      docker-compose -f docker-compose.casdoor-external-db.yml up -d"
    echo ""
    echo "   4. 访问 Casdoor 管理后台创建测试用户:"
    echo "      http://localhost:8000"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
}

# 主流程
main() {
    # 检查 Casdoor 服务
    if ! check_casdoor; then
        exit 1
    fi
    
    # 执行配置
    if ! run_setup; then
        exit 1
    fi
    
    # 更新项目配置
    if ! update_project_config; then
        echo -e "${YELLOW}⚠️  配置文件更新失败，但 Casdoor 配置已完成${NC}"
    fi
    
    # 显示下一步
    show_next_steps
    
    echo ""
    echo -e "${GREEN}✅ 全部完成！${NC}"
    echo ""
}

# 运行主流程
main
