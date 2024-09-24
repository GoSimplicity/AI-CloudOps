package dao

import (
	"go.uber.org/zap"
	"gorm.io/gorm"
)

type PrometheusDao interface {
}

type prometheusDao struct {
	db *gorm.DB
	l  *zap.Logger
}

func NewPrometheusDao(db *gorm.DB, l *zap.Logger) PrometheusDao {
	return &prometheusDao{
		db: db,
		l:  l,
	}
}
