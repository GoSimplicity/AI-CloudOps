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

type GCPProviderImpl struct {
}

// AttachDisk implements Provider.
func (g *GCPProviderImpl) AttachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	panic("unimplemented")
}

// CreateDisk implements Provider.
func (g *GCPProviderImpl) CreateDisk(ctx context.Context, region string, config *model.DiskCreationParams) error {
	panic("unimplemented")
}

// CreateInstance implements Provider.
func (g *GCPProviderImpl) CreateInstance(ctx context.Context, region string, config *model.CreateEcsResourceReq) error {
	panic("unimplemented")
}

// CreateVPC implements Provider.
func (g *GCPProviderImpl) CreateVPC(ctx context.Context, region string, config *model.CreateVpcResourceReq) error {
	panic("unimplemented")
}

// DeleteDisk implements Provider.
func (g *GCPProviderImpl) DeleteDisk(ctx context.Context, region string, diskID string) error {
	panic("unimplemented")
}

// DeleteInstance implements Provider.
func (g *GCPProviderImpl) DeleteInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// DeleteVPC implements Provider.
func (g *GCPProviderImpl) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	panic("unimplemented")
}

// DetachDisk implements Provider.
func (g *GCPProviderImpl) DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	panic("unimplemented")
}

// GetInstanceDetail implements Provider.
func (g *GCPProviderImpl) GetInstanceDetail(ctx context.Context, region string, instanceID string) (*model.ResourceEcs, error) {
	panic("unimplemented")
}

// GetZonesByVpc implements Provider.
func (g *GCPProviderImpl) GetZonesByVpc(ctx context.Context, region string, vpcId string) ([]*model.ZoneResp, error) {
	panic("unimplemented")
}

// ListDisks implements Provider.
func (g *GCPProviderImpl) ListDisks(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.PageResp, error) {
	panic("unimplemented")
}

// ListInstanceOptions implements Provider.
func (g *GCPProviderImpl) ListInstanceOptions(ctx context.Context, payType string, region string, zone string, instanceType string, imageId string, systemDiskCategory string, dataDiskCategory string, pageSize int, pageNumber int) ([]*model.ListInstanceOptionsResp, error) {
	panic("unimplemented")
}

// ListInstances implements Provider.
func (g *GCPProviderImpl) ListInstances(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.ResourceEcs, int64, error) {
	panic("unimplemented")
}

// ListRegions implements Provider.
func (g *GCPProviderImpl) ListRegions(ctx context.Context) ([]*model.RegionResp, error) {
	panic("unimplemented")
}

// ListVPCs implements Provider.
func (g *GCPProviderImpl) ListVPCs(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.ResourceVpc, int64, error) {
	panic("unimplemented")
}

// StartInstance implements Provider.
func (g *GCPProviderImpl) StartInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// StopInstance implements Provider.
func (g *GCPProviderImpl) StopInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// SyncResources implements Provider.
func (g *GCPProviderImpl) SyncResources(ctx context.Context, region string) error {
	panic("unimplemented")
}

// RestartInstance implements Provider.
func (g *GCPProviderImpl) RestartInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// GetVpcDetail 获取VPC详情
func (g *GCPProviderImpl) GetVpcDetail(ctx context.Context, region string, vpcID string) (*model.ResourceVpc, error) {
	panic("unimplemented")
}

func NewGCPProvider() *GCPProviderImpl {
	return &GCPProviderImpl{}
}

func (g *GCPProviderImpl) CreateSecurityGroup(ctx context.Context, region string, config *model.CreateSecurityGroupReq) error {
	return nil
}

func (g *GCPProviderImpl) DeleteSecurityGroup(ctx context.Context, region string, securityGroupID string) error {
	return nil
}

func (g *GCPProviderImpl) GetSecurityGroupDetail(ctx context.Context, region string, securityGroupID string) (*model.ResourceSecurityGroup, error) {
	return nil, nil
}

func (g *GCPProviderImpl) ListSecurityGroups(ctx context.Context, region string, pageNumber int, pageSize int) ([]*model.ResourceSecurityGroup, int64, error) {
	return nil, 0, nil
}
