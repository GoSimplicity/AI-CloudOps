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

// CreateSecurityGroupRequest 创建安全组请求参数
type CreateSecurityGroupRequest struct {
	Region             string                      // 地域
	SecurityGroupName  string                      // 安全组名称
	Description        string                      // 描述
	VpcId              string                      // VPC ID
	SecurityGroupType  string                      // 安全组类型
	ResourceGroupId    string                      // 资源组ID
	Tags               map[string]string           // 标签
	SecurityGroupRules []*model.SecurityGroupRule  // 安全组规则
}

// CreateSecurityGroupResponse 创建安全组响应
type CreateSecurityGroupResponse struct {
	SecurityGroupId string // 安全组ID
}

// CreateSecurityGroup 创建安全组
func (s *SecurityGroupService) CreateSecurityGroup(ctx context.Context, req *CreateSecurityGroupRequest) (*CreateSecurityGroupResponse, error) {
	client, err := s.sdk.CreateEcsClient(req.Region)
	if err != nil {
		s.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
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
		return nil, HandleError(err)
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

			if _, err := client.AuthorizeSecurityGroup(authRequest); err != nil {
				s.sdk.logger.Error("添加安全组规则失败", zap.Error(err), zap.Any("rule", rule))
				return nil, HandleError(err)
			}
		}
		s.sdk.logger.Info("添加安全组规则成功", zap.Int("ruleCount", len(req.SecurityGroupRules)))
	}

	return &CreateSecurityGroupResponse{
		SecurityGroupId: securityGroupId,
	}, nil
}

// DeleteSecurityGroupRequest 删除安全组请求参数
type DeleteSecurityGroupRequest struct {
	Region         string // 地域
	SecurityGroupId string // 安全组ID
}

// DeleteSecurityGroupResponse 删除安全组响应
type DeleteSecurityGroupResponse struct {
	Success bool // 是否成功
}

// DeleteSecurityGroup 删除安全组
func (s *SecurityGroupService) DeleteSecurityGroup(ctx context.Context, req *DeleteSecurityGroupRequest) (*DeleteSecurityGroupResponse, error) {
	client, err := s.sdk.CreateEcsClient(req.Region)
	if err != nil {
		s.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	request := &ecs.DeleteSecurityGroupRequest{
		RegionId:        tea.String(req.Region),
		SecurityGroupId: tea.String(req.SecurityGroupId),
	}

	s.sdk.logger.Info("开始删除安全组", zap.String("region", req.Region), zap.String("securityGroupID", req.SecurityGroupId))
	if _, err = client.DeleteSecurityGroup(request); err != nil {
		s.sdk.logger.Error("删除安全组失败", zap.Error(err))
		return nil, HandleError(err)
	}

	s.sdk.logger.Info("删除安全组成功", zap.String("securityGroupID", req.SecurityGroupId))
	return &DeleteSecurityGroupResponse{Success: true}, nil
}

// GetSecurityGroupDetailRequest 获取安全组详情请求参数
type GetSecurityGroupDetailRequest struct {
	Region         string // 地域
	SecurityGroupId string // 安全组ID
}

// GetSecurityGroupDetailResponse 获取安全组详情响应
type GetSecurityGroupDetailResponse struct {
	SecurityGroup *ecs.DescribeSecurityGroupAttributeResponseBody // 安全组详情
}

// GetSecurityGroupDetail 获取安全组详情
func (s *SecurityGroupService) GetSecurityGroupDetail(ctx context.Context, req *GetSecurityGroupDetailRequest) (*GetSecurityGroupDetailResponse, error) {
	client, err := s.sdk.CreateEcsClient(req.Region)
	if err != nil {
		s.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	request := &ecs.DescribeSecurityGroupAttributeRequest{
		RegionId:        tea.String(req.Region),
		SecurityGroupId: tea.String(req.SecurityGroupId),
	}

	s.sdk.logger.Info("开始获取安全组详情", zap.String("region", req.Region), zap.String("securityGroupID", req.SecurityGroupId))
	response, err := client.DescribeSecurityGroupAttribute(request)
	if err != nil {
		s.sdk.logger.Error("获取安全组详情失败", zap.Error(err))
		return nil, HandleError(err)
	}

	return &GetSecurityGroupDetailResponse{SecurityGroup: response.Body}, nil
}

// ListSecurityGroupsRequest 查询安全组列表请求参数
type ListSecurityGroupsRequest struct {
	Region     string // 地域
	PageNumber int    // 页码
	PageSize   int    // 每页大小
}

// ListSecurityGroupsResponse 查询安全组列表响应
type ListSecurityGroupsResponse struct {
	SecurityGroups []*ecs.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup // 安全组列表
	Total          int64                                                               // 总数
}

// ListSecurityGroups 查询安全组列表（支持分页获取全部资源）
func (s *SecurityGroupService) ListSecurityGroups(ctx context.Context, req *ListSecurityGroupsRequest) (*ListSecurityGroupsResponse, error) {
	client, err := s.sdk.CreateEcsClient(req.Region)
	if err != nil {
		s.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	pageNumber := req.PageNumber
	if pageNumber <= 0 {
		pageNumber = 1
	}

	pageSize := req.PageSize
	if pageSize <= 0 {
		pageSize = 100
	}

	request := &ecs.DescribeSecurityGroupsRequest{
		RegionId:   tea.String(req.Region),
		PageNumber: tea.Int32(int32(pageNumber)),
		PageSize:   tea.Int32(int32(pageSize)),
	}

	s.sdk.logger.Info("查询安全组列表", 
		zap.String("region", req.Region),
		zap.Int("page", pageNumber),
		zap.Int("size", pageSize))

	response, err := client.DescribeSecurityGroups(request)
	if err != nil {
		s.sdk.logger.Error("查询安全组列表失败", zap.Error(err))
		return nil, HandleError(err)
	}

	var securityGroups []*ecs.DescribeSecurityGroupsResponseBodySecurityGroupsSecurityGroup
	var totalCount int64

	if response.Body != nil && response.Body.SecurityGroups != nil && response.Body.SecurityGroups.SecurityGroup != nil {
		securityGroups = response.Body.SecurityGroups.SecurityGroup
		totalCount = int64(tea.Int32Value(response.Body.TotalCount))
	}

	s.sdk.logger.Info("查询安全组列表成功", 
		zap.String("region", req.Region), 
		zap.Int64("total", totalCount),
		zap.Int("count", len(securityGroups)))

	return &ListSecurityGroupsResponse{
		SecurityGroups: securityGroups,
		Total:          totalCount,
	}, nil
}
