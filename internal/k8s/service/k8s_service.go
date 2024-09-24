package service

import "github.com/GoSimplicity/AI-CloudOps/internal/k8s/dao"

type K8sService interface {
}

type k8sService struct {
	dao dao.K8sDao
}

func NewK8sService(dao dao.K8sDao) K8sService {
	return &k8sService{
		dao: dao,
	}
}
