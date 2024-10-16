package di

import (
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
	"os"
	"path/filepath"
)

//// InitLogger 将日志输出到控制台
//func InitLogger() *zap.Logger {
//	// 使用NewDevelopmentConfig创建一个适合开发环境的日志记录器
//	cfg := zap.NewDevelopmentConfig()
//	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // 使用彩色输出
//	l, _ := cfg.Build()
//
//	return l
//}

// InitLogger 将日志输出到./logs/cloudops.log，并同时输出到控制台
func InitLogger() *zap.Logger {
	// 创建日志目录
	logDir := viper.GetString("log.dir")
	logFile := filepath.Join(logDir, "cloudops.log")

	if err := os.MkdirAll(logDir, 0755); err != nil {
		panic("无法创建日志目录")
	}

	// 创建文件输出配置
	fileWriter := zapcore.AddSync(&lumberjack.Logger{
		Filename:   logFile,
		MaxSize:    100,  // 每个日志文件最大100MB
		MaxBackups: 5,    // 保留5个旧文件
		MaxAge:     30,   // 文件最多保存30天
		Compress:   true, // 是否压缩旧日志文件
	})

	// 配置日志编码
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder   // 时间格式
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder // 日志等级大写

	// 创建控制台输出
	consoleWriter := zapcore.AddSync(os.Stdout)

	// 创建 Core
	core := zapcore.NewTee(
		zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), consoleWriter, zapcore.WarnLevel), // 控制台
		zapcore.NewCore(zapcore.NewJSONEncoder(encoderConfig), fileWriter, zapcore.InfoLevel),       // 文件
	)

	// 创建 logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger
}
