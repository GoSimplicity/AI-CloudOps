/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	cron "github.com/GoSimplicity/AI-CloudOps/internal/cron"
	cronHandler "github.com/GoSimplicity/AI-CloudOps/internal/cron/handler"
	"github.com/GoSimplicity/AI-CloudOps/internal/startup"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/notification"
	"github.com/GoSimplicity/AI-CloudOps/mock"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"gorm.io/gorm"
)

type App struct {
	cfg                   *config.Config
	logger                *zap.Logger
	db                    *gorm.DB
	redis                 redis.Cmdable
	server                *gin.Engine
	bootstrap             startup.ApplicationBootstrap
	cronManager           cron.CronManager
	asynqServer           *asynq.Server
	scheduler             *asynq.Scheduler
	cronHandlers          *cronHandler.CronHandlers
	notificationProcessor *notification.Processor
}

func NewApp(
	cfg *config.Config,
	logger *zap.Logger,
	db *gorm.DB,
	redisClient *redis.Client,
	server *gin.Engine,
	bootstrap startup.ApplicationBootstrap,
	cronManager cron.CronManager,
	asynqServer *asynq.Server,
	scheduler *asynq.Scheduler,
	cronHandlers *cronHandler.CronHandlers,
	notificationProcessor *notification.Processor,
) (*App, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}
	if logger == nil {
		return nil, fmt.Errorf("日志实例不能为空")
	}
	if db == nil {
		return nil, fmt.Errorf("数据库实例不能为空")
	}
	if redisClient == nil {
		return nil, fmt.Errorf("Redis实例不能为空")
	}
	if server == nil {
		return nil, fmt.Errorf("HTTP服务实例不能为空")
	}
	if bootstrap == nil {
		return nil, fmt.Errorf("启动器不能为空")
	}
	if cronManager == nil {
		return nil, fmt.Errorf("Cron管理器不能为空")
	}
	if asynqServer == nil {
		return nil, fmt.Errorf("Asynq服务不能为空")
	}
	if scheduler == nil {
		return nil, fmt.Errorf("Asynq调度器不能为空")
	}
	if cronHandlers == nil {
		return nil, fmt.Errorf("Cron处理器不能为空")
	}
	if notificationProcessor == nil {
		return nil, fmt.Errorf("通知处理器不能为空")
	}

	return &App{
		cfg:                   cfg,
		logger:                logger,
		db:                    db,
		redis:                 redisClient,
		server:                server,
		bootstrap:             bootstrap,
		cronManager:           cronManager,
		asynqServer:           asynqServer,
		scheduler:             scheduler,
		cronHandlers:          cronHandlers,
		notificationProcessor: notificationProcessor,
	}, nil
}

func (a *App) Run(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("上下文不能为空")
	}

	a.server.Use(gzip.Gzip(gzip.BestCompression))

	if err := a.registerAIOps(); err != nil {
		return err
	}
	a.registerHealthRoutes()
	if err := a.bootstrapAdmin(ctx); err != nil {
		return err
	}
	// SEC-27：在对外提供服务前检查 admin 是否仍在使用旧内置弱口令（生产拒绝启动，非生产告警）。
	if err := a.checkLegacyAdminPassword(ctx); err != nil {
		return err
	}

	a.server.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "AI-CloudOps API 服务运行中",
			"status":  "running",
		})
	})

	if err := a.bootstrapPermissionCatalog(ctx); err != nil {
		return fmt.Errorf("初始化权限目录失败: %w", err)
	}

	if a.cfg.Mock.Enabled {
		if err := a.initMockData(ctx); err != nil {
			a.logger.Warn("Mock数据初始化失败", zap.Error(err))
		}
	}
	// 管理员口令只在启动阶段使用：无论是否读取过，都在启动 cron/asynq 等可能派生子进程的组件前清除。
	a.scrubAdminPasswordEnv()

	if err := a.initK8sClients(ctx); err != nil {
		a.logger.Warn("K8s客户端初始化失败", zap.Error(err))
	}

	srv := &http.Server{
		Addr:              ":" + a.cfg.Server.Port,
		Handler:           a.server,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	a.showBootInfo()

	g, runCtx := errgroup.WithContext(ctx)

	g.Go(func() error {
		a.logger.Info("HTTP服务开始监听", zap.String("port", a.cfg.Server.Port))
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return fmt.Errorf("HTTP服务监听失败: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		mux := NewTaskMux(a.cronHandlers, a.notificationProcessor)
		a.logger.Info("Asynq服务开始运行")

		go func() {
			<-runCtx.Done()
			a.asynqServer.Shutdown()
		}()

		if err := a.asynqServer.Run(mux); err != nil {
			return fmt.Errorf("Asynq服务运行失败: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		a.logger.Info("Asynq调度器开始运行")

		go func() {
			<-runCtx.Done()
			a.scheduler.Shutdown()
		}()

		if err := a.scheduler.Run(); err != nil {
			return fmt.Errorf("Asynq调度器运行失败: %w", err)
		}
		return nil
	})

	g.Go(func() error {
		a.logger.Info("统一Cron管理器开始运行")
		if err := a.cronManager.Start(runCtx); err != nil {
			return fmt.Errorf("统一Cron管理器启动失败: %w", err)
		}
		return nil
	})

	<-runCtx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := a.Shutdown(shutdownCtx, srv); err != nil {
		a.logger.Error("服务关闭失败", zap.Error(err))
	}

	if err := g.Wait(); err != nil && !errors.Is(err, context.Canceled) {
		return err
	}
	return nil
}

func (a *App) Shutdown(ctx context.Context, srv *http.Server) error {
	a.logger.Info("开始关闭服务")

	stopCtx, stopCancel := context.WithTimeout(ctx, 30*time.Second)
	defer stopCancel()
	if err := a.cronManager.Stop(stopCtx); err != nil {
		a.logger.Warn("Cron管理器停止失败", zap.Error(err))
	}

	a.asynqServer.Shutdown()
	a.scheduler.Shutdown()

	if srv != nil {
		if err := srv.Shutdown(ctx); err != nil {
			a.logger.Warn("HTTP服务关闭失败", zap.Error(err))
		}
	}

	if a.db != nil {
		sqlDB, err := a.db.DB()
		if err != nil {
			a.logger.Warn("获取sql.DB失败，跳过关闭数据库连接", zap.Error(err))
		} else if err := sqlDB.Close(); err != nil {
			a.logger.Warn("关闭数据库连接失败", zap.Error(err))
		}
	}

	if a.logger != nil {
		_ = a.logger.Sync()
	}

	a.logger.Info("服务已关闭")
	return nil
}

func (a *App) initK8sClients(ctx context.Context) error {
	startCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	if err := a.bootstrap.InitializeK8sClients(startCtx); err != nil {
		return fmt.Errorf("初始化K8s客户端失败: %w", err)
	}
	return nil
}

func (a *App) initMockData(ctx context.Context) error {
	if a.db == nil {
		return fmt.Errorf("数据库连接为空")
	}
	// 纵深防御：配置校验已拒绝生产环境开启 mock，这里再兜底一次，避免绕过 config.Load 的调用方写入 mock 数据。
	if config.IsProduction() {
		return fmt.Errorf("生产环境禁止初始化 Mock 数据")
	}

	startCtx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()

	select {
	case <-startCtx.Done():
		return fmt.Errorf("Mock初始化超时: %w", startCtx.Err())
	default:
	}

	if err := mock.NewApiMock(a.db).InitApi(); err != nil {
		return fmt.Errorf("初始化API Mock失败: %w", err)
	}
	if err := mock.NewUserMock(a.db).CreateUserAdmin(); err != nil {
		return fmt.Errorf("初始化用户 Mock失败: %w", err)
	}
	return nil
}

func (a *App) showBootInfo() {
	ips, err := base.GetLocalIPs()
	if err != nil {
		a.logger.Warn("获取本机IP失败", zap.Error(err))
		return
	}
	a.logger.Info("服务启动成功", zap.String("port", a.cfg.Server.Port))
	for _, ip := range ips {
		a.logger.Info("服务访问地址", zap.String("url", fmt.Sprintf("http://%s:%s/", ip, a.cfg.Server.Port)))
	}
}
