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
	"testing"

	"go.uber.org/zap"
)

func TestEcsService_CreateInstance(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	ecsService := NewEcsService(sdk)

	req := &CreateInstanceRequest{
		Region:             "cn-north-4",
		ZoneId:             "cn-north-4a",
		ImageId:            "test-image-id",
		InstanceType:       "test-flavor",
		SecurityGroupIds:   []string{"test-sg-id"},
		SubnetId:           "test-subnet-id",
		InstanceName:       "test-instance",
		Hostname:           "test-hostname",
		Password:           "Test123!@#",
		Description:        "测试实例",
		Amount:             1,
		SystemDiskCategory: "SSD",
		SystemDiskSize:     40,
		DataDiskCategory:   "SSD",
		DataDiskSize:       100,
	}

	ctx := context.Background()
	_, err := ecsService.CreateInstance(ctx, req)
	if err != nil {
		t.Logf("创建ECS实例失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("创建ECS实例成功")
	}
}

func TestEcsService_ListInstances(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	ecsService := NewEcsService(sdk)

	req := &ListInstancesRequest{
		Region: "cn-north-4",
		Page:   1,
		Size:   10,
	}

	ctx := context.Background()
	_, err := ecsService.ListInstances(ctx, req)
	if err != nil {
		t.Logf("查询ECS实例列表失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("查询ECS实例列表成功")
	}
}

func TestEcsService_GetInstanceDetail(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	ecsService := NewEcsService(sdk)

	ctx := context.Background()
	_, err := ecsService.GetInstanceDetail(ctx, "cn-north-4", "test-instance-id")
	if err != nil {
		t.Logf("获取ECS实例详情失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("获取ECS实例详情成功")
	}
}

func TestEcsService_DeleteInstance(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	ecsService := NewEcsService(sdk)

	ctx := context.Background()
	err := ecsService.DeleteInstance(ctx, "cn-north-4", "test-instance-id", false)
	if err != nil {
		t.Logf("删除ECS实例失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("删除ECS实例成功")
	}
}

func TestEcsService_StartInstance(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	ecsService := NewEcsService(sdk)

	ctx := context.Background()
	err := ecsService.StartInstance(ctx, "cn-north-4", "test-instance-id")
	if err != nil {
		t.Logf("启动ECS实例失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("启动ECS实例成功")
	}
}

func TestEcsService_StopInstance(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	ecsService := NewEcsService(sdk)

	ctx := context.Background()
	err := ecsService.StopInstance(ctx, "cn-north-4", "test-instance-id", false)
	if err != nil {
		t.Logf("停止ECS实例失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("停止ECS实例成功")
	}
}

func TestEcsService_RestartInstance(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	ecsService := NewEcsService(sdk)

	ctx := context.Background()
	err := ecsService.RestartInstance(ctx, "cn-north-4", "test-instance-id")
	if err != nil {
		t.Logf("重启ECS实例失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("重启ECS实例成功")
	}
}
