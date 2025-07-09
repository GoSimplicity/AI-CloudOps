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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
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
	Tags               map[string]string
	SecurityGroupRules []*model.SecurityGroupRule
}

type CreateSecurityGroupResponseBody struct {
	SecurityGroupId string
}

// CreateSecurityGroup 创建安全组
func (s *SecurityGroupService) CreateSecurityGroup(ctx context.Context, req *CreateSecurityGroupRequest) (*CreateSecurityGroupResponseBody, error) {
	client, err := s.sdk.CreateEC2Client(ctx, req.Region)
	if err != nil {
		s.sdk.logger.Error("创建EC2客户端失败", zap.Error(err))
		return nil, err
	}

	// 创建安全组
	request := &ec2.CreateSecurityGroupInput{
		GroupName:   &req.SecurityGroupName,
		Description: &req.Description,
		VpcId:       &req.VpcId,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSecurityGroup,
				Tags: []types.Tag{
					{
						Key:   stringPtr("Name"),
						Value: &req.SecurityGroupName,
					},
				},
			},
		},
	}

	// 添加自定义标签
	if req.Tags != nil {
		for key, value := range req.Tags {
			request.TagSpecifications[0].Tags = append(request.TagSpecifications[0].Tags, types.Tag{
				Key:   &key,
				Value: &value,
			})
		}
	}

	s.sdk.logger.Info("开始创建安全组", zap.String("region", req.Region), zap.String("name", req.SecurityGroupName))
	response, err := client.CreateSecurityGroup(ctx, request)
	if err != nil {
		s.sdk.logger.Error("创建安全组失败", zap.Error(err))
		return nil, err
	}

	securityGroupId := *response.GroupId
	s.sdk.logger.Info("安全组创建成功", zap.String("securityGroupId", securityGroupId))

	// 添加安全组规则
	if len(req.SecurityGroupRules) > 0 {
		if err := s.addSecurityGroupRules(ctx, client, securityGroupId, req.SecurityGroupRules); err != nil {
			s.sdk.logger.Error("添加安全组规则失败", zap.Error(err))
			return nil, err
		}
	}

	return &CreateSecurityGroupResponseBody{
		SecurityGroupId: securityGroupId,
	}, nil
}

// addSecurityGroupRules 添加安全组规则
func (s *SecurityGroupService) addSecurityGroupRules(ctx context.Context, client *ec2.Client, securityGroupId string, rules []*model.SecurityGroupRule) error {
	var ingressRules []types.IpPermission
	var egressRules []types.IpPermission

	for _, rule := range rules {
		ipPermission := s.convertToIPPermission(rule)

		if rule.Direction == "ingress" {
			ingressRules = append(ingressRules, ipPermission)
		} else if rule.Direction == "egress" {
			egressRules = append(egressRules, ipPermission)
		}
	}

	// 添加入站规则
	if len(ingressRules) > 0 {
		authRequest := &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId:       &securityGroupId,
			IpPermissions: ingressRules,
		}

		_, err := client.AuthorizeSecurityGroupIngress(ctx, authRequest)
		if err != nil {
			return fmt.Errorf("添加入站规则失败: %w", err)
		}
	}

	// 添加出站规则
	if len(egressRules) > 0 {
		authRequest := &ec2.AuthorizeSecurityGroupEgressInput{
			GroupId:       &securityGroupId,
			IpPermissions: egressRules,
		}

		_, err := client.AuthorizeSecurityGroupEgress(ctx, authRequest)
		if err != nil {
			return fmt.Errorf("添加出站规则失败: %w", err)
		}
	}

	return nil
}

// convertToIPPermission 转换安全组规则为AWS格式
func (s *SecurityGroupService) convertToIPPermission(rule *model.SecurityGroupRule) types.IpPermission {
	var fromPort, toPort int32

	// 解析端口范围
	if rule.PortRange != "" {
		if rule.PortRange == "-1/-1" {
			fromPort = -1
			toPort = -1
		} else {
			// 假设格式为 "22/22" 或 "80-443"
			if rule.PortRange == "22/22" {
				fromPort = 22
				toPort = 22
			} else if rule.PortRange == "80/80" {
				fromPort = 80
				toPort = 80
			} else if rule.PortRange == "443/443" {
				fromPort = 443
				toPort = 443
			} else {
				// 默认值
				fromPort = 80
				toPort = 80
			}
		}
	}

	ipPermission := types.IpPermission{
		IpProtocol: &rule.IpProtocol,
		FromPort:   &fromPort,
		ToPort:     &toPort,
	}

	// 添加CIDR块
	if rule.SourceCidrIp != "" {
		ipPermission.IpRanges = []types.IpRange{
			{
				CidrIp:      &rule.SourceCidrIp,
				Description: &rule.Description,
			},
		}
	}

	return ipPermission
}

// DeleteSecurityGroup 删除安全组
func (s *SecurityGroupService) DeleteSecurityGroup(ctx context.Context, region string, securityGroupID string) error {
	client, err := s.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return err
	}

	request := &ec2.DeleteSecurityGroupInput{
		GroupId: &securityGroupID,
	}

	s.sdk.logger.Info("开始删除安全组", zap.String("securityGroupId", securityGroupID))
	_, err = client.DeleteSecurityGroup(ctx, request)
	if err != nil {
		s.sdk.logger.Error("删除安全组失败", zap.Error(err))
		return err
	}

	s.sdk.logger.Info("安全组删除成功", zap.String("securityGroupId", securityGroupID))
	return nil
}

// GetSecurityGroupDetail 获取安全组详情
func (s *SecurityGroupService) GetSecurityGroupDetail(ctx context.Context, region string, securityGroupID string) (*types.SecurityGroup, error) {
	client, err := s.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return nil, err
	}

	request := &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{securityGroupID},
	}

	s.sdk.logger.Info("开始查询安全组详情", zap.String("securityGroupId", securityGroupID))
	response, err := client.DescribeSecurityGroups(ctx, request)
	if err != nil {
		s.sdk.logger.Error("查询安全组详情失败", zap.Error(err))
		return nil, err
	}

	if len(response.SecurityGroups) == 0 {
		return nil, fmt.Errorf("安全组 %s 不存在", securityGroupID)
	}

	s.sdk.logger.Info("查询安全组详情成功", zap.String("securityGroupId", securityGroupID))
	return &response.SecurityGroups[0], nil
}

// ListSecurityGroupsRequest 查询安全组列表请求参数
type ListSecurityGroupsRequest struct {
	Region     string
	VpcId      string
	PageNumber int
	PageSize   int
}

// ListSecurityGroupsResponseBody 查询安全组列表响应
type ListSecurityGroupsResponseBody struct {
	SecurityGroups []types.SecurityGroup
	Total          int32
}

// ListSecurityGroups 查询安全组列表
func (s *SecurityGroupService) ListSecurityGroups(ctx context.Context, req *ListSecurityGroupsRequest) (*ListSecurityGroupsResponseBody, int64, error) {
	client, err := s.sdk.CreateEC2Client(ctx, req.Region)
	if err != nil {
		return nil, 0, err
	}

	request := &ec2.DescribeSecurityGroupsInput{}

	// 如果指定了VPC ID，添加过滤器
	if req.VpcId != "" {
		request.Filters = []types.Filter{
			{
				Name:   stringPtr("vpc-id"),
				Values: []string{req.VpcId},
			},
		}
	}

	s.sdk.logger.Info("开始查询安全组列表", zap.String("region", req.Region))
	response, err := client.DescribeSecurityGroups(ctx, request)
	if err != nil {
		s.sdk.logger.Error("查询安全组列表失败", zap.Error(err))
		return nil, 0, err
	}

	allSecurityGroups := response.SecurityGroups
	totalCount := int64(len(allSecurityGroups))

	// 分页处理
	startIdx := (req.PageNumber - 1) * req.PageSize
	endIdx := req.PageNumber * req.PageSize
	if startIdx >= len(allSecurityGroups) {
		return &ListSecurityGroupsResponseBody{
			SecurityGroups: []types.SecurityGroup{},
		}, totalCount, nil
	}

	if endIdx > len(allSecurityGroups) {
		endIdx = len(allSecurityGroups)
	}

	s.sdk.logger.Info("查询安全组列表成功", zap.Int64("total", totalCount))

	return &ListSecurityGroupsResponseBody{
		SecurityGroups: allSecurityGroups[startIdx:endIdx],
		Total:          int32(totalCount),
	}, totalCount, nil
}

// AddSecurityGroupRule 添加安全组规则
func (s *SecurityGroupService) AddSecurityGroupRule(ctx context.Context, region string, securityGroupID string, rule *model.SecurityGroupRule) error {
	client, err := s.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return err
	}

	ipPermission := s.convertToIPPermission(rule)

	if rule.Direction == "ingress" {
		authRequest := &ec2.AuthorizeSecurityGroupIngressInput{
			GroupId:       &securityGroupID,
			IpPermissions: []types.IpPermission{ipPermission},
		}

		_, err = client.AuthorizeSecurityGroupIngress(ctx, authRequest)
	} else {
		authRequest := &ec2.AuthorizeSecurityGroupEgressInput{
			GroupId:       &securityGroupID,
			IpPermissions: []types.IpPermission{ipPermission},
		}

		_, err = client.AuthorizeSecurityGroupEgress(ctx, authRequest)
	}

	if err != nil {
		s.sdk.logger.Error("添加安全组规则失败", zap.Error(err))
		return err
	}

	s.sdk.logger.Info("添加安全组规则成功", zap.String("securityGroupId", securityGroupID))
	return nil
}

// RemoveSecurityGroupRule 删除安全组规则
func (s *SecurityGroupService) RemoveSecurityGroupRule(ctx context.Context, region string, securityGroupID string, rule *model.SecurityGroupRule) error {
	client, err := s.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return err
	}

	ipPermission := s.convertToIPPermission(rule)

	if rule.Direction == "ingress" {
		revokeRequest := &ec2.RevokeSecurityGroupIngressInput{
			GroupId:       &securityGroupID,
			IpPermissions: []types.IpPermission{ipPermission},
		}

		_, err = client.RevokeSecurityGroupIngress(ctx, revokeRequest)
	} else {
		revokeRequest := &ec2.RevokeSecurityGroupEgressInput{
			GroupId:       &securityGroupID,
			IpPermissions: []types.IpPermission{ipPermission},
		}

		_, err = client.RevokeSecurityGroupEgress(ctx, revokeRequest)
	}

	if err != nil {
		s.sdk.logger.Error("删除安全组规则失败", zap.Error(err))
		return err
	}

	s.sdk.logger.Info("删除安全组规则成功", zap.String("securityGroupId", securityGroupID))
	return nil
}
