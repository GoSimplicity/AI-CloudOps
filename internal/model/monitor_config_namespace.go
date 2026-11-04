package model

// MonitorConfigNamespaceGuard is a durable transaction-lock address, not an
// ownership record. Keep guard rows after retirement; deleting them could split
// competing writers between different lock records. No configuration body lives
// here. DAO writers must validate UTF-8 and at most 45 characters before locking;
// 180 binary bytes retain all 45 four-byte UTF-8 characters without normalization.
type MonitorConfigNamespaceGuard struct {
	ConfigType  int8   `json:"-" gorm:"type:tinyint;primaryKey;autoIncrement:false;not null"`
	InstanceKey []byte `json:"-" gorm:"type:varbinary(180);primaryKey;not null"`
}

func (MonitorConfigNamespaceGuard) TableName() string { return "cl_monitor_config_namespaces" }
