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
	if err := Init(); err != nil {
		log.Fatalf("初始化失败: %v", err)
	}
}

func Init() error {
	// 初始化配置
	if err := di.InitViper(); err != nil {
		return fmt.Errorf("初始化配置失败: %v", err)
	}

	if err := godotenv.Load(); err != nil {
		log.Printf("加载.env文件失败: %v", err)
	}

	// 初始化 Web 服务器和其他组件
	cmd := di.ProvideCmd()

	// 检查数据库状态
	db := di.InitDB()
	if db != nil {
		if err := di.CheckDBHealth(db); err != nil {
			log.Printf("数据库健康检查失败: %v", err)
			log.Printf("程序将以降级模式运行")
		} else {
			log.Printf("数据库健康检查通过")
		}
	} else {
		log.Printf("数据库连接为空，程序将以降级模式运行")
	}

	// 初始化 K8s 集群客户端
	if di.IsDBAvailable(db) {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		if err := cmd.Bootstrap.InitializeK8sClients(ctx); err != nil {
			log.Printf("K8s集群客户端初始化失败: %v", err)
		} else {
			log.Printf("K8s集群客户端初始化成功")
		}
		cancel()
	}

	// 设置中间件
	cmd.Server.Use(cors.Default())
	cmd.Server.Use(gzip.Gzip(gzip.BestCompression))

	// 设置请求头打印路由
	cmd.Server.GET("/headers", printHeaders)

	// 添加健康检查端点
	cmd.Server.GET("/health", func(c *gin.Context) {
		health := gin.H{
			"status":    "ok",
			"timestamp": time.Now().Unix(),
		}

		// 检查数据库状态
		if db != nil {
			if err := di.CheckDBHealth(db); err != nil {
				health["database"] = gin.H{
					"status": "error",
					"error":  err.Error(),
				}
			} else {
				health["database"] = gin.H{
					"status": "ok",
				}
			}
		} else {
			health["database"] = gin.H{
				"status": "unavailable",
				"error":  "数据库连接为空",
			}
		}

		c.JSON(http.StatusOK, health)
	})

	cmd.Server.GET("/", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "AI-CloudOps API 服务运行中",
			"status":  "running",
		})
	})

	// 判断是否需要mock（只有在数据库可用时才执行）
	if viper.GetString("mock.enabled") == "true" && di.IsDBAvailable(db) {
		if err := InitMock(); err != nil {
			log.Printf("初始化Mock数据失败: %v", err)
			// 不返回错误，让程序继续运行
		}
	} else if viper.GetString("mock.enabled") == "true" {
		log.Printf("数据库不可用，跳过Mock数据初始化")
	}

	// 启动定时任务和worker（只有在数据库可用时才启动）
	if di.IsDBAvailable(db) {
		log.Printf("数据库可用，系统启动完成")
	} else {
		log.Printf("数据库不可用，系统以降级模式运行")
	}

	// 创建HTTP服务器
	srv := &http.Server{
		Addr:    ":" + viper.GetString("server.port"),
		Handler: cmd.Server,
	}

	// 创建系统信号接收器
	quit := make(chan os.Signal, 1)
	// 监听 SIGINT 和 SIGTERM 信号
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// 在goroutine中启动服务器
	go func() {
		showBootInfo(viper.GetString("server.port"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	<-quit
	log.Println("正在关闭服务器...")

	// 设置关闭超时时间为30秒
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// 关闭HTTP服务器,等待所有连接处理完成
	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("服务器关闭异常: %v", err)
		return fmt.Errorf("服务器关闭失败: %v", err)
	}

	// 等待所有goroutine完成
	time.Sleep(2 * time.Second)

	log.Println("服务器已成功关闭")
	return nil
}

// printHeaders 打印请求头信息
func printHeaders(c *gin.Context) {
	headers := make(map[string]string)
	for key, values := range c.Request.Header {
		if len(values) > 0 {
			headers[key] = values[0]
		}
	}
	c.JSON(http.StatusOK, headers)
}

func InitMock() error {
	addr := viper.GetString("mysql.addr")

	// 添加重试机制
	maxRetries := 3
	retryDelay := time.Second * 1

	var db *gorm.DB
	var err error

	log.Printf("Mock模式: 正在连接数据库: %s", addr)

	// 重试连接数据库
	for i := 0; i < maxRetries; i++ {
		db, err = gorm.Open(mysql.Open(addr), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Info),
		})

		if err == nil {
			log.Printf("Mock模式: 数据库连接成功")
			break
		}

		log.Printf("Mock模式: 数据库连接失败 (尝试 %d/%d): %v", i+1, maxRetries, err)

		if i < maxRetries-1 {
			time.Sleep(retryDelay)
		}
	}

	if err != nil {
		log.Printf("Mock模式: 数据库连接失败，跳过Mock数据初始化: %v", err)
		return nil // 不返回错误，让程序继续运行
	}

	sqlDB, err := db.DB()
	if err != nil {
		log.Printf("Mock模式: 获取sql.DB失败: %v", err)
		return nil // 不返回错误，让程序继续运行
	}
	defer sqlDB.Close()

	am := mock.NewApiMock(db)
	if err := am.InitApi(); err != nil {
		log.Printf("Mock模式: 初始化API失败: %v", err)
		return nil // 不返回错误，让程序继续运行
	}

	um := mock.NewUserMock(db)
	if err := um.CreateUserAdmin(); err != nil {
		log.Printf("Mock模式: 创建管理员用户失败: %v", err)
		return nil // 不返回错误，让程序继续运行
	}

	log.Printf("Mock模式: 数据初始化完成")
	return nil
}

func showBootInfo(port string) {
	ips, err := utils.GetLocalIPs()
	if err != nil {
		log.Printf("获取本机 IP 失败: %v", err)
		return
	}

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
