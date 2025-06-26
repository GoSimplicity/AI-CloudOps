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
	"fmt"
	"time"

	"github.com/alibabacloud-go/tea/tea"
	vpc "github.com/alibabacloud-go/vpc-20160428/v2/client"
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
	ZoneId           string
	VSwitchName      string
	VSwitchCidrBlock string
}

type CreateVpcResponseBody struct {
	VpcId     string
	VSwitchId string
}

// CreateVPC 创建VPC
func (v *VpcService) CreateVPC(ctx context.Context, req *CreateVpcRequest) (*CreateVpcResponseBody, error) {
	client, err := v.sdk.CreateVpcClient(req.Region)
	if err != nil {
		return nil, err
	}

	// 创建VPC
	vpcRequest := &vpc.CreateVpcRequest{
		RegionId:    tea.String(req.Region),
		VpcName:     tea.String(req.VpcName),
		CidrBlock:   tea.String(req.CidrBlock),
		Description: tea.String(req.Description),
	}

	vpcResponse, err := client.CreateVpc(vpcRequest)
	if err != nil {
		return nil, err
	}

	vpcId := tea.StringValue(vpcResponse.Body.VpcId)

	// 等待VPC可用
	err = v.waitForVpcAvailable(client, req.Region, vpcId)
	if err != nil {
		return nil, err
	}

	// 创建交换机
	vSwitchRequest := &vpc.CreateVSwitchRequest{
		RegionId:    tea.String(req.Region),
		ZoneId:      tea.String(req.ZoneId),
		VpcId:       tea.String(vpcId),
		VSwitchName: tea.String(req.VSwitchName),
		CidrBlock:   tea.String(req.VSwitchCidrBlock),
		Description: tea.String(req.Description),
	}

	vSwitchResponse, err := client.CreateVSwitch(vSwitchRequest)
	if err != nil {
		return nil, err
	}

	vSwitchId := tea.StringValue(vSwitchResponse.Body.VSwitchId)

	return &CreateVpcResponseBody{
		VpcId:     vpcId,
		VSwitchId: vSwitchId,
	}, nil
}

// waitForVpcAvailable 等待VPC可用
func (v *VpcService) waitForVpcAvailable(client *vpc.Client, region string, vpcId string) error {
	request := &vpc.DescribeVpcAttributeRequest{
		RegionId: tea.String(region),
		VpcId:    tea.String(vpcId),
	}

	for i := 0; i < 10; i++ {
		response, err := client.DescribeVpcAttribute(request)
		if err != nil {
			return err
		}

		if tea.StringValue(response.Body.Status) == "Available" {
			return nil
		}

		time.Sleep(5 * time.Second)
	}

	return fmt.Errorf("等待VPC可用超时")
}

// DeleteVPC 删除VPC
func (v *VpcService) DeleteVPC(ctx context.Context, region string, vpcID string) error {
	client, err := v.sdk.CreateVpcClient(region)
	if err != nil {
		return err
	}

	// 查询并删除所有交换机
	vSwitchRequest := &vpc.DescribeVSwitchesRequest{
		RegionId: tea.String(region),
		VpcId:    tea.String(vpcID),
	}

	vSwitchResponse, err := client.DescribeVSwitches(vSwitchRequest)
	if err != nil {
		return err
	}

	// 删除所有交换机
	for _, vSwitch := range vSwitchResponse.Body.VSwitches.VSwitch {
		vSwitchId := tea.StringValue(vSwitch.VSwitchId)
		deleteVSwitchRequest := &vpc.DeleteVSwitchRequest{
			VSwitchId: tea.String(vSwitchId),
		}

		_, err = client.DeleteVSwitch(deleteVSwitchRequest)
		if err != nil {
			return fmt.Errorf("删除交换机(%s)失败: %w", vSwitchId, err)
		}

		time.Sleep(5 * time.Second)
	}

	// 删除VPC
	request := &vpc.DeleteVpcRequest{
		VpcId: tea.String(vpcID),
	}

	_, err = client.DeleteVpc(request)
	if err != nil {
		return err
	}

	return nil
}

// ListVpcsRequest 查询VPC列表请求参数
type ListVpcsRequest struct {
	Region string
	Page   int
	Size   int
}

// ListVpcsResponseBody 查询VPC列表响应
type ListVpcsResponseBody struct {
	Vpcs  []*vpc.DescribeVpcsResponseBodyVpcsVpc
	Total int64
}

// ListVpcs 查询VPC列表（支持分页获取全部资源）
func (v *VpcService) ListVpcs(ctx context.Context, req *ListVpcsRequest) (*ListVpcsResponseBody, error) {
	var allVpcs []*vpc.DescribeVpcsResponseBodyVpcsVpc
	var totalCount int64 = 0
	page := 1
	pageSize := req.Size
	if pageSize <= 0 {
		pageSize = 100
	}

	for {
		client, err := v.sdk.CreateVpcClient(req.Region)
		if err != nil {
			return nil, err
		}

		request := &vpc.DescribeVpcsRequest{
			RegionId:   tea.String(req.Region),
			PageNumber: tea.Int32(int32(page)),
			PageSize:   tea.Int32(int32(pageSize)),
		}

		response, err := client.DescribeVpcs(request)
		if err != nil {
			return nil, err
		}

		if response.Body == nil || response.Body.Vpcs == nil || response.Body.Vpcs.Vpc == nil {
			break
		}

		vpcs := response.Body.Vpcs.Vpc
		if len(vpcs) == 0 {
			break
		}

		allVpcs = append(allVpcs, vpcs...)
		totalCount = int64(tea.Int32Value(response.Body.TotalCount))

		if len(vpcs) < pageSize {
			break
		}

		page++
	}

	startIdx := (req.Page - 1) * req.Size
	endIdx := req.Page * req.Size
	if startIdx >= len(allVpcs) {
		return &ListVpcsResponseBody{
			Vpcs:  []*vpc.DescribeVpcsResponseBodyVpcsVpc{},
			Total: totalCount,
		}, nil
	}

	if endIdx > len(allVpcs) {
		endIdx = len(allVpcs)
	}

	return &ListVpcsResponseBody{
		Vpcs:  allVpcs[startIdx:endIdx],
		Total: totalCount,
	}, nil
}

// GetVpcDetail 获取VPC详情
func (v *VpcService) GetVpcDetail(ctx context.Context, region string, vpcID string) (*vpc.DescribeVpcAttributeResponseBody, error) {
	client, err := v.sdk.CreateVpcClient(region)
	if err != nil {
		return nil, err
	}

	request := &vpc.DescribeVpcAttributeRequest{
		RegionId: tea.String(region),
		VpcId:    tea.String(vpcID),
	}

	response, err := client.DescribeVpcAttribute(request)
	if err != nil {
		return nil, err
	}

	return response.Body, nil
}

// GetZonesByVpc 获取VPC下的可用区
func (v *VpcService) GetZonesByVpc(ctx context.Context, region string, vpcId string) ([]*vpc.DescribeZonesResponseBodyZonesZone, []*vpc.DescribeVSwitchesResponseBodyVSwitchesVSwitch, error) {
	client, err := v.sdk.CreateVpcClient(region)
	if err != nil {
		return nil, nil, err
	}

	// 并行获取可用区信息和VPC关联的交换机信息
	var zonesResponse *vpc.DescribeZonesResponse
	var vSwitchResponse *vpc.DescribeVSwitchesResponse
	var zonesErr, vSwitchErr error

	// 这里可以使用 errgroup 来并行执行
	ch1 := make(chan struct{})
	ch2 := make(chan struct{})

	go func() {
		defer close(ch1)
		request := &vpc.DescribeZonesRequest{
			RegionId: tea.String(region),
		}
		zonesResponse, zonesErr = client.DescribeZones(request)
	}()

	go func() {
		defer close(ch2)
		vSwitchRequest := &vpc.DescribeVSwitchesRequest{
			RegionId: tea.String(region),
			VpcId:    tea.String(vpcId),
		}
		vSwitchResponse, vSwitchErr = client.DescribeVSwitches(vSwitchRequest)
	}()

	<-ch1
	<-ch2

	if zonesErr != nil {
		return nil, nil, zonesErr
	}

	if vSwitchErr != nil {
		return nil, nil, vSwitchErr
	}

	return zonesResponse.Body.Zones.Zone, vSwitchResponse.Body.VSwitches.VSwitch, nil
}
