package model

import "time"

const (
	ConfigPublicationFamilyPrometheusMain      int8 = 1
	ConfigPublicationFamilyAlertManager        int8 = 2
	ConfigPublicationFamilyAlertRule           int8 = 3
	ConfigPublicationFamilyRecordRule          int8 = 4
	ConfigPublicationFamilyAlertManagerWebhook int8 = 5
)
const (
	ConfigPublicationReserved     = "reserved"
	ConfigPublicationCommitted    = "committed"
	ConfigPublicationProjected    = "projected"
	ConfigPublicationFailed       = "failed"
	ConfigPublicationSuperseded   = "superseded"
	ConfigProjectionGenerated     = "generated"
	ConfigProjectionLineageOnly   = "lineage_only"
	ConfigProjectionNotApplicable = "not_applicable"
	ConfigProjectionRender        = "render"
	ConfigProjectionRetire        = "retire"
	ConfigProjectionInvalidate    = "invalidate"
	ConfigProjectionPending       = "pending"
	ConfigProjectionApplied       = "applied"
	ConfigProjectionSuperseded    = "superseded"
)

// MonitorConfigPoolControl is a permanent counter and receipt locator. Reserving
// an attempt must never overwrite either last committed or last projected op.
// There is no soft deletion: removing this address could split writer locks.
type MonitorConfigPoolControl struct {
	Family                    int8      `json:"-" gorm:"type:tinyint;primaryKey;autoIncrement:false;not null"`
	PoolID                    int       `json:"-" gorm:"type:bigint;primaryKey;autoIncrement:false;not null"`
	ReservedGeneration        uint64    `json:"-" gorm:"type:bigint unsigned;not null;default:0"`
	CurrentAttemptOperationID *uint64   `json:"-" gorm:"type:bigint unsigned"`
	LastCommittedOperationID  *uint64   `json:"-" gorm:"type:bigint unsigned"`
	LastProjectedOperationID  *uint64   `json:"-" gorm:"type:bigint unsigned"`
	CreatedAt                 time.Time `json:"-" gorm:"autoCreateTime"`
	UpdatedAt                 time.Time `json:"-" gorm:"autoUpdateTime"`
}

func (MonitorConfigPoolControl) TableName() string { return "cl_monitor_config_pool_controls" }

// MonitorConfigPoolOperation retains every reservation and committed manifest,
// including empty publications. NULL manifest means not committed; JSON [] is a
// legitimate empty manifest. Committed payloads are immutable; only delivery
// state/timestamps evolve. It contains no rendered YAML or credential body.
type MonitorConfigPoolOperation struct {
	OperationID           uint64     `json:"-" gorm:"primaryKey;autoIncrement;index:idx_config_pool_operation_pending,priority:2"`
	Family                int8       `json:"-" gorm:"type:tinyint;not null;uniqueIndex:uk_config_pool_operation_generation,priority:1"`
	PoolID                int        `json:"-" gorm:"type:bigint;not null;uniqueIndex:uk_config_pool_operation_generation,priority:2"`
	Generation            uint64     `json:"-" gorm:"type:bigint unsigned;not null;uniqueIndex:uk_config_pool_operation_generation,priority:3"`
	Nonce                 []byte     `json:"-" gorm:"type:binary(16);not null;uniqueIndex:uk_config_pool_operation_nonce"`
	Phase                 string     `json:"-" gorm:"type:varchar(16);not null;index:idx_config_pool_operation_pending,priority:1"`
	SourceHash            []byte     `json:"-" gorm:"type:binary(32)"`
	Manifest              *string    `json:"-" gorm:"type:json"`
	RecoveryOfOperationID *uint64    `json:"-" gorm:"type:bigint unsigned;index:idx_config_pool_operation_recovery"`
	FailureCode           string     `json:"-" gorm:"type:varchar(32);not null;default:''"`
	CreatedAt             time.Time  `json:"-" gorm:"autoCreateTime"`
	UpdatedAt             time.Time  `json:"-" gorm:"autoUpdateTime"`
	CommittedAt           *time.Time `json:"-"`
	ProjectedAt           *time.Time `json:"-"`
}

func (MonitorConfigPoolOperation) TableName() string { return "cl_monitor_config_pool_operations" }

// MonitorConfigArtifactEvent is an append-only proof of an issued namespace
// revision. Ordinary SQL keys do not authorize projection: generated bindings or
// proven lineage are mandatory, and a custom key may be not_applicable. Binding
// JSON stores typed logical components, never a free-form physical Redis key.
// Delivery fields can evolve; ownership/digest/binding payloads must not change.
type MonitorConfigArtifactEvent struct {
	EventID          uint64     `json:"-" gorm:"primaryKey;autoIncrement;index:idx_config_artifact_namespace,priority:3;index:idx_config_artifact_delivery,priority:2;index:idx_config_artifact_operation,priority:2"`
	OperationID      *uint64    `json:"-" gorm:"type:bigint unsigned;index:idx_config_artifact_operation,priority:1"`
	ConfigType       int8       `json:"-" gorm:"type:tinyint;not null;index:idx_config_artifact_namespace,priority:1"`
	InstanceKey      []byte     `json:"-" gorm:"type:varbinary(180);not null;index:idx_config_artifact_namespace,priority:2"`
	Action           string     `json:"-" gorm:"type:varchar(16);not null"`
	ProjectionPolicy string     `json:"-" gorm:"type:varchar(20);not null"`
	ConfigRowID      *int       `json:"-" gorm:"type:bigint"`
	OldOwnerDigest   []byte     `json:"-" gorm:"type:binary(32)"`
	NewOwnerDigest   []byte     `json:"-" gorm:"type:binary(32)"`
	BodyDigest       []byte     `json:"-" gorm:"type:binary(32)"`
	Binding          *string    `json:"-" gorm:"type:json"`
	BindingEventID   *uint64    `json:"-" gorm:"type:bigint unsigned;index:idx_config_artifact_binding"`
	DeliveryState    string     `json:"-" gorm:"type:varchar(16);not null;index:idx_config_artifact_delivery,priority:1"`
	FailureCode      string     `json:"-" gorm:"type:varchar(32);not null;default:''"`
	CreatedAt        time.Time  `json:"-" gorm:"autoCreateTime"`
	UpdatedAt        time.Time  `json:"-" gorm:"autoUpdateTime"`
	ProjectedAt      *time.Time `json:"-"`
}

func (MonitorConfigArtifactEvent) TableName() string { return "cl_monitor_config_artifact_events" }
