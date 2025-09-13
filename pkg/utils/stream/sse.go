package stream

import (
	"context"
	"io"
	"sync"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type producer func(context.Context, chan interface{})

func SseStream(ctx *gin.Context, prodFunc producer, log *zap.Logger) {
	streamChan := make(chan interface{}, 1)

	cancelCtx, cancel := context.WithCancel(context.Background())

	defer cancel()
	var wg sync.WaitGroup

	wg.Add(1)
	go func() {
		defer wg.Done()

		// 使用上下文监听客户端断开连接
		select {
		case <-ctx.Request.Context().Done():
			log.Info("Connection closed, stopping container Log SSE...")
			cancel()
		case <-cancelCtx.Done():
			log.Info("Producer finished, stopping SSE...")
		}
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		prodFunc(cancelCtx, streamChan) // 协程阻塞处理
		close(streamChan)
	}()

	ctx.Stream(func(w io.Writer) bool {
		if msg, ok := <-streamChan; ok {
			ctx.SSEvent("message", msg)
			return true
		}
		return false
	})
}
