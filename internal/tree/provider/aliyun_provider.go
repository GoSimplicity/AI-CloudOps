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
	"os"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/aliyun"
	"go.uber.org/zap"
)

type AliyunProviderImpl struct {
	logger               *zap.Logger
	sdk                  *aliyun.SDK
	ecsService           *aliyun.EcsService
	vpcService           *aliyun.VpcService
	securityGroupService *aliyun.SecurityGroupService
}

func NewAliyunProvider(logger *zap.Logger) *AliyunProviderImpl {
	accessKey := os.Getenv("ALIYUN_ACCESS_KEY")
	secretKey := os.Getenv("ALIYUN_SECRET_KEY")
	if accessKey == "" || secretKey == "" {
		logger.Error("ALIYUN_ACCESS_KEY or ALIYUN_SECRET_KEY is not set")
		return nil
	}
	sdk := aliyun.NewSDK(accessKey, secretKey)
	return &AliyunProviderImpl{
		logger:               logger,
		sdk:                  sdk,
		ecsService:           aliyun.NewEcsService(sdk),
		vpcService:           aliyun.NewVpcService(sdk),
		securityGroupService: aliyun.NewSecurityGroupService(sdk),
	}
}
// AttachDisk 实现Provider接口
func (a *AliyunProviderImpl) AttachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	panic("未实现")
}

// CreateDisk 实现Provider接口
func (a *AliyunProviderImpl) CreateDisk(ctx context.Context, region string, config *model.CreateDiskReq) (*model.ResourceDisk, error) {
	panic("未实现")
}

// CreateInstance 实现Provider接口
func (a *AliyunProviderImpl) CreateInstance(ctx context.Context, region string, config *model.CreateEcsResourceReq) error {
	panic("未实现")
}

// CreateSecurityGroup 实现Provider接口
func (a *AliyunProviderImpl) CreateSecurityGroup(ctx context.Context, region string, config *model.CreateSecurityGroupReq) (*model.ResourceSecurityGroup, error) {
	panic("未实现")
}

// CreateVPC 实现Provider接口
func (a *AliyunProviderImpl) CreateVPC(ctx context.Context, region string, config *model.CreateVpcResourceReq) (*model.ResourceVpc, error) {
	panic("未实现")
}

// DeleteDisk 实现Provider接口
func (a *AliyunProviderImpl) DeleteDisk(ctx context.Context, region string, diskID string) error {
	panic("未实现")
}

// DeleteInstance 实现Provider接口
func (a *AliyunProviderImpl) DeleteInstance(ctx context.Context, region string, instanceID string) error {
	panic("未实现")
}

// DeleteSecurityGroup 实现Provider接口
func (a *AliyunProviderImpl) DeleteSecurityGroup(ctx context.Context, region string, securityGroupID string) error {
	panic("未实现")
}

// DeleteVPC 实现Provider接口
func (a *AliyunProviderImpl) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	panic("未实现")
}

// DetachDisk 实现Provider接口
func (a *AliyunProviderImpl) DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	panic("未实现")
}

// GetDisk 实现Provider接口
func (a *AliyunProviderImpl) GetDisk(ctx context.Context, region string, diskID string) (*model.ResourceDisk, error) {
	panic("未实现")
}

// GetInstance 实现Provider接口
func (a *AliyunProviderImpl) GetInstance(ctx context.Context, region string, instanceID string) (*model.ResourceEcs, error) {
	panic("未实现")
}

// GetSecurityGroup 实现Provider接口
func (a *AliyunProviderImpl) GetSecurityGroup(ctx context.Context, region string, securityGroupID string) (*model.ResourceSecurityGroup, error) {
	panic("未实现")
}

// GetVPC 实现Provider接口
func (a *AliyunProviderImpl) GetVPC(ctx context.Context, region string, vpcID string) (*model.ResourceVpc, error) {
	panic("未实现")
}

// ListDisks 实现Provider接口
func (a *AliyunProviderImpl) ListDisks(ctx context.Context, region string, pageNumber int, pageSize int) ([]*model.ResourceDisk, int64, error) {
	panic("未实现")
}

// ListInstances 实现Provider接口
func (a *AliyunProviderImpl) ListInstances(ctx context.Context, region string, page int, size int) ([]*model.ResourceEcs, int64, error) {
	panic("未实现")
}

// ListRegionDataDiskCategories 实现Provider接口
func (a *AliyunProviderImpl) ListRegionDataDiskCategories(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	panic("未实现")
}

// ListRegionImages 实现Provider接口
func (a *AliyunProviderImpl) ListRegionImages(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	panic("未实现")
}

// ListRegionInstanceTypes 实现Provider接口
func (a *AliyunProviderImpl) ListRegionInstanceTypes(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	panic("未实现")
}

// ListRegionOptions 实现Provider接口
func (a *AliyunProviderImpl) ListRegionOptions(ctx context.Context) ([]*model.ListEcsResourceOptionsResp, error) {
	panic("未实现")
}

// ListRegionSystemDiskCategories 实现Provider接口
func (a *AliyunProviderImpl) ListRegionSystemDiskCategories(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	panic("未实现")
}

// ListRegionZones 实现Provider接口
func (a *AliyunProviderImpl) ListRegionZones(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	panic("未实现")
}

// ListRegions 实现Provider接口
func (a *AliyunProviderImpl) ListRegions(ctx context.Context) ([]*model.RegionResp, error) {
	panic("未实现")
}

// ListSecurityGroups 实现Provider接口
func (a *AliyunProviderImpl) ListSecurityGroups(ctx context.Context, region string, pageNumber int, pageSize int) ([]*model.ResourceSecurityGroup, int64, error) {
	panic("未实现")
}

// ListVPCs 实现Provider接口
func (a *AliyunProviderImpl) ListVPCs(ctx context.Context, region string, pageNumber int, pageSize int) ([]*model.ResourceVpc, int64, error) {
	panic("未实现")
}

// RestartInstance 实现Provider接口
func (a *AliyunProviderImpl) RestartInstance(ctx context.Context, region string, instanceID string) error {
	panic("未实现")
}

// StartInstance 实现Provider接口
func (a *AliyunProviderImpl) StartInstance(ctx context.Context, region string, instanceID string) error {
	panic("未实现")
}

// StopInstance 实现Provider接口
func (a *AliyunProviderImpl) StopInstance(ctx context.Context, region string, instanceID string) error {
	panic("未实现")
}

// SyncResources 实现Provider接口
func (a *AliyunProviderImpl) SyncResources(ctx context.Context, region string) error {
	panic("未实现")
}
