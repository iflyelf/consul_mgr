#!/bin/bash

# Casdoor 配置向导
# 功能：引导用户完成 Casdoor 配置，并生成配置文件
# 作者: iflyelf
# 创建时间: 2024-01-15

set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎯 Casdoor 配置向导"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 检查 Casdoor 是否运行
check_casdoor() {
    echo ""
    echo "📡 检查 Casdoor 服务状态..."
    
    if curl -s -f "http://localhost:8000/" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Casdoor 服务运行正常${NC}"
        echo "   访问地址: http://localhost:8000"
        return 0
    else
        echo -e "${RED}❌ Casdoor 服务未运行${NC}"
        echo ""
        echo "请先启动 Casdoor:"
        echo "  docker-compose -f docker-compose.casdoor-external-db.yml up -d casdoor"
        echo ""
        exit 1
    fi
}

# 打开 Casdoor 管理后台
open_casdoor() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📋 步骤 1: 打开 Casdoor 管理后台"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "请在浏览器中打开："
    echo -e "${BLUE}http://localhost:8000${NC}"
    echo ""
    echo "默认管理员账号："
    echo "  用户名: admin"
    echo "  密码: 123"
    echo ""
    read -p "按 Enter 继续..."
}

# 创建组织
create_organization() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📋 步骤 2: 创建组织"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "1. 点击左侧菜单 'Organizations'"
    echo "2. 点击右上角 'Add' 按钮"
    echo "3. 填写以下信息："
    echo ""
    echo "   Name:         consul_mgr"
    echo "   Display name: Consul Manager"
    echo "   Website URL:  http://localhost:8080"
    echo ""
    echo "4. 点击 'Save' 保存"
    echo ""
    read -p "完成后按 Enter 继续..."
}

# 创建应用
create_application() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📋 步骤 3: 创建应用（重要！）"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "1. 点击左侧菜单 'Applications'"
    echo "2. 点击右上角 'Add' 按钮"
    echo "3. 填写以下信息："
    echo ""
    echo "   Name:         consul_manager"
    echo "   Display name: Consul Manager 应用"
    echo "   Organization: consul_mgr (选择刚创建的)"
    echo "   Homepage URL: http://localhost:8080"
    echo ""
    echo "4. 重要：在 'Redirect URIs' 中添加："
    echo "   http://localhost:8080/api/auth/callback"
    echo ""
    echo "5. Token 配置："
    echo "   Token format: JWT"
    echo "   Token expire: 24 hours"
    echo ""
    echo "6. 点击 'Save' 保存"
    echo ""
    echo -e "${YELLOW}⚠️  注意：保存后会生成 Client ID 和 Client Secret${NC}"
    echo ""
    read -p "完成后按 Enter 继续..."
}

# 获取配置信息
get_config() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📋 步骤 4: 获取配置信息"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "在 Applications 列表中，找到 'consul_manager'"
    echo "查看或编辑应用，可以看到："
    echo ""
    
    read -p "请输入 Client ID: " CLIENT_ID
    read -p "请输入 Client Secret: " CLIENT_SECRET
    
    echo ""
    echo -e "${GREEN}✅ 配置信息已记录${NC}"
}

# 创建角色和权限
create_roles_permissions() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📋 步骤 5: 创建角色和权限（可选）"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "建议创建以下角色："
    echo ""
    echo "1. admin (管理员)"
    echo "   - 所有权限"
    echo ""
    echo "2. operator (运维)"
    echo "   - 服务组管理"
    echo "   - 实例管理"
    echo ""
    echo "3. viewer (访客)"
    echo "   - 只读权限"
    echo ""
    echo "详细步骤请参考："
    echo "  docs/CASDOOR_MANUAL_SETUP.md"
    echo ""
    read -p "是否现在创建？(y/n): " CREATE_ROLES
    
    if [ "$CREATE_ROLES" = "y" ]; then
        echo ""
        echo "请按照手册完成角色和权限配置..."
        echo ""
        read -p "完成后按 Enter 继续..."
    else
        echo ""
        echo -e "${YELLOW}跳过角色权限配置，稍后可以在 Casdoor 中配置${NC}"
    fi
}

# 生成配置文件
generate_config() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "💾 生成配置文件"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    
    # 生成 .env.casdoor
    cat > .env.casdoor << EOF
# Casdoor 配置（向导生成）
# 生成时间: $(date)

CASDOOR_ENDPOINT=http://localhost:8000
CASDOOR_CLIENT_ID=$CLIENT_ID
CASDOOR_CLIENT_SECRET=$CLIENT_SECRET
CASDOOR_ORGANIZATION=consul_mgr
CASDOOR_APPLICATION=consul_manager
EOF
    
    echo -e "${GREEN}✅ 配置已保存到 .env.casdoor${NC}"
    
    # 合并到 .env
    if [ -f ".env" ]; then
        echo ""
        read -p "是否合并到 .env 文件？(y/n): " MERGE_ENV
        if [ "$MERGE_ENV" = "y" ]; then
            echo "" >> .env
            cat .env.casdoor >> .env
            echo -e "${GREEN}✅ 配置已合并到 .env${NC}"
        fi
    else
        cp .env.casdoor .env
        echo -e "${GREEN}✅ 配置已保存到 .env${NC}"
    fi
    
    # 更新 config.yaml
    if [ -f "etc/config.yaml" ] && ! grep -q "^Casdoor:" etc/config.yaml; then
        echo ""
        read -p "是否更新 etc/config.yaml？(y/n): " UPDATE_YAML
        if [ "$UPDATE_YAML" = "y" ]; then
            cat >> etc/config.yaml << 'EOF'

# Casdoor 配置（向导添加）
Casdoor:
  Endpoint: ${CASDOOR_ENDPOINT:http://localhost:8000}
  ClientId: ${CASDOOR_CLIENT_ID}
  ClientSecret: ${CASDOOR_CLIENT_SECRET}
  Certificate: ${CASDOOR_CERTIFICATE:}
  OrganizationName: ${CASDOOR_ORGANIZATION:consul_mgr}
  ApplicationName: ${CASDOOR_APPLICATION:consul_manager}

# 权限配置
Permission:
  EnableServiceGroupAuth: true
  DefaultPermissions: ["read"]
EOF
            echo -e "${GREEN}✅ etc/config.yaml 已更新${NC}"
        fi
    fi
}

# 显示总结
show_summary() {
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "🎉 Casdoor 配置完成！"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "📋 配置摘要："
    echo "   Endpoint:      http://localhost:8000"
    echo "   Organization:  consul_mgr"
    echo "   Application:   consul_manager"
    echo "   Client ID:     $CLIENT_ID"
    echo ""
    echo "📄 配置文件："
    echo "   .env.casdoor   - Casdoor 环境变量"
    echo "   .env           - 主配置文件"
    echo "   etc/config.yaml - 服务配置"
    echo ""
    echo "📚 参考文档："
    echo "   docs/CASDOOR_MANUAL_SETUP.md - 详细配置手册"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "📝 下一步："
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo ""
    echo "1. 加载环境变量:"
    echo "   source .env"
    echo ""
    echo "2. 继续开发（Phase 2）:"
    echo "   - 后端 OAuth2 集成"
    echo "   - 前端 Casdoor 集成"
    echo ""
    echo "3. 或查看详细手册:"
    echo "   cat docs/CASDOOR_MANUAL_SETUP.md"
    echo ""
}

# 主流程
main() {
    check_casdoor
    open_casdoor
    create_organization
    create_application
    get_config
    create_roles_permissions
    generate_config
    show_summary
    
    echo -e "${GREEN}✅ 配置向导完成！${NC}"
    echo ""
}

# 运行
main
