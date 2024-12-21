FROM golang:1.22.1-alpine

ENV GOPROXY=https://goproxy.cn,direct
ENV TZ=Asia/Shanghai

RUN apk update --no-cache && apk add --no-cache tzdata
RUN ln -snf /usr/share/zoneinfo/$TZ /etc/localtime &&  echo $TZ >/etc/timezone
RUN go install github.com/cortesi/modd/cmd/modd@latest

WORKDIR /go

CMD ["modd"]