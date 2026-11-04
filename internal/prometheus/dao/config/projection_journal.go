package config

import (
	"context"
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

var (
	ErrJournalInvalid          = errors.New("发布日志请求无效")
	ErrJournalConflict         = errors.New("发布日志状态或归属已变化")
	ErrJournalNotFound         = errors.New("发布日志记录不存在")
	ErrJournalWrite            = errors.New("发布日志写入失败")
	ErrJournalUncertain        = errors.New("发布日志提交结果不确定，请按nonce核对")
	ErrJournalCounterExhausted = errors.New("发布generation已耗尽")
)

type PoolOperationRequest struct {
	Family                int8
	PoolID                int
	Nonce                 [16]byte
	RecoveryOfOperationID *uint64
}
type PoolReservation struct {
	OperationID uint64
	Family      int8
	PoolID      int
	Generation  uint64
	Nonce       [16]byte
}
type ProjectionBinding struct {
	Version    int    `json:"version"`
	Role       string `json:"role"`
	ConfigType int8   `json:"config_type"`
	InstanceIP string `json:"instance_ip"`
	GroupID    *int   `json:"group_id,omitempty"`
}
type GeneratedPlanItem struct {
	SourceID uint64
	Action   string
	Expected *GeneratedConfigOwner
	Desired  *model.MonitorConfig
	Binding  *ProjectionBinding
}
type GeneratedPlanRequest struct {
	Reservation PoolReservation
	SourceHash  [32]byte
	Items       []GeneratedPlanItem
}
type ArtifactReceipt struct {
	SourceID         uint64
	EventID          uint64
	OperationID      *uint64
	ConfigType       int8
	InstanceKey      string
	Action           string
	ProjectionPolicy string
	ConfigRowID      *int
	OldOwnerDigest   []byte
	NewOwnerDigest   []byte
	BodyDigest       []byte
	Binding          *ProjectionBinding
	BindingEventID   *uint64
}
type PoolReceipt struct {
	Reservation PoolReservation
	SourceHash  [32]byte
	Phase       string
	Artifacts   []ArtifactReceipt
}
type PoolOperationSnapshot struct {
	Reservation PoolReservation
	Phase       string
	SourceHash  *[32]byte
	Receipt     *PoolReceipt
	FailureCode string
}

type ProjectionJournalDAO interface {
	ReservePoolOperation(context.Context, PoolOperationRequest) (PoolReservation, error)
	CommitGeneratedPlan(context.Context, GeneratedPlanRequest) (PoolReceipt, error)
	LoadOperationByNonce(context.Context, [16]byte) (PoolOperationSnapshot, error)
	LoadLastCommitted(context.Context, int8, int) (*PoolReceipt, error)
	LoadLastProjected(context.Context, int8, int) (*PoolReceipt, error)
	MarkPoolOperationFailed(context.Context, PoolReservation, string) (bool, error)
	MarkArtifactDeliveryIfCurrent(context.Context, ArtifactReceipt, string) (bool, error)
	MarkPoolProjectedIfCurrent(context.Context, PoolReceipt) (bool, error)
}

func NewProjectionJournalDAO(logger *zap.Logger, db *gorm.DB) ProjectionJournalDAO {
	return &monitorConfigDAO{db: db, l: logger}
}
