FROM golang:1.21-alpine AS builder

WORKDIR /build

# 安装依赖
RUN apk --no-cache add ca-certificates git

# 复制go模块文件
COPY go.mod go.sum ./
RUN go mod download

# 复制源代码
COPY . .

# 构建应用
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o webdav-proxy .

FROM alpine:latest
RUN apk --no-cache add ca-certificates tzdata

WORKDIR /app

# 复制构建好的二进制文件
COPY --from=builder /build/webdav-proxy .

# 创建非root用户
RUN addgroup -S appgroup && adduser -S appuser -G appgroup
USER appuser

EXPOSE 8080

# 健康检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
  CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

ENTRYPOINT ["./webdav-proxy"]
CMD ["--listen", ":8080"]