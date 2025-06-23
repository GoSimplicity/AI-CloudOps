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

package provider

import (
	"context"
	"os"
	"testing"

	ecsmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
	evsmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/evs/v2/model"
	vpcmodel "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3/model"
	"go.uber.org/zap"
)

func TestNewHuaweiProvider(t *testing.T) {
	logger := zap.NewNop()

	// 测试创建华为云Provider
	provider := NewHuaweiProvider(logger)

	// 验证provider不为nil且正确初始化
	if provider == nil {
		t.Error("Expected provider to be not nil")
	}

	// 验证默认配置
	if provider.config == nil {
		t.Error("Expected config to be initialized")
	}

	// 验证logger设置
	if provider.logger == nil {
		t.Error("Expected logger to be set")
	}
}

func TestHuaweiProviderImpl_ListRegions(t *testing.T) {
	logger := zap.NewNop()
	provider := &HuaweiProviderImpl{
		logger: logger,
		config: getDefaultHuaweiConfig(),
	}

	ctx := context.Background()
	regions, err := provider.ListRegions(ctx)

	// 由于没有初始化SDK，应该返回错误
	if err == nil {
		t.Error("Expected error when SDK is not initialized")
	}

	if regions != nil {
		t.Error("Expected regions to be nil when SDK is not initialized")
	}
}

func TestHuaweiProviderImpl_ListInstances(t *testing.T) {
	logger := zap.NewNop()
	provider := &HuaweiProviderImpl{
		logger: logger,
		config: getDefaultHuaweiConfig(),
	}

	ctx := context.Background()
	instances, total, err := provider.ListInstances(ctx, "cn-north-4", 1, 10)

	// 由于没有初始化SDK，应该返回错误
	if err == nil {
		t.Error("Expected error when SDK is not initialized")
	}

	if instances != nil {
		t.Error("Expected instances to be nil when SDK is not initialized")
	}

	if total != 0 {
		t.Error("Expected total to be 0 when SDK is not initialized")
	}
}

func TestHuaweiProviderImpl_ListVPCs(t *testing.T) {
	logger := zap.NewNop()
	provider := &HuaweiProviderImpl{
		logger: logger,
		config: getDefaultHuaweiConfig(),
	}

	ctx := context.Background()
	vpcs, err := provider.ListVPCs(ctx, "cn-north-4", 1, 10)

	// 由于没有初始化SDK，应该返回错误
	if err == nil {
		t.Error("Expected error when SDK is not initialized")
	}

	if vpcs != nil {
		t.Error("Expected vpcs to be nil when SDK is not initialized")
	}
}

func TestHuaweiProviderImpl_ListSecurityGroups(t *testing.T) {
	logger := zap.NewNop()
	provider := &HuaweiProviderImpl{
		logger: logger,
		config: getDefaultHuaweiConfig(),
	}

	ctx := context.Background()
	sgs, err := provider.ListSecurityGroups(ctx, "cn-north-4", 1, 10)

	// 由于没有初始化SDK，应该返回错误
	if err == nil {
		t.Error("Expected error when SDK is not initialized")
	}

	if sgs != nil {
		t.Error("Expected security groups to be nil when SDK is not initialized")
	}
}

func TestHuaweiProviderImpl_ListDisks(t *testing.T) {
	logger := zap.NewNop()
	provider := &HuaweiProviderImpl{
		logger: logger,
		config: getDefaultHuaweiConfig(),
	}

	ctx := context.Background()
	disks, err := provider.ListDisks(ctx, "cn-north-4", 1, 10)

	// 由于没有初始化SDK，应该返回错误
	if err == nil {
		t.Error("Expected error when SDK is not initialized")
	}

	if disks != nil {
		t.Error("Expected disks to be nil when SDK is not initialized")
	}
}

// 测试资源转换方法
func TestHuaweiProviderImpl_ConvertMethods(t *testing.T) {
	logger := zap.NewNop()
	provider := &HuaweiProviderImpl{
		logger: logger,
		config: getDefaultHuaweiConfig(),
	}

	// 测试ECS转换方法 - 使用空的ServerDetail
	emptyServer := ecsmodel.ServerDetail{}
	ecs := provider.convertToResourceEcsFromListInstance(emptyServer)
	if ecs == nil {
		t.Error("Expected ECS to be not nil for empty server")
	}

	// 测试ECS详情转换方法
	ecsDetail := provider.convertToResourceEcsFromInstanceDetail(&emptyServer)
	if ecsDetail == nil {
		t.Error("Expected ECS detail to be not nil for empty server")
	}

	// 测试VPC转换方法 - 使用空的Vpc
	emptyVpc := vpcmodel.Vpc{}
	vpc := provider.convertToResourceVpcFromListVpc(emptyVpc, "cn-north-4")
	if vpc == nil {
		t.Error("Expected VPC to be not nil for empty vpc")
	}

	// 测试VPC详情转换方法
	vpcDetail := provider.convertToResourceVpcFromDetail(&emptyVpc, "cn-north-4")
	if vpcDetail == nil {
		t.Error("Expected VPC detail to be not nil for empty vpc")
	}

	// 测试安全组转换方法 - 使用空的SecurityGroup
	emptySg := vpcmodel.SecurityGroup{}
	sg := provider.convertToResourceSecurityGroupFromList(emptySg, "cn-north-4")
	if sg == nil {
		t.Error("Expected security group to be not nil for empty security group")
	}

	// 测试安全组详情转换方法
	emptySgInfo := &vpcmodel.SecurityGroupInfo{}
	sgDetail := provider.convertToResourceSecurityGroupFromDetail(emptySgInfo, "cn-north-4")
	if sgDetail == nil {
		t.Error("Expected security group detail to be not nil for empty security group")
	}

	// 测试磁盘转换方法 - 使用空的VolumeDetail
	emptyDisk := evsmodel.VolumeDetail{}
	disk := provider.convertToResourceDiskFromList(emptyDisk, "cn-north-4")
	if disk == nil {
		t.Error("Expected disk to be not nil for empty disk")
	}

	// 测试磁盘详情转换方法
	diskDetail := provider.convertToResourceDiskFromDetail(&emptyDisk, "cn-north-4")
	if diskDetail == nil {
		t.Error("Expected disk detail to be not nil for empty disk")
	}
}

// 测试区域发现相关方法
func TestHuaweiProviderImpl_RegionDiscovery(t *testing.T) {
	logger := zap.NewNop()
	provider := &HuaweiProviderImpl{
		logger: logger,
		config: getDefaultHuaweiConfig(),
	}

	// mock: 设置环境变量让 getRegionPatternsFromConfig 返回非空
	os.Setenv("HUAWEI_CLOUD_REGION_PATTERNS", "cn-north-1")
	defer os.Unsetenv("HUAWEI_CLOUD_REGION_PATTERNS")

	// 测试区域名称生成
	localName := provider.generateRegionLocalName("cn-north-1")
	if localName == "" {
		t.Error("Expected local name to be not empty")
	}

	// 测试区域模式生成
	patterns := provider.generateRegionPatterns()
	if len(patterns) == 0 {
		t.Error("Expected patterns to be not empty")
	}

	// 测试区域探测
	valid := provider.probeRegion("cn-north-1")
	// 由于没有SDK，应该返回false
	if valid {
		t.Error("Expected probe to return false when SDK is not initialized")
	}
}

// 测试配置相关方法
func TestHuaweiProviderImpl_ConfigMethods(t *testing.T) {
	logger := zap.NewNop()
	provider := &HuaweiProviderImpl{
		logger: logger,
		config: getDefaultHuaweiConfig(),
	}

	// 测试获取配置
	config := provider.GetConfig()
	if config == nil {
		t.Error("Expected config to be not nil")
	}

	// 测试重置配置
	provider.ResetToDefaults()
	if provider.config == nil {
		t.Error("Expected config to be not nil after reset")
	}

	// 测试导出配置
	configData, err := provider.ExportConfig()
	if err != nil {
		t.Errorf("Expected no error when exporting config: %v", err)
	}
	if len(configData) == 0 {
		t.Error("Expected config data to be not empty")
	}
}
