package service

import "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao"

type PrometheusService interface {
}

type prometheusService struct {
	dao dao.PrometheusDao
}

func NewPrometheusService(dao dao.PrometheusDao) PrometheusService {
	return &prometheusService{
		dao: dao,
	}
}
