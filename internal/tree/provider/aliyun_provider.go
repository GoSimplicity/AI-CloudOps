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
	sdk := aliyun.NewSDK(os.Getenv("ALIYUN_ACCESS_KEY"), os.Getenv("ALIYUN_SECRET_KEY"))
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
	req := &aliyun.CreateInstanceRequest{
		Region:             region,
		ZoneId:             config.ZoneId,
		ImageId:            config.ImageId,
		InstanceType:       config.InstanceType,
		SecurityGroupIds:   config.SecurityGroupIds,
		VSwitchId:          config.VSwitchId,
		InstanceName:       config.InstanceName,
		Hostname:           config.Hostname,
		Password:           config.Password,
		Description:        config.Description,
		Amount:             config.Amount,
		DryRun:             false,
		InstanceChargeType: config.InstanceChargeType,
		SystemDiskCategory: config.SystemDiskCategory,
		SystemDiskSize:     config.SystemDiskSize,
		DataDiskCategory:   config.DataDiskCategory,
		DataDiskSize:       config.DataDiskSize,
	}

	_, err := a.ecsService.CreateInstance(ctx, req)
	return err
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
	req := &aliyun.DeleteInstanceRequest{
		Region:     region,
		InstanceID: instanceID,
		Force:      true,
	}

	_, err := a.ecsService.DeleteInstance(ctx, req)
	return err
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
	req := &aliyun.GetInstanceDetailRequest{
		Region:     region,
		InstanceID: instanceID,
	}

	resp, err := a.ecsService.GetInstanceDetail(ctx, req)
	if err != nil {
		return nil, err
	}

	instance := resp.Instance
	return &model.ResourceEcs{
		InstanceId:   *instance.InstanceId,
		InstanceName: *instance.InstanceName,
		Status:       *instance.Status,
		RegionId:     *instance.RegionId,
		ZoneId:       *instance.ZoneId,
		InstanceType: *instance.InstanceType,
		// 其他字段根据需要映射
	}, nil
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
	req := &aliyun.ListDisksRequest{
		Region: region,
		Page:   pageNumber,
		Size:   pageSize,
	}

	resp, err := a.ecsService.ListDisks(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	disks := make([]*model.ResourceDisk, 0, len(resp.Disks))
	for _, disk := range resp.Disks {
		disks = append(disks, &model.ResourceDisk{
			DiskID:     *disk.DiskId,
			DiskName:   *disk.DiskName,
			Size:       int(*disk.Size),
			Status:     *disk.Status,
			DiskType:   *disk.Type,
			RegionId:   *disk.RegionId,
			ZoneId:     *disk.ZoneId,
			InstanceID: *disk.InstanceId,
			// 其他字段根据需要映射
		})
	}

	return disks, resp.Total, nil
}

// ListInstances 实现Provider接口
func (a *AliyunProviderImpl) ListInstances(ctx context.Context, region string, page int, size int) ([]*model.ResourceEcs, int64, error) {
	req := &aliyun.ListInstancesRequest{
		Region: region,
		Page:   page,
		Size:   size,
	}

	resp, err := a.ecsService.ListInstances(ctx, req)
	if err != nil {
		return nil, 0, err
	}

	instances := make([]*model.ResourceEcs, 0, len(resp.Instances))
	for _, instance := range resp.Instances {
		instances = append(instances, &model.ResourceEcs{
			InstanceId:   *instance.InstanceId,
			InstanceName: *instance.InstanceName,
			Status:       *instance.Status,
			RegionId:     *instance.RegionId,
			ZoneId:       *instance.ZoneId,
			InstanceType: *instance.InstanceType,
			// 其他字段根据需要映射
		})
	}

	return instances, resp.Total, nil
}

// ListRegionDataDiskCategories 实现Provider接口
func (a *AliyunProviderImpl) ListRegionDataDiskCategories(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	panic("未实现")
}

// ListRegionImages 实现Provider接口
func (a *AliyunProviderImpl) ListRegionImages(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	req := &aliyun.ListImagesRequest{
		Region:          region,
		ImageOwnerAlias: "system",
		Status:          "Available",
	}

	resp, err := a.ecsService.ListImages(ctx, req)
	if err != nil {
		return nil, err
	}

	images := make([]*model.ListEcsResourceOptionsResp, 0, len(resp.Images))
	for _, image := range resp.Images {
		images = append(images, &model.ListEcsResourceOptionsResp{
			Value: *image.ImageId,
			Label: *image.ImageName,
		})
	}

	return images, nil
}

// ListRegionInstanceTypes 实现Provider接口
func (a *AliyunProviderImpl) ListRegionInstanceTypes(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	req := &aliyun.ListInstanceTypesRequest{
		Region:     region,
		MaxResults: 100,
	}

	resp, err := a.ecsService.ListInstanceTypes(ctx, req)
	if err != nil {
		return nil, err
	}

	types := make([]*model.ListEcsResourceOptionsResp, 0, len(resp.InstanceTypes))
	for _, instanceType := range resp.InstanceTypes {
		types = append(types, &model.ListEcsResourceOptionsResp{
			Value: *instanceType.InstanceTypeId,
			Label: *instanceType.InstanceTypeId,
		})
	}

	return types, nil
}

// ListRegionOptions 实现Provider接口
func (a *AliyunProviderImpl) ListRegionOptions(ctx context.Context) ([]*model.ListEcsResourceOptionsResp, error) {
	regions, err := a.ListRegions(ctx)
	if err != nil {
		return nil, err
	}

	options := make([]*model.ListEcsResourceOptionsResp, 0, len(regions))
	for _, region := range regions {
		options = append(options, &model.ListEcsResourceOptionsResp{
			Value: region.RegionId,
			Label: region.LocalName,
		})
	}

	return options, nil
}

// ListRegionSystemDiskCategories 实现Provider接口
func (a *AliyunProviderImpl) ListRegionSystemDiskCategories(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	panic("未实现")
}

// ListRegionZones 实现Provider接口
func (a *AliyunProviderImpl) ListRegionZones(ctx context.Context, region string) ([]*model.ListEcsResourceOptionsResp, error) {
	req := &aliyun.ListZonesRequest{
		Region:         region,
		AcceptLanguage: "zh-CN",
	}

	resp, err := a.ecsService.ListZones(ctx, req)
	if err != nil {
		return nil, err
	}

	zones := make([]*model.ListEcsResourceOptionsResp, 0, len(resp.Zones))
	for _, zone := range resp.Zones {
		zones = append(zones, &model.ListEcsResourceOptionsResp{
			Value: *zone.ZoneId,
			Label: *zone.LocalName,
		})
	}

	return zones, nil
}

// ListRegions 实现Provider接口
func (a *AliyunProviderImpl) ListRegions(ctx context.Context) ([]*model.RegionResp, error) {
	req := &aliyun.ListRegionsRequest{
		AcceptLanguage: "zh-CN",
	}

	resp, err := a.ecsService.ListRegions(ctx, req)
	if err != nil {
		return nil, err
	}

	regions := make([]*model.RegionResp, 0, len(resp.Regions))
	for _, region := range resp.Regions {
		regions = append(regions, &model.RegionResp{
			RegionId:  *region.RegionId,
			LocalName: *region.LocalName,
		})
	}

	return regions, nil
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
	req := &aliyun.RestartInstanceRequest{
		Region:     region,
		InstanceID: instanceID,
	}

	_, err := a.ecsService.RestartInstance(ctx, req)
	return err
}

// StartInstance 实现Provider接口
func (a *AliyunProviderImpl) StartInstance(ctx context.Context, region string, instanceID string) error {
	req := &aliyun.StartInstanceRequest{
		Region:     region,
		InstanceID: instanceID,
	}

	_, err := a.ecsService.StartInstance(ctx, req)
	return err
}

// StopInstance 实现Provider接口
func (a *AliyunProviderImpl) StopInstance(ctx context.Context, region string, instanceID string) error {
	req := &aliyun.StopInstanceRequest{
		Region:     region,
		InstanceID: instanceID,
		ForceStop:  false,
	}

	_, err := a.ecsService.StopInstance(ctx, req)
	return err
}

// SyncResources 实现Provider接口
func (a *AliyunProviderImpl) SyncResources(ctx context.Context, region string) error {
	panic("未实现")
}
