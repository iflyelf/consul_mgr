#!/bin/bash

# Consul_Mgr 快速启动脚本
# 用途：初始化数据库并启动服务

set -e

echo "=========================================="
echo "  Consul_Mgr 快速启动脚本"
echo "=========================================="
echo ""

# 检查环境变量
if [ -z "$DATABASE_URL" ]; then
    echo "❌ 错误: 未设置 DATABASE_URL 环境变量"
    echo ""
    echo "请设置以下环境变量："
    echo "  export DATABASE_URL='postgresql://iflyelf:1q23l@Yc45j@10.0.51.88:6000/consul_mgr?sslmode=disable'"
    echo "  export JWT_SECRET='consul_mgr_jwt_secret_2024_min_32_chars'"
    echo "  export ADMIN_PASSWORD='ysyh!9Sky'"
    echo ""
    exit 1
fi

if [ -z "$JWT_SECRET" ]; then
    echo "❌ 错误: 未设置 JWT_SECRET 环境变量"
    exit 1
fi

if [ -z "$ADMIN_PASSWORD" ]; then
    echo "❌ 错误: 未设置 ADMIN_PASSWORD 环境变量"
    exit 1
fi

echo "✅ 环境变量检查通过"
echo ""

# 检查数据库连接
echo "🔍 检查数据库连接..."
if ! command -v psql &> /dev/null; then
    echo "⚠️  警告: 未安装 psql，跳过数据库连接测试"
else
    if psql "$DATABASE_URL" -c "SELECT 1" &> /dev/null; then
        echo "✅ 数据库连接成功"
    else
        echo "❌ 错误: 数据库连接失败"
        exit 1
    fi
fi
echo ""

# 检查二进制文件
if [ ! -f "./consul_mgr" ]; then
    echo "🔨 编译项目..."
    make build
    echo "✅ 编译完成"
else
    echo "✅ 二进制文件已存在"
fi
echo ""

# 启动服务
echo "🚀 启动 Consul_Mgr 服务..."
echo ""
echo "服务信息:"
echo "  - API 地址: http://localhost:8080"
echo "  - 健康检查: http://localhost:8080/health"
echo "  - 管理员账号: $ADMIN_USERNAME (默认: iflyelf)"
echo ""
echo "按 Ctrl+C 停止服务"
echo "=========================================="
echo ""

# 启动服务
./consul_mgr -c etc/config.yaml
