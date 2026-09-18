.PHONY: build build-linux-amd64 build-linux-arm64 build-all clean run test schema

VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

# 数据库连接串（导出表结构用，可通过环境变量覆盖）
DATABASE_URL ?= postgresql://postgres:ysyh!9Sky@localhost:5432/consul_mgr?sslmode=disable

# 本地构建
build:
	CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)" -o consul_mgr ./cmd/api

# Linux AMD64
build-linux-amd64:
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o consul_mgr-linux-amd64 ./cmd/api

# Linux ARM64
build-linux-arm64:
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o consul_mgr-linux-arm64 ./cmd/api

# 构建所有架构
build-all: build-linux-amd64 build-linux-arm64

# 清理
clean:
	rm -f consul_mgr consul_mgr-linux-* dist/*.tar.gz dist/checksums.txt

# 运行
run:
	go run ./cmd/api -c deploy/config/config.yaml

# 测试
test:
	go test -v ./...

# 安装依赖
deps:
	go mod tidy
	go mod download

# 格式化代码
fmt:
	go fmt ./...
	gofmt -s -w .

# 代码检查
lint:
	golangci-lint run

# 从运行中的数据库导出表结构快照（避免手工维护过期）
schema:
	@command -v pg_dump >/dev/null 2>&1 || { echo "❌ 需要 pg_dump（postgresql-client）"; exit 1; }
	@echo "📤 导出表结构到 deploy/sql/schema.sql ..."
	@printf '%s\n' \
	  '-- =============================================================================' \
	  '-- Consul Manager 数据库表结构（自动生成，请勿手工编辑）' \
	  '--' \
	  '-- 生成方式：make schema（pg_dump --schema-only）' \
	  '-- 权威来源：程序启动时按代码内嵌 DDL 自动建表；本文件仅为审计/参考快照。' \
	  '-- =============================================================================' \
	  > deploy/sql/schema.sql
	pg_dump --schema-only --no-owner --no-privileges "$(DATABASE_URL)" >> deploy/sql/schema.sql
	@echo "✅ 已生成: deploy/sql/schema.sql"

# 生成发布包
release: build-all
	mkdir -p dist
	cd dist && \
	for f in ../consul_mgr-linux-*; do \
		base=$$(basename $$f); \
		tar -czf "$${base}.tar.gz" -C .. "$${base}"; \
		sha256sum "$${base}.tar.gz" >> checksums.txt; \
	done
