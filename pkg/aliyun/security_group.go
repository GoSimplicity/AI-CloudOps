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

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	ecs "github.com/alibabacloud-go/ecs-20140526/v2/client"
	"github.com/alibabacloud-go/tea/tea"
	"go.uber.org/zap"
)

type SecurityGroupService struct {
	sdk *SDK
}

func NewSecurityGroupService(sdk *SDK) *SecurityGroupService {
	return &SecurityGroupService{sdk: sdk}
}

type CreateSecurityGroupRequest struct {
	Region             string
	SecurityGroupName  string
	Description        string
	VpcId              string
	SecurityGroupType  string
	ResourceGroupId    string
	Tags               map[string]string
	SecurityGroupRules []*model.SecurityGroupRule
}

type CreateSecurityGroupResponseBody struct {
	SecurityGroupId string
}

// CreateSecurityGroup 创建安全组
func (s *SecurityGroupService) CreateSecurityGroup(ctx context.Context, req *CreateSecurityGroupRequest) (*CreateSecurityGroupResponseBody, error) {
	client, err := s.sdk.CreateEcsClient(req.Region)
	if err != nil {
		s.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	request := &ecs.CreateSecurityGroupRequest{
		RegionId:          tea.String(req.Region),
		SecurityGroupName: tea.String(req.SecurityGroupName),
		Description:       tea.String(req.Description),
		VpcId:             tea.String(req.VpcId),
		SecurityGroupType: tea.String(req.SecurityGroupType),
		ResourceGroupId:   tea.String(req.ResourceGroupId),
	}

	// 设置标签
	if len(req.Tags) > 0 {
		tags := make([]*ecs.CreateSecurityGroupRequestTag, 0, len(req.Tags))
		for k, v := range req.Tags {
			tags = append(tags, &ecs.CreateSecurityGroupRequestTag{
				Key:   tea.String(k),
				Value: tea.String(v),
			})
		}
		request.Tag = tags
	}

	s.sdk.logger.Info("开始创建安全组", zap.String("region", req.Region), zap.Any("request", req))
	response, err := client.CreateSecurityGroup(request)
	if err != nil {
		s.sdk.logger.Error("创建安全组失败", zap.Error(err))
		return nil, err
	}

	securityGroupId := tea.StringValue(response.Body.SecurityGroupId)
	s.sdk.logger.Info("创建安全组成功", zap.String("securityGroupID", securityGroupId))

	// 如果有安全组规则，添加规则
	if len(req.SecurityGroupRules) > 0 {
		for _, rule := range req.SecurityGroupRules {
			authRequest := &ecs.AuthorizeSecurityGroupRequest{
				RegionId:        tea.String(req.Region),
				SecurityGroupId: tea.String(securityGroupId),
				IpProtocol:      tea.String(rule.IpProtocol),
				PortRange:       tea.String(rule.PortRange),
				SourceCidrIp:    tea.String(rule.SourceCidrIp),
				Description:     tea.String(rule.Description),
			}

			_, err := client.AuthorizeSecurityGroup(authRequest)
			if err != nil {
				s.sdk.logger.Error("添加安全组规则失败", zap.Error(err), zap.Any("rule", rule))
				return nil, err
			}
		}
		s.sdk.logger.Info("添加安全组规则成功", zap.Int("ruleCount", len(req.SecurityGroupRules)))
	}

	return &CreateSecurityGroupResponseBody{
		SecurityGroupId: securityGroupId,
	}, nil
}

// DeleteSecurityGroup 删除安全组
func (s *SecurityGroupService) DeleteSecurityGroup(ctx context.Context, region string, securityGroupID string) error {
	client, err := s.sdk.CreateEcsClient(region)
	if err != nil {
		s.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return err
	}

	request := &ecs.DeleteSecurityGroupRequest{
		RegionId:        tea.String(region),
		SecurityGroupId: tea.String(securityGroupID),
	}

	s.sdk.logger.Info("开始删除安全组", zap.String("region", region), zap.String("securityGroupID", securityGroupID))
	_, err = client.DeleteSecurityGroup(request)
	if err != nil {
		s.sdk.logger.Error("删除安全组失败", zap.Error(err))
		return err
	}

	s.sdk.logger.Info("删除安全组成功", zap.String("securityGroupID", securityGroupID))
	return nil
}

// GetSecurityGroupDetail 获取安全组详情
func (s *SecurityGroupService) GetSecurityGroupDetail(ctx context.Context, region string, securityGroupID string) (*ecs.DescribeSecurityGroupAttributeResponseBody, error) {
	client, err := s.sdk.CreateEcsClient(region)
	if err != nil {
		s.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, err
	}

	request := &ecs.DescribeSecurityGroupAttributeRequest{
		RegionId:        tea.String(region),
		SecurityGroupId: tea.String(securityGroupID),
	}

	s.sdk.logger.Info("开始获取安全组详情", zap.String("region", region), zap.String("securityGroupID", securityGroupID))
	response, err := client.DescribeSecurityGroupAttribute(request)
	if err != nil {
		s.sdk.logger.Error("获取安全组详情失败", zap.Error(err))
		return nil, err
	}

	return response.Body, nil
}

// ListSecurityGroupsRequest 查询安全组列表请求参数
type ListSecurityGroupsRequest struct {
	Region     string
	PageNumber int
	PageSize   int
}

// ListSecurityGroupsResponseBody 查询安全组列表响应
type ListSecurityGroupsResponseBody struct {
	SecurityGroups []*ecs.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup
	Total          int64
}

// ListSecurityGroups 查询安全组列表（支持分页获取全部资源）
func (s *SecurityGroupService) ListSecurityGroups(ctx context.Context, req *ListSecurityGroupsRequest) (*ListSecurityGroupsResponseBody, error) {
	var allSecurityGroups []*ecs.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup
	var totalCount int64 = 0
	page := 1
	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	for {
		client, err := s.sdk.CreateEcsClient(req.Region)
		if err != nil {
			return nil, err
		}

		request := &ecs.DescribeSecurityGroupsRequest{
			RegionId:   tea.String(req.Region),
			PageNumber: tea.Int32(int32(page)),
			PageSize:   tea.Int32(int32(pageSize)),
		}

		response, err := client.DescribeSecurityGroups(request)
		if err != nil {
			return nil, err
		}

		if response.Body == nil || response.Body.SecurityGroups == nil || response.Body.SecurityGroups.SecurityGroup == nil {
			break
		}

		securityGroups := response.Body.SecurityGroups.SecurityGroup
		if len(securityGroups) == 0 {
			break
		}

		allSecurityGroups = append(allSecurityGroups, securityGroups...)
		totalCount = int64(tea.Int32Value(response.Body.TotalCount))

		if len(securityGroups) < pageSize {
			break
		}

		page++
	}

	startIdx := (req.PageNumber - 1) * req.PageSize
	endIdx := req.PageNumber * req.PageSize
	if startIdx >= len(allSecurityGroups) {
		return &ListSecurityGroupsResponseBody{
			SecurityGroups: []*ecs.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup{},
			Total:          totalCount,
		}, nil
	}

	if endIdx > len(allSecurityGroups) {
		endIdx = len(allSecurityGroups)
	}

	return &ListSecurityGroupsResponseBody{
		SecurityGroups: allSecurityGroups[startIdx:endIdx],
		Total:          totalCount,
	}, nil
}
