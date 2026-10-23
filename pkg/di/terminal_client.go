package di

import (
	treeAPI "github.com/GoSimplicity/AI-CloudOps/internal/tree/api"
	"github.com/GoSimplicity/AI-CloudOps/pkg/ssh"
	"go.uber.org/zap"
)

func InitTerminalClientFactory(logger *zap.Logger) treeAPI.TerminalClientFactory {
	return func() treeAPI.TerminalClient { return ssh.NewClient(logger) }
}
