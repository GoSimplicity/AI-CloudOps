# syntax=docker/dockerfile:1
FROM golang:1.26.10-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
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
    go build -o main main.go
    
FROM alpine:latest
ENV TZ=Asia/Shanghai
RUN apk --no-cache add ca-certificates tzdata wget curl bash python3 nodejs && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone
WORKDIR /app
COPY --from=builder /app/main .
RUN chmod +x ./main
EXPOSE 8889
CMD ["./main"]
