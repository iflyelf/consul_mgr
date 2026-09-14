.PHONY: build build-linux-amd64 build-linux-arm64 build-all clean run test

VERSION ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)

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

# 生成发布包
release: build-all
	mkdir -p dist
	cd dist && \
	for f in ../consul_mgr-linux-*; do \
		base=$$(basename $$f); \
		tar -czf "$${base}.tar.gz" -C .. "$${base}"; \
		sha256sum "$${base}.tar.gz" >> checksums.txt; \
	done
