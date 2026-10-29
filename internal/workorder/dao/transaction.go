package dao

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type transactionKey struct{}

// DBWithContext 让同一工单事务中的 DAO 共用连接
func DBWithContext(ctx context.Context, db *gorm.DB) *gorm.DB {
	if tx, ok := ctx.Value(transactionKey{}).(*gorm.DB); ok {
		return tx.WithContext(ctx)
	}
	return db.WithContext(ctx)
}
func InTransaction(ctx context.Context) bool {
	_, ok := ctx.Value(transactionKey{}).(*gorm.DB)
	return ok
}
func (d *workorderInstanceDAO) InTransaction(ctx context.Context, fn func(context.Context) error) error {
	if InTransaction(ctx) {
		return fn(ctx)
	}
	return d.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error { return fn(context.WithValue(ctx, transactionKey{}, tx)) })
}
func (d *workorderInstanceDAO) SaveNotificationEvent(ctx context.Context, event *model.WorkorderNotificationEvent) error {
	if !InTransaction(ctx) {
		return errors.New("工单通知事件必须在工单事务中保存")
	}
	return DBWithContext(ctx, d.db).Create(event).Error
}
