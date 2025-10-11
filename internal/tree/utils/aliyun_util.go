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
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/aliyun/alibaba-cloud-sdk-go/sdk/requests"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"
	"go.uber.org/zap"
)

// AliyunClient 阿里云客户端封装
type AliyunClient struct {
	client *ecs.Client
	region string
	logger *zap.Logger
}

// NewAliyunClient 创建阿里云客户端
func NewAliyunClient(accessKey, secretKey, region string, logger *zap.Logger) (*AliyunClient, error) {
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

// VerifyCredentials 验证阿里云凭证
func (c *AliyunClient) VerifyCredentials(ctx context.Context) error {
	request := ecs.CreateDescribeRegionsRequest()
	request.Scheme = "https"

	_, err := c.client.DescribeRegions(request)
	if err != nil {
		return fmt.Errorf("阿里云凭证验证失败: %w", err)
	}

	c.logger.Info("阿里云凭证验证成功")
	return nil
}

// ListECSInstances 获取ECS实例列表（支持分页获取所有实例）
func (c *AliyunClient) ListECSInstances(ctx context.Context, instanceIDs []string) ([]*model.TreeCloudResource, error) {
	var allResources []*model.TreeCloudResource
	pageNumber := 1
	pageSize := 100

	for {
		request := ecs.CreateDescribeInstancesRequest()
		request.Scheme = "https"
		request.PageSize = requests.NewInteger(pageSize)
		request.PageNumber = requests.NewInteger(pageNumber)

		// 如果指定了实例ID，则只获取指定的实例
		if len(instanceIDs) > 0 {
			request.InstanceIds = fmt.Sprintf(`["%s"]`, strings.Join(instanceIDs, `","`))
		}

		response, err := c.client.DescribeInstances(request)
		if err != nil {
			return nil, fmt.Errorf("获取ECS实例列表失败(页码:%d): %w", pageNumber, err)
		}

		// 转换实例数据
		for _, instance := range response.Instances.Instance {
			resource := c.convertECSToResource(&instance)
			allResources = append(allResources, resource)
		}

		// 检查是否还有下一页
		if len(response.Instances.Instance) < pageSize {
			break
		}

		// 如果指定了实例ID，不需要分页
		if len(instanceIDs) > 0 {
			break
		}

		pageNumber++
	}

	c.logger.Info("获取阿里云ECS实例成功", zap.Int("count", len(allResources)))
	return allResources, nil
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

	if len(instance.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
		resource.PrivateIP = instance.VpcAttributes.PrivateIpAddress.IpAddress[0]
	}

	// 设置镜像名称（如果OSName为空，使用ImageId）
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
			expireTime, err := time.Parse("2006-01-02T15:04Z", instance.ExpiredTime)
			if err == nil {
				resource.ExpireTime = &expireTime
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
