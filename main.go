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
	"syscall"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/mock"
	"github.com/GoSimplicity/AI-CloudOps/pkg/di"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/fatih/color"
	"github.com/gin-contrib/cors"
	"github.com/gin-contrib/gzip"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

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

	// 路由
	cmd.Server.GET("/headers", func(c *gin.Context) {
		headers := make(map[string]string)
		for k, v := range c.Request.Header {
			if len(v) > 0 {
				headers[k] = v[0]
			}
		}
		c.JSON(http.StatusOK, headers)
	})

	cmd.Server.GET("/health", func(c *gin.Context) {
		status := "ok"
		dbStatus := "ok"
		dbErr := ""
		if db == nil {
			dbStatus = "unavailable"
			dbErr = "数据库连接为空"
		} else if err := di.CheckDBHealth(db); err != nil {
			dbStatus = "error"
			dbErr = err.Error()
		}
		c.JSON(http.StatusOK, gin.H{
			"status":    status,
			"timestamp": time.Now().Unix(),
			"database": gin.H{
				"status": dbStatus,
				"error":  dbErr,
			},
		})
	})

	cmd.Server.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "AI-CloudOps API 服务运行中",
			"status":  "running",
		})
	})

	// mock数据
	if viper.GetBool("mock.enabled") && di.IsDBAvailable(db) {
		if err := initMock(); err != nil {
			log.Printf("Mock数据初始化失败: %v", err)
		}
	} else if viper.GetBool("mock.enabled") {
		log.Printf("数据库不可用，跳过Mock数据初始化")
	}

	// 启动定时任务
	if di.IsDBAvailable(db) {
		go func() {
			_ = cmd.Cron.StartOnDutyHistoryManager(context.Background())
		}()
		go func() {
			_ = cmd.Cron.StartPrometheusConfigRefreshManager(context.Background())
		}()
		log.Printf("系统启动完成")
	} else {
		log.Printf("降级模式运行")
	}

	// 启动HTTP服务
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

	shutdownCtx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	time.Sleep(2 * time.Second)
	log.Println("服务器已关闭")
	return nil
}

func initMock() error {
	addr := viper.GetString("mysql.addr")
	var db *gorm.DB
	var err error
	for i := 0; i < 3; i++ {
		db, err = gorm.Open(mysql.Open(addr), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})
		if err == nil {
			break
		}
		time.Sleep(time.Second)
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
	for _, ip := range ips {
		fmt.Printf("%s  ", color.GreenString("➜"))
		fmt.Printf("%s  ", color.New(color.Bold).Sprint("Network:"))
		fmt.Printf("%s\n", color.MagentaString("http://%s:%s/", ip, port))
	}
}
