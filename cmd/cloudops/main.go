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
	"log"
	"net/http"

	"github.com/GoSimplicity/AI-CloudOps/mock"
	"github.com/casbin/casbin/v2"
	gormadapter "github.com/casbin/gorm-adapter/v3"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/GoSimplicity/AI-CloudOps/pkg/di"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	Init()
}

func Init() {
	// 初始化配置
	di.InitViper()
	// 初始化 Web 服务器和其他组件
	cmd := di.InitWebServer()
	// 初始化翻译器
	if err := di.InitTrans(); err != nil {
		log.Printf("初始化翻译器失败: %v\n", err)
		return
	}

	// 设置请求头打印路由
	cmd.Server.GET("/headers", printHeaders)

	// 判断是否需要mock
	e := viper.GetString("mock.enabled")
	if e == "true" {
		InitMock()
	}

	sp := viper.GetString("server.port")

	go cmd.Cron.Start() // 启动定时任务
	go cmd.Start.StartWorker()

	// 启动 Web 服务器
	if err := cmd.Server.Run(":" + sp); err != nil {
		zap.L().Fatal("Failed to start web server", zap.Error(err))
	}

}

// printHeaders 打印请求头信息
func printHeaders(c *gin.Context) {
	headers := c.Request.Header
	for key, values := range headers {
		for _, value := range values {
			c.String(http.StatusOK, "%s: %s\n", key, value)
		}
	}
}

func InitMock() {
	addr := viper.GetString("mysql.addr")
	db, err := gorm.Open(mysql.Open(addr), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})

	adapter, err := gormadapter.NewAdapterByDB(db)
	if err != nil {
		log.Fatalf("Failed to create adapter: %v", err)
	}
	enforcer, err := casbin.NewEnforcer("config/model.conf", adapter)
	if err != nil {
		log.Fatalf("Failed to create enforcer: %v", err)
	}

	if err != nil {
		log.Println("mock db error")
	}

	// 确保在函数退出时关闭数据库连接
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB from gorm.DB: %v", err)
	}

	defer sqlDB.Close()

	mm := mock.NewMenuMock(db)
	mm.InitMenu()

	am := mock.NewApiMock(db)
	am.InitApi()

	um := mock.NewUserMock(db, enforcer)
	um.CreateUserAdmin()
}
