FROM golang:1.23.1-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
ENV TZ=Asia/Shanghai
ARG TARGETOS
ARG TARGETARCH

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} \
    go build -a -installsuffix cgo -o main main.go
    
FROM alpine:latest
ENV TZ=Asia/Shanghai
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app
COPY --from=builder /app/main .
EXPOSE 8889
CMD ["./main"]
