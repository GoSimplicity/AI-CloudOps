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
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/GoSimplicity/AI-CloudOps/pkg/di"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
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

	// 初始化 Web 服务器和其他组件
	cmd := di.InitWebServer()

	// 初始化翻译器
	//if err := di.InitTrans(); err != nil {
	//	return fmt.Errorf("初始化翻译器失败: %v", err)
	//}

	// 设置请求头打印路由
	cmd.Server.GET("/headers", printHeaders)

	// 判断是否需要mock
	if viper.GetString("mock.enabled") == "true" {
		if err := InitMock(); err != nil {
			return fmt.Errorf("初始化Mock数据失败: %v", err)
		}
	}

	// 启动定时任务和worker
	go func() {
		if err := cmd.Scheduler.RegisterTimedTasks(); err != nil {
			log.Fatalf("注册定时任务失败: %v", err)
		}

		if err := cmd.Scheduler.Run(); err != nil {
			log.Fatalf("启动定时任务失败: %v", err)
		}
	}()

	go cmd.Start.StartWorker()

	// 启动异步任务服务器
	go func() {
		mux := cmd.Routes.RegisterHandlers()
		if err := cmd.Asynq.Run(mux); err != nil {
			log.Fatalf("启动异步任务服务器失败: %v", err)
		}
	}()

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
		log.Printf("服务器启动成功，监听端口: %s", viper.GetString("server.port"))
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %v", err)
		}
	}()

	// 等待中断信号
	<-quit
	log.Println("正在关闭服务器...")

	// 先停止定时任务
	cmd.Scheduler.Stop()

	// 关闭异步任务服务器
	cmd.Asynq.Shutdown()

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
	db, err := gorm.Open(mysql.Open(addr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		return fmt.Errorf("连接数据库失败: %v", err)
	}

	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		return fmt.Errorf("创建适配器失败: %v", err)
	}

	enforcer, err := casbin.NewEnforcer("config/model.conf", adapter)
	if err != nil {
		return fmt.Errorf("创建enforcer失败: %v", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("获取sql.DB失败: %v", err)
	}
	defer sqlDB.Close()

	am := mock.NewApiMock(db)
	if err := am.InitApi(); err != nil {
		return fmt.Errorf("初始化API失败: %v", err)
	}

	um := mock.NewUserMock(db, enforcer)
	if err := um.CreateUserAdmin(); err != nil {
		return fmt.Errorf("创建管理员用户失败: %v", err)
	}

	rm := mock.NewRoleMock(db, enforcer)
	if err := rm.InitRole(); err != nil {
		return fmt.Errorf("创建角色失败: %v", err)
	}

	return nil
}
