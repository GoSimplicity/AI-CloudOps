package dao

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrRegionSyncBusy = errors.New("该地域已有同步任务，请稍后重试")
var ErrRegionSyncLeaseLost = errors.New("地域同步租约已过期或被取代")
var ErrRegionSyncCommitUnknown = errors.New("地域同步提交结果未知，请核对后处理")

type RegionSyncLease struct {
	RegionID       int
	Token          string
	Generation     uint64
	ExpiresEpochMS int64
}

func databaseEpochMS(db *gorm.DB) (string, error) {
	switch db.Dialector.Name() {
	case "mysql":
		return "CAST(UNIX_TIMESTAMP(CURRENT_TIMESTAMP(3)) * 1000 AS SIGNED)", nil
	case "sqlite":
		return "CAST((julianday('now') - 2440587.5) * 86400000 AS INTEGER)", nil
	default:
		return "", errors.New("不支持的同步数据库")
	}
}

func (d *treeCloudDAO) AcquireRegionSyncLease(ctx context.Context, regionID int, duration time.Duration) (*RegionSyncLease, error) {
	if regionID <= 0 || duration <= 0 || duration > 10*time.Minute {
		return nil, errors.New("地域租约参数无效")
	}
	clock, err := databaseEpochMS(d.db)
	if err != nil {
		return nil, err
	}
	var random [16]byte
	if _, err = rand.Read(random[:]); err != nil {
		return nil, err
	}
	token := hex.EncodeToString(random[:])
	// The insert and conditional update are short statements, not an external-call transaction.
	if err := d.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&model.CloudRegionSyncLease{RegionID: regionID}).Error; err != nil {
		return nil, err
	}
	result := d.db.WithContext(ctx).Model(&model.CloudRegionSyncLease{}).Where("region_id = ? AND expires_epoch_ms <= "+clock, regionID).Updates(map[string]any{"token": token, "generation": gorm.Expr("generation + 1"), "expires_epoch_ms": gorm.Expr(clock+" + ?", duration.Milliseconds())})
	if result.Error != nil {
		return nil, result.Error
	}
	if result.RowsAffected != 1 {
		return nil, ErrRegionSyncBusy
	}
	var lease model.CloudRegionSyncLease
	if err := d.db.WithContext(ctx).Where("region_id = ? AND token = ?", regionID, token).First(&lease).Error; err != nil {
		return nil, err
	}
	return &RegionSyncLease{RegionID: regionID, Token: token, Generation: lease.Generation, ExpiresEpochMS: lease.ExpiresEpochMS}, nil
}

func (d *treeCloudDAO) ReleaseRegionSyncLease(ctx context.Context, lease *RegionSyncLease) error {
	if lease == nil {
		return nil
	}
	return d.db.WithContext(ctx).Model(&model.CloudRegionSyncLease{}).Where("region_id = ? AND token = ? AND generation = ?", lease.RegionID, lease.Token, lease.Generation).Updates(map[string]any{"token": "", "expires_epoch_ms": 0}).Error
}

// ApplyRegionSync protects account/region changes and lease fencing in one short transaction.
func (d *treeCloudDAO) ApplyRegionSync(ctx context.Context, account *model.CloudAccount, region *model.CloudAccountRegion, lease *RegionSyncLease, apply func(TreeCloudDAO) error) error {
	if account == nil || region == nil || lease == nil || region.ID != lease.RegionID {
		return errors.New("地域同步参数无效")
	}
	clock, err := databaseEpochMS(d.db)
	if err != nil {
		return err
	}
	tx := d.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return tx.Error
	}
	defer tx.Rollback()
	var current model.CloudAccount
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, account.ID).Error; err != nil {
		return err
	}
	if current.Status != model.CloudAccountEnabled || current.Provider != account.Provider || current.CredentialVersion != account.CredentialVersion {
		return errors.New("云账户或凭据已变化，请重新同步")
	}
	var actual model.CloudAccountRegion
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ? AND cloud_account_id = ?", region.ID, account.ID).First(&actual).Error; err != nil {
		return err
	}
	if actual.Status != model.CloudAccountRegionEnabled || actual.Region != region.Region || !sameOptionalString(actual.ProjectID, region.ProjectID) {
		return errors.New("地域配置已变化，请重新同步")
	}
	var locked model.CloudRegionSyncLease
	if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("region_id = ? AND token = ? AND generation = ? AND expires_epoch_ms > "+clock, lease.RegionID, lease.Token, lease.Generation).First(&locked).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrRegionSyncLeaseLost
		}
		return err
	}
	if err := apply(&treeCloudDAO{db: tx, logger: d.logger}); err != nil {
		return err
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	var count int64
	if err := tx.Model(&model.CloudRegionSyncLease{}).Where("region_id = ? AND token = ? AND generation = ? AND expires_epoch_ms > "+clock, lease.RegionID, lease.Token, lease.Generation).Count(&count).Error; err != nil {
		return err
	}
	if count != 1 {
		return ErrRegionSyncLeaseLost
	}
	if err := tx.Commit().Error; err != nil {
		return fmt.Errorf("%w", ErrRegionSyncCommitUnknown)
	}
	return nil
}

func sameOptionalString(a, b *string) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	return *a == *b
}

func (d *treeCloudDAO) EnsureTreeNode(ctx context.Context, id int) error {
	var node model.TreeNode
	if id <= 0 {
		return errors.New("绑定节点ID无效")
	}
	if err := d.db.WithContext(ctx).Clauses(clause.Locking{Strength: "UPDATE"}).First(&node, id).Error; err != nil {
		return errors.New("绑定节点不存在")
	}
	return nil
}

func (d *treeCloudDAO) GetAnyByRegionAndInstanceID(ctx context.Context, regionID int, instanceID string) (*model.TreeCloudResource, error) {
	var resource model.TreeCloudResource
	err := d.db.WithContext(ctx).Unscoped().Where("cloud_account_region_id = ? AND instance_id = ?", regionID, instanceID).First(&resource).Error
	if err != nil {
		return nil, err
	}
	return &resource, nil
}

func (d *treeCloudDAO) RestoreCloudResource(ctx context.Context, id int, fields map[string]interface{}) error {
	values := make(map[string]interface{}, len(fields)+1)
	for key, value := range fields {
		values[key] = value
	}
	values["deleted_at"] = 0
	result := d.db.WithContext(ctx).Unscoped().Model(&model.TreeCloudResource{}).Where("id = ? AND deleted_at <> 0", id).Updates(values)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("资源恢复冲突")
	}
	return nil
}

func (d *treeCloudDAO) DeleteCloudResourceWithBindings(ctx context.Context, id int) error {
	if err := d.db.WithContext(ctx).Exec("DELETE FROM cl_tree_node_cloud WHERE tree_cloud_resource_id = ?", id).Error; err != nil {
		return err
	}
	result := d.db.WithContext(ctx).Delete(&model.TreeCloudResource{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("资源删除冲突")
	}
	return nil
}

func (d *treeCloudDAO) CompleteRegionSyncHistory(ctx context.Context, id int, token string, stats model.CloudRegionSyncResult) error {
	if id <= 0 || token == "" {
		return errors.New("地域同步历史参数无效")
	}
	now := time.Now().UTC()
	result := d.db.WithContext(ctx).Model(&model.CloudResourceSyncHistory{}).Where("id = ? AND operation_id = ? AND sync_status = ?", id, token, "running").Updates(map[string]any{
		"sync_status": "success", "total_count": stats.TotalCount, "new_count": stats.NewCount, "update_count": stats.UpdateCount, "delete_count": stats.DeleteCount, "failed_count": 0, "end_time": now, "error_message": "",
	})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected != 1 {
		return errors.New("地域同步历史状态冲突")
	}
	return nil
}

func (d *treeCloudDAO) FinishRegionSyncHistory(ctx context.Context, id int, token, status, category string) error {
	if id <= 0 || token == "" || (status != "failed" && status != "cancelled" && status != "unknown") {
		return errors.New("地域同步历史参数无效")
	}
	now := time.Now().UTC()
	// A committed success must never be replaced when the commit acknowledgement was lost.
	result := d.db.WithContext(ctx).Model(&model.CloudResourceSyncHistory{}).Where("id = ? AND operation_id = ? AND sync_status = ?", id, token, "running").Updates(map[string]any{"sync_status": status, "error_message": category, "failed_count": 1, "end_time": now})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		var history model.CloudResourceSyncHistory
		if err := d.db.WithContext(ctx).Where("id = ? AND operation_id = ?", id, token).First(&history).Error; err != nil {
			return err
		}
		if history.SyncStatus != "success" && history.SyncStatus != "failed" && history.SyncStatus != "cancelled" && history.SyncStatus != "unknown" {
			return errors.New("地域同步历史状态冲突")
		}
	}
	return nil
}
