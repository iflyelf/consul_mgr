#############################
#     设置公共的变量         #
#############################
ARG BASE_IMAGE_TAG=resolute
FROM ubuntu:${BASE_IMAGE_TAG}

# 作者描述信息
LABEL org.opencontainers.image.authors="iflyelf" \
      org.opencontainers.image.vendor="iflyelf"

ARG TARGETARCH
ARG TARGETVARIANT

# 时区设置
ARG TZ=Asia/Shanghai
ENV TZ=$TZ
# 语言设置
ARG LANG=zh_CN.UTF-8
ENV LANG=$LANG

# 镜像变量
ARG DOCKER_IMAGE=iflyelf/consul-mgr
ENV DOCKER_IMAGE=$DOCKER_IMAGE
ARG DOCKER_IMAGE_OS=ubuntu
ENV DOCKER_IMAGE_OS=$DOCKER_IMAGE_OS
ARG DOCKER_IMAGE_TAG=resolute
ENV DOCKER_IMAGE_TAG=$DOCKER_IMAGE_TAG

# 环境设置
ARG DEBIAN_FRONTEND=noninteractive
ENV DEBIAN_FRONTEND=$DEBIAN_FRONTEND

# GO环境变量
ARG GO_VERSION=1.21.0
ENV GO_VERSION=$GO_VERSION
ARG GOROOT=/opt/go
ENV GOROOT=$GOROOT
ARG GOPATH=/opt/golang
ENV GOPATH=$GOPATH
ENV PATH=$PATH:$GOROOT/bin:$GOPATH/bin

# 工作目录
ARG DOWNLOAD_SRC=/tmp/src
ENV DOWNLOAD_SRC=$DOWNLOAD_SRC
RUN mkdir -p ${DOWNLOAD_SRC}

# 源设置（阿里云Ubuntu镜像）
RUN sed -i 's@//.*archive.ubuntu.com@//mirrors.aliyun.com@g' /etc/apt/sources.list && \
    sed -i 's/security.ubuntu.com/mirrors.aliyun.com/g' /etc/apt/sources.list

# 安装基础软件
RUN apt-get update && \
    apt-get install -y \
        tzdata \
        locales \
        ca-certificates \
        wget \
        curl \
        tar \
        gzip \
        vim \
        net-tools \
        iputils-ping \
        dnsutils \
        telnet \
        iproute2 \
        procps && \
    # 设置时区
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone && \
    # 设置语言
    locale-gen ${LANG} && \
    update-locale LANG=${LANG} && \
    # 清理
    rm -rf /var/lib/apt/lists/*

# 安装 Go（根据架构选择）
RUN case ${TARGETARCH} in \
        amd64)   GO_ARCH=amd64 ;; \
        arm64)   GO_ARCH=arm64 ;; \
        *)       echo "不支持的架构: ${TARGETARCH}" && exit 1 ;; \
    esac && \
    echo "目标架构: ${TARGETARCH} => Go 包: linux-${GO_ARCH}" && \
    wget --no-check-certificate https://go.dev/dl/go${GO_VERSION}.linux-${GO_ARCH}.tar.gz \
         -O /tmp/go-${GO_ARCH}.tar.gz && \
    tar xzf /tmp/go-${GO_ARCH}.tar.gz -C /opt && \
    mkdir -pv ${GOPATH}/bin && \
    # 仅删除 Go 压缩包, 不清空整个 /tmp (避免误删 DOWNLOAD_SRC=/tmp/src)
    rm -f /tmp/go-${GO_ARCH}.tar.gz && \
    # 软链 go 到 /usr/bin, 后续 RUN 层无需配 PATH
    ln -sf /opt/go/bin/* /usr/bin/ && \
    # 验证版本
    go version

# 设置 Go 代理
ENV GOPROXY=https://goproxy.cn,direct

# 复制源码
WORKDIR /build
COPY . .

# 编译
RUN go mod download && \
    CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} \
    go build -trimpath -ldflags "-s -w" -o consul_mgr ./cmd/api

# 创建运行目录
RUN mkdir -p /app/logs /app/etc && \
    mv consul_mgr /app/ && \
    cp etc/config.yaml /app/etc/

# 工作目录
WORKDIR /app

# 暴露端口
EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# 启动命令
CMD ["/app/consul_mgr", "-c", "/app/etc/config.yaml"]
