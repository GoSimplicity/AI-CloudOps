package model

import "time"

// Managed operation records are durable deduplication tombstones. Never remove
// them while old clients or delayed requests can still present an operation ID.
const CloudManagedOperationReviewRoute = "/api/tree/cloud/managed/operations/:operation_id/resolve"

type CloudManagedOperation struct {
	Key                  string     `gorm:"primaryKey;type:varchar(64)" json:"-"`
	OperationID          string     `gorm:"type:varchar(36);not null" json:"operation_id"`
	OperatorID           int        `gorm:"not null;index" json:"-"`
	CloudAccountID       int        `gorm:"not null;index;index:idx_managed_op_scope,priority:1" json:"cloud_account_id"`
	CloudAccountRegionID int        `gorm:"not null;index;index:idx_managed_op_scope,priority:2" json:"cloud_account_region_id"`
	ResourceIDs          StringList `json:"resource_ids" gorm:"type:text"`
	Kind                 string     `gorm:"type:varchar(20);not null;index:idx_managed_op_scope,priority:3" json:"kind"`
	AdmissionVersion     int        `json:"-" gorm:"not null;default:0"`
	StartedAt            *time.Time `json:"started_at,omitempty"`
	ClaimToken           string     `gorm:"type:varchar(64);not null" json:"-"`
	RequestHash          string     `gorm:"type:varchar(64);not null" json:"-"`
	State                string     `gorm:"type:varchar(20);not null" json:"state"`
	Result               []byte     `gorm:"type:text" json:"-"`
	ReviewedAt           *time.Time `json:"reviewed_at,omitempty"`
	ReviewedBy           int        `json:"reviewed_by,omitempty"`
	ReviewReason         string     `json:"review_reason,omitempty" gorm:"type:varchar(1000)"`
	PreviousState        string     `json:"previous_state,omitempty" gorm:"type:varchar(20)"`
	CreatedAt            time.Time  `json:"created_at" gorm:"index:idx_managed_op_scope,priority:4"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

func (CloudManagedOperation) TableName() string { return "cl_tree_cloud_managed_operation" }

type CloudManagedOperationView struct {
	OwnerID              int                     `json:"owner_id"`
	OperationID          string                  `json:"operation_id"`
	CloudAccountID       int                     `json:"cloud_account_id"`
	CloudAccountRegionID int                     `json:"cloud_account_region_id"`
	ResourceIDs          StringList              `json:"resource_ids"`
	Kind                 string                  `json:"kind"`
	State                string                  `json:"state"`
	Result               *CloudManagedSyncResult `json:"result,omitempty"`
	ReviewedAt           *time.Time              `json:"reviewed_at,omitempty"`
	ReviewedBy           int                     `json:"reviewed_by,omitempty"`
	ReviewReason         string                  `json:"review_reason,omitempty"`
	PreviousState        string                  `json:"previous_state,omitempty"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
}

type ReviewCloudManagedOperationReq struct {
	CloudAccountID       int      `json:"cloud_account_id" binding:"required,gt=0"`
	CloudAccountRegionID int      `json:"cloud_account_region_id" binding:"required,gt=0"`
	Kind                 string   `json:"kind" binding:"required,oneof=rds clb cache"`
	ResourceIDs          []string `json:"resource_ids" binding:"required,min=1,max=100,dive,required"`
	ResourcesVerified    bool     `json:"resources_verified" binding:"required"`
	Reason               string   `json:"reason" binding:"required,min=10,max=1000"`
	OperationID          string   `json:"-"`
	OperatorID           int      `json:"-"`
	OwnerID              int      `json:"owner_id,omitempty" binding:"omitempty,gt=0"`
	TrustedSuper         bool     `json:"-"`
}

type ListCloudManagedOperationsReq struct {
	ListCloudManagedResourcesReq
	OwnerID         int  `form:"owner_id" binding:"omitempty,gt=0"`
	AllOwners       bool `form:"all_owners"`
	UnconfirmedOnly bool `form:"unconfirmed_only"`
	ViewerID        int  `form:"-" json:"-"`
	TrustedSuper    bool `form:"-" json:"-"`
}
