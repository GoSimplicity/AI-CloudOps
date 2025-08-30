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

package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/GoSimplicity/AI-CloudOps/docs"
	"github.com/GoSimplicity/AI-CloudOps/mock"
	"github.com/GoSimplicity/AI-CloudOps/pkg/di"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/fatih/color"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// @title           AI-CloudOps API
// @version         1.0
// @description     AI-CloudOps云原生运维平台API文档
// @termsOfService  http://swagger.io/terms/

// @contact.name   Bamboo Team
// @contact.url    https://github.com/GoSimplicity/AI-CloudOps
// @contact.email  support@example.com

// @license.name  MIT
// @license.url   https://opensource.org/licenses/MIT

// @host      localhost:8889
// @BasePath  /

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description					Bearer Token认证

// 检查环境变量是否为true
func isEnvTrue(key string) bool {
	value := strings.ToLower(os.Getenv(key))
	return value == "true" || value == "1" || value == "yes" || value == "y" || value == "on"
}

// 检查是否应该启用Swagger
func shouldEnableSwagger() bool {
	// 优先检查环境变量
	if swaggerEnabled := os.Getenv("SWAGGER_ENABLED"); swaggerEnabled != "" {
		return isEnvTrue("SWAGGER_ENABLED")
	}

	// 检查配置文件
	if viper.IsSet("swagger.enabled") {
		return viper.GetBool("swagger.enabled")
	}

	// 默认情况下，开发环境启用，生产环境禁用
	env := strings.ToLower(os.Getenv("GIN_MODE"))
	return env != "release" && env != "production"
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("启动失败: %v", err)
	}
}

func run() error {
	// 加载配置
	if err := di.InitViper(); err != nil {
		return fmt.Errorf("配置加载失败: %v", err)
	}
	_ = godotenv.Load()

	// 初始化依赖
	cmd := di.ProvideCmd()
	db := di.InitDB()

	// 数据库健康检查
	if db != nil && di.CheckDBHealth(db) == nil {
		log.Printf("数据库健康检查通过")
	} else {
		log.Printf("数据库不可用，降级模式")
	}

	// 初始化K8s客户端
	if di.IsDBAvailable(db) {
		if err := cmd.Bootstrap.InitializeK8sClients(context.Background()); err != nil {
			log.Printf("K8s客户端初始化失败: %v", err)
		}
	}

	// 中间件
	cmd.Server.Use(cors.Default())
	cmd.Server.Use(gzip.Gzip(gzip.BestCompression))

	cmd.Server.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "AI-CloudOps API 服务运行中",
			"status":  "running",
		})
	})

	// 条件注册Swagger文档路由
	if shouldEnableSwagger() {
		cmd.Server.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))
		if viper.GetBool("server.debug") {
			log.Printf("Swagger文档已启用: http://localhost:%s/swagger/index.html", viper.GetString("server.port"))
		}
	} else {
		if viper.GetBool("server.debug") {
			log.Printf("Swagger文档已禁用")
		}
	}

	// mock数据
	if viper.GetBool("mock.enabled") && di.IsDBAvailable(db) {
		if err := initMock(); err != nil {
			log.Printf("Mock数据初始化失败: %v", err)
		}
	} else if viper.GetBool("mock.enabled") {
		log.Printf("数据库不可用，跳过Mock数据初始化")
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 启动定时任务
	if di.IsDBAvailable(db) {
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("StartOnDutyHistoryManager panic: %v", r)
				}
			}()
			_ = cmd.Cron.StartOnDutyHistoryManager(ctx)
		}()
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("StartPrometheusConfigRefreshManager panic: %v", r)
				}
			}()
			_ = cmd.Cron.StartPrometheusConfigRefreshManager(ctx)
		}()
		go func() {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("StartCheckK8sStatusManager panic: %v", r)
				}
			}()
			_ = cmd.Cron.StartCheckK8sStatusManager(ctx)
		}()
		log.Printf("系统启动完成")
	} else {
		log.Printf("降级模式运行")
	}

	srv := &http.Server{
		Addr:    ":" + viper.GetString("server.port"),
		Handler: cmd.Server,
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	go func() {
		showBootInfo(viper.GetString("server.port"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	<-quit
	log.Println("正在关闭服务器...")

	cancel()

	shutdownCtx, shutdownCancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer shutdownCancel()
	_ = srv.Shutdown(shutdownCtx)
	time.Sleep(2 * time.Second)
	log.Println("服务器已关闭")
	return nil
}

func initMock() error {
	addr := viper.GetString("mysql.addr")
	var db *gorm.DB
	var err error
	for i := 0; i < 5; i++ {
		db, err = gorm.Open(mysql.Open(addr), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}

	if err != nil {
		return fmt.Errorf("数据库连接失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取sql.DB失败: %v", err)
	}
	defer sqlDB.Close()
	if err := mock.NewApiMock(db).InitApi(); err != nil {
		return fmt.Errorf("初始化API失败: %v", err)
	}
	if err := mock.NewUserMock(db).CreateUserAdmin(); err != nil {
		return fmt.Errorf("创建管理员用户失败: %v", err)
	}
	log.Printf("Mock数据初始化完成")
	return nil
}

func showBootInfo(port string) {
	ips, _ := utils.GetLocalIPs()
	color.Green("AI-CloudOps API 服务启动成功")
	fmt.Printf("%s  ", color.GreenString("➜"))
	fmt.Printf("%s    ", color.New(color.Bold).Sprint("Local:"))
	fmt.Printf("%s\n", color.MagentaString("http://localhost:%s/", port))
	if shouldEnableSwagger() {
		fmt.Printf("%s  ", color.GreenString("➜"))
		fmt.Printf("%s  ", color.New(color.Bold).Sprint("Swagger:"))
		fmt.Printf("%s\n", color.MagentaString("http://localhost:%s/swagger/index.html", port))
	}
	for _, ip := range ips {
		fmt.Printf("%s  ", color.GreenString("➜"))
		fmt.Printf("%s  ", color.New(color.Bold).Sprint("Network:"))
		fmt.Printf("%s\n", color.MagentaString("http://%s:%s/", ip, port))
	}
}
