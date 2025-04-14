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
	"fmt"
	"os"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	openapiv2 "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	vpc "github.com/alibabacloud-go/vpc-20160428/v2/client"
	"go.uber.org/zap"
)

type AliyunProvider interface {
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

type aliyunProvider struct {
	logger          *zap.Logger
	accessKeyId     string
	accessKeySecret string
}

func NewAliyunProvider(logger *zap.Logger) AliyunProvider {
	accessKeyId := os.Getenv("ALIYUN_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ALIYUN_ACCESS_KEY_SECRET")
	return &aliyunProvider{
		logger:          logger,
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
	}
}

// 创建ECS客户端
func (a *aliyunProvider) createEcsClient(region string) (*ecs.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(a.accessKeyId),
		AccessKeySecret: tea.String(a.accessKeySecret),
		RegionId:        tea.String(region),
	}
	return ecs.NewClient(config)
}

// 创建VPC客户端
func (a *aliyunProvider) createVpcClient(region string) (*vpc.Client, error) {
	config := &openapiv2.Config{
		AccessKeyId:     tea.String(a.accessKeyId),
		AccessKeySecret: tea.String(a.accessKeySecret),
		RegionId:        tea.String(region),
	}
	return vpc.NewClient(config)
}

// CreateInstance 创建ECS实例
func (a *aliyunProvider) CreateInstance(ctx context.Context, region string, config *model.EcsCreationParams) error {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.RunInstancesRequest{
		RegionId:         tea.String(region),
		ZoneId:           tea.String(config.ZoneId),
		ImageId:          tea.String(config.ImageId),
		InstanceType:     tea.String(config.InstanceType),
		SecurityGroupIds: tea.StringSlice(config.SecurityGroupIds),
		VSwitchId:        tea.String(config.VSwitchId),
		InstanceName:     tea.String(config.InstanceName),
		HostName:         tea.String(config.HostnamePrefix),
		Description:      tea.String(config.Description),
		Amount:           tea.Int32(int32(config.Quantity)),
		DryRun:           tea.Bool(config.DryRun),
	}

	// 设置系统盘
	if config.SystemDiskCategory != "" {
		request.SystemDisk = &ecs.RunInstancesRequestSystemDisk{
			Category: tea.String(config.SystemDiskCategory),
		}
	}

	// 设置标签
	if len(config.Tags) > 0 {
		tags := make([]*ecs.RunInstancesRequestTag, 0, len(config.Tags))
		for k, v := range config.Tags {
			tags = append(tags, &ecs.RunInstancesRequestTag{
				Key:   tea.String(k),
				Value: tea.String(v),
			})
		}
		request.Tag = tags
	}

	a.logger.Info("开始创建ECS实例", zap.String("region", region), zap.Any("config", config))
	response, err := client.RunInstances(request)
	if err != nil {
		a.logger.Error("创建ECS实例失败", zap.Error(err))
		return err
	}

	a.logger.Info("创建ECS实例成功",
		zap.Strings("instanceIds", tea.StringSliceValue(response.Body.InstanceIdSets.InstanceIdSet)))
	return nil
}

// StartInstance 启动ECS实例
func (a *aliyunProvider) StartInstance(ctx context.Context, region string, instanceID string) error {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.StartInstanceRequest{
		InstanceId: tea.String(instanceID),
	}

	a.logger.Info("开始启动ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
	_, err = client.StartInstance(request)
	if err != nil {
		a.logger.Error("启动ECS实例失败", zap.Error(err))
		return err
	}

	a.logger.Info("启动ECS实例成功", zap.String("instanceID", instanceID))
	return nil
}

// StopInstance 停止ECS实例
func (a *aliyunProvider) StopInstance(ctx context.Context, region string, instanceID string) error {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.StopInstanceRequest{
		InstanceId: tea.String(instanceID),
		ForceStop:  tea.Bool(false),
	}

	a.logger.Info("开始停止ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
	_, err = client.StopInstance(request)
	if err != nil {
		a.logger.Error("停止ECS实例失败", zap.Error(err))
		return err
	}

	a.logger.Info("停止ECS实例成功", zap.String("instanceID", instanceID))
	return nil
}

// DeleteInstance 删除ECS实例
func (a *aliyunProvider) DeleteInstance(ctx context.Context, region string, instanceID string) error {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.DeleteInstanceRequest{
		InstanceId: tea.String(instanceID),
		Force:      tea.Bool(true),
	}

	a.logger.Info("开始删除ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
	_, err = client.DeleteInstance(request)
	if err != nil {
		a.logger.Error("删除ECS实例失败", zap.Error(err))
		return err
	}

	a.logger.Info("删除ECS实例成功", zap.String("instanceID", instanceID))
	return nil
}

// AttachDisk 挂载磁盘
func (a *aliyunProvider) AttachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.AttachDiskRequest{
		DiskId:     tea.String(diskID),
		InstanceId: tea.String(instanceID),
	}

	a.logger.Info("开始挂载磁盘", zap.String("region", region), zap.String("diskID", diskID), zap.String("instanceID", instanceID))
	_, err = client.AttachDisk(request)
	if err != nil {
		a.logger.Error("挂载磁盘失败", zap.Error(err))
		return err
	}

	a.logger.Info("挂载磁盘成功", zap.String("diskID", diskID), zap.String("instanceID", instanceID))
	return nil
}

// CreateDisk 创建磁盘
func (a *aliyunProvider) CreateDisk(ctx context.Context, region string, config *model.DiskCreationParams) error {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.CreateDiskRequest{
		RegionId:     tea.String(region),
		ZoneId:       tea.String(config.ZoneId),
		DiskName:     tea.String(config.DiskName),
		DiskCategory: tea.String(config.DiskCategory),
		Size:         tea.Int32(int32(config.Size)),
		Description:  tea.String(config.Description),
	}

	// 设置标签
	if len(config.Tags) > 0 {
		tags := make([]*ecs.CreateDiskRequestTag, 0, len(config.Tags))
		for k, v := range config.Tags {
			tags = append(tags, &ecs.CreateDiskRequestTag{
				Key:   tea.String(k),
				Value: tea.String(v),
			})
		}
		request.Tag = tags
	}

	a.logger.Info("开始创建磁盘", zap.String("region", region), zap.Any("config", config))
	response, err := client.CreateDisk(request)
	if err != nil {
		a.logger.Error("创建磁盘失败", zap.Error(err))
		return err
	}

	a.logger.Info("创建磁盘成功", zap.String("diskID", tea.StringValue(response.Body.DiskId)))
	return nil
}

// CreateVPC 创建VPC
func (a *aliyunProvider) CreateVPC(ctx context.Context, region string, config *model.VpcCreationParams) error {
	client, err := a.createVpcClient(region)
	if err != nil {
		a.logger.Error("创建VPC客户端失败", zap.Error(err))
		return err
	}

	// 创建VPC
	vpcRequest := &vpc.CreateVpcRequest{
		RegionId:    tea.String(region),
		VpcName:     tea.String(config.VpcName),
		CidrBlock:   tea.String(config.CidrBlock),
		Description: tea.String(config.Description),
	}

	a.logger.Info("开始创建VPC", zap.String("region", region), zap.Any("config", config))
	vpcResponse, err := client.CreateVpc(vpcRequest)
	if err != nil {
		a.logger.Error("创建VPC失败", zap.Error(err))
		return err
	}

	vpcId := tea.StringValue(vpcResponse.Body.VpcId)
	a.logger.Info("创建VPC成功", zap.String("vpcID", vpcId))

	// 等待VPC可用
	err = a.waitForVpcAvailable(client, region, vpcId)
	if err != nil {
		a.logger.Error("等待VPC可用失败", zap.Error(err))
		return err
	}

	// 创建交换机
	vSwitchRequest := &vpc.CreateVSwitchRequest{
		RegionId:    tea.String(region),
		ZoneId:      tea.String(config.ZoneId),
		VpcId:       tea.String(vpcId),
		VSwitchName: tea.String(config.VSwitchName),
		CidrBlock:   tea.String(config.VSwitchCidrBlock),
		Description: tea.String(config.Description),
	}

	a.logger.Info("开始创建交换机", zap.String("vpcID", vpcId), zap.String("vSwitchName", config.VSwitchName))
	vSwitchResponse, err := client.CreateVSwitch(vSwitchRequest)
	if err != nil {
		a.logger.Error("创建交换机失败", zap.Error(err))
		return err
	}

	a.logger.Info("创建交换机成功", zap.String("vSwitchID", tea.StringValue(vSwitchResponse.Body.VSwitchId)))
	return nil
}

// 等待VPC可用
func (a *aliyunProvider) waitForVpcAvailable(client *vpc.Client, region string, vpcId string) error {
	request := &vpc.DescribeVpcAttributeRequest{
		RegionId: tea.String(region),
		VpcId:    tea.String(vpcId),
	}

	for i := 0; i < 10; i++ {
		response, err := client.DescribeVpcAttribute(request)
		if err != nil {
			return err
		}

		if tea.StringValue(response.Body.Status) == "Available" {
			return nil
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("等待VPC可用超时")
}

// DeleteDisk 删除磁盘
func (a *aliyunProvider) DeleteDisk(ctx context.Context, region string, diskID string) error {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.DeleteDiskRequest{
		DiskId: tea.String(diskID),
	}

	a.logger.Info("开始删除磁盘", zap.String("region", region), zap.String("diskID", diskID))
	_, err = client.DeleteDisk(request)
	if err != nil {
		a.logger.Error("删除磁盘失败", zap.Error(err))
		return err
	}

	a.logger.Info("删除磁盘成功", zap.String("diskID", diskID))
	return nil
}

// DeleteVPC 删除VPC
func (a *aliyunProvider) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	client, err := a.createVpcClient(region)
	if err != nil {
		a.logger.Error("创建VPC客户端失败", zap.Error(err))
		return err
	}

	// 先查询并删除所有交换机
	vSwitchRequest := &vpc.DescribeVSwitchesRequest{
		RegionId: tea.String(region),
		VpcId:    tea.String(vpcID),
	}

	a.logger.Info("查询VPC下的交换机", zap.String("region", region), zap.String("vpcID", vpcID))
	vSwitchResponse, err := client.DescribeVSwitches(vSwitchRequest)
	if err != nil {
		a.logger.Error("查询交换机失败", zap.Error(err))
		return err
	}

	for _, vSwitch := range vSwitchResponse.Body.VSwitches.VSwitch {
		deleteVSwitchRequest := &vpc.DeleteVSwitchRequest{
			VSwitchId: vSwitch.VSwitchId,
		}

		a.logger.Info("删除交换机", zap.String("vSwitchID", tea.StringValue(vSwitch.VSwitchId)))
		_, err = client.DeleteVSwitch(deleteVSwitchRequest)
		if err != nil {
			a.logger.Error("删除交换机失败", zap.Error(err))
			return err
		}
	}

	// 删除VPC
	request := &vpc.DeleteVpcRequest{
		VpcId: tea.String(vpcID),
	}

	a.logger.Info("开始删除VPC", zap.String("region", region), zap.String("vpcID", vpcID))
	_, err = client.DeleteVpc(request)
	if err != nil {
		a.logger.Error("删除VPC失败", zap.Error(err))
		return err
	}

	a.logger.Info("删除VPC成功", zap.String("vpcID", vpcID))
	return nil
}

// DetachDisk 卸载磁盘
func (a *aliyunProvider) DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.DetachDiskRequest{
		DiskId:     tea.String(diskID),
		InstanceId: tea.String(instanceID),
	}

	a.logger.Info("开始卸载磁盘", zap.String("region", region), zap.String("diskID", diskID), zap.String("instanceID", instanceID))
	_, err = client.DetachDisk(request)
	if err != nil {
		a.logger.Error("卸载磁盘失败", zap.Error(err))
		return err
	}

	a.logger.Info("卸载磁盘成功", zap.String("diskID", diskID), zap.String("instanceID", instanceID))
	return nil
}

// ListDisks 列出磁盘
func (a *aliyunProvider) ListDisks(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.PageResp, error) {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	request := &ecs.DescribeDisksRequest{
		RegionId:   tea.String(region),
		PageSize:   tea.Int32(int32(pageSize)),
		PageNumber: tea.Int32(int32(pageNumber)),
	}

	a.logger.Info("开始查询磁盘列表", zap.String("region", region))
	response, err := client.DescribeDisks(request)
	if err != nil {
		a.logger.Error("查询磁盘列表失败", zap.Error(err))
		return nil, err
	}

	total := int64(tea.Int32Value(response.Body.TotalCount))
	a.logger.Info("查询磁盘列表成功", zap.Int64("total", total))

	// 这里需要根据实际情况转换为PageResp
	result := []*model.PageResp{
		{
			Total:    total,
			Page:     pageNumber,
			PageSize: pageSize,
			Data:     response.Body.Disks.Disk,
		},
	}

	return result, nil
}

// ListInstances 列出ECS实例
func (a *aliyunProvider) ListInstances(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.ResourceECSResp, error) {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	request := &ecs.DescribeInstancesRequest{
		RegionId:   tea.String(region),
		PageSize:   tea.Int32(int32(pageSize)),
		PageNumber: tea.Int32(int32(pageNumber)),
	}

	a.logger.Info("开始查询ECS实例列表", zap.String("region", region))
	response, err := client.DescribeInstances(request)
	if err != nil {
		a.logger.Error("查询ECS实例列表失败", zap.Error(err))
		return nil, err
	}

	total := int64(tea.Int32Value(response.Body.TotalCount))
	a.logger.Info("查询ECS实例列表成功", zap.Int64("total", total))

	// 转换为ResourceECSResp
	result := make([]*model.ResourceECSResp, 0, len(response.Body.Instances.Instance))
	for _, instance := range response.Body.Instances.Instance {
		// 安全处理IP地址，避免空切片导致的索引越界
		privateIp := ""
		if len(instance.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
			privateIp = tea.StringValue(instance.VpcAttributes.PrivateIpAddress.IpAddress[0])
		}

		publicIp := ""
		if len(instance.PublicIpAddress.IpAddress) > 0 {
			publicIp = tea.StringValue(instance.PublicIpAddress.IpAddress[0])
		}

		ecsResp := &model.ResourceECSResp{
			ResourceEcs: model.ResourceEcs{
				ComputeResource: model.ComputeResource{
					ResourceBase: model.ResourceBase{
						InstanceName:     tea.StringValue(instance.InstanceName),
						InstanceId:       tea.StringValue(instance.InstanceId),
						Provider:         model.CloudProvider(tea.StringValue(instance.RegionId)),
						Region:           tea.StringValue(instance.RegionId),
						ZoneId:           tea.StringValue(instance.ZoneId),
						VpcId:            tea.StringValue(instance.VpcAttributes.VpcId),
						Status:           model.ResourceStatus(tea.StringValue(instance.Status)),
						CreationTime:     tea.StringValue(instance.CreationTime),
						Description:      tea.StringValue(instance.Description),
						PrivateIpAddress: privateIp,
						PublicIpAddress:  publicIp,
					},
					Cpu:          int(tea.Int32Value(instance.Cpu)),
					Memory:       int(tea.Int32Value(instance.Memory)) / 1024, // 转换为GB
					InstanceType: tea.StringValue(instance.InstanceType),
					IpAddr:       privateIp,
				},
				OsType:   tea.StringValue(instance.OSType),
				OSName:   tea.StringValue(instance.OSName),
				Hostname: tea.StringValue(instance.HostName),
			},
			CreatedAt: tea.StringValue(instance.CreationTime),
			UpdatedAt: time.Now().Format(time.RFC3339),
		}
		result = append(result, ecsResp)
	}

	return result, nil
}

// ListVPCs 列出VPC
func (a *aliyunProvider) ListVPCs(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.VpcResp, error) {
	client, err := a.createVpcClient(region)
	if err != nil {
		a.logger.Error("创建VPC客户端失败", zap.Error(err))
		return nil, err
	}

	request := &vpc.DescribeVpcsRequest{
		RegionId:   tea.String(region),
		PageSize:   tea.Int32(int32(pageSize)),
		PageNumber: tea.Int32(int32(pageNumber)),
	}

	a.logger.Info("开始查询VPC列表", zap.String("region", region))
	response, err := client.DescribeVpcs(request)
	if err != nil {
		a.logger.Error("查询VPC列表失败", zap.Error(err))
		return nil, err
	}

	total := int64(tea.Int32Value(response.Body.TotalCount))
	a.logger.Info("查询VPC列表成功", zap.Int64("total", total))

	// 转换为VpcResp
	result := make([]*model.VpcResp, 0, len(response.Body.Vpcs.Vpc))
	for _, vpc := range response.Body.Vpcs.Vpc {
		vpcResp := &model.VpcResp{
			VpcId:       tea.StringValue(vpc.VpcId),
			VpcName:     tea.StringValue(vpc.VpcName),
			CidrBlock:   tea.StringValue(vpc.CidrBlock),
			Description: tea.StringValue(vpc.Description),
		}
		result = append(result, vpcResp)
	}

	return result, nil
}
