FROM golang:1.23.1-alpine AS builder
ENV GOPROXY=https://goproxy.cn,direct
ENV TZ=Asia/Shanghai
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o main main.go

FROM alpine:latest
ENV TZ=Asia/Shanghai
RUN apk --no-cache add ca-certificates tzdata wget curl && \
    ln -snf /usr/share/zoneinfo/$TZ /etc/localtime && \
    echo $TZ > /etc/timezone
WORKDIR /app
COPY --from=builder /app/main .
RUN chmod +x ./main
EXPOSE 8889
CMD ["./main"]
