package di

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
