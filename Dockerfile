#############################
#  Consul Manager 多阶段构建  #
#  builder: iflyelf/ubuntu:latest（含 Go/Node/工具链）#
#  runtime: iflyelf/ubuntu:lite（精简体）             #
#############################

# 构建基础镜像（含 Go / Node / Python / 编译工具链，无需额外安装）
ARG BUILDER_IMAGE=iflyelf/ubuntu:latest
# 运行基础镜像（精简，仅含运行时所需基础包）
ARG RUNTIME_IMAGE=iflyelf/ubuntu:lite

# =============================================================================
# 阶段一：构建（编译 Go 二进制）
# =============================================================================
FROM ${BUILDER_IMAGE} AS builder

ARG TARGETARCH
ARG TARGETVARIANT

# 版本号（由 CI 通过 --build-arg VERSION=<git tag> 注入，缺省为 dev）
ARG VERSION=dev

# Go 模块代理（构建基础镜像已内置，这里允许覆盖）
ARG GOPROXY=https://goproxy.cn,direct

# 说明：构建基础镜像已预装 Go/Node/Python 及全部依赖，无需再 apt 更新/安装。

WORKDIR /src

# 先复制依赖清单，利用层缓存加速 go mod download
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/opt/golang/pkg/mod \
    go mod download

# 复制源码（web/dist 为前端构建产物，供 web 包 go:embed 使用）
COPY cmd/ ./cmd/
COPY internal/ ./internal/
COPY web/ ./web/

# 交叉编译 consul_mgr（CGO_ENABLED=0 纯静态二进制）
RUN --mount=type=cache,target=/root/.cache/go-build \
    --mount=type=cache,target=/opt/golang/pkg/mod \
    set -eux && \
    CGO_ENABLED=0 go build -trimpath \
        -ldflags "-s -w -X main.version=${VERSION}" \
        -o /out/consul_mgr ./cmd/api && \
    /out/consul_mgr --version

# =============================================================================
# 阶段二：运行（仅拷贝构建产物到精简镜像）
# =============================================================================
FROM ${RUNTIME_IMAGE} AS runtime

LABEL org.opencontainers.image.authors="iflyelf" \
      org.opencontainers.image.vendor="iflyelf"

ARG TZ=Asia/Shanghai
ENV TZ=$TZ
ARG LANG=zh_CN.UTF-8
ENV LANG=$LANG

# 复制编译产物
COPY --from=builder /out/consul_mgr /usr/local/bin/consul_mgr

# 内置默认配置（仓库 etc/config.yaml 已脱敏；运行时可用环境变量覆盖或挂载卷替换）
RUN mkdir -p /etc/consul_mgr
COPY etc/config.yaml /etc/consul_mgr/config.yaml

WORKDIR /

EXPOSE 8080

# 默认配置文件路径，可通过挂载卷覆盖
ENTRYPOINT ["/usr/bin/tini", "--", "/usr/local/bin/consul_mgr"]
CMD ["-c", "/etc/consul_mgr/config.yaml"]
