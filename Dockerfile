# ============================================================
# BlockExplore 多阶段构建 Dockerfile
# 阶段1: 编译 Go 程序
# 阶段2: 运行时镜像（仅包含二进制文件，体积小）
# ============================================================

# ---------- 阶段1: 编译阶段 ----------
FROM golang:1.21-alpine AS builder

# 设置工作目录
WORKDIR /app

# 安装 Git（go mod 需要）
RUN apk add --no-cache git

# 国内构建加速：使用 goproxy.cn，并清除可能从宿主继承的代理设置
ENV GOPROXY=https://goproxy.cn,direct
ENV HTTP_PROXY=
ENV HTTPS_PROXY=
ENV http_proxy=
ENV https_proxy=

# 先复制依赖文件，利用 Docker 缓存层
# 只有 go.mod/go.sum 变化时才重新下载依赖
COPY go.mod go.sum ./
RUN go mod download

# 复制所有源代码
COPY . .

# 编译所有微服务的二进制文件
# CGO_ENABLED=0 禁用 CGO，生成静态链接的二进制文件
# -ldflags="-s -w" 去除调试信息，减小体积
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/query-api ./cmd/query-api/ && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/search-api ./cmd/search-api/ && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/price-api ./cmd/price-api/ && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/eth-sync-worker ./cmd/eth-sync-worker/ && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/btc-sync-worker ./cmd/btc-sync-worker/ && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/sol-sync-worker ./cmd/sol-sync-worker/ && \
    CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /app/bin/block-processor ./cmd/block-processor/

# ---------- 阶段2: 运行阶段 ----------
FROM alpine:3.19

# 设置工作目录
WORKDIR /app

# 安装必要的运行时依赖
# ca-certificates: HTTPS 证书（访问区块链节点需要）
# tzdata: 时区数据
RUN apk add --no-cache ca-certificates tzdata

# 从编译阶段复制二进制文件
COPY --from=builder /app/bin/ /app/

# 复制配置文件（运行时通过 env_file 覆盖）
COPY .env.docker /app/.env

# 复制数据库迁移文件
COPY migrations/ /app/migrations/

# 暴露端口
# 8080: query-api
# 8081: search-api
# 8082: price-api
EXPOSE 8080 8081 8082

# 默认启动 query-api（可通过 docker-compose 覆盖）
CMD ["/app/query-api"]
