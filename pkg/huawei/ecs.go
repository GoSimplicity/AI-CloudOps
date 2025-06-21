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

package huawei

import (
	"context"

	ecsmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
	"go.uber.org/zap"
)

type EcsService struct {
	sdk *SDK
}

func NewEcsService(sdk *SDK) *EcsService {
	return &EcsService{sdk: sdk}
}

type CreateInstanceRequest struct {
	Region             string
	ZoneId             string
	ImageId            string
	InstanceType       string
	SecurityGroupIds   []string
	SubnetId           string
	InstanceName       string
	Hostname           string
	Password           string
	Description        string
	Amount             int
	DryRun             bool
	InstanceChargeType string
	SystemDiskCategory string
	SystemDiskSize     int
	DataDiskCategory   string
	DataDiskSize       int
}

type CreateInstanceResponseBody struct {
	InstanceIds []string
}

// CreateInstance 创建ECS实例
func (e *EcsService) CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*CreateInstanceResponseBody, error) {
	client, err := e.sdk.CreateEcsClient(req.Region, e.sdk.accessKey)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	// 根据磁盘类型获取对应的枚举值
	var systemDiskType ecsmodel.PostPaidServerRootVolumeVolumetype
	volumeTypeEnum := ecsmodel.GetPostPaidServerRootVolumeVolumetypeEnum()
	switch req.SystemDiskCategory {
	case "SSD":
		systemDiskType = volumeTypeEnum.SSD
	case "GPSSD":
		systemDiskType = volumeTypeEnum.GPSSD
	case "SAS":
		systemDiskType = volumeTypeEnum.SAS
	case "SATA":
		systemDiskType = volumeTypeEnum.SATA
	case "ESSD":
		systemDiskType = volumeTypeEnum.ESSD
	case "GPSSD2":
		systemDiskType = volumeTypeEnum.GPSSD2
	case "ESSD2":
		systemDiskType = volumeTypeEnum.ESSD2
	default:
		systemDiskType = volumeTypeEnum.SSD // 默认使用SSD
	}

	// 构建系统盘配置
	systemDiskSize := int32(req.SystemDiskSize)
	systemDisk := &ecsmodel.PostPaidServerRootVolume{
		Volumetype: systemDiskType,
		Size:       &systemDiskSize,
	}

	// 构建数据盘配置
	var dataVolumes []ecsmodel.PostPaidServerDataVolume
	if req.DataDiskCategory != "" && req.DataDiskSize > 0 {
		// 根据磁盘类型获取对应的枚举值
		var dataDiskType ecsmodel.PostPaidServerDataVolumeVolumetype
		dataVolumeTypeEnum := ecsmodel.GetPostPaidServerDataVolumeVolumetypeEnum()
		switch req.DataDiskCategory {
		case "SSD":
			dataDiskType = dataVolumeTypeEnum.SSD
		case "GPSSD":
			dataDiskType = dataVolumeTypeEnum.GPSSD
		case "SAS":
			dataDiskType = dataVolumeTypeEnum.SAS
		case "SATA":
			dataDiskType = dataVolumeTypeEnum.SATA
		case "ESSD":
			dataDiskType = dataVolumeTypeEnum.ESSD
		case "GPSSD2":
			dataDiskType = dataVolumeTypeEnum.GPSSD2
		case "ESSD2":
			dataDiskType = dataVolumeTypeEnum.ESSD2
		default:
			dataDiskType = dataVolumeTypeEnum.SSD // 默认使用SSD
		}

		dataDiskSize := int32(req.DataDiskSize)
		dataVolumes = []ecsmodel.PostPaidServerDataVolume{
			{
				Volumetype: dataDiskType,
				Size:       dataDiskSize,
			},
		}
	}

	// 构建网络配置
	nics := []ecsmodel.PostPaidServerNic{
		{
			SubnetId: &req.SubnetId,
		},
	}

	// 构建安全组配置
	var securityGroups []ecsmodel.PostPaidServerSecurityGroup
	for _, sgId := range req.SecurityGroupIds {
		securityGroups = append(securityGroups, ecsmodel.PostPaidServerSecurityGroup{
			Id: &sgId,
		})
	}

	// 构建请求参数
	availabilityZone := req.ZoneId
	description := req.Description
	count := int32(req.Amount)
	adminPass := req.Password

	request := &ecsmodel.CreatePostPaidServersRequest{
		Body: &ecsmodel.CreatePostPaidServersRequestBody{
			Server: &ecsmodel.PostPaidServer{
				Name:             req.InstanceName,
				ImageRef:         req.ImageId,
				FlavorRef:        req.InstanceType,
				AvailabilityZone: &availabilityZone,
				RootVolume:       systemDisk,
				DataVolumes:      &dataVolumes,
				Nics:             nics,
				SecurityGroups:   &securityGroups,
				AdminPass:        &adminPass,
				Description:      &description,
				Count:            &count,
			},
		},
	}

	e.sdk.logger.Info("开始创建ECS实例", zap.String("region", req.Region), zap.Any("request", req))
	response, err := client.CreatePostPaidServers(request)
	if err != nil {
		e.sdk.logger.Error("创建ECS实例失败", zap.Error(err))
		return nil, err
	}

	instanceIds := make([]string, 0)
	if response.ServerIds != nil {
		instanceIds = *response.ServerIds
	}

	e.sdk.logger.Info("创建ECS实例成功", zap.Strings("instanceIds", instanceIds))

	return &CreateInstanceResponseBody{
		InstanceIds: instanceIds,
	}, nil
}

// StartInstance 启动ECS实例
func (e *EcsService) StartInstance(ctx context.Context, region string, instanceID string) error {
	client, err := e.sdk.CreateEcsClient(region, e.sdk.accessKey)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecsmodel.BatchStartServersRequest{
		Body: &ecsmodel.BatchStartServersRequestBody{
			OsStart: &ecsmodel.BatchStartServersOption{
				Servers: []ecsmodel.ServerId{
					{
						Id: instanceID,
					},
				},
			},
		},
	}

	e.sdk.logger.Info("开始启动ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
	_, err = client.BatchStartServers(request)
	if err != nil {
		e.sdk.logger.Error("启动ECS实例失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("启动ECS实例成功", zap.String("instanceID", instanceID))
	return nil
}

// StopInstance 停止ECS实例
func (e *EcsService) StopInstance(ctx context.Context, region string, instanceID string, forceStop bool) error {
	client, err := e.sdk.CreateEcsClient(region, e.sdk.accessKey)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	// 根据forceStop参数选择停止类型
	var stopType ecsmodel.BatchStopServersOptionType
	stopTypeEnum := ecsmodel.GetBatchStopServersOptionTypeEnum()
	if forceStop {
		stopType = stopTypeEnum.HARD
	} else {
		stopType = stopTypeEnum.SOFT
	}

	request := &ecsmodel.BatchStopServersRequest{
		Body: &ecsmodel.BatchStopServersRequestBody{
			OsStop: &ecsmodel.BatchStopServersOption{
				Servers: []ecsmodel.ServerId{
					{
						Id: instanceID,
					},
				},
				Type: &stopType,
			},
		},
	}

	e.sdk.logger.Info("开始停止ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
	_, err = client.BatchStopServers(request)
	if err != nil {
		e.sdk.logger.Error("停止ECS实例失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("停止ECS实例成功", zap.String("instanceID", instanceID))
	return nil
}

// RestartInstance 重启ECS实例
func (e *EcsService) RestartInstance(ctx context.Context, region string, instanceID string) error {
	client, err := e.sdk.CreateEcsClient(region, e.sdk.accessKey)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecsmodel.BatchRebootServersRequest{
		Body: &ecsmodel.BatchRebootServersRequestBody{
			Reboot: &ecsmodel.BatchRebootSeversOption{
				Servers: []ecsmodel.ServerId{
					{
						Id: instanceID,
					},
				},
			},
		},
	}

	e.sdk.logger.Info("开始重启ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
	_, err = client.BatchRebootServers(request)
	if err != nil {
		e.sdk.logger.Error("重启ECS实例失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("重启ECS实例成功", zap.String("instanceID", instanceID))
	return nil
}

// DeleteInstance 删除ECS实例
func (e *EcsService) DeleteInstance(ctx context.Context, region string, instanceID string, force bool) error {
	client, err := e.sdk.CreateEcsClient(region, e.sdk.accessKey)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecsmodel.DeleteServersRequest{
		Body: &ecsmodel.DeleteServersRequestBody{
			Servers: []ecsmodel.ServerId{
				{
					Id: instanceID,
				},
			},
		},
	}

	e.sdk.logger.Info("开始删除ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
	_, err = client.DeleteServers(request)
	if err != nil {
		e.sdk.logger.Error("删除ECS实例失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("删除ECS实例成功", zap.String("instanceID", instanceID))
	return nil
}

// ListInstancesRequest 查询实例列表请求参数
type ListInstancesRequest struct {
	Region string
	Page   int
	Size   int
}

// ListInstancesResponseBody 查询实例列表响应
type ListInstancesResponseBody struct {
	Instances []ecsmodel.ServerDetail
	Total     int32
}

// ListInstances 查询ECS实例列表
func (e *EcsService) ListInstances(ctx context.Context, req *ListInstancesRequest) (*ListInstancesResponseBody, error) {
	client, err := e.sdk.CreateEcsClient(req.Region, e.sdk.accessKey)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	limit := int32(req.Size)
	request := &ecsmodel.ListServersDetailsRequest{
		Limit: &limit,
	}

	e.sdk.logger.Info("开始查询ECS实例列表", zap.String("region", req.Region))
	response, err := client.ListServersDetails(request)
	if err != nil {
		e.sdk.logger.Error("查询ECS实例列表失败", zap.Error(err))
		return nil, err
	}

	var total int32
	var instances []ecsmodel.ServerDetail
	if response.Servers != nil {
		instances = *response.Servers
		total = int32(len(instances))
	}
	if response.Count != nil {
		total = *response.Count
	}

	e.sdk.logger.Info("查询ECS实例列表成功", zap.Int32("total", total))

	return &ListInstancesResponseBody{
		Instances: instances,
		Total:     total,
	}, nil
}

// GetInstanceDetail 获取ECS实例详情
func (e *EcsService) GetInstanceDetail(ctx context.Context, region string, instanceID string) (*ecsmodel.ServerDetail, error) {
	client, err := e.sdk.CreateEcsClient(region, e.sdk.accessKey)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	request := &ecsmodel.ShowServerRequest{
		ServerId: instanceID,
	}

	e.sdk.logger.Info("开始获取ECS实例详情", zap.String("region", region), zap.String("instanceID", instanceID))
	response, err := client.ShowServer(request)
	if err != nil {
		e.sdk.logger.Error("获取ECS实例详情失败", zap.Error(err))
		return nil, err
	}

	return response.Server, nil
}
