#!/bin/bash

# Consul Manager 启动脚本

# 颜色定义
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}   Consul Manager 启动脚本${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""

# 检查环境变量
if [ -z "$DATABASE_URL" ]; then
    echo -e "${RED}错误: 未设置 DATABASE_URL 环境变量${NC}"
    echo "请设置: export DATABASE_URL='postgresql://user:password@host:port/database'"
    exit 1
fi

if [ -z "$JWT_SECRET" ]; then
    echo -e "${YELLOW}警告: 未设置 JWT_SECRET，使用默认值${NC}"
    export JWT_SECRET="consul_mgr_jwt_secret_2024_min_32_chars"
fi

if [ -z "$ADMIN_USERNAME" ]; then
    echo -e "${YELLOW}警告: 未设置 ADMIN_USERNAME，使用默认值 admin${NC}"
    export ADMIN_USERNAME="admin"
fi

if [ -z "$ADMIN_PASSWORD" ]; then
    echo -e "${RED}错误: 未设置 ADMIN_PASSWORD 环境变量${NC}"
    exit 1
fi

if [ -z "$ADMIN_EMAIL" ]; then
    export ADMIN_EMAIL="${ADMIN_USERNAME}@example.com"
fi

# 获取脚本所在目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
cd "$SCRIPT_DIR"

# 检查可执行文件是否存在
if [ ! -f "./consul_mgr" ]; then
    echo -e "${RED}错误: consul_mgr 可执行文件不存在${NC}"
    echo "请先运行: make build"
    exit 1
fi

# 停止已有进程
echo -e "${YELLOW}停止已有进程...${NC}"
pkill -9 consul_mgr 2>/dev/null || true
sleep 1

# 启动后端服务
echo -e "${GREEN}启动后端服务...${NC}"
nohup ./consul_mgr -c etc/config.yaml > /tmp/consul_mgr.log 2>&1 &
BACKEND_PID=$!

# 等待启动
sleep 3

# 检查进程是否存在
if ps -p $BACKEND_PID > /dev/null; then
    echo -e "${GREEN}✓ 后端服务启动成功 (PID: $BACKEND_PID)${NC}"
    echo -e "  API 地址: http://localhost:8080"
    echo -e "  健康检查: http://localhost:8080/health"
    echo -e "  日志文件: /tmp/consul_mgr.log"
else
    echo -e "${RED}✗ 后端服务启动失败${NC}"
    echo "查看日志: tail -f /tmp/consul_mgr.log"
    exit 1
fi

# 启动前端服务（如果存在）
if [ -d "./web" ] && [ -f "./web/package.json" ]; then
    echo ""
    echo -e "${GREEN}启动前端服务...${NC}"
    cd web
    
    # 检查 node_modules 是否存在
    if [ ! -d "node_modules" ]; then
        echo -e "${YELLOW}首次运行，安装依赖...${NC}"
        npm install
    fi
    
    # 停止已有前端进程
    pkill -f "vite" 2>/dev/null || true
    
    # 启动前端
    nohup npm run dev > /tmp/vite.log 2>&1 &
    FRONTEND_PID=$!
    sleep 5
    
    if ps -p $FRONTEND_PID > /dev/null; then
        echo -e "${GREEN}✓ 前端服务启动成功 (PID: $FRONTEND_PID)${NC}"
        echo -e "  访问地址: http://localhost:5173"
        echo -e "  日志文件: /tmp/vite.log"
    else
        echo -e "${YELLOW}! 前端服务启动失败（但后端正常运行）${NC}"
    fi
fi

echo ""
echo -e "${GREEN}======================================${NC}"
echo -e "${GREEN}启动完成！${NC}"
echo -e "${GREEN}======================================${NC}"
echo ""
echo -e "管理员账号: ${ADMIN_USERNAME}"
echo -e "访问地址: http://localhost:5173"
echo ""
echo -e "查看日志:"
echo -e "  后端: tail -f /tmp/consul_mgr.log"
echo -e "  前端: tail -f /tmp/vite.log"
echo ""
echo -e "停止服务:"
echo -e "  pkill -9 consul_mgr"
echo -e "  pkill -f vite"
echo ""
