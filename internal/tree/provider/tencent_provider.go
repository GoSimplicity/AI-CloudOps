/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

package provider

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

type TencentProvider interface {
	SyncResources(ctx context.Context, region string) error

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

	ListInstanceOptions(ctx context.Context, payType string, region string, zone string, instanceType string, systemDiskCategory string, dataDiskCategory string) ([]interface{}, error)
}

type tencentProvider struct {
}

func NewTencentProvider() TencentProvider {
	return &tencentProvider{}
}

// AttachDisk implements TencentProvider.
func (t *tencentProvider) AttachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	panic("unimplemented")
}

// CreateDisk implements TencentProvider.
func (t *tencentProvider) CreateDisk(ctx context.Context, region string, config *model.DiskCreationParams) error {
	panic("unimplemented")
}

// CreateInstance implements TencentProvider.
func (t *tencentProvider) CreateInstance(ctx context.Context, region string, config *model.EcsCreationParams) error {
	panic("unimplemented")
}

// CreateVPC implements TencentProvider.
func (t *tencentProvider) CreateVPC(ctx context.Context, region string, config *model.VpcCreationParams) error {
	panic("unimplemented")
}

// DeleteDisk implements TencentProvider.
func (t *tencentProvider) DeleteDisk(ctx context.Context, region string, diskID string) error {
	panic("unimplemented")
}

// DeleteInstance implements TencentProvider.
func (t *tencentProvider) DeleteInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// DeleteVPC implements TencentProvider.
func (t *tencentProvider) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	panic("unimplemented")
}

// DetachDisk implements TencentProvider.
func (t *tencentProvider) DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	panic("unimplemented")
}

// ListDisks implements TencentProvider.
func (t *tencentProvider) ListDisks(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.PageResp, error) {
	panic("unimplemented")
}

// ListInstances implements TencentProvider.
func (t *tencentProvider) ListInstances(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.ResourceECSResp, error) {
	panic("unimplemented")
}

// ListVPCs implements TencentProvider.
func (t *tencentProvider) ListVPCs(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.VpcResp, error) {
	panic("unimplemented")
}

// StartInstance implements TencentProvider.
func (t *tencentProvider) StartInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// StopInstance implements TencentProvider.
func (t *tencentProvider) StopInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

func (t *tencentProvider) SyncResources(ctx context.Context, region string) error {
	panic("unimplemented")
}

func (t *tencentProvider) ListInstanceOptions(ctx context.Context, payType string, region string, zone string, instanceType string, systemDiskCategory string, dataDiskCategory string) ([]interface{}, error) {
	panic("unimplemented")
}
