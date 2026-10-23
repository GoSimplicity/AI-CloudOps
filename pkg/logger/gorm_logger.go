package logger

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"
	glogger "gorm.io/gorm/logger"
)

// gormLogger 只记录固定诊断元数据，SQL和驱动错误均可能含凭据
// SQL占位符不能阻止Duplicate entry等驱动错误携带原文
// 不将第三方日志格式或变参转发给zap
type gormLogger struct {
	output        *zap.Logger
	level         glogger.LogLevel
	slowThreshold time.Duration
}

func NewGORMLogger(output *zap.Logger) *gormLogger {
	if output == nil {
		output = zap.NewNop()
	}
	return &gormLogger{output: output.Named("database"), level: glogger.Warn, slowThreshold: 200 * time.Millisecond}
}
func (l *gormLogger) LogMode(level glogger.LogLevel) glogger.Interface {
	copy := *l
	copy.level = level
	return &copy
}
func (l *gormLogger) Info(context.Context, string, ...interface{}) {
	if l.level >= glogger.Info {
		l.output.Info("数据库框架信息")
	}
}
func (l *gormLogger) Warn(context.Context, string, ...interface{}) {
	if l.level >= glogger.Warn {
		l.output.Warn("数据库框架警告")
	}
}
func (l *gormLogger) Error(context.Context, string, ...interface{}) {
	if l.level >= glogger.Error {
		l.output.Error("数据库框架错误")
	}
}

func (l *gormLogger) Trace(_ context.Context, begin time.Time, query func() (string, int64), err error) {
	if l.level <= glogger.Silent {
		return
	}
	elapsed := time.Since(begin)
	failed := err != nil && l.level >= glogger.Error
	slow := l.slowThreshold > 0 && elapsed > l.slowThreshold && l.level >= glogger.Warn
	if !failed && !slow && l.level < glogger.Info {
		return
	}
	_, rows := query()
	fields := []zap.Field{zap.Duration("elapsed", elapsed), zap.Int64("rows", rows)}
	switch {
	case failed:
		fields = append(fields, zap.String("error_type", DatabaseErrorType(err)))
		l.output.Error("数据库操作失败", fields...)
	case slow:
		l.output.Warn("数据库操作耗时过长", fields...)
	default:
		l.output.Info("数据库操作完成", fields...)
	}
}

// ParamsFilter 减少框架日志回调中的参数展开，Trace本身也绝不输出SQL
func (*gormLogger) ParamsFilter(_ context.Context, query string, _ ...interface{}) (string, []interface{}) {
	return query, nil
}

func DatabaseErrorType(err error) string {
	switch {
	case errors.Is(err, context.Canceled):
		return "context_canceled"
	case errors.Is(err, context.DeadlineExceeded):
		return "deadline_exceeded"
	case errors.Is(err, gorm.ErrRecordNotFound), errors.Is(err, sql.ErrNoRows):
		return "record_not_found"
	default:
		return fmt.Sprintf("%T", err)
	}
}

var _ glogger.Interface = (*gormLogger)(nil)
