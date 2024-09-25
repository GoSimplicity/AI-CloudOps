package service

import (
	"context"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type K8sService interface {
	ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error)
}

type k8sService struct {
	dao dao.K8sDAO
}

func NewK8sService(dao dao.K8sDAO) K8sService {
	return &k8sService{
		dao: dao,
	}
}

func (k *k8sService) ListAllClusters(ctx context.Context) ([]*model.K8sCluster, error) {
	return k.dao.ListAllClusters(ctx)
}
