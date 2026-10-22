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

package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"go.uber.org/zap"
)

// AliyunClient 阿里云客户端封装
type aliyunECSAPI interface {
	DescribeInstances(*ecs.DescribeInstancesRequest) (*ecs.DescribeInstancesResponse, error)
	DescribeRegions(*ecs.DescribeRegionsRequest) (*ecs.DescribeRegionsResponse, error)
}

type AliyunClient struct {
	client aliyunECSAPI
	region string
	logger *zap.Logger
}

func NewAliyunClient(accessKey, secretKey, region string, logger *zap.Logger) (*AliyunClient, error) {
	if strings.TrimSpace(accessKey) == "" || strings.TrimSpace(secretKey) == "" {
		return nil, fmt.Errorf("阿里云AccessKey和SecretKey不能为空")
	}
	if region == "" {
		return nil, fmt.Errorf("阿里云Region不能为空")
	}
	if logger == nil {
		logger = zap.NewNop()
	}

	client, err := ecs.NewClientWithAccessKey(region, accessKey, secretKey)
	if err != nil {
		return nil, fmt.Errorf("创建阿里云客户端失败: %w", err)
	}

	return &AliyunClient{
		client: client,
		region: region,
		logger: logger,
	}, nil
}

// prepareAliyunRequest bounds each legacy SDK request. This SDK has no
// context-aware API; cancellation is checked between calls and deadlines cap I/O.
func prepareAliyunRequest(ctx context.Context, request requests.AcsRequest) error {
	if ctx == nil {
		return fmt.Errorf("请求上下文不能为空")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	timeout := 30 * time.Second
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < timeout {
		timeout = time.Until(deadline)
	}
	if timeout <= 0 {
		return context.DeadlineExceeded
	}
	request.SetReadTimeout(timeout)
	connect := 5 * time.Second
	if timeout < connect {
		connect = timeout
	}
	request.SetConnectTimeout(connect)
	return nil
}

func (c *AliyunClient) VerifyCredentials(ctx context.Context) error {
	// DescribeRegions requires no RAM authorization. A bounded read confirms the
	// exact ecs:DescribeInstances permission required by resource synchronization.
	request := ecs.CreateDescribeInstancesRequest()
	request.Scheme = "https"
	request.RegionId = c.region
	request.MaxResults = requests.NewInteger(10)
	if err := prepareAliyunRequest(ctx, request); err != nil {
		return err
	}
	response, err := c.client.DescribeInstances(request)
	if err != nil {
		return fmt.Errorf("阿里云ECS读取凭证/权限验证失败: %w", err)
	}
	if response == nil {
		return fmt.Errorf("阿里云返回空响应")
	}
	return ctx.Err()
}

// ListECSInstances uses token pagination, including filtered batches of <=100 IDs.
func (c *AliyunClient) ListECSInstances(ctx context.Context, instanceIDs []string) ([]*model.TreeCloudResource, error) {
	batches := [][]string{nil}
	if len(instanceIDs) > 0 {
		batches = nil
		for _, id := range instanceIDs {
			if strings.TrimSpace(id) == "" {
				return nil, fmt.Errorf("实例ID不能为空")
			}
		}
		for start := 0; start < len(instanceIDs); start += 100 {
			end := start + 100
			if end > len(instanceIDs) {
				end = len(instanceIDs)
			}
			batches = append(batches, instanceIDs[start:end])
		}
	}
	resources := make([]*model.TreeCloudResource, 0)
	seenIDs := make(map[string]bool)
	for _, batch := range batches {
		next := ""
		tokens := make(map[string]bool)
		for {
			request := ecs.CreateDescribeInstancesRequest()
			request.Scheme = "https"
			request.RegionId = c.region
			request.MaxResults = requests.NewInteger(100)
			request.NextToken = next
			if len(batch) > 0 {
				encoded, _ := json.Marshal(batch)
				request.InstanceIds = string(encoded)
			}
			if err := prepareAliyunRequest(ctx, request); err != nil {
				return nil, err
			}
			response, err := c.client.DescribeInstances(request)
			if err != nil {
				return nil, fmt.Errorf("获取ECS实例失败: %w", err)
			}
			if err := ctx.Err(); err != nil {
				return nil, err
			}
			if response == nil {
				return nil, fmt.Errorf("阿里云返回空响应")
			}
			for _, instance := range response.Instances.Instance {
				if !seenIDs[instance.InstanceId] {
					resources = append(resources, c.convertECSToResource(&instance))
					seenIDs[instance.InstanceId] = true
				}
			}
			next = response.NextToken
			if next == "" {
				break
			}
			if tokens[next] {
				return nil, fmt.Errorf("阿里云返回重复分页令牌，终止不完整同步")
			}
			tokens[next] = true
		}
	}
	return resources, nil
}

// convertECSToResource 将阿里云ECS实例转换为内部资源模型
func (c *AliyunClient) convertECSToResource(instance *ecs.Instance) *model.TreeCloudResource {
	resource := &model.TreeCloudResource{
		Name:         instance.InstanceName,
		ResourceType: model.ResourceTypeECS,
		InstanceID:   instance.InstanceId,
		InstanceType: instance.InstanceType,
		Status:       c.convertECSStatus(instance.Status),
		Region:       c.region,
		ZoneID:       instance.ZoneId,
		VpcID:        instance.VpcAttributes.VpcId,
		OSType:       instance.OSType,
		OSName:       instance.OSName,
		ImageID:      instance.ImageId,
		Cpu:          instance.Cpu,
		Memory:       instance.Memory / 1024, // 阿里云返回的是MB，转换为GB
	}

	if len(instance.PublicIpAddress.IpAddress) > 0 {
		resource.PublicIP = instance.PublicIpAddress.IpAddress[0]
	}

	if resource.PublicIP == "" {
		resource.PublicIP = instance.EipAddress.IpAddress
	}

	if len(instance.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
		resource.PrivateIP = instance.VpcAttributes.PrivateIpAddress.IpAddress[0]
	}

	if instance.OSName != "" {
		resource.ImageName = instance.OSName
	} else {
		resource.ImageName = instance.ImageId
	}

	switch instance.InstanceChargeType {
	case "PostPaid":
		resource.ChargeType = model.ChargeTypePostPaid
	case "PrePaid":
		resource.ChargeType = model.ChargeTypePrePaid
		if instance.ExpiredTime != "" {
			for _, layout := range []string{time.RFC3339, "2006-01-02T15:04Z"} {
				if expireTime, err := time.Parse(layout, instance.ExpiredTime); err == nil {
					resource.ExpireTime = &expireTime
					break
				}
			}
		}
	}

	resource.Currency = model.CurrencyCNY

	var tags model.KeyValueList
	for _, tag := range instance.Tags.Tag {
		tags = append(tags, model.KeyValue{
			Key:   tag.TagKey,
			Value: tag.TagValue,
		})
	}
	resource.Tags = tags

	resource.Port = 22
	resource.Username = "root"

	return resource
}

// convertECSStatus 转换阿里云ECS状态到内部状态
func (c *AliyunClient) convertECSStatus(status string) model.CloudResourceStatus {
	switch status {
	case "Running":
		return model.CloudResourceRunning
	case "Stopped":
		return model.CloudResourceStopped
	case "Starting":
		return model.CloudResourceStarting
	case "Stopping":
		return model.CloudResourceStopping
	default:
		return model.CloudResourceUnknown
	}
}

// GetECSInstanceByID 根据实例ID获取单个ECS实例
func (c *AliyunClient) GetECSInstanceByID(ctx context.Context, instanceID string) (*model.TreeCloudResource, error) {
	resources, err := c.ListECSInstances(ctx, []string{instanceID})
	if err != nil {
		return nil, err
	}

	if len(resources) == 0 {
		return nil, fmt.Errorf("未找到实例: %s", instanceID)
	}

	return resources[0], nil
}

// GetAvailableRegions 获取阿里云所有可用区域列表
func (c *AliyunClient) GetAvailableRegions(ctx context.Context) ([]model.AvailableRegion, error) {
	request := ecs.CreateDescribeRegionsRequest()
	request.Scheme = "https"
	request.AcceptLanguage = "zh-CN"
	request.ResourceType = "instance"
	if err := prepareAliyunRequest(ctx, request); err != nil {
		return nil, err
	}

	c.logger.Debug("调用阿里云DescribeRegions API")

	response, err := c.client.DescribeRegions(request)
	if err != nil {
		c.logger.Error("调用阿里云DescribeRegions API失败", zap.Error(err))
		return nil, fmt.Errorf("获取阿里云区域列表失败: %w", err)
	}

	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if response == nil {
		return nil, fmt.Errorf("阿里云返回空区域响应")
	}
	c.logger.Info("成功获取阿里云区域列表",
		zap.Int("regionCount", len(response.Regions.Region)))

	var regions []model.AvailableRegion
	for _, region := range response.Regions.Region {
		regions = append(regions, model.AvailableRegion{
			Region:     region.RegionId,
			RegionName: region.LocalName,
			Available:  region.Status == "available",
		})
	}

	return regions, nil
}
