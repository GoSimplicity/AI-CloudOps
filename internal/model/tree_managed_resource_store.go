package model

import "time"

// CloudManagedResource stores typed cloud metadata separately from ECS/SSH fields.
type CloudManagedResource struct {
	Model
	CloudAccountID       int                  `json:"cloud_account_id" gorm:"not null;uniqueIndex:idx_managed_identity,priority:1;index"`
	CloudAccountRegionID int                  `json:"cloud_account_region_id" gorm:"not null;uniqueIndex:idx_managed_identity,priority:2;index"`
	Kind                 string               `json:"kind" gorm:"type:varchar(20);not null;uniqueIndex:idx_managed_identity,priority:3"`
	ProviderID           string               `json:"provider_id" gorm:"type:varchar(100);not null;uniqueIndex:idx_managed_identity,priority:4"`
	RDS                  *AliyunRDSInstance   `json:"rds,omitempty" gorm:"serializer:json;type:json"`
	CLB                  *AliyunLoadBalancer  `json:"clb,omitempty" gorm:"serializer:json;type:json"`
	Cache                *AliyunCacheInstance `json:"cache,omitempty" gorm:"serializer:json;type:json"`
	LocalDescription     string               `json:"local_description" gorm:"type:text"`
	LastSyncTime         time.Time            `json:"last_sync_time" gorm:"not null"`
}

func (CloudManagedResource) TableName() string { return "cl_tree_cloud_managed_resource" }

const CloudManagedResolveRoute = "/api/tree/cloud/managed/history/:id/resolve"

type CloudManagedSyncHistory struct {
	Model
	CloudAccountID       int        `json:"cloud_account_id" gorm:"not null;index"`
	CloudAccountRegionID int        `json:"cloud_account_region_id" gorm:"not null;index"`
	Kind                 string     `json:"kind" gorm:"type:varchar(20);not null"`
	ResourceIDs          StringList `json:"resource_ids" gorm:"type:text;not null"`
	OperationID          string     `json:"-" gorm:"type:varchar(64);index"`
	LeaseExpiresMS       int64      `json:"-" gorm:"not null;default:0"`
	Status               string     `json:"status" gorm:"type:varchar(20);not null;index"`
	Count                int        `json:"count" gorm:"not null;default:0"`
	NewCount             int        `json:"new_count" gorm:"not null;default:0"`
	UpdateCount          int        `json:"update_count" gorm:"not null;default:0"`
	ErrorCategory        string     `json:"error_category,omitempty" gorm:"type:varchar(50)"`
	OperatorID           int        `json:"operator_id" gorm:"not null"`
	StartedAt            time.Time  `json:"started_at" gorm:"not null"`
	FinishedAt           *time.Time `json:"finished_at,omitempty"`
	ResolvedAt           *time.Time `json:"resolved_at,omitempty"`
	ResolvedBy           int        `json:"resolved_by,omitempty"`
	ResolutionReason     string     `json:"resolution_reason,omitempty" gorm:"type:varchar(1000)"`
	RetrySafe            *bool      `json:"retry_safe,omitempty" gorm:"-"`
}

func (CloudManagedSyncHistory) TableName() string { return "cl_tree_cloud_managed_sync_history" }

type ListCloudManagedResourcesReq struct {
	CloudAccountID       int    `form:"cloud_account_id" binding:"required,gt=0"`
	CloudAccountRegionID int    `form:"cloud_account_region_id" binding:"required,gt=0"`
	Kind                 string `form:"kind" binding:"required,oneof=rds clb cache"`
	Page                 int    `form:"page"`
	Size                 int    `form:"size"`
}
type SyncCloudManagedResourcesReq struct {
	OperationID          string   `json:"operation_id,omitempty" binding:"omitempty,max=36"`
	CloudAccountID       int      `json:"cloud_account_id" binding:"required,gt=0"`
	CloudAccountRegionID int      `json:"cloud_account_region_id" binding:"required,gt=0"`
	Kind                 string   `json:"kind" binding:"required,oneof=rds clb cache"`
	ResourceIDs          []string `json:"resource_ids" binding:"required,min=1,max=100,dive,required"`
	OperatorID           int      `json:"-"`
}
type CloudManagedSyncResult struct {
	OperationID   string `json:"operation_id,omitempty"`
	Status        string `json:"status"`
	HistoryID     int    `json:"history_id,omitempty"`
	Count         int    `json:"count"`
	NewCount      int    `json:"new_count"`
	UpdateCount   int    `json:"update_count"`
	ErrorCategory string `json:"error_category,omitempty"`
	RetrySafe     bool   `json:"retry_safe"`
}

type ResolveCloudManagedHistoryReq struct {
	CloudAccountID       int    `json:"cloud_account_id" binding:"required,gt=0"`
	CloudAccountRegionID int    `json:"cloud_account_region_id" binding:"required,gt=0"`
	Kind                 string `json:"kind" binding:"required,oneof=rds clb cache"`
	Reason               string `json:"reason" binding:"required,min=10,max=1000"`
	ResourcesVerified    bool   `json:"resources_verified" binding:"required"`
	HistoryID            int    `json:"-"`
	OperatorID           int    `json:"-"`
}
