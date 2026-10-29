package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"gorm.io/gorm"
)

// reconcileRegion only receives a transaction-bound DAO; every change and log
// either commits together or is rolled back. Cloud reads occur before it.
func reconcileRegion(ctx context.Context, store dao.TreeCloudDAO, regionID int, resources []*model.TreeCloudResource, full, autoBind bool, bindNodeID, operatorID int, operatorName string) (model.CloudRegionSyncResult, error) {
	stats := model.CloudRegionSyncResult{RegionID: regionID, Status: "success"}
	byID := make(map[string]bool, len(resources))
	for _, resource := range resources {
		if resource == nil || strings.TrimSpace(resource.InstanceID) == "" || byID[resource.InstanceID] || resource.CloudAccountRegionID != regionID || resource.ResourceType != model.ResourceTypeECS {
			return stats, errors.New("云端资源快照无效或包含重复实例")
		}
		byID[resource.InstanceID] = true
	}
	if autoBind {
		if err := store.EnsureTreeNode(ctx, bindNodeID); err != nil {
			return stats, err
		}
	}
	if full {
		existing, err := store.GetResourcesByRegion(ctx, regionID, model.ResourceTypeECS)
		if err != nil {
			return stats, err
		}
		for _, resource := range existing {
			if err := ctx.Err(); err != nil {
				return stats, err
			}
			if !byID[resource.InstanceID] {
				if err := store.DeleteCloudResourceWithBindings(ctx, resource.ID); err != nil {
					return stats, err
				}
				if err := createSyncChange(ctx, store, resource, model.ChangeTypeDeleted, "", resource.Name, "", operatorID, operatorName); err != nil {
					return stats, err
				}
				stats.DeleteCount++
			}
		}
	}
	for _, resource := range resources {
		if err := ctx.Err(); err != nil {
			return stats, err
		}
		stats.TotalCount++
		now := time.Now().UTC()
		resource.LastSyncTime = &now
		old, err := store.GetAnyByRegionAndInstanceID(ctx, regionID, resource.InstanceID)
		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return stats, err
		}
		switch {
		case old == nil:
			created := newProviderResource(resource)
			if err := store.Create(ctx, created); err != nil {
				return stats, err
			}
			resource.ID = created.ID
			if autoBind {
				if err := store.BindTreeNodes(ctx, resource.ID, []int{bindNodeID}); err != nil {
					return stats, err
				}
			}
			if err := createSyncChange(ctx, store, resource, model.ChangeTypeCreated, "", "", resource.Name, operatorID, operatorName); err != nil {
				return stats, err
			}
			stats.NewCount++
		case old.DeletedAt != 0:
			resource.ID = old.ID
			if err := store.RestoreCloudResource(ctx, old.ID, providerResourceFields(resource)); err != nil {
				return stats, err
			}
			if autoBind {
				if err := store.BindTreeNodes(ctx, resource.ID, []int{bindNodeID}); err != nil {
					return stats, err
				}
			}
			if err := createSyncChange(ctx, store, resource, model.ChangeTypeCreated, "", "", resource.Name, operatorID, operatorName); err != nil {
				return stats, err
			}
			stats.NewCount++
		default:
			resource.ID = old.ID
			if err := store.UpdateMetadata(ctx, resource.ID, providerResourceFields(resource)); err != nil {
				return stats, err
			}
			if err := createSyncChange(ctx, store, resource, model.ChangeTypeUpdated, "", old.Name, resource.Name, operatorID, operatorName); err != nil {
				return stats, err
			}
			if old.Status != resource.Status {
				if err := createSyncChange(ctx, store, resource, model.ChangeTypeStatusChanged, "status", fmt.Sprint(old.Status), fmt.Sprint(resource.Status), operatorID, operatorName); err != nil {
					return stats, err
				}
			}
			stats.UpdateCount++
		}
	}
	return stats, nil
}

// Cloud response data cannot set local credentials, metadata or associations.
func newProviderResource(r *model.TreeCloudResource) *model.TreeCloudResource {
	return &model.TreeCloudResource{
		Name: r.Name, ResourceType: model.ResourceTypeECS, Status: r.Status,
		CloudAccountID: r.CloudAccountID, CloudAccountRegionID: r.CloudAccountRegionID,
		Region: r.Region, InstanceID: r.InstanceID, InstanceType: r.InstanceType,
		Cpu: r.Cpu, Memory: r.Memory, Disk: r.Disk, PublicIP: r.PublicIP, PrivateIP: r.PrivateIP,
		VpcID: r.VpcID, ZoneID: r.ZoneID, ChargeType: r.ChargeType, ExpireTime: r.ExpireTime,
		Currency: r.Currency, OSType: r.OSType, OSName: r.OSName, ImageID: r.ImageID,
		ImageName: r.ImageName, Tags: r.Tags, LastSyncTime: r.LastSyncTime,
	}
}

func createSyncChange(ctx context.Context, store dao.TreeCloudDAO, resource *model.TreeCloudResource, kind, field, oldValue, newValue string, operatorID int, operatorName string) error {
	return store.CreateChangeLog(ctx, &model.CloudResourceChangeLog{ResourceID: resource.ID, InstanceID: resource.InstanceID, ChangeType: kind, FieldName: field, OldValue: oldValue, NewValue: newValue, ChangeSource: model.ChangeSourceSync, OperatorID: operatorID, OperatorName: operatorName, ChangeTime: time.Now().UTC()})
}
