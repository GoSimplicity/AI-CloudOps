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

package aws

import (
	"context"
	"encoding/base64"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.uber.org/zap"
)

// EC2Service AWS EC2服务客户端
type EC2Service struct {
	sdk *SDK
}

// NewEC2Service 创建EC2服务实例
func NewEC2Service(sdk *SDK) *EC2Service {
	return &EC2Service{sdk: sdk}
}

// CreateInstanceRequest 创建实例请求
type CreateInstanceRequest struct {
	Region             string
	AvailabilityZone   string
	ImageId            string
	InstanceType       string
	SecurityGroupIds   []string
	SubnetId           string
	InstanceName       string
	KeyName            string
	UserData           string
	Description        string
	MinCount           int
	MaxCount           int
	DryRun             bool
	IamInstanceProfile string
	SystemDiskSize     int32
	SystemDiskType     string // gp2, gp3, io1, io2, st1, sc1
	DataDisks          []DataDisk
	Tags               map[string]string
}

type DataDisk struct {
	Size       int32
	VolumeType string // gp2, gp3, io1, io2, st1, sc1
	Device     string
	Iops       int32
	Throughput int32
}

type CreateInstanceResponseBody struct {
	InstanceIds []string
}

// CreateInstance 创建EC2实例
func (e *EC2Service) CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*CreateInstanceResponseBody, error) {
	client, err := e.sdk.CreateEC2Client(ctx, req.Region)
	if err != nil {
		e.sdk.logger.Error("创建EC2客户端失败", zap.Error(err))
		return nil, err
	}

	// 构建系统盘配置
	var systemDiskType types.VolumeType
	switch req.SystemDiskType {
	case "gp2":
		systemDiskType = types.VolumeTypeGp2
	case "gp3":
		systemDiskType = types.VolumeTypeGp3
	case "io1":
		systemDiskType = types.VolumeTypeIo1
	case "io2":
		systemDiskType = types.VolumeTypeIo2
	case "st1":
		systemDiskType = types.VolumeTypeSt1
	case "sc1":
		systemDiskType = types.VolumeTypeSc1
	default:
		systemDiskType = types.VolumeTypeGp3 // 默认使用gp3
	}

	blockDeviceMappings := []types.BlockDeviceMapping{
		{
			DeviceName: stringPtr("/dev/sda1"),
			Ebs: &types.EbsBlockDevice{
				VolumeSize:          &req.SystemDiskSize,
				VolumeType:          systemDiskType,
				DeleteOnTermination: boolPtr(true),
			},
		},
	}

	// 添加数据盘
	for i, dataDisk := range req.DataDisks {
		var dataVolumeType types.VolumeType
		switch dataDisk.VolumeType {
		case "gp2":
			dataVolumeType = types.VolumeTypeGp2
		case "gp3":
			dataVolumeType = types.VolumeTypeGp3
		case "io1":
			dataVolumeType = types.VolumeTypeIo1
		case "io2":
			dataVolumeType = types.VolumeTypeIo2
		case "st1":
			dataVolumeType = types.VolumeTypeSt1
		case "sc1":
			dataVolumeType = types.VolumeTypeSc1
		default:
			dataVolumeType = types.VolumeTypeGp3
		}

		device := dataDisk.Device
		if device == "" {
			device = fmt.Sprintf("/dev/sdf%d", i)
		}

		ebsDevice := &types.EbsBlockDevice{
			VolumeSize:          &dataDisk.Size,
			VolumeType:          dataVolumeType,
			DeleteOnTermination: boolPtr(false),
		}

		if dataDisk.Iops > 0 {
			ebsDevice.Iops = &dataDisk.Iops
		}

		if dataDisk.Throughput > 0 {
			ebsDevice.Throughput = &dataDisk.Throughput
		}

		blockDeviceMappings = append(blockDeviceMappings, types.BlockDeviceMapping{
			DeviceName: &device,
			Ebs:        ebsDevice,
		})
	}

	// 构建标签规范
	tagSpecifications := []types.TagSpecification{
		{
			ResourceType: types.ResourceTypeInstance,
			Tags: []types.Tag{
				{
					Key:   stringPtr("Name"),
					Value: &req.InstanceName,
				},
			},
		},
	}

	// 添加自定义标签
	if req.Tags != nil {
		for key, value := range req.Tags {
			tagSpecifications[0].Tags = append(tagSpecifications[0].Tags, types.Tag{
				Key:   &key,
				Value: &value,
			})
		}
	}

	// 构建网络接口规范
	networkInterfaces := []types.InstanceNetworkInterfaceSpecification{
		{
			DeviceIndex:              int32Ptr(0),
			SubnetId:                 &req.SubnetId,
			Groups:                   req.SecurityGroupIds,
			DeleteOnTermination:      boolPtr(true),
			AssociatePublicIpAddress: boolPtr(true),
		},
	}

	// 构建运行实例请求
	runRequest := &ec2.RunInstancesInput{
		ImageId:             &req.ImageId,
		InstanceType:        types.InstanceType(req.InstanceType),
		MinCount:            int32Ptr(int32(req.MinCount)),
		MaxCount:            int32Ptr(int32(req.MaxCount)),
		BlockDeviceMappings: blockDeviceMappings,
		NetworkInterfaces:   networkInterfaces,
		TagSpecifications:   tagSpecifications,
		DryRun:              &req.DryRun,
	}

	// 设置密钥对
	if req.KeyName != "" {
		runRequest.KeyName = &req.KeyName
	}

	// 设置用户数据
	if req.UserData != "" {
		userData := base64.StdEncoding.EncodeToString([]byte(req.UserData))
		runRequest.UserData = &userData
	}

	// 设置IAM实例配置文件
	if req.IamInstanceProfile != "" {
		runRequest.IamInstanceProfile = &types.IamInstanceProfileSpecification{
			Name: &req.IamInstanceProfile,
		}
	}

	// 设置可用区
	if req.AvailabilityZone != "" {
		runRequest.Placement = &types.Placement{
			AvailabilityZone: &req.AvailabilityZone,
		}
	}

	e.sdk.logger.Info("开始创建EC2实例", zap.String("region", req.Region), zap.String("instanceType", req.InstanceType), zap.Int("count", req.MaxCount))
	response, err := client.RunInstances(ctx, runRequest)
	if err != nil {
		e.sdk.logger.Error("创建EC2实例失败", zap.Error(err))
		return nil, err
	}

	instanceIds := make([]string, len(response.Instances))
	for i, instance := range response.Instances {
		instanceIds[i] = *instance.InstanceId
	}

	e.sdk.logger.Info("EC2实例创建成功", zap.Strings("instanceIds", instanceIds))

	return &CreateInstanceResponseBody{
		InstanceIds: instanceIds,
	}, nil
}

// StartInstance 启动EC2实例
func (e *EC2Service) StartInstance(ctx context.Context, region string, instanceID string) error {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return err
	}

	request := &ec2.StartInstancesInput{
		InstanceIds: []string{instanceID},
	}

	e.sdk.logger.Info("开始启动EC2实例", zap.String("instanceId", instanceID))
	_, err = client.StartInstances(ctx, request)
	if err != nil {
		e.sdk.logger.Error("启动EC2实例失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("EC2实例启动成功", zap.String("instanceId", instanceID))
	return nil
}

// StopInstance 停止EC2实例
func (e *EC2Service) StopInstance(ctx context.Context, region string, instanceID string, force bool) error {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return err
	}

	request := &ec2.StopInstancesInput{
		InstanceIds: []string{instanceID},
		Force:       &force,
	}

	e.sdk.logger.Info("开始停止EC2实例", zap.String("instanceId", instanceID), zap.Bool("force", force))
	_, err = client.StopInstances(ctx, request)
	if err != nil {
		e.sdk.logger.Error("停止EC2实例失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("EC2实例停止成功", zap.String("instanceId", instanceID))
	return nil
}

// RestartInstance 重启EC2实例
func (e *EC2Service) RestartInstance(ctx context.Context, region string, instanceID string) error {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return err
	}

	request := &ec2.RebootInstancesInput{
		InstanceIds: []string{instanceID},
	}

	e.sdk.logger.Info("开始重启EC2实例", zap.String("instanceId", instanceID))
	_, err = client.RebootInstances(ctx, request)
	if err != nil {
		e.sdk.logger.Error("重启EC2实例失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("EC2实例重启成功", zap.String("instanceId", instanceID))
	return nil
}

// DeleteInstance 终止EC2实例
func (e *EC2Service) DeleteInstance(ctx context.Context, region string, instanceID string, force bool) error {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return err
	}

	request := &ec2.TerminateInstancesInput{
		InstanceIds: []string{instanceID},
	}

	e.sdk.logger.Info("开始终止EC2实例", zap.String("instanceId", instanceID))
	_, err = client.TerminateInstances(ctx, request)
	if err != nil {
		e.sdk.logger.Error("终止EC2实例失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("EC2实例终止成功", zap.String("instanceId", instanceID))
	return nil
}

type ListInstancesRequest struct {
	Region string
	Page   int
	Size   int
	VpcId  string
}

type ListInstancesResponseBody struct {
	Instances []types.Instance
	Total     int64
}

// ListInstances 查询EC2实例列表
func (e *EC2Service) ListInstances(ctx context.Context, req *ListInstancesRequest) (*ListInstancesResponseBody, int64, error) {
	client, err := e.sdk.CreateEC2Client(ctx, req.Region)
	if err != nil {
		return nil, 0, err
	}

	request := &ec2.DescribeInstancesInput{}

	// 如果指定了VPC ID，添加过滤器
	if req.VpcId != "" {
		request.Filters = []types.Filter{
			{
				Name:   stringPtr("vpc-id"),
				Values: []string{req.VpcId},
			},
		}
	}

	e.sdk.logger.Info("开始查询EC2实例列表", zap.String("region", req.Region))
	response, err := client.DescribeInstances(ctx, request)
	if err != nil {
		e.sdk.logger.Error("查询EC2实例列表失败", zap.Error(err))
		return nil, 0, err
	}

	var allInstances []types.Instance
	for _, reservation := range response.Reservations {
		allInstances = append(allInstances, reservation.Instances...)
	}

	totalCount := int64(len(allInstances))

	// 分页处理
	startIdx := (req.Page - 1) * req.Size
	endIdx := req.Page * req.Size
	if startIdx >= len(allInstances) {
		return &ListInstancesResponseBody{
			Instances: []types.Instance{},
		}, totalCount, nil
	}

	if endIdx > len(allInstances) {
		endIdx = len(allInstances)
	}

	e.sdk.logger.Info("查询EC2实例列表成功", zap.Int64("total", totalCount))

	return &ListInstancesResponseBody{
		Instances: allInstances[startIdx:endIdx],
		Total:     totalCount,
	}, totalCount, nil
}

// GetInstanceDetail 获取EC2实例详情
func (e *EC2Service) GetInstanceDetail(ctx context.Context, region string, instanceID string) (*types.Instance, error) {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return nil, err
	}

	request := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceID},
	}

	e.sdk.logger.Info("开始查询EC2实例详情", zap.String("instanceId", instanceID))
	response, err := client.DescribeInstances(ctx, request)
	if err != nil {
		e.sdk.logger.Error("查询EC2实例详情失败", zap.Error(err))
		return nil, err
	}

	if len(response.Reservations) == 0 || len(response.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("EC2实例 %s 不存在", instanceID)
	}

	instance := response.Reservations[0].Instances[0]
	e.sdk.logger.Info("查询EC2实例详情成功", zap.String("instanceId", instanceID))
	return &instance, nil
}

// ModifyInstanceAttribute 修改实例属性
func (e *EC2Service) ModifyInstanceAttribute(ctx context.Context, region string, instanceID string, instanceType string) error {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return err
	}

	request := &ec2.ModifyInstanceAttributeInput{
		InstanceId: &instanceID,
		InstanceType: &types.AttributeValue{
			Value: &instanceType,
		},
	}

	e.sdk.logger.Info("开始修改EC2实例属性", zap.String("instanceId", instanceID), zap.String("instanceType", instanceType))
	_, err = client.ModifyInstanceAttribute(ctx, request)
	if err != nil {
		e.sdk.logger.Error("修改EC2实例属性失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("EC2实例属性修改成功", zap.String("instanceId", instanceID))
	return nil
}

// DescribeInstanceType 查询指定实例类型的vCPU和内存（GB）
func (e *EC2Service) DescribeInstanceType(ctx context.Context, region, instanceType string) (int, int, error) {
	client, err := e.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return 0, 0, err
	}
	input := &ec2.DescribeInstanceTypesInput{
		InstanceTypes: []types.InstanceType{types.InstanceType(instanceType)},
	}
	output, err := client.DescribeInstanceTypes(ctx, input)
	if err != nil || len(output.InstanceTypes) == 0 {
		return 0, 0, err
	}
	vcpu := int(*output.InstanceTypes[0].VCpuInfo.DefaultVCpus)
	memMiB := output.InstanceTypes[0].MemoryInfo.SizeInMiB
	memory := 0
	if memMiB != nil {
		memory = int(*memMiB / 1024)
		if memory == 0 {
			memory = int(*memMiB)
		}
	}
	return vcpu, memory, nil
}

// waitForInstanceRunning 等待实例运行
func (e *EC2Service) waitForInstanceRunning(ctx context.Context, client *ec2.Client, instanceId string) error {
	request := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceId},
	}

	for i := 0; i < 30; i++ {
		response, err := client.DescribeInstances(ctx, request)
		if err != nil {
			return err
		}

		if len(response.Reservations) > 0 && len(response.Reservations[0].Instances) > 0 {
			instance := response.Reservations[0].Instances[0]
			if instance.State.Name == types.InstanceStateNameRunning {
				return nil
			}
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("等待EC2实例运行超时")
}

// waitForInstanceStopped 等待实例停止
func (e *EC2Service) waitForInstanceStopped(ctx context.Context, client *ec2.Client, instanceId string) error {
	request := &ec2.DescribeInstancesInput{
		InstanceIds: []string{instanceId},
	}

	for i := 0; i < 30; i++ {
		response, err := client.DescribeInstances(ctx, request)
		if err != nil {
			return err
		}

		if len(response.Reservations) > 0 && len(response.Reservations[0].Instances) > 0 {
			instance := response.Reservations[0].Instances[0]
			if instance.State.Name == types.InstanceStateNameStopped {
				return nil
			}
		}

		time.Sleep(10 * time.Second)
	}

	return fmt.Errorf("等待EC2实例停止超时")
}

// 辅助函数
func boolPtr(b bool) *bool {
	return &b
}

func int32Ptr(i int32) *int32 {
	return &i
}
