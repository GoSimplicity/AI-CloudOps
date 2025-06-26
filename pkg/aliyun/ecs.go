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

package aliyun

import (
	"context"
	"strconv"

	ecs "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"go.uber.org/zap"
)

type EcsService struct {
	sdk *SDK
}

func NewEcsService(sdk *SDK) *EcsService {
	return &EcsService{sdk: sdk}
}

type CreateInstanceRequest struct {
	Region             string   // 地域
	ZoneId             string   // 可用区ID
	ImageId            string   // 镜像ID
	InstanceType       string   // 实例类型
	SecurityGroupIds   []string // 安全组ID
	VSwitchId          string   // 交换机ID
	InstanceName       string   // 实例名称
	Hostname           string   // 主机名
	Password           string   // 密码
	Description        string   // 描述
	Amount             int      // 数量
	DryRun             bool     // 是否只预检
	InstanceChargeType string   // 实例付费类型
	SystemDiskCategory string   // 系统盘类型
	SystemDiskSize     int      // 系统盘大小
	DataDiskCategory   string   // 数据盘类型
	DataDiskSize       int      // 数据盘大小
}

type CreateInstanceResponse struct {
	InstanceIds []string
}

func (e *EcsService) CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*CreateInstanceResponse, error) {
	client, err := e.sdk.CreateEcsClient(req.Region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	request := &ecs.RunInstancesRequest{
		RegionId:           tea.String(req.Region),
		ZoneId:             tea.String(req.ZoneId),
		ImageId:            tea.String(req.ImageId),
		InstanceType:       tea.String(req.InstanceType),
		SecurityGroupIds:   tea.StringSlice(req.SecurityGroupIds),
		VSwitchId:          tea.String(req.VSwitchId),
		InstanceName:       tea.String(req.InstanceName),
		HostName:           tea.String(req.Hostname),
		Password:           tea.String(req.Password),
		Description:        tea.String(req.Description),
		Amount:             tea.Int32(int32(req.Amount)),
		DryRun:             tea.Bool(req.DryRun),
		InstanceChargeType: tea.String(req.InstanceChargeType),
	}

	// 设置系统盘
	if req.SystemDiskCategory != "" {
		request.SystemDisk = &ecs.RunInstancesRequestSystemDisk{
			Category: tea.String(req.SystemDiskCategory),
			Size:     tea.String(strconv.Itoa(req.SystemDiskSize)),
		}
	}

	// 设置数据盘
	if req.DataDiskCategory != "" {
		request.DataDisk = []*ecs.RunInstancesRequestDataDisk{
			{
				Category: tea.String(req.DataDiskCategory),
				Size:     tea.Int32(int32(req.DataDiskSize)),
			},
		}
	}

	e.sdk.logger.Info("开始创建ECS实例", zap.String("region", req.Region), zap.Any("request", req))
	response, err := client.RunInstances(request)
	if err != nil {
		e.sdk.logger.Error("创建ECS实例失败", zap.Error(err))
		return nil, HandleError(err)
	}

	instanceIds := tea.StringSliceValue(response.Body.InstanceIdSets.InstanceIdSet)
	e.sdk.logger.Info("创建ECS实例成功", zap.Strings("instanceIds", instanceIds))

	return &CreateInstanceResponse{
		InstanceIds: instanceIds,
	}, nil
}

type StartInstanceRequest struct {
	Region     string // 地域
	InstanceID string // 实例ID
}

type StartInstanceResponse struct {
	Success bool
}

func (e *EcsService) StartInstance(ctx context.Context, req *StartInstanceRequest) (*StartInstanceResponse, error) {
	client, err := e.sdk.CreateEcsClient(req.Region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	request := &ecs.StartInstanceRequest{
		InstanceId: tea.String(req.InstanceID),
	}

	e.sdk.logger.Info("开始启动ECS实例", zap.String("region", req.Region), zap.String("instanceID", req.InstanceID))
	_, err = client.StartInstance(request)
	if err != nil {
		e.sdk.logger.Error("启动ECS实例失败", zap.Error(err))
		return nil, HandleError(err)
	}

	e.sdk.logger.Info("启动ECS实例成功", zap.String("instanceID", req.InstanceID))
	return &StartInstanceResponse{Success: true}, nil
}

type StopInstanceRequest struct {
	Region     string // 地域
	InstanceID string // 实例ID
	ForceStop  bool   // 是否强制停止
}

type StopInstanceResponse struct {
	Success bool
}

func (e *EcsService) StopInstance(ctx context.Context, req *StopInstanceRequest) (*StopInstanceResponse, error) {
	client, err := e.sdk.CreateEcsClient(req.Region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	request := &ecs.StopInstanceRequest{
		InstanceId: tea.String(req.InstanceID),
		ForceStop:  tea.Bool(req.ForceStop),
	}

	e.sdk.logger.Info("开始停止ECS实例", zap.String("region", req.Region), zap.String("instanceID", req.InstanceID))
	_, err = client.StopInstance(request)
	if err != nil {
		e.sdk.logger.Error("停止ECS实例失败", zap.Error(err))
		return nil, HandleError(err)
	}

	e.sdk.logger.Info("停止ECS实例成功", zap.String("instanceID", req.InstanceID))
	return &StopInstanceResponse{Success: true}, nil
}

type RestartInstanceRequest struct {
	Region     string // 地域
	InstanceID string // 实例ID
}

type RestartInstanceResponse struct {
	Success bool
}

func (e *EcsService) RestartInstance(ctx context.Context, req *RestartInstanceRequest) (*RestartInstanceResponse, error) {
	client, err := e.sdk.CreateEcsClient(req.Region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	request := &ecs.RebootInstanceRequest{
		InstanceId: tea.String(req.InstanceID),
	}

	e.sdk.logger.Info("开始重启ECS实例", zap.String("region", req.Region), zap.String("instanceID", req.InstanceID))
	_, err = client.RebootInstance(request)
	if err != nil {
		e.sdk.logger.Error("重启ECS实例失败", zap.Error(err))
		return nil, HandleError(err)
	}

	e.sdk.logger.Info("重启ECS实例成功", zap.String("instanceID", req.InstanceID))
	return &RestartInstanceResponse{Success: true}, nil
}

type DeleteInstanceRequest struct {
	Region     string // 地域
	InstanceID string // 实例ID
	Force      bool   // 是否强制删除
}

type DeleteInstanceResponse struct {
	Success bool
}

func (e *EcsService) DeleteInstance(ctx context.Context, req *DeleteInstanceRequest) (*DeleteInstanceResponse, error) {
	client, err := e.sdk.CreateEcsClient(req.Region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	request := &ecs.DeleteInstanceRequest{
		InstanceId: tea.String(req.InstanceID),
		Force:      tea.Bool(req.Force),
	}

	e.sdk.logger.Info("开始删除ECS实例", zap.String("region", req.Region), zap.String("instanceID", req.InstanceID))
	_, err = client.DeleteInstance(request)
	if err != nil {
		e.sdk.logger.Error("删除ECS实例失败", zap.Error(err))
		return nil, HandleError(err)
	}

	e.sdk.logger.Info("删除ECS实例成功", zap.String("instanceID", req.InstanceID))
	return &DeleteInstanceResponse{Success: true}, nil
}

type ListInstancesRequest struct {
	Region string // 地域
	ZoneId string // 可用区
	Page   int    // 页码
	Size   int    // 每页大小
}

type ListInstancesResponse struct {
	Instances []*ecs.DescribeInstancesResponseBodyInstancesInstance
	Total     int64
}

func (e *EcsService) ListInstances(ctx context.Context, req *ListInstancesRequest) (*ListInstancesResponse, error) {
	client, err := e.sdk.CreateEcsClient(req.Region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	pageNumber := req.Page
	if pageNumber <= 0 {
		pageNumber = 1
	}

	pageSize := req.Size
	if pageSize <= 0 {
		pageSize = 100
	}

	request := &ecs.DescribeInstancesRequest{
		RegionId:   tea.String(req.Region),
		PageNumber: tea.Int32(int32(pageNumber)),
		PageSize:   tea.Int32(int32(pageSize)),
		ZoneId:     tea.String(req.ZoneId),
	}

	e.sdk.logger.Info("查询ECS实例列表",
		zap.String("region", req.Region),
		zap.String("zoneId", req.ZoneId),
		zap.Int("page", pageNumber),
		zap.Int("size", pageSize))

	response, err := client.DescribeInstances(request)
	if err != nil {
		e.sdk.logger.Error("查询ECS实例列表失败", zap.Error(err))
		return nil, HandleError(err)
	}

	var instances []*ecs.DescribeInstancesResponseBodyInstancesInstance
	var totalCount int64

	if response.Body != nil && response.Body.Instances != nil && response.Body.Instances.Instance != nil {
		instances = response.Body.Instances.Instance
		totalCount = int64(tea.Int32Value(response.Body.TotalCount))
	}

	return &ListInstancesResponse{
		Instances: instances,
		Total:     totalCount,
	}, nil
}

type GetInstanceDetailRequest struct {
	Region     string // 地域
	InstanceID string // 实例ID
}

type GetInstanceDetailResponse struct {
	Instance *ecs.DescribeInstanceAttributeResponseBody
}

func (e *EcsService) GetInstanceDetail(ctx context.Context, req *GetInstanceDetailRequest) (*GetInstanceDetailResponse, error) {
	client, err := e.sdk.CreateEcsClient(req.Region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	request := &ecs.DescribeInstanceAttributeRequest{
		InstanceId: tea.String(req.InstanceID),
	}

	response, err := client.DescribeInstanceAttribute(request)
	if err != nil {
		e.sdk.logger.Error("查询ECS实例详情失败", zap.Error(err))
		return nil, HandleError(err)
	}

	return &GetInstanceDetailResponse{
		Instance: response.Body,
	}, nil
}

type ListRegionsRequest struct {
	AcceptLanguage string // 语言
}

type ListRegionsResponse struct {
	Regions []*ecs.DescribeRegionsResponseBodyRegionsRegion
	Total   int64
}

func (e *EcsService) ListRegions(ctx context.Context, req *ListRegionsRequest) (*ListRegionsResponse, error) {
	client, err := e.sdk.CreateEcsClient("cn-hangzhou")
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	language := "zh-CN"
	if req != nil && req.AcceptLanguage != "" {
		language = req.AcceptLanguage
	}

	request := &ecs.DescribeRegionsRequest{
		AcceptLanguage: tea.String(language),
	}

	e.sdk.logger.Info("开始查询区域列表")
	response, err := client.DescribeRegions(request)
	if err != nil {
		e.sdk.logger.Error("查询区域列表失败", zap.Error(err))
		return nil, HandleError(err)
	}

	regions := response.Body.Regions.Region
	e.sdk.logger.Info("查询区域列表成功", zap.Int("count", len(regions)))

	return &ListRegionsResponse{
		Regions: regions,
		Total:   int64(len(regions)),
	}, nil
}

type ListZonesRequest struct {
	Region         string // 地域
	AcceptLanguage string // 语言
}

type ListZonesResponse struct {
	Zones []*ecs.DescribeZonesResponseBodyZonesZone
	Total int64
}

func (e *EcsService) ListZones(ctx context.Context, req *ListZonesRequest) (*ListZonesResponse, error) {
	client, err := e.sdk.CreateEcsClient(req.Region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	language := "zh-CN"
	if req.AcceptLanguage != "" {
		language = req.AcceptLanguage
	}

	request := &ecs.DescribeZonesRequest{
		RegionId:       tea.String(req.Region),
		AcceptLanguage: tea.String(language),
	}

	e.sdk.logger.Info("开始查询可用区列表", zap.String("region", req.Region))
	response, err := client.DescribeZones(request)
	if err != nil {
		e.sdk.logger.Error("查询可用区列表失败", zap.String("region", req.Region), zap.Error(err))
		return nil, HandleError(err)
	}

	zones := response.Body.Zones.Zone
	e.sdk.logger.Info("查询可用区列表成功", zap.String("region", req.Region), zap.Int("count", len(zones)))

	return &ListZonesResponse{
		Zones: zones,
		Total: int64(len(zones)),
	}, nil
}

type ListImagesRequest struct {
	Region          string // 地域
	InstanceType    string // 实例类型，镜像受实例类型影响
	ImageOwnerAlias string // 镜像来源，默认为system
	Status          string // 镜像状态，默认为Available
}

type ListImagesResponse struct {
	Images []*ecs.DescribeImagesResponseBodyImagesImage
	Total  int64
}

func (e *EcsService) ListImages(ctx context.Context, req *ListImagesRequest) (*ListImagesResponse, error) {
	client, err := e.sdk.CreateEcsClient(req.Region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	request := &ecs.DescribeImagesRequest{
		RegionId:        tea.String(req.Region),
		Status:          tea.String(req.Status),
		ImageOwnerAlias: tea.String(req.ImageOwnerAlias),
	}

	// 如果指定了实例类型，则筛选适用于该实例类型的镜像
	if req.InstanceType != "" {
		request.InstanceType = tea.String(req.InstanceType)
	}

	e.sdk.logger.Info("开始查询镜像列表", zap.String("region", req.Region), zap.String("instanceType", req.InstanceType))
	response, err := client.DescribeImages(request)
	if err != nil {
		e.sdk.logger.Error("查询镜像列表失败", zap.String("region", req.Region), zap.Error(err))
		return nil, HandleError(err)
	}

	images := response.Body.Images.Image
	e.sdk.logger.Info("查询镜像列表成功", zap.String("region", req.Region), zap.Int("count", len(images)))

	return &ListImagesResponse{
		Images: images,
		Total:  int64(len(images)),
	}, nil
}

type ListDisksRequest struct {
	Region     string // 地域
	ZoneId     string // 可用区
	InstanceId string // 实例ID
	DiskType   string // 磁盘类型
	Status     string // 磁盘状态
	Page       int    // 页码
	Size       int    // 每页大小
}

type ListDisksResponse struct {
	Disks []*ecs.DescribeDisksResponseBodyDisksDisk
	Total int64
}

func (e *EcsService) ListDisks(ctx context.Context, req *ListDisksRequest) (*ListDisksResponse, error) {
	client, err := e.sdk.CreateEcsClient(req.Region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	request := &ecs.DescribeDisksRequest{
		RegionId:   tea.String(req.Region),
		PageNumber: tea.Int32(int32(req.Page)),
		PageSize:   tea.Int32(int32(req.Size)),
	}

	if req.ZoneId != "" {
		request.ZoneId = tea.String(req.ZoneId)
	}
	if req.InstanceId != "" {
		request.InstanceId = tea.String(req.InstanceId)
	}
	if req.DiskType != "" {
		request.DiskType = tea.String(req.DiskType)
	}
	if req.Status != "" {
		request.Status = tea.String(req.Status)
	}

	e.sdk.logger.Info("开始查询磁盘列表", zap.String("region", req.Region), zap.String("zoneId", req.ZoneId))
	response, err := client.DescribeDisks(request)
	if err != nil {
		e.sdk.logger.Error("查询磁盘列表失败", zap.String("region", req.Region), zap.Error(err))
		return nil, HandleError(err)
	}

	disks := response.Body.Disks.Disk
	totalCount := int64(tea.Int32Value(response.Body.TotalCount))
	e.sdk.logger.Info("查询磁盘列表成功", zap.String("region", req.Region), zap.Int("count", len(disks)))

	return &ListDisksResponse{
		Disks: disks,
		Total: totalCount,
	}, nil
}

type ListInstanceTypesRequest struct {
	Region             string // 地域
	ZoneId             string // 可用区
	InstanceChargeType string // 付费类型：PrePaid(包年包月)、PostPaid(按量付费)
	MaxResults         int64  // 最大返回数量
	NextToken          string // 下一页的token
}

type ListInstanceTypesResponse struct {
	InstanceTypes []*ecs.DescribeInstanceTypesResponseBodyInstanceTypesInstanceType
	Total         int64
}

func (e *EcsService) ListInstanceTypes(ctx context.Context, req *ListInstanceTypesRequest) (*ListInstanceTypesResponse, error) {
	client, err := e.sdk.CreateEcsClient(req.Region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	request := &ecs.DescribeInstanceTypesRequest{
		MaxResults: tea.Int64(req.MaxResults),
		NextToken:  tea.String(req.NextToken),
	}

	e.sdk.logger.Info("开始查询实例类型列表", zap.String("region", req.Region))
	response, err := client.DescribeInstanceTypes(request)
	if err != nil {
		e.sdk.logger.Error("查询实例类型列表失败", zap.String("region", req.Region), zap.Error(err))
		return nil, HandleError(err)
	}

	// 获取实例类型列表
	instanceTypes := response.Body.InstanceTypes.InstanceType

	// 如果没有指定可用区和付费类型，直接返回所有实例类型
	if req.ZoneId == "" && req.InstanceChargeType == "" {
		e.sdk.logger.Info("查询实例类型列表成功", zap.String("region", req.Region), zap.Int("count", len(instanceTypes)))
		return &ListInstanceTypesResponse{
			InstanceTypes: instanceTypes,
			Total:         int64(len(instanceTypes)),
		}, nil
	}

	// 查询可用资源
	availableRequest := &ecs.DescribeAvailableResourceRequest{
		RegionId:            tea.String(req.Region),
		DestinationResource: tea.String("InstanceType"),
	}

	if req.ZoneId != "" {
		availableRequest.ZoneId = tea.String(req.ZoneId)
	}

	if req.InstanceChargeType != "" {
		availableRequest.InstanceChargeType = tea.String(req.InstanceChargeType)
	}

	availableResponse, err := client.DescribeAvailableResource(availableRequest)
	if err != nil {
		e.sdk.logger.Error("查询可用资源失败", zap.String("region", req.Region), zap.Error(err))
		return nil, HandleError(err)
	}

	availableTypes := make(map[string]bool)

	if availableResponse.Body != nil && availableResponse.Body.AvailableZones != nil {
		for _, zone := range availableResponse.Body.AvailableZones.AvailableZone {
			if zone.AvailableResources == nil {
				continue
			}

			for _, resource := range zone.AvailableResources.AvailableResource {
				if resource.SupportedResources == nil {
					continue
				}

				// 批量处理支持的资源
				for _, supportedResource := range resource.SupportedResources.SupportedResource {
					if supportedResource != nil && tea.StringValue(supportedResource.Status) == "Available" {
						availableTypes[tea.StringValue(supportedResource.Value)] = true
					}
				}
			}
		}
	}

	filteredTypes := make([]*ecs.DescribeInstanceTypesResponseBodyInstanceTypesInstanceType, 0, len(instanceTypes))

	for _, instanceType := range instanceTypes {
		if instanceType != nil {
			typeID := tea.StringValue(instanceType.InstanceTypeId)
			if availableTypes[typeID] {
				filteredTypes = append(filteredTypes, instanceType)
			}
		}
	}

	e.sdk.logger.Info("查询实例类型列表成功", zap.String("region", req.Region), zap.Int("count", len(filteredTypes)))
	return &ListInstanceTypesResponse{
		InstanceTypes: filteredTypes,
		Total:         int64(len(filteredTypes)),
	}, nil
}
