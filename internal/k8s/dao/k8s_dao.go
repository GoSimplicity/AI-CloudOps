package dao

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type K8sDao interface {
}

type k8sDao struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewK8sDao(db *gorm.DB, l *zap.Logger) K8sDao {
	return &k8sDao{
		db: db,
		l:  l,
	}
}
