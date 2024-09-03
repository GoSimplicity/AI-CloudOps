package di

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// InitLogger 将日志输出到控制台
func InitLogger() *zap.Logger {
	// 使用NewDevelopmentConfig创建一个适合开发环境的日志记录器
	cfg := zap.NewDevelopmentConfig()
	cfg.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder // 使用彩色输出
	l, _ := cfg.Build()
	return l
}
