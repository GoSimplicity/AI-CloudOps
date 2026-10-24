# syntax=docker/dockerfile:1
# 告警 webhook 服务（cmd/webhook）镜像；构建上下文为后端仓库根目录（compose.yaml 中 context: ../..）
FROM golang:1.26.10-alpine AS builder
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -o webhook ./cmd/webhook

FROM alpine:latest
ENV TZ=Asia/Shanghai
RUN apk --no-cache add ca-certificates tzdata && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone && \
    addgroup -S cloudops && adduser -S -G cloudops -H cloudops
WORKDIR /app
COPY --from=builder /app/webhook ./webhook
# 默认读取 ./config/webhook.yaml（--config 可改）；DSN 不在文件里，由环境变量 WEBHOOK_MYSQL_ADDR 注入
COPY --from=builder /app/config/webhook.yaml ./config/webhook.yaml
USER cloudops
EXPOSE 8889
CMD ["./webhook"]
