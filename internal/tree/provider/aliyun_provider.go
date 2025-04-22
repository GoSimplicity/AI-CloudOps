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
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	openapi "github.com/alibabacloud-go/darabonba-openapi/client"
	openapiv2 "github.com/alibabacloud-go/darabonba-openapi/v2/client"
	ecs "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	vpc "github.com/alibabacloud-go/vpc-20160428/v2/client"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
)

type AliyunProviderImpl struct {
	logger          *zap.Logger
	dao             dao.ResourceDAO
	accessKeyId     string
	accessKeySecret string
}

func NewAliyunProvider(logger *zap.Logger, dao dao.ResourceDAO) *AliyunProviderImpl {
	accessKeyId := os.Getenv("ALIYUN_ACCESS_KEY_ID")
	accessKeySecret := os.Getenv("ALIYUN_ACCESS_KEY_SECRET")
	return &AliyunProviderImpl{
		logger:          logger,
		dao:             dao,
		accessKeyId:     accessKeyId,
		accessKeySecret: accessKeySecret,
	}
}

// 创建ECS客户端
func (a *AliyunProviderImpl) createEcsClient(region string) (*ecs.Client, error) {
	config := &openapi.Config{
		AccessKeyId:     tea.String(a.accessKeyId),
		AccessKeySecret: tea.String(a.accessKeySecret),
		RegionId:        tea.String(region),
		Endpoint:        tea.String("ecs.aliyuncs.com"),
	}
	return ecs.NewClient(config)
}

// 创建VPC客户端
func (a *AliyunProviderImpl) createVpcClient(region string) (*vpc.Client, error) {
	config := &openapiv2.Config{
		AccessKeyId:     tea.String(a.accessKeyId),
		AccessKeySecret: tea.String(a.accessKeySecret),
		RegionId:        tea.String(region),
	}
	return vpc.NewClient(config)
}

// CreateInstance 创建ECS实例
func (a *AliyunProviderImpl) CreateInstance(ctx context.Context, region string, config *model.CreateEcsResourceReq) error {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.RunInstancesRequest{
		RegionId:           tea.String(region),
		ZoneId:             tea.String(config.ZoneId),
		ImageId:            tea.String(config.ImageId),
		InstanceType:       tea.String(config.InstanceType),
		SecurityGroupIds:   tea.StringSlice(config.SecurityGroupIds),
		VSwitchId:          tea.String(config.VSwitchId),
		InstanceName:       tea.String(config.InstanceName),
		HostName:           tea.String(config.Hostname),
		Password:           tea.String(config.Password),
		Description:        tea.String(config.Description),
		Amount:             tea.Int32(int32(config.Amount)),
		DryRun:             tea.Bool(config.DryRun),
		InstanceChargeType: tea.String(string(config.InstanceChargeType)),
	}

	// 设置系统盘
	if config.SystemDiskCategory != "" {
		request.SystemDisk = &ecs.RunInstancesRequestSystemDisk{
			Category: tea.String(config.SystemDiskCategory),
			Size:     tea.String(strconv.Itoa(config.SystemDiskSize)),
		}
	}

	// 设置数据盘
	if config.DataDiskCategory != "" {
		request.DataDisk = []*ecs.RunInstancesRequestDataDisk{
			{
				Category: tea.String(config.DataDiskCategory),
				Size:     tea.Int32(int32(config.DataDiskSize)),
			},
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
func (a *AliyunProviderImpl) StartInstance(ctx context.Context, region string, instanceID string) error {
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
func (a *AliyunProviderImpl) StopInstance(ctx context.Context, region string, instanceID string) error {
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

// RestartInstance 重启ECS实例
func (a *AliyunProviderImpl) RestartInstance(ctx context.Context, region string, instanceID string) error {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.RebootInstanceRequest{
		InstanceId: tea.String(instanceID),
	}

	a.logger.Info("开始重启ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
	_, err = client.RebootInstance(request)
	if err != nil {
		a.logger.Error("重启ECS实例失败", zap.Error(err))
		return err
	}

	a.logger.Info("重启ECS实例成功", zap.String("instanceID", instanceID))
	return nil
}

// DeleteInstance 删除ECS实例
func (a *AliyunProviderImpl) DeleteInstance(ctx context.Context, region string, instanceID string) error {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.DeleteInstanceRequest{
		InstanceId: tea.String(instanceID),
		Force:      tea.Bool(true), // 强制删除
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
func (a *AliyunProviderImpl) AttachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
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
func (a *AliyunProviderImpl) CreateDisk(ctx context.Context, region string, config *model.DiskCreationParams) error {
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
func (a *AliyunProviderImpl) CreateVPC(ctx context.Context, region string, config *model.CreateVpcResourceReq) error {
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
func (a *AliyunProviderImpl) waitForVpcAvailable(client *vpc.Client, region string, vpcId string) error {
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
func (a *AliyunProviderImpl) DeleteDisk(ctx context.Context, region string, diskID string) error {
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
func (a *AliyunProviderImpl) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	client, err := a.createVpcClient(region)
	if err != nil {
		a.logger.Error("创建VPC客户端失败", zap.Error(err))
		return err
	}

	// 先查询VPC详情，获取相关资源
	vpcDetailRequest := &vpc.DescribeVpcAttributeRequest{
		RegionId: tea.String(region),
		VpcId:    tea.String(vpcID),
	}

	_, err = client.DescribeVpcAttribute(vpcDetailRequest)
	if err != nil {
		a.logger.Error("查询VPC详情失败", zap.Error(err))
		return err
	}

	// 查询并删除所有交换机
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

	// 检查交换机是否有依赖资源
	for _, vSwitch := range vSwitchResponse.Body.VSwitches.VSwitch {
		vSwitchId := tea.StringValue(vSwitch.VSwitchId)

		// 检查交换机下是否有ECS实例
		ecsClient, err := a.createEcsClient(region)
		if err == nil {
			ecsRequest := &ecs.DescribeInstancesRequest{
				RegionId:  tea.String(region),
				VSwitchId: tea.String(vSwitchId),
			}
			ecsResponse, err := ecsClient.DescribeInstances(ecsRequest)
			if err == nil && ecsResponse.Body != nil && len(ecsResponse.Body.Instances.Instance) > 0 {
				a.logger.Warn("交换机下存在ECS实例，无法删除", zap.String("vSwitchID", vSwitchId))
				return fmt.Errorf("交换机(%s)下存在ECS实例，请先删除相关实例", vSwitchId)
			}
		}

		// 删除交换机
		deleteVSwitchRequest := &vpc.DeleteVSwitchRequest{
			VSwitchId: tea.String(vSwitchId),
		}

		a.logger.Info("删除交换机", zap.String("vSwitchID", vSwitchId))
		_, err = client.DeleteVSwitch(deleteVSwitchRequest)
		if err != nil {
			a.logger.Error("删除交换机失败", zap.Error(err))
			return fmt.Errorf("删除交换机(%s)失败: %w", vSwitchId, err)
		}

		time.Sleep(5 * time.Second)
	}

	// 检查并删除NAT网关
	natClient, err := a.createVpcClient(region)
	if err == nil {
		natRequest := &vpc.DescribeNatGatewaysRequest{
			RegionId: tea.String(region),
			VpcId:    tea.String(vpcID),
		}
		natResponse, err := natClient.DescribeNatGateways(natRequest)
		if err == nil && natResponse.Body != nil {
			for _, nat := range natResponse.Body.NatGateways.NatGateway {
				natId := tea.StringValue(nat.NatGatewayId)
				deleteNatRequest := &vpc.DeleteNatGatewayRequest{
					RegionId:     tea.String(region),
					NatGatewayId: tea.String(natId),
				}
				a.logger.Info("删除NAT网关", zap.String("natGatewayID", natId))
				_, err = natClient.DeleteNatGateway(deleteNatRequest)
				if err != nil {
					a.logger.Error("删除NAT网关失败", zap.Error(err))
					return fmt.Errorf("删除NAT网关(%s)失败: %w", natId, err)
				}
				time.Sleep(5 * time.Second)
			}
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
func (a *AliyunProviderImpl) DetachDisk(ctx context.Context, region string, diskID string, instanceID string) error {
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
func (a *AliyunProviderImpl) ListDisks(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.PageResp, error) {
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
			Total: total,
			Data:  response.Body.Disks.Disk,
		},
	}

	return result, nil
}

// ListInstances 列出ECS实例
func (a *AliyunProviderImpl) ListInstances(ctx context.Context, region string, pageSize int, pageNumber int) ([]*model.ResourceEcs, int64, error) {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, 0, err
	}
	request := &ecs.DescribeInstancesRequest{
		RegionId:   tea.String(region),
		PageSize:   tea.Int32(int32(pageSize)),
		PageNumber: tea.Int32(int32(pageNumber)),
	}

	response, err := client.DescribeInstances(request)
	if err != nil {
		a.logger.Error("查询ECS实例列表失败", zap.Error(err))
		return nil, 0, err
	}

	total := int64(tea.Int32Value(response.Body.TotalCount))

	instances := response.Body.Instances.Instance
	result := make([]*model.ResourceEcs, len(instances))
	for i, instance := range instances {
		// 将阿里云ECS实例转换为我们的模型
		result[i] = &model.ResourceEcs{
			ComputeResource: model.ComputeResource{
				ResourceBase: model.ResourceBase{
					InstanceName:       tea.StringValue(instance.InstanceName),
					InstanceId:         tea.StringValue(instance.InstanceId),
					Provider:           model.CloudProviderAliyun,
					RegionId:           tea.StringValue(instance.RegionId),
					ZoneId:             tea.StringValue(instance.ZoneId),
					VpcId:              tea.StringValue(instance.VpcAttributes.VpcId),
					Status:             tea.StringValue(instance.Status),
					CreationTime:       tea.StringValue(instance.CreationTime),
					InstanceChargeType: tea.StringValue(instance.InstanceChargeType),
					Description:        tea.StringValue(instance.Description),
					SecurityGroupIds:   model.StringList(tea.StringSliceValue(instance.SecurityGroupIds.SecurityGroupId)),
					PrivateIpAddress:   model.StringList(tea.StringSliceValue(instance.VpcAttributes.PrivateIpAddress.IpAddress)),
					PublicIpAddress:    model.StringList(tea.StringSliceValue(instance.PublicIpAddress.IpAddress)),
					LastSyncTime:       time.Now(),
				},
				Cpu:          int(tea.Int32Value(instance.Cpu)),
				Memory:       int(tea.Int32Value(instance.Memory)) / 1024,
				InstanceType: tea.StringValue(instance.InstanceType),
				ImageId:      tea.StringValue(instance.ImageId),
				HostName:     tea.StringValue(instance.HostName),
				IpAddr:       tea.StringValue(instance.VpcAttributes.PrivateIpAddress.IpAddress[0]),
			},
			OsType:          tea.StringValue(instance.OSType),
			OSName:          tea.StringValue(instance.OSName),
			StartTime:       tea.StringValue(instance.StartTime),
			AutoReleaseTime: tea.StringValue(instance.AutoReleaseTime),
		}
	}

	if len(result) == 0 {
		return []*model.ResourceEcs{}, 0, nil
	}

	return result, total, nil
}

// GetInstanceDetail 获取ECS实例详情
func (a *AliyunProviderImpl) GetInstanceDetail(ctx context.Context, region string, instanceID string) (*model.ResourceEcs, error) {
	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	request := &ecs.DescribeInstanceAttributeRequest{
		InstanceId: tea.String(instanceID),
	}

	response, err := client.DescribeInstanceAttribute(request)
	if err != nil {
		a.logger.Error("查询ECS实例详情失败", zap.Error(err))
		return nil, err
	}

	instance := response.Body
	result := &model.ResourceEcs{
		ComputeResource: model.ComputeResource{
			ResourceBase: model.ResourceBase{
				InstanceName:       tea.StringValue(instance.InstanceName),
				InstanceId:         tea.StringValue(instance.InstanceId),
				Provider:           model.CloudProviderAliyun,
				RegionId:           tea.StringValue(instance.RegionId),
				ZoneId:             tea.StringValue(instance.ZoneId),
				VpcId:              tea.StringValue(instance.VpcAttributes.VpcId),
				Status:             tea.StringValue(instance.Status),
				CreationTime:       tea.StringValue(instance.CreationTime),
				InstanceChargeType: tea.StringValue(instance.InstanceChargeType),
				Description:        tea.StringValue(instance.Description),
				SecurityGroupIds:   model.StringList(tea.StringSliceValue(instance.SecurityGroupIds.SecurityGroupId)),
				PrivateIpAddress:   model.StringList(tea.StringSliceValue(instance.VpcAttributes.PrivateIpAddress.IpAddress)),
				PublicIpAddress:    model.StringList(tea.StringSliceValue(instance.PublicIpAddress.IpAddress)),
				LastSyncTime:       time.Now(),
			},
			Cpu:          int(tea.Int32Value(instance.Cpu)),
			Memory:       int(tea.Int32Value(instance.Memory)) / 1024,
			InstanceType: tea.StringValue(instance.InstanceType),
			ImageId:      tea.StringValue(instance.ImageId),
			HostName:     tea.StringValue(instance.HostName),
			IpAddr:       tea.StringValue(instance.VpcAttributes.PrivateIpAddress.IpAddress[0]),
		},
	}

	return result, nil
}

// ListVPCs 获取VPC列表
func (a *AliyunProviderImpl) ListVPCs(ctx context.Context, region string, pageNumber int, pageSize int) ([]*model.ResourceVpc, int64, error) {
	client, err := a.createVpcClient(region)
	if err != nil {
		a.logger.Error("创建VPC客户端失败", zap.Error(err))
		return nil, 0, err
	}

	request := &vpc.DescribeVpcsRequest{
		RegionId:   tea.String(region),
		PageNumber: tea.Int32(int32(pageNumber)),
		PageSize:   tea.Int32(int32(pageSize)),
	}

	response, err := client.DescribeVpcs(request)
	if err != nil {
		a.logger.Error("查询VPC列表失败", zap.Error(err))
		return nil, 0, err
	}

	total := int64(tea.Int32Value(response.Body.TotalCount))
	a.logger.Info("查询VPC列表成功", zap.Int64("total", total))

	result := make([]*model.ResourceVpc, len(response.Body.Vpcs.Vpc))
	for i, vpc := range response.Body.Vpcs.Vpc {
		var tags []string
		if vpc.Tags != nil && vpc.Tags.Tag != nil {
			tags = make([]string, 0, len(vpc.Tags.Tag))
			for _, tag := range vpc.Tags.Tag {
				if tag == nil || tag.Key == nil || tag.Value == nil {
					continue
				}
				tags = append(tags, fmt.Sprintf("%s=%s", tea.StringValue(tag.Key), tea.StringValue(tag.Value)))
			}
		}
		result[i] = &model.ResourceVpc{
			ResourceBase: model.ResourceBase{
				InstanceName: tea.StringValue(vpc.VpcName),
				InstanceId:   tea.StringValue(vpc.VpcId),
				Provider:     model.CloudProviderAliyun,
				RegionId:     tea.StringValue(vpc.RegionId),
				VpcId:        tea.StringValue(vpc.VpcId),
				Status:       tea.StringValue(vpc.Status),
				CreationTime: tea.StringValue(vpc.CreationTime),
				Description:  tea.StringValue(vpc.Description),
				LastSyncTime: time.Now(),
				Tags:         model.StringList(tags),
			},
			VpcName:         tea.StringValue(vpc.VpcName),
			CidrBlock:       tea.StringValue(vpc.CidrBlock),
			Ipv6CidrBlock:   tea.StringValue(vpc.Ipv6CidrBlock),
			VSwitchIds:      model.StringList(tea.StringSliceValue(vpc.VSwitchIds.VSwitchId)),
			RouteTableIds:   model.StringList(tea.StringSliceValue(vpc.RouterTableIds.RouterTableIds)),
			NatGatewayIds:   model.StringList(tea.StringSliceValue(vpc.NatGatewayIds.NatGatewayIds)),
			IsDefault:       tea.BoolValue(vpc.IsDefault),
			ResourceGroupId: tea.StringValue(vpc.ResourceGroupId),
		}
	}

	return result, total, nil
}

// SyncResources 同步资源
func (a *AliyunProviderImpl) SyncResources(ctx context.Context, region string) error {
	return nil
}

// ListRegions 列出区域
func (a *AliyunProviderImpl) ListRegions(ctx context.Context) ([]*model.RegionResp, error) {
	client, err := a.createEcsClient("cn-hangzhou")
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	request := &ecs.DescribeRegionsRequest{
		AcceptLanguage: tea.String("zh-CN"),
	}

	a.logger.Info("开始查询区域列表")
	response, err := client.DescribeRegions(request)
	if err != nil {
		a.logger.Error("查询区域列表失败", zap.Error(err))
		return nil, err
	}

	result := make([]*model.RegionResp, 0, len(response.Body.Regions.Region))
	for _, region := range response.Body.Regions.Region {
		result = append(result, &model.RegionResp{
			RegionId:       tea.StringValue(region.RegionId),
			LocalName:      tea.StringValue(region.LocalName),
			RegionEndpoint: tea.StringValue(region.RegionEndpoint),
		})
	}

	a.logger.Info("查询区域列表成功", zap.Int("count", len(result)))
	return result, nil
}

// GetZonesByVpc 获取VPC下的可用区
func (a *AliyunProviderImpl) GetZonesByVpc(ctx context.Context, region string, vpcId string) ([]*model.ZoneResp, error) {
	client, err := a.createVpcClient(region)
	if err != nil {
		a.logger.Error("创建VPC客户端失败", zap.Error(err))
		return nil, err
	}

	// 首先获取VPC信息
	vpcRequest := &vpc.DescribeVpcsRequest{
		RegionId: tea.String(region),
		VpcId:    tea.String(vpcId),
	}

	a.logger.Info("开始查询VPC信息", zap.String("region", region), zap.String("vpcId", vpcId))
	vpcResponse, err := client.DescribeVpcs(vpcRequest)
	if err != nil {
		a.logger.Error("查询VPC信息失败", zap.Error(err))
		return nil, err
	}

	if len(vpcResponse.Body.Vpcs.Vpc) == 0 {
		a.logger.Error("未找到指定的VPC", zap.String("vpcId", vpcId))
		return nil, fmt.Errorf("未找到指定的VPC: %s", vpcId)
	}

	// 并行获取可用区信息和VPC关联的交换机信息
	var zonesResponse *vpc.DescribeZonesResponse
	var vSwitchResponse *vpc.DescribeVSwitchesResponse
	var zonesErr, vSwitchErr error

	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		request := &vpc.DescribeZonesRequest{
			RegionId: tea.String(region),
		}
		zonesResponse, zonesErr = client.DescribeZones(request)
	}()

	go func() {
		defer wg.Done()
		vSwitchRequest := &vpc.DescribeVSwitchesRequest{
			RegionId: tea.String(region),
			VpcId:    tea.String(vpcId),
		}
		vSwitchResponse, vSwitchErr = client.DescribeVSwitches(vSwitchRequest)
	}()

	wg.Wait()

	if zonesErr != nil {
		a.logger.Error("查询可用区列表失败", zap.Error(zonesErr))
		return nil, zonesErr
	}

	if vSwitchErr != nil {
		a.logger.Error("查询交换机列表失败", zap.Error(vSwitchErr))
		return nil, vSwitchErr
	}

	// 创建一个map来存储VPC关联的可用区
	vpcZones := make(map[string]bool, len(vSwitchResponse.Body.VSwitches.VSwitch))
	for _, vSwitch := range vSwitchResponse.Body.VSwitches.VSwitch {
		vpcZones[tea.StringValue(vSwitch.ZoneId)] = true
	}

	// 过滤出VPC关联的可用区
	result := make([]*model.ZoneResp, 0, len(vpcZones))
	for _, zone := range zonesResponse.Body.Zones.Zone {
		zoneId := tea.StringValue(zone.ZoneId)
		if _, exists := vpcZones[zoneId]; exists {
			result = append(result, &model.ZoneResp{
				ZoneId:    zoneId,
				LocalName: tea.StringValue(zone.LocalName),
			})
		}
	}

	a.logger.Info("查询VPC关联的可用区成功", zap.Int("count", len(result)))
	return result, nil
}

// ListInstanceOptions 列出实例选项
func (a *AliyunProviderImpl) ListInstanceOptions(_ context.Context, payType string, region string, zone string, instanceType string, imageId string, systemDiskCategory string, dataDiskCategory string, pageSize int, pageNumber int) ([]*model.ListInstanceOptionsResp, error) {
	a.logger.Info("开始查询实例选项",
		zap.String("payType", payType),
		zap.String("region", region),
		zap.String("zone", zone),
		zap.String("instanceType", instanceType),
		zap.String("systemDiskCategory", systemDiskCategory),
		zap.String("dataDiskCategory", dataDiskCategory))

	// 依次判断每个选项是否为空，实现扁平化流程控制
	if payType == "" {
		return a.listAvailablePayTypes()
	}

	client, err := a.createEcsClient(region)
	if err != nil {
		a.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	if region == "" {
		return a.listAvailableRegions(client)
	}

	if zone == "" {
		return a.listAvailableZones(client, region)
	}

	if instanceType == "" {
		return a.listAvailableInstanceTypes(client, region, zone, payType)
	}

	if imageId == "" {
		return a.listAvailableInstanceTypeImages(client, region, zone, payType, instanceType, pageSize, pageNumber)
	}

	if systemDiskCategory == "" {
		return a.listAvailableSystemDiskCategories(client, region, zone, instanceType)
	}

	if dataDiskCategory == "" {
		return a.listAvailableDataDiskCategories(client, region, zone, instanceType)
	}

	// 所有选项都已选择，返回完整的配置信息
	return a.getCompleteConfiguration(payType, region, zone, instanceType, imageId, systemDiskCategory, dataDiskCategory)
}

// listAvailablePayTypes 获取可用的付费类型
func (a *AliyunProviderImpl) listAvailablePayTypes() ([]*model.ListInstanceOptionsResp, error) {
	return []*model.ListInstanceOptionsResp{
		{
			PayType: "PrePaid",
			Valid:   true,
		},
		{
			PayType: "PostPaid",
			Valid:   true,
		},
	}, nil
}

// listAvailableRegions 获取可用地域列表
func (a *AliyunProviderImpl) listAvailableRegions(client *ecs.Client) ([]*model.ListInstanceOptionsResp, error) {
	request := &ecs.DescribeRegionsRequest{
		AcceptLanguage: tea.String("zh-CN"),
	}

	response, err := client.DescribeRegions(request)
	if err != nil {
		a.logger.Error("获取地域列表失败", zap.Error(err))
		return nil, err
	}

	if response == nil || response.Body == nil || response.Body.Regions == nil {
		return []*model.ListInstanceOptionsResp{}, nil
	}

	regions := make([]*model.ListInstanceOptionsResp, 0, len(response.Body.Regions.Region))
	for _, region := range response.Body.Regions.Region {
		if region == nil || region.RegionId == nil {
			continue
		}

		regions = append(regions, &model.ListInstanceOptionsResp{
			Region: *region.RegionId,
			Valid:  true,
		})
	}

	return regions, nil
}

// listAvailableZones 获取指定地域下的可用区列表
func (a *AliyunProviderImpl) listAvailableZones(client *ecs.Client, region string) ([]*model.ListInstanceOptionsResp, error) {
	request := &ecs.DescribeZonesRequest{
		RegionId: tea.String(region),
	}

	response, err := client.DescribeZones(request)
	if err != nil {
		a.logger.Error("获取可用区列表失败", zap.String("region", region), zap.Error(err))
		return nil, err
	}

	if response == nil || response.Body == nil || response.Body.Zones == nil {
		return []*model.ListInstanceOptionsResp{}, nil
	}

	zones := make([]*model.ListInstanceOptionsResp, 0, len(response.Body.Zones.Zone))
	for _, zone := range response.Body.Zones.Zone {
		if zone == nil || zone.ZoneId == nil {
			continue
		}

		zones = append(zones, &model.ListInstanceOptionsResp{
			Region: region,
			Zone:   *zone.ZoneId,
			Valid:  true,
		})
	}

	return zones, nil
}

// listAvailableInstanceTypes 获取指定地域和可用区下可用的实例规格
func (a *AliyunProviderImpl) listAvailableInstanceTypes(client *ecs.Client, region string, zone string, payType string) ([]*model.ListInstanceOptionsResp, error) {
	request := &ecs.DescribeAvailableResourceRequest{
		RegionId:            tea.String(region),
		ZoneId:              tea.String(zone),
		DestinationResource: tea.String("InstanceType"),
		InstanceChargeType:  tea.String(payType),
	}

	response, err := client.DescribeAvailableResource(request)
	if err != nil {
		a.logger.Error("获取可用实例规格失败", zap.String("region", region), zap.String("zone", zone), zap.Error(err))
		return nil, err
	}

	// 提前分配容量，减少内存重新分配
	availableInstanceTypeMap := make(map[string]bool)

	// 添加空指针检查
	if response == nil || response.Body == nil || response.Body.AvailableZones == nil || response.Body.AvailableZones.AvailableZone == nil {
		a.logger.Warn("API响应数据为空", zap.String("region", region), zap.String("zone", zone))
		return []*model.ListInstanceOptionsResp{}, nil
	}

	// 扁平化处理可用实例类型收集
	for _, availableZone := range response.Body.AvailableZones.AvailableZone {
		// 跳过不匹配的可用区
		if availableZone == nil || availableZone.ZoneId == nil || *availableZone.ZoneId != zone {
			continue
		}

		// 跳过无资源的可用区
		if availableZone.AvailableResources == nil || availableZone.AvailableResources.AvailableResource == nil {
			continue
		}

		// 扁平化处理资源遍历和实例类型收集
		for _, resource := range availableZone.AvailableResources.AvailableResource {
			if resource == nil || resource.SupportedResources == nil || resource.SupportedResources.SupportedResource == nil {
				continue
			}

			for _, supportedResource := range resource.SupportedResources.SupportedResource {
				if supportedResource != nil && supportedResource.Status != nil &&
					supportedResource.Value != nil && *supportedResource.Status == "Available" {
					availableInstanceTypeMap[*supportedResource.Value] = true
				}
			}
		}
	}

	// 如果没有可用实例类型，直接返回
	if len(availableInstanceTypeMap) == 0 {
		return []*model.ListInstanceOptionsResp{}, nil
	}

	// 批量查询实例类型详情
	return a.batchFetchInstanceTypeDetails(client, availableInstanceTypeMap, region, zone, payType)
}

// listAvailableInstanceTypeImages 获取指定地域和可用区下可用的实例类型和镜像信息
func (a *AliyunProviderImpl) listAvailableInstanceTypeImages(client *ecs.Client, region string, zone string, payType string, instanceType string, pageSize int, pageNumber int) ([]*model.ListInstanceOptionsResp, error) {
	// 获取可用镜像信息
	imagesRequest := &ecs.DescribeImagesRequest{
		RegionId:        tea.String(region),
		Status:          tea.String("Available"),
		ImageOwnerAlias: tea.String("system"),
		PageSize:        tea.Int32(int32(pageSize)),
		PageNumber:      tea.Int32(int32(pageNumber)),
	}

	imagesResponse, imagesErr := client.DescribeImages(imagesRequest)
	if imagesErr != nil {
		a.logger.Error("查询可用镜像失败", zap.Error(imagesErr))
		return nil, imagesErr
	}

	// 处理镜像数据
	type ImageInfo struct {
		ImageId      string
		OSName       string
		OSType       string
		Architecture string
	}

	availableImages := make([]*ImageInfo, 0)
	if imagesResponse != nil && imagesResponse.Body != nil &&
		imagesResponse.Body.Images != nil && imagesResponse.Body.Images.Image != nil {
		for _, image := range imagesResponse.Body.Images.Image {
			if image == nil || image.ImageId == nil || image.OSName == nil {
				continue
			}

			availableImages = append(availableImages, &ImageInfo{
				ImageId:      tea.StringValue(image.ImageId),
				OSName:       tea.StringValue(image.OSName),
				OSType:       tea.StringValue(image.OSType),
				Architecture: tea.StringValue(image.Architecture),
			})
		}
	}

	// 组合结果 - 为每个镜像关联指定的实例类型
	result := make([]*model.ListInstanceOptionsResp, 0, len(availableImages))
	for _, image := range availableImages {
		result = append(result, &model.ListInstanceOptionsResp{
			InstanceType: instanceType,
			Region:       region,
			Zone:         zone,
			PayType:      payType,
			ImageId:      image.ImageId,
			OSName:       image.OSName,
			OSType:       image.OSType,
			Architecture: image.Architecture,
			Valid:        true,
		})
	}

	return result, nil
}

// batchFetchInstanceTypeDetails 批量获取实例类型详情
func (a *AliyunProviderImpl) batchFetchInstanceTypeDetails(client *ecs.Client, instanceTypeMap map[string]bool, region string, zone string, payType string) ([]*model.ListInstanceOptionsResp, error) {
	// 将map转换为切片，便于批量查询
	instanceTypeIds := make([]*string, 0, len(instanceTypeMap))
	for typeId := range instanceTypeMap {
		id := typeId // 创建局部变量避免闭包问题
		instanceTypeIds = append(instanceTypeIds, &id)
	}

	// 批量获取实例类型详情，提高查询效率
	instanceTypes := make([]*model.ListInstanceOptionsResp, 0, len(instanceTypeIds))
	batchSize := 10 // 阿里云API批量查询上限

	// 计算需要的批次数
	batchCount := (len(instanceTypeIds) + batchSize - 1) / batchSize

	// 使用错误组合
	var errGroup errgroup.Group
	resultCh := make(chan *model.ListInstanceOptionsResp, len(instanceTypeIds))

	// 并行请求各批次
	for i := 0; i < batchCount; i++ {
		startIdx := i * batchSize
		endIdx := (i + 1) * batchSize
		if endIdx > len(instanceTypeIds) {
			endIdx = len(instanceTypeIds)
		}

		batchIds := instanceTypeIds[startIdx:endIdx]

		// 为每个批次创建一个goroutine
		errGroup.Go(func() error {
			batchRequest := &ecs.DescribeInstanceTypesRequest{
				InstanceTypes: batchIds,
			}

			batchResponse, err := client.DescribeInstanceTypes(batchRequest)
			if err != nil {
				a.logger.Warn("批量获取实例规格详情失败", zap.Error(err))
				return nil // 不中断其他批次
			}

			if batchResponse == nil || batchResponse.Body == nil ||
				batchResponse.Body.InstanceTypes == nil ||
				batchResponse.Body.InstanceTypes.InstanceType == nil {
				return nil
			}

			// 处理返回的实例类型信息
			for _, info := range batchResponse.Body.InstanceTypes.InstanceType {
				if info == nil || info.InstanceTypeId == nil {
					continue
				}

				resultCh <- &model.ListInstanceOptionsResp{
					PayType:      payType,
					Region:       region,
					Zone:         zone,
					InstanceType: *info.InstanceTypeId,
					Cpu:          int(tea.Int32Value(info.CpuCoreCount)),
					Memory:       int(tea.Float32Value(info.MemorySize)),
					Valid:        true,
				}
			}

			return nil
		})
	}

	// 等待所有goroutine完成
	go func() {
		errGroup.Wait()
		close(resultCh)
	}()

	// 收集结果
	for result := range resultCh {
		instanceTypes = append(instanceTypes, result)
	}

	return instanceTypes, nil
}

// 扁平化处理磁盘类型查询
func (a *AliyunProviderImpl) listAvailableDiskCategories(client *ecs.Client, region string, zone string, instanceType string, diskType string) ([]*model.ListInstanceOptionsResp, error) {
	request := &ecs.DescribeAvailableResourceRequest{
		RegionId:            tea.String(region),
		ZoneId:              tea.String(zone),
		InstanceType:        tea.String(instanceType),
		DestinationResource: tea.String(diskType), // SystemDisk 或 DataDisk
	}

	response, err := client.DescribeAvailableResource(request)
	if err != nil {
		a.logger.Error("获取可用磁盘类型失败",
			zap.String("region", region),
			zap.String("zone", zone),
			zap.String("instanceType", instanceType),
			zap.String("diskType", diskType),
			zap.Error(err))
		return nil, err
	}

	// 检查响应是否为空
	if response == nil || response.Body == nil || response.Body.AvailableZones == nil ||
		response.Body.AvailableZones.AvailableZone == nil {
		return []*model.ListInstanceOptionsResp{}, nil
	}

	// 使用map去重
	diskTypesMap := make(map[string]bool)

	// 扁平化处理
	for _, availableZone := range response.Body.AvailableZones.AvailableZone {
		// 只处理指定可用区
		if availableZone == nil || availableZone.ZoneId == nil || *availableZone.ZoneId != zone {
			continue
		}

		// 缺少资源信息
		if availableZone.AvailableResources == nil || availableZone.AvailableResources.AvailableResource == nil {
			continue
		}

		// 遍历资源
		for _, resource := range availableZone.AvailableResources.AvailableResource {
			// 缺少支持的资源
			if resource == nil || resource.SupportedResources == nil || resource.SupportedResources.SupportedResource == nil {
				continue
			}

			// 遍历支持的资源
			for _, supportedResource := range resource.SupportedResources.SupportedResource {
				if supportedResource == nil || supportedResource.Status == nil || supportedResource.Value == nil {
					continue
				}

				// 只添加可用状态的资源
				if *supportedResource.Status != "Available" {
					continue
				}

				diskTypesMap[*supportedResource.Value] = true
			}
		}
	}

	// 转换为结果列表
	result := make([]*model.ListInstanceOptionsResp, 0, len(diskTypesMap))
	for diskCategory := range diskTypesMap {
		if diskType == "SystemDisk" {
			result = append(result, &model.ListInstanceOptionsResp{
				Region:             region,
				Zone:               zone,
				InstanceType:       instanceType,
				SystemDiskCategory: diskCategory,
				Valid:              true,
			})
		} else {
			result = append(result, &model.ListInstanceOptionsResp{
				Region:           region,
				Zone:             zone,
				InstanceType:     instanceType,
				DataDiskCategory: diskCategory,
				Valid:            true,
			})
		}
	}

	return result, nil
}

// listAvailableSystemDiskCategories 获取可用的系统盘类型
func (a *AliyunProviderImpl) listAvailableSystemDiskCategories(client *ecs.Client, region string, zone string, instanceType string) ([]*model.ListInstanceOptionsResp, error) {
	return a.listAvailableDiskCategories(client, region, zone, instanceType, "SystemDisk")
}

// listAvailableDataDiskCategories 获取可用的数据盘类型
func (a *AliyunProviderImpl) listAvailableDataDiskCategories(client *ecs.Client, region string, zone string, instanceType string) ([]*model.ListInstanceOptionsResp, error) {
	return a.listAvailableDiskCategories(client, region, zone, instanceType, "DataDisk")
}

// getCompleteConfiguration 获取完整的配置信息
func (a *AliyunProviderImpl) getCompleteConfiguration(payType string, region string, zone string, instanceType string, imageId string, systemDiskCategory string, dataDiskCategory string) ([]*model.ListInstanceOptionsResp, error) {
	return []*model.ListInstanceOptionsResp{
		{
			PayType:            payType,
			Region:             region,
			Zone:               zone,
			InstanceType:       instanceType,
			ImageId:            imageId,
			SystemDiskCategory: systemDiskCategory,
			DataDiskCategory:   dataDiskCategory,
			Valid:              true,
		},
	}, nil
}

// GetVpcDetail 获取VPC详情
func (a *AliyunProviderImpl) GetVpcDetail(ctx context.Context, region string, vpcID string) (*model.ResourceVpc, error) {
	client, err := a.createVpcClient(region)
	if err != nil {
		a.logger.Error("创建VPC客户端失败", zap.Error(err))
		return nil, err
	}

	request := &vpc.DescribeVpcAttributeRequest{
		RegionId: tea.String(region),
		VpcId:    tea.String(vpcID),
	}

	response, err := client.DescribeVpcAttribute(request)
	if err != nil {
		a.logger.Error("获取VPC详情失败", zap.Error(err))
		return nil, err
	}

	if response == nil || response.Body == nil {
		return nil, errors.New("获取VPC详情失败，响应为空")
	}

	tagList := make([]string, 0, len(response.Body.Tags.Tag))
	for _, tag := range response.Body.Tags.Tag {
		tagList = append(tagList, fmt.Sprintf("%s=%s", tea.StringValue(tag.Key), tea.StringValue(tag.Value)))
	}

	return &model.ResourceVpc{
		ResourceBase: model.ResourceBase{
			InstanceName: tea.StringValue(response.Body.VpcName),
			InstanceId:   tea.StringValue(response.Body.VpcId),
			Provider:     model.CloudProviderAliyun,
			RegionId:     tea.StringValue(response.Body.RegionId),
			Status:       tea.StringValue(response.Body.Status),
			Description:  tea.StringValue(response.Body.Description),
			CreationTime: tea.StringValue(response.Body.CreationTime),
			Tags:         model.StringList(tagList),
		},
		VpcName:         tea.StringValue(response.Body.VpcName),
		CidrBlock:       tea.StringValue(response.Body.CidrBlock),
		Ipv6CidrBlock:   tea.StringValue(response.Body.Ipv6CidrBlock),
		VSwitchIds:      model.StringList(tea.StringSliceValue(response.Body.VSwitchIds.VSwitchId)),
		RouteTableIds:   []string{tea.StringValue(response.Body.VRouterId)},
		IsDefault:       tea.BoolValue(response.Body.IsDefault),
		ResourceGroupId: tea.StringValue(response.Body.ResourceGroupId),
	}, nil
}
