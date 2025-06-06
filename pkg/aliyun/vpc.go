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
		v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
		return nil, err
	}

	// 创建VPC
	vpcRequest := &vpc.CreateVpcRequest{
		RegionId:    tea.String(req.Region),
		VpcName:     tea.String(req.VpcName),
		CidrBlock:   tea.String(req.CidrBlock),
		Description: tea.String(req.Description),
	}

	v.sdk.logger.Info("开始创建VPC", zap.String("region", req.Region), zap.Any("request", req))
	vpcResponse, err := client.CreateVpc(vpcRequest)
	if err != nil {
		v.sdk.logger.Error("创建VPC失败", zap.Error(err))
		return nil, err
	}

	vpcId := tea.StringValue(vpcResponse.Body.VpcId)
	v.sdk.logger.Info("创建VPC成功", zap.String("vpcID", vpcId))

	// 等待VPC可用
	err = v.waitForVpcAvailable(client, req.Region, vpcId)
	if err != nil {
		v.sdk.logger.Error("等待VPC可用失败", zap.Error(err))
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

	v.sdk.logger.Info("开始创建交换机", zap.String("vpcID", vpcId), zap.String("vSwitchName", req.VSwitchName))
	vSwitchResponse, err := client.CreateVSwitch(vSwitchRequest)
	if err != nil {
		v.sdk.logger.Error("创建交换机失败", zap.Error(err))
		return nil, err
	}

	vSwitchId := tea.StringValue(vSwitchResponse.Body.VSwitchId)
	v.sdk.logger.Info("创建交换机成功", zap.String("vSwitchID", vSwitchId))

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
		v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
		return err
	}

	// 查询并删除所有交换机
	vSwitchRequest := &vpc.DescribeVSwitchesRequest{
		RegionId: tea.String(region),
		VpcId:    tea.String(vpcID),
	}

	v.sdk.logger.Info("查询VPC下的交换机", zap.String("region", region), zap.String("vpcID", vpcID))
	vSwitchResponse, err := client.DescribeVSwitches(vSwitchRequest)
	if err != nil {
		v.sdk.logger.Error("查询交换机失败", zap.Error(err))
		return err
	}

	// 删除所有交换机
	for _, vSwitch := range vSwitchResponse.Body.VSwitches.VSwitch {
		vSwitchId := tea.StringValue(vSwitch.VSwitchId)
		deleteVSwitchRequest := &vpc.DeleteVSwitchRequest{
			VSwitchId: tea.String(vSwitchId),
		}

		v.sdk.logger.Info("删除交换机", zap.String("vSwitchID", vSwitchId))
		_, err = client.DeleteVSwitch(deleteVSwitchRequest)
		if err != nil {
			v.sdk.logger.Error("删除交换机失败", zap.Error(err))
			return fmt.Errorf("删除交换机(%s)失败: %w", vSwitchId, err)
		}

		time.Sleep(5 * time.Second)
	}

	// 删除VPC
	request := &vpc.DeleteVpcRequest{
		VpcId: tea.String(vpcID),
	}

	v.sdk.logger.Info("开始删除VPC", zap.String("region", region), zap.String("vpcID", vpcID))
	_, err = client.DeleteVpc(request)
	if err != nil {
		v.sdk.logger.Error("删除VPC失败", zap.Error(err))
		return err
	}

	v.sdk.logger.Info("删除VPC成功", zap.String("vpcID", vpcID))
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

// ListVpcs 查询VPC列表
func (v *VpcService) ListVpcs(ctx context.Context, req *ListVpcsRequest) (*ListVpcsResponseBody, error) {
	client, err := v.sdk.CreateVpcClient(req.Region)
	if err != nil {
		v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
		return nil, err
	}

	request := &vpc.DescribeVpcsRequest{
		RegionId:   tea.String(req.Region),
		PageNumber: tea.Int32(int32(req.Page)),
		PageSize:   tea.Int32(int32(req.Size)),
	}

	response, err := client.DescribeVpcs(request)
	if err != nil {
		v.sdk.logger.Error("查询VPC列表失败", zap.Error(err))
		return nil, err
	}

	return &ListVpcsResponseBody{
		Vpcs:  response.Body.Vpcs.Vpc,
		Total: int64(tea.Int32Value(response.Body.TotalCount)),
	}, nil
}

// GetVpcDetail 获取VPC详情
func (v *VpcService) GetVpcDetail(ctx context.Context, region string, vpcID string) (*vpc.DescribeVpcAttributeResponseBody, error) {
	client, err := v.sdk.CreateVpcClient(region)
	if err != nil {
		v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
		return nil, err
	}

	request := &vpc.DescribeVpcAttributeRequest{
		RegionId: tea.String(region),
		VpcId:    tea.String(vpcID),
	}

	response, err := client.DescribeVpcAttribute(request)
	if err != nil {
		v.sdk.logger.Error("获取VPC详情失败", zap.Error(err))
		return nil, err
	}

	return response.Body, nil
}

// GetZonesByVpc 获取VPC下的可用区
func (v *VpcService) GetZonesByVpc(ctx context.Context, region string, vpcId string) ([]*vpc.DescribeZonesResponseBodyZonesZone, []*vpc.DescribeVSwitchesResponseBodyVSwitchesVSwitch, error) {
	client, err := v.sdk.CreateVpcClient(region)
	if err != nil {
		v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
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
		v.sdk.logger.Error("查询可用区列表失败", zap.Error(zonesErr))
		return nil, nil, zonesErr
	}

	if vSwitchErr != nil {
		v.sdk.logger.Error("查询交换机列表失败", zap.Error(vSwitchErr))
		return nil, nil, vSwitchErr
	}

	return zonesResponse.Body.Zones.Zone, vSwitchResponse.Body.VSwitches.VSwitch, nil
}
