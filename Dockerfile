# syntax=docker/dockerfile:1
# golang 1.26.11-alpine，多架构索引 digest（library/golang:1.26.11-alpine，2026-11-10）
FROM golang:1.26.11-alpine@sha256:ae52b5bc6ed045a9ed26ab03dc19327a802a9cc9aa8b27ed57f5c6b60579b0bb AS builder
ARG GOPROXY=https://proxy.golang.org,direct
ENV GOPROXY=${GOPROXY}
ENV TZ=Asia/Shanghai
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY go.mod go.sum ./
RUN --mount=type=cache,target=/go/pkg/mod go mod download
COPY . .
RUN --mount=type=cache,target=/go/pkg/mod \
    --mount=type=cache,target=/root/.cache/go-build \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -o main main.go && \
    CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -trimpath -o schema ./cmd/schema

# alpine 3.24.4，多架构索引 digest（library/alpine:3.24.4，2026-11-08）。
# schema 是维护 CLI，仍以 root 运行；这里只固定基础镜像。
FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS schema
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/schema ./schema
ENTRYPOINT ["/app/schema"]

# alpine 3.24.4，与 schema 阶段同一 digest。
FROM alpine:3.24.4@sha256:f59abe656b692d901e184be3445e084786602108599d2a51d229c0f5a48c71bf AS runtime
ENV TZ=Asia/Shanghai
# SEC-27：镜像默认按生产启动（ENV 未设置会拒绝启动）；本地/开发运行请在 compose 或 docker run -e 显式覆盖为 development
ENV ENV=production
# 固定 UID/GID，与 deploy/release/compose.yaml 的 user 对齐。家目录放 /tmp，解释器写缓存时不必改应用目录。
ENV HOME=/tmp
RUN apk --no-cache add ca-certificates tzdata wget curl bash python3 nodejs && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone && \
    addgroup -g 10001 -S cloudops && \
    adduser -S -D -H -u 10001 -G cloudops -h /tmp cloudops && \
    mkdir -p /app/logs /tmp/cloudops-cron && \
    chown 10001:10001 /app/logs /tmp/cloudops-cron && \
    chmod 0755 /app/logs && \
    chmod 0750 /tmp/cloudops-cron
WORKDIR /app
# 二进制保持 root 所有、不可被运行用户改写。
# /app/logs 归 10001。未填 work_dir 的 Cron 命令和脚本写 /tmp/cloudops-cron，不写 /app。
COPY --from=builder /app/main .
RUN chmod 0755 ./main
USER 10001:10001
EXPOSE 8889
CMD ["./main"]
