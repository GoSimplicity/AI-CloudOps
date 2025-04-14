package provider

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type HuaweiProvider interface {
	// 资源管理
	ListInstances(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.ResourceECSResp, error)
	CreateInstance(ctx context.Context, region string, config *model.EcsCreationParams) error
	DeleteInstance(ctx context.Context, region string, instanceID string) error
	StartInstance(ctx context.Context, region string, instanceID string) error
	StopInstance(ctx context.Context, region string, instanceID string) error

	// 网络管理
	ListVPCs(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.VpcResp, error)
	CreateVPC(ctx context.Context, region string, config *model.VpcCreationParams) error
	DeleteVPC(ctx context.Context, region string, vpcID string) error

	// 存储管理
	ListDisks(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.PageResp, error)
	CreateDisk(ctx context.Context, region string, config *model.DiskCreationParams) error
	DeleteDisk(ctx context.Context, region string, diskID string) error
	AttachDisk(ctx context.Context, region string, diskID string, instanceID string) error
	DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error
}

type huaweiProvider struct {
}

func NewHuaweiProvider() HuaweiProvider {
	return &huaweiProvider{}
}

// AttachDisk implements HuaweiProvider.
func (h *huaweiProvider) AttachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	panic("unimplemented")
}

// CreateDisk implements HuaweiProvider.
func (h *huaweiProvider) CreateDisk(ctx context.Context, region string, config *model.DiskCreationParams) error {
	panic("unimplemented")
}

// CreateInstance implements HuaweiProvider.
func (h *huaweiProvider) CreateInstance(ctx context.Context, region string, config *model.EcsCreationParams) error {
	panic("unimplemented")
}

// CreateVPC implements HuaweiProvider.
func (h *huaweiProvider) CreateVPC(ctx context.Context, region string, config *model.VpcCreationParams) error {
	panic("unimplemented")
}

// DeleteDisk implements HuaweiProvider.
func (h *huaweiProvider) DeleteDisk(ctx context.Context, region string, diskID string) error {
	panic("unimplemented")
}

// DeleteInstance implements HuaweiProvider.
func (h *huaweiProvider) DeleteInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// DeleteVPC implements HuaweiProvider.
func (h *huaweiProvider) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	panic("unimplemented")
}

// DetachDisk implements HuaweiProvider.
func (h *huaweiProvider) DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	panic("unimplemented")
}

// ListDisks implements HuaweiProvider.
func (h *huaweiProvider) ListDisks(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.PageResp, error) {
	panic("unimplemented")
}

// ListInstances implements HuaweiProvider.
func (h *huaweiProvider) ListInstances(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.ResourceECSResp, error) {
	panic("unimplemented")
}

// ListVPCs implements HuaweiProvider.
func (h *huaweiProvider) ListVPCs(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.VpcResp, error) {
	panic("unimplemented")
}

// StartInstance implements HuaweiProvider.
func (h *huaweiProvider) StartInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// StopInstance implements HuaweiProvider.
func (h *huaweiProvider) StopInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}