# syntax=docker/dockerfile:1
FROM golang:1.26.10-alpine AS builder
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

# Explicit maintenance tool. It never inherits application or database secrets.
FROM alpine:latest AS schema
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/schema ./schema
ENTRYPOINT ["/app/schema"]
    
FROM alpine:latest AS runtime
ENV TZ=Asia/Shanghai
# SEC-27：镜像默认按生产启动（ENV 未设置会拒绝启动）；本地/开发运行请在 compose 或 docker run -e 显式覆盖为 development
ENV ENV=production
RUN apk --no-cache add ca-certificates tzdata wget curl bash python3 nodejs && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone
WORKDIR /app
COPY --from=builder /app/main .
RUN chmod +x ./main
EXPOSE 8889
CMD ["./main"]
