package logger

import "fmt"

// databaseError 保留原始错误链供程序判别，字符串只含静态阶段和类型
type databaseError struct {
	stage string
	cause error
}

func (e *databaseError) Error() string {
	return fmt.Sprintf("%s (%s)", e.stage, DatabaseErrorType(e.cause))
}
func (e *databaseError) Unwrap() error { return e.cause }

// WrapDatabaseError stage只能由调用方静态指定，不应包含配置、SQL或请求值
func WrapDatabaseError(stage string, cause error) error {
	if cause == nil {
		return nil
	}
	return &databaseError{stage: stage, cause: cause}
}
