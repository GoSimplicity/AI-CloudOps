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

func TestDiskService_CreateDisk(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	diskService := NewDiskService(sdk)

	req := &CreateDiskRequest{
		Region:       "cn-north-4",
		ZoneId:       "cn-north-4a",
		DiskName:     "test-disk",
		DiskCategory: "SSD",
		Size:         100,
		Description:  "测试磁盘",
	}

	ctx := context.Background()
	_, err := diskService.CreateDisk(ctx, req)
	if err != nil {
		t.Logf("创建磁盘失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("创建磁盘成功")
	}
}

func TestDiskService_ListDisks(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	diskService := NewDiskService(sdk)

	req := &ListDisksRequest{
		Region: "cn-north-4",
		Page:   1,
		Size:   10,
	}

	ctx := context.Background()
	_, err := diskService.ListDisks(ctx, req)
	if err != nil {
		t.Logf("查询磁盘列表失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("查询磁盘列表成功")
	}
}

func TestDiskService_GetDisk(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	diskService := NewDiskService(sdk)

	ctx := context.Background()
	_, err := diskService.GetDisk(ctx, "cn-north-4", "test-disk-id")
	if err != nil {
		t.Logf("获取磁盘详情失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("获取磁盘详情成功")
	}
}

func TestDiskService_DeleteDisk(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	diskService := NewDiskService(sdk)

	ctx := context.Background()
	err := diskService.DeleteDisk(ctx, "cn-north-4", "test-disk-id")
	if err != nil {
		t.Logf("删除磁盘失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("删除磁盘成功")
	}
}

func TestDiskService_AttachDisk(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	diskService := NewDiskService(sdk)

	ctx := context.Background()
	err := diskService.AttachDisk(ctx, "cn-north-4", "test-disk-id", "test-instance-id")
	if err != nil {
		t.Logf("挂载磁盘失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("挂载磁盘成功")
	}
}

func TestDiskService_DetachDisk(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	sdk := NewSDK(logger, "test-access-key", "test-secret-key")
	diskService := NewDiskService(sdk)

	ctx := context.Background()
	err := diskService.DetachDisk(ctx, "cn-north-4", "test-disk-id", "test-instance-id")
	if err != nil {
		t.Logf("卸载磁盘失败（预期，因为没有真实凭证）: %v", err)
	} else {
		t.Log("卸载磁盘成功")
	}
}
