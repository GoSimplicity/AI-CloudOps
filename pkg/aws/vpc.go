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
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.uber.org/zap"
)

type VpcService struct {
	sdk *SDK
}

func NewVpcService(sdk *SDK) *VpcService {
	return &VpcService{sdk: sdk}
}

type CreateVpcRequest struct {
	Region           string
	VpcName          string
	CidrBlock        string
	Description      string
	AvailabilityZone string
	SubnetName       string
	SubnetCidrBlock  string
	Tags             map[string]string
}

type CreateVpcResponseBody struct {
	VpcId    string
	SubnetId string
}

// CreateVPC 创建VPC和子网
func (v *VpcService) CreateVPC(ctx context.Context, req *CreateVpcRequest) (*CreateVpcResponseBody, error) {
	client, err := v.sdk.CreateEC2Client(ctx, req.Region)
	if err != nil {
		v.sdk.logger.Error("创建EC2客户端失败", zap.Error(err))
		return nil, err
	}

	// 创建VPC
	vpcRequest := &ec2.CreateVpcInput{
		CidrBlock: &req.CidrBlock,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeVpc,
				Tags: []types.Tag{
					{
						Key:   stringPtr("Name"),
						Value: &req.VpcName,
					},
					{
						Key:   stringPtr("Description"),
						Value: &req.Description,
					},
				},
			},
		},
	}

	// 添加自定义标签
	if req.Tags != nil {
		for key, value := range req.Tags {
			vpcRequest.TagSpecifications[0].Tags = append(vpcRequest.TagSpecifications[0].Tags, types.Tag{
				Key:   &key,
				Value: &value,
			})
		}
	}

	v.sdk.logger.Info("开始创建VPC", zap.String("region", req.Region), zap.String("cidr", req.CidrBlock))
	vpcResponse, err := client.CreateVpc(ctx, vpcRequest)
	if err != nil {
		v.sdk.logger.Error("创建VPC失败", zap.Error(err))
		return nil, err
	}

	vpcId := *vpcResponse.Vpc.VpcId
	v.sdk.logger.Info("VPC创建成功", zap.String("vpcId", vpcId))

	// 等待VPC可用
	if err := v.waitForVpcAvailable(ctx, client, vpcId); err != nil {
		v.sdk.logger.Error("等待VPC可用失败", zap.Error(err))
		return nil, err
	}

	// 创建子网
	subnetRequest := &ec2.CreateSubnetInput{
		VpcId:            &vpcId,
		CidrBlock:        &req.SubnetCidrBlock,
		AvailabilityZone: &req.AvailabilityZone,
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeSubnet,
				Tags: []types.Tag{
					{
						Key:   stringPtr("Name"),
						Value: &req.SubnetName,
					},
					{
						Key:   stringPtr("Description"),
						Value: &req.Description,
					},
				},
			},
		},
	}

	v.sdk.logger.Info("开始创建子网", zap.String("vpcId", vpcId), zap.String("cidr", req.SubnetCidrBlock))
	subnetResponse, err := client.CreateSubnet(ctx, subnetRequest)
	if err != nil {
		v.sdk.logger.Error("创建子网失败", zap.Error(err))
		return nil, err
	}

	subnetId := *subnetResponse.Subnet.SubnetId
	v.sdk.logger.Info("子网创建成功", zap.String("subnetId", subnetId))

	return &CreateVpcResponseBody{
		VpcId:    vpcId,
		SubnetId: subnetId,
	}, nil
}

// waitForVpcAvailable 等待VPC可用
func (v *VpcService) waitForVpcAvailable(ctx context.Context, client *ec2.Client, vpcId string) error {
	request := &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcId},
	}

	for i := 0; i < 10; i++ {
		response, err := client.DescribeVpcs(ctx, request)
		if err != nil {
			return err
		}

		if len(response.Vpcs) > 0 && response.Vpcs[0].State == types.VpcStateAvailable {
			return nil
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("等待VPC可用超时")
}

// DeleteVPC 删除VPC
func (v *VpcService) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	client, err := v.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return err
	}

	// 1. 先删除所有子网
	if err := v.deleteSubnets(ctx, client, vpcID); err != nil {
		return err
	}

	// 2. 删除网关
	if err := v.deleteInternetGateways(ctx, client, vpcID); err != nil {
		return err
	}

	// 3. 删除VPC
	deleteReq := &ec2.DeleteVpcInput{
		VpcId: &vpcID,
	}

	v.sdk.logger.Info("开始删除VPC", zap.String("vpcId", vpcID))
	_, err = client.DeleteVpc(ctx, deleteReq)
	if err != nil {
		v.sdk.logger.Error("删除VPC失败", zap.Error(err))
		return err
	}

	v.sdk.logger.Info("VPC删除成功", zap.String("vpcId", vpcID))
	return nil
}

// deleteSubnets 删除VPC下的所有子网
func (v *VpcService) deleteSubnets(ctx context.Context, client *ec2.Client, vpcID string) error {
	// 查询VPC下的所有子网
	listReq := &ec2.DescribeSubnetsInput{
		Filters: []types.Filter{
			{
				Name:   stringPtr("vpc-id"),
				Values: []string{vpcID},
			},
		},
	}

	response, err := client.DescribeSubnets(ctx, listReq)
	if err != nil {
		return fmt.Errorf("查询子网失败: %w", err)
	}

	// 删除所有子网
	for _, subnet := range response.Subnets {
		subnetID := *subnet.SubnetId

		deleteSubnetReq := &ec2.DeleteSubnetInput{
			SubnetId: &subnetID,
		}

		v.sdk.logger.Info("删除子网", zap.String("subnetId", subnetID))
		_, err := client.DeleteSubnet(ctx, deleteSubnetReq)
		if err != nil {
			return fmt.Errorf("删除子网 %s 失败: %w", subnetID, err)
		}
	}

	return nil
}

// deleteInternetGateways 删除VPC的网关
func (v *VpcService) deleteInternetGateways(ctx context.Context, client *ec2.Client, vpcID string) error {
	// 查询VPC的网关
	listReq := &ec2.DescribeInternetGatewaysInput{
		Filters: []types.Filter{
			{
				Name:   stringPtr("attachment.vpc-id"),
				Values: []string{vpcID},
			},
		},
	}

	response, err := client.DescribeInternetGateways(ctx, listReq)
	if err != nil {
		return fmt.Errorf("查询网关失败: %w", err)
	}

	// 分离并删除所有网关
	for _, igw := range response.InternetGateways {
		igwID := *igw.InternetGatewayId

		// 分离网关
		detachReq := &ec2.DetachInternetGatewayInput{
			InternetGatewayId: &igwID,
			VpcId:             &vpcID,
		}

		v.sdk.logger.Info("分离网关", zap.String("igwId", igwID))
		_, err := client.DetachInternetGateway(ctx, detachReq)
		if err != nil {
			return fmt.Errorf("分离网关 %s 失败: %w", igwID, err)
		}

		// 删除网关
		deleteReq := &ec2.DeleteInternetGatewayInput{
			InternetGatewayId: &igwID,
		}

		v.sdk.logger.Info("删除网关", zap.String("igwId", igwID))
		_, err = client.DeleteInternetGateway(ctx, deleteReq)
		if err != nil {
			return fmt.Errorf("删除网关 %s 失败: %w", igwID, err)
		}
	}

	return nil
}

type ListVpcsRequest struct {
	Region string
	Page   int
	Size   int
}

type ListVpcsResponseBody struct {
	Vpcs []types.Vpc
}

// ListVpcs 查询VPC列表
func (v *VpcService) ListVpcs(ctx context.Context, req *ListVpcsRequest) (*ListVpcsResponseBody, int64, error) {
	client, err := v.sdk.CreateEC2Client(ctx, req.Region)
	if err != nil {
		return nil, 0, err
	}

	request := &ec2.DescribeVpcsInput{}

	v.sdk.logger.Info("开始查询VPC列表", zap.String("region", req.Region))
	response, err := client.DescribeVpcs(ctx, request)
	if err != nil {
		v.sdk.logger.Error("查询VPC列表失败", zap.Error(err))
		return nil, 0, err
	}

	allVpcs := response.Vpcs
	totalCount := int64(len(allVpcs))

	// 分页处理
	startIdx := (req.Page - 1) * req.Size
	endIdx := req.Page * req.Size
	if startIdx >= len(allVpcs) {
		return &ListVpcsResponseBody{
			Vpcs: []types.Vpc{},
		}, totalCount, nil
	}

	if endIdx > len(allVpcs) {
		endIdx = len(allVpcs)
	}

	v.sdk.logger.Info("查询VPC列表成功", zap.Int64("total", totalCount))

	return &ListVpcsResponseBody{
		Vpcs: allVpcs[startIdx:endIdx],
	}, totalCount, nil
}

// GetVpcDetail 获取VPC详情
func (v *VpcService) GetVpcDetail(ctx context.Context, region string, vpcID string) (*types.Vpc, error) {
	client, err := v.sdk.CreateEC2Client(ctx, region)
	if err != nil {
		return nil, err
	}

	request := &ec2.DescribeVpcsInput{
		VpcIds: []string{vpcID},
	}

	v.sdk.logger.Info("开始查询VPC详情", zap.String("vpcId", vpcID))
	response, err := client.DescribeVpcs(ctx, request)
	if err != nil {
		v.sdk.logger.Error("查询VPC详情失败", zap.Error(err))
		return nil, err
	}

	if len(response.Vpcs) == 0 {
		return nil, fmt.Errorf("VPC %s 不存在", vpcID)
	}

	v.sdk.logger.Info("查询VPC详情成功", zap.String("vpcId", vpcID))
	return &response.Vpcs[0], nil
}

// stringPtr 返回字符串指针
func stringPtr(s string) *string {
	return &s
}
