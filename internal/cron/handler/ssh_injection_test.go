package handler

import (
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/cron/executor"
	"go.uber.org/zap"
)

func TestCronHandlersRetainInjectedSSHExecutor(t *testing.T) {
	supplied, err := executor.NewSSHExecutor(&config.Config{}, zap.NewNop(), nil)
	if err != nil {
		t.Fatal(err)
	}
	handlers := NewCronHandlers(zap.NewNop(), nil, supplied, nil, nil, nil, nil, nil)
	if handlers.sshExecutor != supplied {
		t.Fatal("handler replaced its configured SSH executor")
	}
}
