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
	Region             string
	ZoneId             string
	ImageId            string
	InstanceType       string
	SecurityGroupIds   []string
	VSwitchId          string
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
	client, err := e.sdk.CreateEcsClient(req.Region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
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
		return nil, err
	}

	instanceIds := tea.StringSliceValue(response.Body.InstanceIdSets.InstanceIdSet)
	e.sdk.logger.Info("创建ECS实例成功", zap.Strings("instanceIds", instanceIds))

	return &CreateInstanceResponseBody{
		InstanceIds: instanceIds,
	}, nil
}

// StartInstance 启动ECS实例
func (e *EcsService) StartInstance(ctx context.Context, region string, instanceID string) error {
	client, err := e.sdk.CreateEcsClient(region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.StartInstanceRequest{
		InstanceId: tea.String(instanceID),
	}

	e.sdk.logger.Info("开始启动ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
	_, err = client.StartInstance(request)
	if err != nil {
		e.sdk.logger.Error("启动ECS实例失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("启动ECS实例成功", zap.String("instanceID", instanceID))
	return nil
}

// StopInstance 停止ECS实例
func (e *EcsService) StopInstance(ctx context.Context, region string, instanceID string, forceStop bool) error {
	client, err := e.sdk.CreateEcsClient(region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.StopInstanceRequest{
		InstanceId: tea.String(instanceID),
		ForceStop:  tea.Bool(forceStop),
	}

	e.sdk.logger.Info("开始停止ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
	_, err = client.StopInstance(request)
	if err != nil {
		e.sdk.logger.Error("停止ECS实例失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("停止ECS实例成功", zap.String("instanceID", instanceID))
	return nil
}

// RestartInstance 重启ECS实例
func (e *EcsService) RestartInstance(ctx context.Context, region string, instanceID string) error {
	client, err := e.sdk.CreateEcsClient(region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.RebootInstanceRequest{
		InstanceId: tea.String(instanceID),
	}

	e.sdk.logger.Info("开始重启ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
	_, err = client.RebootInstance(request)
	if err != nil {
		e.sdk.logger.Error("重启ECS实例失败", zap.Error(err))
		return err
	}

	e.sdk.logger.Info("重启ECS实例成功", zap.String("instanceID", instanceID))
	return nil
}

// DeleteInstance 删除ECS实例
func (e *EcsService) DeleteInstance(ctx context.Context, region string, instanceID string, force bool) error {
	client, err := e.sdk.CreateEcsClient(region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.DeleteInstanceRequest{
		InstanceId: tea.String(instanceID),
		Force:      tea.Bool(force),
	}

	e.sdk.logger.Info("开始删除ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
	_, err = client.DeleteInstance(request)
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
	Instances []*ecs.DescribeInstancesResponseBodyInstancesInstance
	Total     int64
}

// ListInstances 查询ECS实例列表（支持分页获取全部资源）
func (e *EcsService) ListInstances(ctx context.Context, req *ListInstancesRequest) (*ListInstancesResponseBody, error) {
	var allInstances []*ecs.DescribeInstancesResponseBodyInstancesInstance
	var totalCount int64 = 0
	page := 1
	pageSize := req.Size
	if pageSize <= 0 {
		pageSize = 100
	}

	for {
		client, err := e.sdk.CreateEcsClient(req.Region)
		if err != nil {
			return nil, err
		}

		request := &ecs.DescribeInstancesRequest{
			RegionId:   tea.String(req.Region),
			PageNumber: tea.Int32(int32(page)),
			PageSize:   tea.Int32(int32(pageSize)),
		}

		response, err := client.DescribeInstances(request)
		if err != nil {
			return nil, err
		}

		if response.Body == nil || response.Body.Instances == nil || response.Body.Instances.Instance == nil {
			break
		}

		instances := response.Body.Instances.Instance
		if len(instances) == 0 {
			break
		}

		allInstances = append(allInstances, instances...)
		totalCount = int64(tea.Int32Value(response.Body.TotalCount))

		if len(instances) < pageSize {
			break
		}

		page++
	}

	startIdx := (req.Page - 1) * req.Size
	endIdx := req.Page * req.Size
	if startIdx >= len(allInstances) {
		return &ListInstancesResponseBody{
			Instances: []*ecs.DescribeInstancesResponseBodyInstancesInstance{},
			Total:     totalCount,
		}, nil
	}

	if endIdx > len(allInstances) {
		endIdx = len(allInstances)
	}

	return &ListInstancesResponseBody{
		Instances: allInstances[startIdx:endIdx],
		Total:     totalCount,
	}, nil
}

// GetInstanceDetail 获取ECS实例详情
func (e *EcsService) GetInstanceDetail(ctx context.Context, region string, instanceID string) (*ecs.DescribeInstanceAttributeResponseBody, error) {
	client, err := e.sdk.CreateEcsClient(region)
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	request := &ecs.DescribeInstanceAttributeRequest{
		InstanceId: tea.String(instanceID),
	}

	response, err := client.DescribeInstanceAttribute(request)
	if err != nil {
		e.sdk.logger.Error("查询ECS实例详情失败", zap.Error(err))
		return nil, err
	}

	return response.Body, nil
}

// ListRegions 查询地域列表
func (e *EcsService) ListRegions(ctx context.Context) ([]*ecs.DescribeRegionsResponseBodyRegionsRegion, error) {
	client, err := e.sdk.CreateEcsClient("cn-hangzhou")
	if err != nil {
		e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	request := &ecs.DescribeRegionsRequest{
		AcceptLanguage: tea.String("zh-CN"),
	}

	e.sdk.logger.Info("开始查询区域列表")
	response, err := client.DescribeRegions(request)
	if err != nil {
		e.sdk.logger.Error("查询区域列表失败", zap.Error(err))
		return nil, err
	}

	e.sdk.logger.Info("查询区域列表成功", zap.Int("count", len(response.Body.Regions.Region)))
	return response.Body.Regions.Region, nil
}
