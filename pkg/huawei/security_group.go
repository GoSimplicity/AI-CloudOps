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
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	vpcmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3/model"
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
	client, err := s.sdk.CreateVpcClient(req.Region, s.sdk.accessKey)
	if err != nil {
		return nil, err
	}

	request := &vpcmodel.CreateSecurityGroupRequest{
		Body: &vpcmodel.CreateSecurityGroupRequestBody{
			SecurityGroup: &vpcmodel.CreateSecurityGroupOption{
				Name:        req.SecurityGroupName,
				Description: &req.Description,
			},
		},
	}

	response, err := client.CreateSecurityGroup(request)
	if err != nil {
		return nil, err
	}

	securityGroupId := ""
	if response.SecurityGroup != nil {
		securityGroupId = response.SecurityGroup.Id
	}
	if securityGroupId == "" {
		return nil, fmt.Errorf("未获取到安全组ID")
	}

	// 如果有安全组规则，添加规则
	if len(req.SecurityGroupRules) > 0 {
		for _, rule := range req.SecurityGroupRules {
			authRequest := &vpcmodel.CreateSecurityGroupRuleRequest{
				Body: &vpcmodel.CreateSecurityGroupRuleRequestBody{
					SecurityGroupRule: &vpcmodel.CreateSecurityGroupRuleOption{
						SecurityGroupId: securityGroupId,
						Direction:       rule.Direction,
						Protocol:        &rule.IpProtocol,
						Ethertype:       nil, // 使用默认值
						Multiport:       &rule.PortRange,
						RemoteIpPrefix:  &rule.SourceCidrIp,
						Description:     &rule.Description,
					},
				},
			}

			_, err := client.CreateSecurityGroupRule(authRequest)
			if err != nil {
				return nil, err
			}
		}
	}

	return &CreateSecurityGroupResponseBody{
		SecurityGroupId: securityGroupId,
	}, nil
}

// DeleteSecurityGroup 删除安全组
func (s *SecurityGroupService) DeleteSecurityGroup(ctx context.Context, region string, securityGroupID string) error {
	client, err := s.sdk.CreateVpcClient(region, s.sdk.accessKey)
	if err != nil {
		return err
	}

	request := &vpcmodel.DeleteSecurityGroupRequest{
		SecurityGroupId: securityGroupID,
	}

	_, err = client.DeleteSecurityGroup(request)
	if err != nil {
		return err
	}

	return nil
}

// GetSecurityGroupDetail 获取安全组详情
func (s *SecurityGroupService) GetSecurityGroupDetail(ctx context.Context, region string, securityGroupID string) (*vpcmodel.SecurityGroupInfo, error) {
	client, err := s.sdk.CreateVpcClient(region, s.sdk.accessKey)
	if err != nil {
		return nil, err
	}

	request := &vpcmodel.ShowSecurityGroupRequest{
		SecurityGroupId: securityGroupID,
	}

	response, err := client.ShowSecurityGroup(request)
	if err != nil {
		return nil, err
	}

	return response.SecurityGroup, nil
}

// ListSecurityGroupsRequest 查询安全组列表请求参数
type ListSecurityGroupsRequest struct {
	Region     string
	PageNumber int
	PageSize   int
}

// ListSecurityGroupsResponseBody 查询安全组列表响应
type ListSecurityGroupsResponseBody struct {
	SecurityGroups []vpcmodel.SecurityGroup
	Total          int32
}

// ListSecurityGroups 查询安全组列表（支持分页获取全部资源）
func (s *SecurityGroupService) ListSecurityGroups(ctx context.Context, req *ListSecurityGroupsRequest) (*ListSecurityGroupsResponseBody, int64, error) {
	var allSecurityGroups []vpcmodel.SecurityGroup
	var totalCount int64 = 0
	
	client, err := s.sdk.CreateVpcClient(req.Region, s.sdk.accessKey)
	if err != nil {
		return nil, 0, err
	}

	limit := int32(req.PageSize)
	if limit <= 0 {
		limit = 100
	}

	request := &vpcmodel.ListSecurityGroupsRequest{
		Limit: &limit,
	}

	response, err := client.ListSecurityGroups(request)
	if err != nil {
		return nil, 0, err
	}

	if response.SecurityGroups != nil {
		allSecurityGroups = *response.SecurityGroups
		totalCount = int64(len(allSecurityGroups))
	}

	startIdx := (req.PageNumber - 1) * req.PageSize
	endIdx := req.PageNumber * req.PageSize
	if startIdx >= len(allSecurityGroups) {
		return &ListSecurityGroupsResponseBody{
			SecurityGroups: []vpcmodel.SecurityGroup{},
		}, totalCount, nil
	}

	if endIdx > len(allSecurityGroups) {
		endIdx = len(allSecurityGroups)
	}

	return &ListSecurityGroupsResponseBody{
		SecurityGroups: allSecurityGroups[startIdx:endIdx],
	}, totalCount, nil
}
