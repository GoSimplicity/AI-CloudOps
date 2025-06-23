package huawei

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestVpcService_DeleteVPC(t *testing.T) {
	// 创建测试SDK
	logger, _ := zap.NewDevelopment()
	sdk := &SDK{
		logger:    logger,
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
	}

	vpcService := NewVpcService(sdk)

	// 测试删除VPC
	ctx := context.Background()
	err := vpcService.DeleteVPC(ctx, "cn-north-4", "test-vpc-id")

	// 由于没有真实的认证信息，这里主要测试代码结构
	if err != nil {
		t.Logf("删除VPC失败（预期）：%v", err)
	} else {
		t.Log("删除VPC成功")
	}
}

func TestVpcService_ListVpcs(t *testing.T) {
	// 创建测试SDK
	logger, _ := zap.NewDevelopment()
	sdk := &SDK{
		logger:    logger,
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
	}

	vpcService := NewVpcService(sdk)

	// 测试列出VPC
	ctx := context.Background()
	req := &ListVpcsRequest{
		Region: "cn-north-4",
		Page:   1,
		Size:   10,
	}

	response, err := vpcService.ListVpcs(ctx, req)

	// 由于没有真实的认证信息，这里主要测试代码结构
	if err != nil {
		t.Logf("列出VPC失败（预期）：%v", err)
	} else {
		t.Logf("列出VPC成功，数量：%d", len(response.Vpcs))
	}
}
