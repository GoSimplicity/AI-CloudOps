package service

import (
	"context"
	treeNode "github.com/GoSimplicity/AI-CloudOps/internal/tree/dao/tree_node"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"go.uber.org/zap"
)

type NotAuthService interface {
	BuildPrometheusServiceDiscovery(ctx context.Context, leafNodeIdList []string, port int) ([]*targetgroup.Group, error)
}

type notAuthService struct {
	l           *zap.Logger
	treeNodeDao treeNode.TreeNodeDAO
}

func NewNotAuthService(l *zap.Logger, treeNodeDao treeNode.TreeNodeDAO) NotAuthService {
	return &notAuthService{
		l:           l,
		treeNodeDao: treeNodeDao,
	}
}

func (n *notAuthService) BuildPrometheusServiceDiscovery(ctx context.Context, leafNodeIdList []string, port int) ([]*targetgroup.Group, error) {
	//TODO implement me
	panic("implement me")
}
