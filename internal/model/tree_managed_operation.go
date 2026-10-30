package model

import "time"

// Managed operation records are durable deduplication tombstones. Never remove
// them while old clients or delayed requests can still present an operation ID.
type CloudManagedOperation struct {
	Key                  string    `gorm:"primaryKey;type:varchar(64)" json:"-"`
	OperationID          string    `gorm:"type:varchar(36);not null" json:"operation_id"`
	OperatorID           int       `gorm:"not null;index" json:"-"`
	CloudAccountID       int       `gorm:"not null;index" json:"cloud_account_id"`
	CloudAccountRegionID int       `gorm:"not null;index" json:"cloud_account_region_id"`
	Kind                 string    `gorm:"type:varchar(20);not null" json:"kind"`
	ClaimToken           string    `gorm:"type:varchar(64);not null" json:"-"`
	RequestHash          string    `gorm:"type:varchar(64);not null" json:"-"`
	State                string    `gorm:"type:varchar(20);not null" json:"state"`
	Result               []byte    `gorm:"type:text" json:"-"`
	CreatedAt            time.Time `json:"created_at"`
	UpdatedAt            time.Time `json:"updated_at"`
}

func (CloudManagedOperation) TableName() string { return "cl_tree_cloud_managed_operation" }

type CloudManagedOperationView struct {
	OperationID          string                  `json:"operation_id"`
	CloudAccountID       int                     `json:"cloud_account_id"`
	CloudAccountRegionID int                     `json:"cloud_account_region_id"`
	Kind                 string                  `json:"kind"`
	State                string                  `json:"state"`
	Result               *CloudManagedSyncResult `json:"result,omitempty"`
	CreatedAt            time.Time               `json:"created_at"`
	UpdatedAt            time.Time               `json:"updated_at"`
}
