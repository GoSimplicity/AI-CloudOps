package dao

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type ManagedCloudDAO interface {
	List(context.Context, *model.ListCloudManagedResourcesReq) ([]*model.CloudManagedResource, int64, error)
	Get(context.Context, *model.ListCloudManagedResourcesReq, int) (*model.CloudManagedResource, error)
	Upsert(context.Context, *model.CloudManagedResource) (bool, error)
	CreateHistory(context.Context, *model.CloudManagedSyncHistory) error
	CompleteHistory(context.Context, int, string, model.CloudManagedSyncResult) error
	FinishHistory(context.Context, int, string, string, string) error
	ListHistory(context.Context, *model.ListCloudManagedResourcesReq) ([]*model.CloudManagedSyncHistory, int64, error)
}

func (d *managedCloudDAO) Get(ctx context.Context, req *model.ListCloudManagedResourcesReq, id int) (*model.CloudManagedResource, error) {
	var row model.CloudManagedResource
	err := d.db.WithContext(ctx).Where("id = ? AND cloud_account_id = ? AND cloud_account_region_id = ? AND kind = ?", id, req.CloudAccountID, req.CloudAccountRegionID, req.Kind).First(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

type managedCloudDAO struct{ db *gorm.DB }

func NewManagedCloudDAO(db *gorm.DB) ManagedCloudDAO { return &managedCloudDAO{db: db} }

func (d *managedCloudDAO) List(ctx context.Context, req *model.ListCloudManagedResourcesReq) ([]*model.CloudManagedResource, int64, error) {
	query := d.db.WithContext(ctx).Model(&model.CloudManagedResource{}).Where("cloud_account_id = ? AND cloud_account_region_id = ? AND kind = ?", req.CloudAccountID, req.CloudAccountRegionID, req.Kind)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]*model.CloudManagedResource, 0)
	err := query.Order("id ASC").Limit(req.Size).Offset((req.Page - 1) * req.Size).Find(&rows).Error
	return rows, total, err
}

// Upsert updates only provider metadata and leaves local descriptions intact.
func (d *managedCloudDAO) Upsert(ctx context.Context, item *model.CloudManagedResource) (bool, error) {
	if item == nil || item.CloudAccountID <= 0 || item.CloudAccountRegionID <= 0 || item.ProviderID == "" {
		return false, errors.New("资源身份无效")
	}
	var payload any
	switch item.Kind {
	case "rds":
		if item.RDS == nil || item.CLB != nil || item.Cache != nil || item.RDS.InstanceID != item.ProviderID {
			return false, errors.New("RDS资源类型无效")
		}
		payload = item.RDS
	case "clb":
		if item.CLB == nil || item.RDS != nil || item.Cache != nil || item.CLB.InstanceID != item.ProviderID {
			return false, errors.New("CLB资源类型无效")
		}
		payload = item.CLB
	case "cache":
		if item.Cache == nil || item.RDS != nil || item.CLB != nil || item.Cache.InstanceID != item.ProviderID {
			return false, errors.New("缓存资源类型无效")
		}
		payload = item.Cache
	default:
		return false, errors.New("资源类型无效")
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return false, errors.New("资源元数据无效")
	}
	var existing model.CloudManagedResource
	err = d.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).Where("cloud_account_id = ? AND cloud_account_region_id = ? AND kind = ? AND provider_id = ?", item.CloudAccountID, item.CloudAccountRegionID, item.Kind, item.ProviderID).First(&existing).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		if err := d.db.WithContext(ctx).Create(item).Error; err != nil {
			return false, err
		}
		return true, nil
	}
	if err != nil {
		return false, err
	}
	// Force explicit column selection, so neither local_description nor other
	// type payloads can be changed by a cloud response.
	update := map[string]any{item.Kind: string(encoded), "last_sync_time": item.LastSyncTime}
	result := d.db.WithContext(ctx).Model(&model.CloudManagedResource{}).Where("id = ?", existing.ID).Updates(update)
	if result.Error != nil {
		return false, result.Error
	}
	return false, nil
}
func (d *managedCloudDAO) CreateHistory(ctx context.Context, row *model.CloudManagedSyncHistory) error {
	return d.db.WithContext(ctx).Create(row).Error
}
func (d *managedCloudDAO) CompleteHistory(ctx context.Context, id int, token string, result model.CloudManagedSyncResult) error {
	now := time.Now().UTC()
	update := d.db.WithContext(ctx).Model(&model.CloudManagedSyncHistory{}).Where("id = ? AND operation_id = ? AND status = ?", id, token, "running").Updates(map[string]any{"status": "success", "count": result.Count, "new_count": result.NewCount, "update_count": result.UpdateCount, "error_category": "", "finished_at": now})
	if update.Error != nil {
		return update.Error
	}
	if update.RowsAffected != 1 {
		return errors.New("同步历史状态冲突")
	}
	return nil
}
func (d *managedCloudDAO) FinishHistory(ctx context.Context, id int, token, status, category string) error {
	if status != "failed" && status != "cancelled" && status != "unknown" {
		return errors.New("同步历史状态无效")
	}
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&model.CloudManagedSyncHistory{}).Where("id = ? AND operation_id = ? AND status = ?", id, token, "running").Updates(map[string]any{"status": status, "error_category": category, "finished_at": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var row model.CloudManagedSyncHistory
		if err := d.db.WithContext(ctx).Where("id = ? AND operation_id = ?", id, token).First(&row).Error; err != nil {
			return err
		}
		if row.Status != "success" && row.Status != "failed" && row.Status != "unknown" && row.Status != "cancelled" {
			return errors.New("同步历史状态冲突")
		}
	}
	return nil
}
func (d *managedCloudDAO) ListHistory(ctx context.Context, req *model.ListCloudManagedResourcesReq) ([]*model.CloudManagedSyncHistory, int64, error) {
	clock, err := databaseEpochMS(d.db)
	if err != nil {
		return nil, 0, err
	}
	var now int64
	if err := d.db.WithContext(ctx).Raw("SELECT " + clock).Scan(&now).Error; err != nil {
		return nil, 0, err
	}
	query := d.db.WithContext(ctx).Model(&model.CloudManagedSyncHistory{}).Where("cloud_account_id = ? AND cloud_account_region_id = ? AND kind = ?", req.CloudAccountID, req.CloudAccountRegionID, req.Kind)
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	rows := make([]*model.CloudManagedSyncHistory, 0)
	if err := query.Order("id DESC").Limit(req.Size).Offset((req.Page - 1) * req.Size).Find(&rows).Error; err != nil {
		return nil, 0, err
	}
	for _, row := range rows {
		if row.Status == "running" && row.LeaseExpiresMS <= now {
			row.Status = "unknown"
			row.ErrorCategory = "lease_expired"
		}
		if row.Status != "success" {
			safe := row.Status == "failed" || row.Status == "cancelled"
			row.RetrySafe = &safe
		}
	}
	return rows, total, nil
}
