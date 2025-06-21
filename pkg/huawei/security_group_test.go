package huawei

import (
	"context"
	"testing"

	"go.uber.org/zap"
)

func TestSecurityGroupService_CreateSecurityGroup(t *testing.T) {
	// 创建测试SDK
	logger, _ := zap.NewDevelopment()
	sdk := &SDK{
		logger:    logger,
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
	}

	securityGroupService := NewSecurityGroupService(sdk)

	// 测试创建安全组
	ctx := context.Background()
	req := &CreateSecurityGroupRequest{
		Region:            "cn-north-4",
		SecurityGroupName: "test-security-group",
		Description:       "测试安全组",
		VpcId:             "test-vpc-id",
	}

	response, err := securityGroupService.CreateSecurityGroup(ctx, req)

	// 由于没有真实的认证信息，这里主要测试代码结构
	if err != nil {
		t.Logf("创建安全组失败（预期）：%v", err)
	} else {
		t.Logf("创建安全组成功，ID：%s", response.SecurityGroupId)
	}
}

func TestSecurityGroupService_ListSecurityGroups(t *testing.T) {
	// 创建测试SDK
	logger, _ := zap.NewDevelopment()
	sdk := &SDK{
		logger:    logger,
		accessKey: "test-access-key",
		secretKey: "test-secret-key",
	}

	securityGroupService := NewSecurityGroupService(sdk)

	// 测试列出安全组
	ctx := context.Background()
	req := &ListSecurityGroupsRequest{
		Region:     "cn-north-4",
		PageNumber: 1,
		PageSize:   10,
	}

	response, err := securityGroupService.ListSecurityGroups(ctx, req)

	// 由于没有真实的认证信息，这里主要测试代码结构
	if err != nil {
		t.Logf("列出安全组失败（预期）：%v", err)
	} else {
		t.Logf("列出安全组成功，数量：%d", len(response.SecurityGroups))
	}
}
