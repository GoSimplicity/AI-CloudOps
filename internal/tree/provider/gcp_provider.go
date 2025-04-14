package provider

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type GcpProvider interface {
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

type gcpProvider struct {
}

func NewGcpProvider() GcpProvider {
	return &gcpProvider{}
}


// AttachDisk implements GcpProvider.
func (g *gcpProvider) AttachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	panic("unimplemented")
}

// CreateDisk implements GcpProvider.
func (g *gcpProvider) CreateDisk(ctx context.Context, region string, config *model.DiskCreationParams) error {
	panic("unimplemented")
}

// CreateInstance implements GcpProvider.
func (g *gcpProvider) CreateInstance(ctx context.Context, region string, config *model.EcsCreationParams) error {
	panic("unimplemented")
}

// CreateVPC implements GcpProvider.
func (g *gcpProvider) CreateVPC(ctx context.Context, region string, config *model.VpcCreationParams) error {
	panic("unimplemented")
}

// DeleteDisk implements GcpProvider.
func (g *gcpProvider) DeleteDisk(ctx context.Context, region string, diskID string) error {
	panic("unimplemented")
}

// DeleteInstance implements GcpProvider.
func (g *gcpProvider) DeleteInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// DeleteVPC implements GcpProvider.
func (g *gcpProvider) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	panic("unimplemented")
}

// DetachDisk implements GcpProvider.
func (g *gcpProvider) DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	panic("unimplemented")
}

// ListDisks implements GcpProvider.
func (g *gcpProvider) ListDisks(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.PageResp, error) {
	panic("unimplemented")
}

// ListInstances implements GcpProvider.
func (g *gcpProvider) ListInstances(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.ResourceECSResp, error) {
	panic("unimplemented")
}

// ListVPCs implements GcpProvider.
func (g *gcpProvider) ListVPCs(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.VpcResp, error) {
	panic("unimplemented")
}

// StartInstance implements GcpProvider.
func (g *gcpProvider) StartInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// StopInstance implements GcpProvider.
func (g *gcpProvider) StopInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}