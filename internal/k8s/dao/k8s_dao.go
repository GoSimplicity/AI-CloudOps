package dao

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type K8sDAO interface {
	ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error)
}

type k8sDAO struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewK8sDAO(db *gorm.DB, l *zap.Logger) K8sDAO {
	return &k8sDAO{
		db: db,
		l:  l,
	}
}

func (k *k8sDAO) ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error) {
	var clusters []*model.K8sCluster

	if err := k.db.WithContext(ctx).Find(&clusters).Error; err != nil {
		k.l.Error("ListAllClusters 查询所有集群失败", zap.Error(err))
		return nil, err
	}

	return clusters, nil
}
