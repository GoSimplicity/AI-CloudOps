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

// ListECSInstances 获取ECS实例列表
func (c *AliyunClient) ListECSInstances(ctx context.Context, instanceIDs []string) ([]*model.TreeCloudResource, error) {
	var allResources []*model.TreeCloudResource
	pageNumber := 1
	pageSize := 100

	c.logger.Info("开始获取阿里云ECS实例",
		zap.String("region", c.region),
		zap.Int("specifiedInstanceCount", len(instanceIDs)))

	for {
		request := ecs.CreateDescribeInstancesRequest()
		request.Scheme = "https"
		request.PageSize = requests.NewInteger(pageSize)
		request.PageNumber = requests.NewInteger(pageNumber)

		// 如果指定了实例ID，则只获取指定的实例
		if len(instanceIDs) > 0 {
			request.InstanceIds = fmt.Sprintf(`["%s"]`, strings.Join(instanceIDs, `","`))
		}

		c.logger.Debug("调用阿里云API",
			zap.Int("pageNumber", pageNumber),
			zap.Int("pageSize", pageSize))

		response, err := c.client.DescribeInstances(request)
		if err != nil {
			c.logger.Error("阿里云API调用失败",
				zap.Int("pageNumber", pageNumber),
				zap.Error(err))
			return nil, fmt.Errorf("获取ECS实例列表失败(页码:%d): %w", pageNumber, err)
		}

		c.logger.Info("阿里云API返回",
			zap.Int("pageNumber", pageNumber),
			zap.Int("totalCount", response.TotalCount),
			zap.Int("currentPageCount", len(response.Instances.Instance)),
			zap.Int("pageSize", response.PageSize))

		// 转换实例数据
		for _, instance := range response.Instances.Instance {
			resource := c.convertECSToResource(&instance)
			allResources = append(allResources, resource)
		}

		// 使用TotalCount来判断是否还有下一页
		// 如果已经获取的资源数量大于等于总数，停止分页
		if len(allResources) >= response.TotalCount {
			c.logger.Info("已获取所有实例",
				zap.Int("totalCount", response.TotalCount),
				zap.Int("fetchedCount", len(allResources)))
			break
		}

		// 判断是否为最后一页
		if len(response.Instances.Instance) < pageSize {
			c.logger.Info("到达最后一页",
				zap.Int("currentPageCount", len(response.Instances.Instance)),
				zap.Int("pageSize", pageSize))
			break
		}

		// 如果指定了实例ID，不需要分页
		if len(instanceIDs) > 0 {
			break
		}

		pageNumber++
	}

	c.logger.Info("获取阿里云ECS实例成功",
		zap.Int("count", len(allResources)),
		zap.String("region", c.region))
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

	// 设置镜像名称
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

// GetAvailableRegions 获取阿里云所有可用区域列表
func (c *AliyunClient) GetAvailableRegions(ctx context.Context) ([]model.AvailableRegion, error) {
	request := ecs.CreateDescribeRegionsRequest()
	request.Scheme = "https"

	c.logger.Debug("调用阿里云DescribeRegions API")

	response, err := c.client.DescribeRegions(request)
	if err != nil {
		c.logger.Error("调用阿里云DescribeRegions API失败", zap.Error(err))
		return nil, fmt.Errorf("获取阿里云区域列表失败: %w", err)
	}

	c.logger.Info("成功获取阿里云区域列表",
		zap.Int("regionCount", len(response.Regions.Region)))

	var regions []model.AvailableRegion
	for _, region := range response.Regions.Region {
		regions = append(regions, model.AvailableRegion{
			Region:     region.RegionId,
			RegionName: region.LocalName,
			Available:  true, // 阿里云API返回的都是可用的区域
		})
	}

	return regions, nil
}
