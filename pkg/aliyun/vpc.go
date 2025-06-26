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
	"go.uber.org/zap"
)

type VpcService struct {
	sdk *SDK
}

func NewVpcService(sdk *SDK) *VpcService {
	return &VpcService{sdk: sdk}
}

// CreateVpcRequest 创建VPC请求参数
type CreateVpcRequest struct {
	Region           string
	VpcName          string
	CidrBlock        string
	Description      string
	ZoneId           string
	VSwitchName      string
	VSwitchCidrBlock string
}

// CreateVpcResponse 创建VPC响应
type CreateVpcResponse struct {
	VpcId     string
	VSwitchId string
}

// CreateVPC 创建VPC
func (v *VpcService) CreateVPC(ctx context.Context, req *CreateVpcRequest) (*CreateVpcResponse, error) {
	client, err := v.sdk.CreateVpcClient(req.Region)
	if err != nil {
		v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	// 创建VPC
	vpcRequest := &vpc.CreateVpcRequest{
		RegionId:    tea.String(req.Region),
		VpcName:     tea.String(req.VpcName),
		CidrBlock:   tea.String(req.CidrBlock),
		Description: tea.String(req.Description),
	}

	v.sdk.logger.Info("开始创建VPC", zap.String("region", req.Region), zap.String("vpcName", req.VpcName))
	vpcResponse, err := client.CreateVpc(vpcRequest)
	if err != nil {
		v.sdk.logger.Error("创建VPC失败", zap.Error(err))
		return nil, HandleError(err)
	}

	vpcId := tea.StringValue(vpcResponse.Body.VpcId)
	v.sdk.logger.Info("VPC创建成功", zap.String("vpcId", vpcId))

	// 等待VPC可用
	if err := v.waitForVpcAvailable(client, req.Region, vpcId); err != nil {
		v.sdk.logger.Error("等待VPC可用失败", zap.Error(err))
		return nil, HandleError(err)
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

	v.sdk.logger.Info("开始创建交换机", zap.String("vpcId", vpcId), zap.String("zoneId", req.ZoneId))
	vSwitchResponse, err := client.CreateVSwitch(vSwitchRequest)
	if err != nil {
		v.sdk.logger.Error("创建交换机失败", zap.Error(err))
		return nil, HandleError(err)
	}

	vSwitchId := tea.StringValue(vSwitchResponse.Body.VSwitchId)
	v.sdk.logger.Info("交换机创建成功", zap.String("vSwitchId", vSwitchId))

	return &CreateVpcResponse{
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

	v.sdk.logger.Info("等待VPC变为可用状态", zap.String("vpcId", vpcId))
	for i := 0; i < 10; i++ {
		response, err := client.DescribeVpcAttribute(request)
		if err != nil {
			v.sdk.logger.Error("查询VPC属性失败", zap.Error(err))
			return HandleError(err)
		}

		if tea.StringValue(response.Body.Status) == "Available" {
			v.sdk.logger.Info("VPC已变为可用状态", zap.String("vpcId", vpcId))
			return nil
		}

		v.sdk.logger.Debug("VPC尚未可用，等待中", zap.String("vpcId", vpcId), zap.Int("attempt", i+1))
		time.Sleep(5 * time.Second)
	}

	err := fmt.Errorf("等待VPC可用超时")
	v.sdk.logger.Error("等待VPC可用超时", zap.String("vpcId", vpcId))
	return err
}

// DeleteVpcRequest 删除VPC请求参数
type DeleteVpcRequest struct {
	Region string
	VpcID  string
}

// DeleteVpcResponse 删除VPC响应
type DeleteVpcResponse struct {
	Success bool
}

// DeleteVPC 删除VPC
func (v *VpcService) DeleteVPC(ctx context.Context, req *DeleteVpcRequest) (*DeleteVpcResponse, error) {
	client, err := v.sdk.CreateVpcClient(req.Region)
	if err != nil {
		v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	// 查询并删除所有交换机
	vSwitchRequest := &vpc.DescribeVSwitchesRequest{
		RegionId: tea.String(req.Region),
		VpcId:    tea.String(req.VpcID),
	}

	v.sdk.logger.Info("开始查询VPC下的交换机", zap.String("region", req.Region), zap.String("vpcId", req.VpcID))
	vSwitchResponse, err := client.DescribeVSwitches(vSwitchRequest)
	if err != nil {
		v.sdk.logger.Error("查询交换机失败", zap.Error(err))
		return nil, HandleError(err)
	}

	// 删除所有交换机
	for _, vSwitch := range vSwitchResponse.Body.VSwitches.VSwitch {
		vSwitchId := tea.StringValue(vSwitch.VSwitchId)
		deleteVSwitchRequest := &vpc.DeleteVSwitchRequest{
			VSwitchId: tea.String(vSwitchId),
		}

		v.sdk.logger.Info("开始删除交换机", zap.String("vSwitchId", vSwitchId))
		if _, err = client.DeleteVSwitch(deleteVSwitchRequest); err != nil {
			v.sdk.logger.Error("删除交换机失败", zap.String("vSwitchId", vSwitchId), zap.Error(err))
			return nil, HandleError(err)
		}

		v.sdk.logger.Info("交换机删除成功", zap.String("vSwitchId", vSwitchId))
		time.Sleep(5 * time.Second)
	}

	// 删除VPC
	request := &vpc.DeleteVpcRequest{
		VpcId: tea.String(req.VpcID),
	}

	v.sdk.logger.Info("开始删除VPC", zap.String("vpcId", req.VpcID))
	if _, err = client.DeleteVpc(request); err != nil {
		v.sdk.logger.Error("删除VPC失败", zap.Error(err))
		return nil, HandleError(err)
	}

	v.sdk.logger.Info("VPC删除成功", zap.String("vpcId", req.VpcID))
	return &DeleteVpcResponse{Success: true}, nil
}

// ListVpcsRequest 查询VPC列表请求参数
type ListVpcsRequest struct {
	Region string
	Page   int
	Size   int
}

// ListVpcsResponse 查询VPC列表响应
type ListVpcsResponse struct {
	Vpcs  []*vpc.DescribeVpcsResponseBodyVpcsVpc
	Total int64
}

// ListVpcs 查询VPC列表（支持分页获取全部资源）
func (v *VpcService) ListVpcs(ctx context.Context, req *ListVpcsRequest) (*ListVpcsResponse, error) {
	var allVpcs []*vpc.DescribeVpcsResponseBodyVpcsVpc
	var totalCount int64
	page := 1
	pageSize := req.Size
	if pageSize <= 0 {
		pageSize = 100
	}

	v.sdk.logger.Info("开始查询VPC列表", zap.String("region", req.Region))
	
	for {
		client, err := v.sdk.CreateVpcClient(req.Region)
		if err != nil {
			v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
			return nil, HandleError(err)
		}

		request := &vpc.DescribeVpcsRequest{
			RegionId:   tea.String(req.Region),
			PageNumber: tea.Int32(int32(page)),
			PageSize:   tea.Int32(int32(pageSize)),
		}

		response, err := client.DescribeVpcs(request)
		if err != nil {
			v.sdk.logger.Error("查询VPC列表失败", zap.Error(err))
			return nil, HandleError(err)
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
		v.sdk.logger.Info("查询VPC列表成功，但当前页无数据", zap.String("region", req.Region), zap.Int("page", req.Page), zap.Int("total", len(allVpcs)))
		return &ListVpcsResponse{
			Vpcs:  []*vpc.DescribeVpcsResponseBodyVpcsVpc{},
			Total: totalCount,
		}, nil
	}

	if endIdx > len(allVpcs) {
		endIdx = len(allVpcs)
	}

	v.sdk.logger.Info("查询VPC列表成功", zap.String("region", req.Region), zap.Int("count", endIdx-startIdx), zap.Int("total", len(allVpcs)))
	return &ListVpcsResponse{
		Vpcs:  allVpcs[startIdx:endIdx],
		Total: totalCount,
	}, nil
}

// GetVpcDetailRequest 获取VPC详情请求参数
type GetVpcDetailRequest struct {
	Region string
	VpcID  string
}

// GetVpcDetailResponse 获取VPC详情响应
type GetVpcDetailResponse struct {
	VpcDetail *vpc.DescribeVpcAttributeResponseBody
}

// GetVpcDetail 获取VPC详情
func (v *VpcService) GetVpcDetail(ctx context.Context, req *GetVpcDetailRequest) (*GetVpcDetailResponse, error) {
	client, err := v.sdk.CreateVpcClient(req.Region)
	if err != nil {
		v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
		return nil, HandleError(err)
	}

	request := &vpc.DescribeVpcAttributeRequest{
		RegionId: tea.String(req.Region),
		VpcId:    tea.String(req.VpcID),
	}

	v.sdk.logger.Info("开始查询VPC详情", zap.String("region", req.Region), zap.String("vpcId", req.VpcID))
	response, err := client.DescribeVpcAttribute(request)
	if err != nil {
		v.sdk.logger.Error("查询VPC详情失败", zap.Error(err))
		return nil, HandleError(err)
	}

	v.sdk.logger.Info("查询VPC详情成功", zap.String("vpcId", req.VpcID))
	return &GetVpcDetailResponse{
		VpcDetail: response.Body,
	}, nil
}

// GetZonesByVpcRequest 获取VPC下的可用区请求参数
type GetZonesByVpcRequest struct {
	Region string
	VpcID  string
}

// GetZonesByVpcResponse 获取VPC下的可用区响应
type GetZonesByVpcResponse struct {
	Zones     []*vpc.DescribeZonesResponseBodyZonesZone
	VSwitches []*vpc.DescribeVSwitchesResponseBodyVSwitchesVSwitch
}

// GetZonesByVpc 获取VPC下的可用区
func (v *VpcService) GetZonesByVpc(ctx context.Context, req *GetZonesByVpcRequest) (*GetZonesByVpcResponse, error) {
	client, err := v.sdk.CreateVpcClient(req.Region)
	if err != nil {
		v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
		return nil, HandleError(err)
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
			RegionId: tea.String(req.Region),
		}
		v.sdk.logger.Debug("开始查询可用区信息", zap.String("region", req.Region))
		zonesResponse, zonesErr = client.DescribeZones(request)
	}()

	go func() {
		defer close(ch2)
		vSwitchRequest := &vpc.DescribeVSwitchesRequest{
			RegionId: tea.String(req.Region),
			VpcId:    tea.String(req.VpcID),
		}
		v.sdk.logger.Debug("开始查询VPC下的交换机", zap.String("region", req.Region), zap.String("vpcId", req.VpcID))
		vSwitchResponse, vSwitchErr = client.DescribeVSwitches(vSwitchRequest)
	}()

	<-ch1
	<-ch2

	if zonesErr != nil {
		v.sdk.logger.Error("获取可用区信息失败", zap.Error(zonesErr))
		return nil, HandleError(zonesErr)
	}

	if vSwitchErr != nil {
		v.sdk.logger.Error("获取交换机信息失败", zap.Error(vSwitchErr))
		return nil, HandleError(vSwitchErr)
	}

	v.sdk.logger.Info("获取VPC可用区和交换机信息成功", 
		zap.String("vpcId", req.VpcID), 
		zap.Int("zonesCount", len(zonesResponse.Body.Zones.Zone)),
		zap.Int("vSwitchesCount", len(vSwitchResponse.Body.VSwitches.VSwitch)))
	
	return &GetZonesByVpcResponse{
		Zones:     zonesResponse.Body.Zones.Zone,
		VSwitches: vSwitchResponse.Body.VSwitches.VSwitch,
	}, nil
}
