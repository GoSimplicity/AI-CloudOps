package dao

import (
	"context"
	"errors"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"strings"
	"time"
)

func (d *managedCloudDAO) safeOperationResultSQL() (string, error) {
	var kind, text, number func(string) string
	var integerType, stringType, boolCheck string
	switch d.db.Dialector.Name() {
	case "sqlite":
		kind = func(field string) string { return "json_type(result, '$." + field + "')" }
		text = func(field string) string { return "json_extract(result, '$." + field + "')" }
		number = text
		integerType = "integer"
		stringType = "text"
		boolCheck = kind("retry_safe") + " = 'true'"
	case "mysql":
		kind = func(field string) string { return "JSON_TYPE(JSON_EXTRACT(result, '$." + field + "'))" }
		text = func(field string) string { return "JSON_UNQUOTE(JSON_EXTRACT(result, '$." + field + "'))" }
		number = func(field string) string { return "CAST(" + text(field) + " AS SIGNED)" }
		integerType = "INTEGER"
		stringType = "STRING"
		boolCheck = kind("retry_safe") + " = 'BOOLEAN' AND " + text("retry_safe") + " = 'true'"
	default:
		return "", errors.New("不支持的操作存储数据库")
	}
	parts := []string{boolCheck, kind("operation_id") + " = '" + stringType + "'", text("operation_id") + " = operation_id", text("status") + " IN ('success','failed','cancelled')"}
	for _, field := range []string{"count", "new_count", "update_count"} {
		parts = append(parts, kind(field)+" = '"+integerType+"'", number(field)+" BETWEEN 0 AND 100")
	}
	sum := number("new_count") + " + " + number("update_count")
	parts = append(parts, "("+sum+") <= "+number("count"), "("+text("status")+" <> 'success' OR ("+number("count")+" > 0 AND ("+sum+") = "+number("count")+"))")
	return strings.Join(parts, " AND "), nil
}
func (d *managedCloudDAO) unresolvedOperationSQL(current *model.CloudManagedOperation) (string, []any, error) {
	safe, err := d.safeOperationResultSQL()
	if err != nil {
		return "", nil, err
	}
	pending := "state = 'running'"
	args := []any{}
	if current != nil {
		// A later reservation cannot have side effects before durable admission.
		// Legacy reservations without this protocol are always treated as uncertain.
		pending = "(state = 'running' AND (admission_version = 0 OR started_at IS NOT NULL OR created_at < ? OR (created_at = ? AND `key` < ?)))"
		args = []any{current.CreatedAt, current.CreatedAt, current.Key}
	}
	reviewLength := "length(trim(review_reason))"
	if d.db.Dialector.Name() == "mysql" {
		reviewLength = "CHAR_LENGTH(TRIM(review_reason))"
	}
	reviewValid := "reviewed_at IS NOT NULL AND reviewed_by > 0 AND " + reviewLength + " BETWEEN 10 AND 1000 AND previous_state IN ('not_recorded','running','completed')"
	condition := "(" + pending + " OR (state = 'completed' AND NOT COALESCE((" + safe + "), FALSE)) OR (state = 'reviewed' AND NOT COALESCE((" + reviewValid + "),FALSE)) OR state NOT IN ('running','completed','reviewed'))"
	return condition, args, nil
}
func (d *managedCloudDAO) HasUnresolvedOperations(ctx context.Context, accountID, regionID int, kind, currentKey string) (bool, error) {
	var current *model.CloudManagedOperation
	if currentKey != "" {
		var row model.CloudManagedOperation
		if err := d.db.WithContext(ctx).Where("`key` = ?", currentKey).First(&row).Error; err != nil {
			return false, err
		}
		if row.CloudAccountID != accountID || row.CloudAccountRegionID != regionID {
			return false, errors.New("操作范围不一致")
		}
		if kind == "" {
			kind = row.Kind
		} else if row.Kind != kind {
			return false, errors.New("操作类型不一致")
		}
		current = &row
	}
	condition, args, err := d.unresolvedOperationSQL(current)
	if err != nil {
		return false, err
	}
	query := d.db.WithContext(ctx).Model(&model.CloudManagedOperation{}).Where("cloud_account_id = ? AND cloud_account_region_id = ? AND kind = ?", accountID, regionID, kind).Where(condition, args...)
	if currentKey != "" {
		query = query.Where("`key` <> ?", currentKey)
	}
	var count int64
	err = query.Count(&count).Error
	return count > 0, err
}
func (d *managedCloudDAO) MarkOperationStarted(ctx context.Context, key, token string) error {
	update := d.db.WithContext(ctx).Model(&model.CloudManagedOperation{}).Where("`key` = ? AND claim_token = ? AND state = ? AND started_at IS NULL", key, token, "running").Updates(map[string]any{"started_at": time.Now().UTC(), "updated_at": time.Now().UTC()})
	if update.Error != nil {
		return update.Error
	}
	if update.RowsAffected != 1 {
		return errors.New("操作已接纳或状态失效")
	}
	return nil
}
func (d *managedCloudDAO) ListOperations(ctx context.Context, req *model.ListCloudManagedOperationsReq, owner int) ([]*model.CloudManagedOperation, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.CloudManagedOperation{}).Where("cloud_account_id = ? AND cloud_account_region_id = ? AND kind = ?", req.CloudAccountID, req.CloudAccountRegionID, req.Kind)
	if owner > 0 {
		query = query.Where("operator_id = ?", owner)
	}
	if req.UnconfirmedOnly {
		condition, args, err := d.unresolvedOperationSQL(nil)
		if err != nil {
			return nil, 0, err
		}
		query = query.Where(condition, args...)
	}
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]*model.CloudManagedOperation, 0)
	err := query.Order("created_at DESC, `key` DESC").Limit(req.Size).Offset((req.Page - 1) * req.Size).Find(&rows).Error
	return rows, total, err
}
