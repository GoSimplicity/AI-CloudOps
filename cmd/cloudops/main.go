package main

import (
	"github.com/GoSimplicity/CloudOps/mock"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"log"
	"net/http"

	"github.com/GoSimplicity/CloudOps/config"
	"github.com/GoSimplicity/CloudOps/pkg/di"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func main() {
	Init()
}

func Init() {
	// 初始化配置
	config.InitViper()
	// 初始化 Web 服务器和其他组件
	server := di.InitWebServer()

	// 初始化翻译器
	if err := di.InitTrans(); err != nil {
		log.Printf("初始化翻译器失败: %v\n", err)
		return
	}

	// 设置请求头打印路由
	server.GET("/headers", printHeaders)

	// 判断是否需要mock
	e := viper.GetString("mock.enabled")
	if e == "true" {
		InitMock()
	}

	sp := viper.GetString("server.port")

	// 启动 Web 服务器
	if err := server.Run(":" + sp); err != nil {
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

	if err != nil {
		log.Println("mock db error")
	}

	// 确保在函数退出时关闭数据库连接
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatalf("Failed to get sql.DB from gorm.DB: %v", err)
	}

	defer sqlDB.Close()

	um := mock.NewUserMock(db)
	um.CreateUserAdmin()
}
