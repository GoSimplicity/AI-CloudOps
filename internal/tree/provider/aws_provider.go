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

type AWSProviderImpl struct {
}

// AttachDisk implements Provider.
func (a *AWSProviderImpl) AttachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	panic("unimplemented")
}

// CreateDisk implements Provider.
func (a *AWSProviderImpl) CreateDisk(ctx context.Context, region string, config *model.DiskCreationParams) error {
	panic("unimplemented")
}

// CreateInstance implements Provider.
func (a *AWSProviderImpl) CreateInstance(ctx context.Context, region string, config *model.CreateEcsResourceReq) error {
	panic("unimplemented")
}

// CreateVPC implements Provider.
func (a *AWSProviderImpl) CreateVPC(ctx context.Context, region string, config *model.CreateVpcResourceReq) error {
	panic("unimplemented")
}

// DeleteDisk implements Provider.
func (a *AWSProviderImpl) DeleteDisk(ctx context.Context, region string, diskID string) error {
	panic("unimplemented")
}

// DeleteInstance implements Provider.
func (a *AWSProviderImpl) DeleteInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// DeleteVPC implements Provider.
func (a *AWSProviderImpl) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	panic("unimplemented")
}

// DetachDisk implements Provider.
func (a *AWSProviderImpl) DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	panic("unimplemented")
}

// GetInstanceDetail implements Provider.
func (a *AWSProviderImpl) GetInstanceDetail(ctx context.Context, region string, instanceID string) (*model.ResourceEcs, error) {
	panic("unimplemented")
}

// GetZonesByVpc implements Provider.
func (a *AWSProviderImpl) GetZonesByVpc(ctx context.Context, region string, vpcId string) ([]*model.ZoneResp, error) {
	panic("unimplemented")
}

// ListDisks implements Provider.
func (a *AWSProviderImpl) ListDisks(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.PageResp, error) {
	panic("unimplemented")
}

// ListInstanceOptions implements Provider.
func (a *AWSProviderImpl) ListInstanceOptions(ctx context.Context, payType string, region string, zone string, instanceType string, imageId string, systemDiskCategory string, dataDiskCategory string, pageSize int, pageNumber int) ([]*model.ListInstanceOptionsResp, error) {
	panic("unimplemented")
}

// ListInstances implements Provider.
func (a *AWSProviderImpl) ListInstances(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.ResourceEcs, int64, error) {
	panic("unimplemented")
}

// ListRegions implements Provider.
func (a *AWSProviderImpl) ListRegions(ctx context.Context) ([]*model.RegionResp, error) {
	panic("unimplemented")
}

// ListVPCs implements Provider.
func (a *AWSProviderImpl) ListVPCs(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.ResourceVpc, int64, error) {
	panic("unimplemented")
}

// StartInstance implements Provider.
func (a *AWSProviderImpl) StartInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// StopInstance implements Provider.
func (a *AWSProviderImpl) StopInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// SyncResources implements Provider.
func (a *AWSProviderImpl) SyncResources(ctx context.Context, region string) error {
	panic("unimplemented")
}

// RestartInstance implements Provider.
func (a *AWSProviderImpl) RestartInstance(ctx context.Context, region string, instanceID string) error {
	panic("unimplemented")
}

// GetVpcDetail 获取VPC详情
func (a *AWSProviderImpl) GetVpcDetail(ctx context.Context, region string, vpcID string) (*model.ResourceVpc, error) {
	panic("unimplemented")
}

func NewAWSProvider() *AWSProviderImpl {
	return &AWSProviderImpl{}
}

func (a *AWSProviderImpl) CreateSecurityGroup(ctx context.Context, region string, config *model.CreateSecurityGroupReq) error {
	return nil
}

func (a *AWSProviderImpl) DeleteSecurityGroup(ctx context.Context, region string, securityGroupID string) error {
	return nil
}

func (a *AWSProviderImpl) GetSecurityGroupDetail(ctx context.Context, region string, securityGroupID string) (*model.ResourceSecurityGroup, error) {
	return nil, nil
}

func (a *AWSProviderImpl) ListSecurityGroups(ctx context.Context, region string, pageNumber int, pageSize int) ([]*model.ResourceSecurityGroup, int64, error) {
	return nil, 0, nil
}
