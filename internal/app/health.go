package app

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/sync/errgroup"
)

const readinessTimeout = 2 * time.Second

func (a *App) registerHealthRoutes() {
	a.server.GET("/health", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"status": "ok"}) })
	a.server.GET("/ready", readinessHandler(func(ctx context.Context) error {
		db, err := a.db.DB()
		if err != nil {
			return err
		}
		return db.PingContext(ctx)
	}, func(ctx context.Context) error {
		return a.redis.Ping(ctx).Err()
	}))
}

// readinessHandler 使用同一请求期限并发检查依赖，响应不暴露连接信息
func readinessHandler(databasePing, cachePing func(context.Context) error) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), readinessTimeout)
		defer cancel()
		group, probeCtx := errgroup.WithContext(ctx)
		group.Go(func() error { return databasePing(probeCtx) })
		group.Go(func() error { return cachePing(probeCtx) })
		if err := group.Wait(); err != nil || ctx.Err() != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "not_ready"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ready"})
	}
}
