#!/bin/bash

set -e

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Consul Manager 数据库清理和 Casdoor 集成"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# 数据库连接信息
DB_HOST="10.0.51.88"
DB_PORT="6000"
DB_USER="iflyelf"
DB_PASS="1q23l@Yc45j"
DB_NAME="consul_mgr"

echo ""
echo "⚠️  警告：此操作将删除以下表："
echo "   - users"
echo "   - roles"
echo "   - permissions"
echo "   - user_roles"
echo "   - role_permissions"
echo "   - role_group_permissions"
echo ""
echo "✅ 将创建以下新表："
echo "   - service_group_users"
echo "   - service_group_roles"
echo ""

# 安装 postgresql-client（如果需要）
if ! command -v psql &> /dev/null; then
    echo "正在安装 PostgreSQL 客户端..."
    apt-get update -qq && apt-get install -y -qq postgresql-client > /dev/null 2>&1 || {
        echo "❌ 无法安装 postgresql-client，请手动安装"
        echo "   执行: apt-get install postgresql-client"
        exit 1
    }
fi

echo "开始执行数据库迁移..."
echo ""

# 执行 SQL 脚本
export PGPASSWORD="$DB_PASS"

psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" \
     -f deploy/sql/cleanup_and_migrate.sql

if [ $? -eq 0 ]; then
    echo ""
    echo "✅ 数据库迁移完成！"
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "当前数据库表："
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -c "
        SELECT 
            table_name,
            pg_size_pretty(pg_total_relation_size(quote_ident(table_name)::regclass)) as size
        FROM information_schema.tables 
        WHERE table_schema = 'public' 
        AND table_type = 'BASE TABLE'
        ORDER BY table_name;
    "
    
    echo ""
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "下一步："
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    echo "1. 启动 Casdoor："
    echo "   docker-compose -f docker-compose.casdoor-external-db.yml up -d casdoor"
    echo ""
    echo "2. 访问 Casdoor 管理界面："
    echo "   http://localhost:8000"
    echo "   默认账号: admin / 123"
    echo ""
    echo "3. 配置 Casdoor（参考 QUICK_START_CASDOOR.md）"
    echo ""
else
    echo ""
    echo "❌ 数据库迁移失败！"
    exit 1
fi

unset PGPASSWORD

