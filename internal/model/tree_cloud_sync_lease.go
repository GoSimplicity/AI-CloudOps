package model

// CloudRegionSyncLease fences all sync writers for one account region.
type CloudRegionSyncLease struct {
	RegionID       int    `gorm:"primaryKey"`
	Token          string `gorm:"type:varchar(64);not null;default:''"`
	Generation     uint64 `gorm:"not null;default:0"`
	ExpiresEpochMS int64  `gorm:"not null;default:0"`
}

func (CloudRegionSyncLease) TableName() string { return "cl_tree_cloud_sync_lease" }
