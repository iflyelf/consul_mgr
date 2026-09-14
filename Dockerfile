#############################
#     多阶段构建             #
#############################

# 阶段1: 基于 ubuntu-docker 编译
FROM iflyelf/ubuntu:resolute AS builder

ARG TARGETARCH
ARG VERSION=dev

WORKDIR /build

# 复制源代码
COPY . .

# 编译后端（静态编译）
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w -X main.version=${VERSION}" \
    -o consul_mgr ./cmd/api

# 使用 upx 压缩二进制（可选）
RUN upx-ucl --best --lzma consul_mgr || true

# 阶段2: 运行时镜像（最小化）
FROM alpine:latest

ARG TARGETARCH

# 安装必要的运行时依赖
RUN apk add --no-cache ca-certificates tzdata

# 设置时区
ENV TZ=Asia/Shanghai
RUN ln -sf /usr/share/zoneinfo/${TZ} /etc/localtime

# 创建非 root 用户
RUN addgroup -g 1000 consul && \
    adduser -D -u 1000 -G consul consul

WORKDIR /app

# 从构建阶段复制二进制文件
COPY --from=builder /build/consul_mgr .
COPY --from=builder /build/deploy/config/config.yaml ./etc/config.yaml

# 修改文件权限
RUN chown -R consul:consul /app

USER consul

EXPOSE 8080

ENTRYPOINT ["./consul_mgr"]
CMD ["-c", "etc/config.yaml"]
