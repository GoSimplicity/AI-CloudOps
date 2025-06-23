# 华为云集成文档

## 概述

本文档描述了AI-CloudOps项目中华为云集成的实现。华为云集成提供了对华为云ECS、VPC、安全组和磁盘等核心服务的支持。

## 架构

华为云集成采用分层架构设计：

```
┌─────────────────────────────────────────────────────────────┐
│                    Provider Layer                          │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              HuaweiProviderImpl                     │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│                     Service Layer                          │
│  ┌─────────────┐ ┌─────────────┐ ┌─────────────┐ ┌─────────┐ │
│  │ EcsService  │ │ VpcService  │ │SecurityGroup│ │DiskServ │ │
│  └─────────────┘ └─────────────┘ └─────────────┘ └─────────┘ │
└─────────────────────────────────────────────────────────────┘
┌─────────────────────────────────────────────────────────────┐
│                      SDK Layer                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                    Huawei SDK                       │   │
│  └─────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────┘
```

## 目录结构

```
pkg/huawei/
├── sdk.go                    # 华为云SDK客户端
├── types.go                  # 类型定义
├── ecs.go                    # ECS服务
├── vpc.go                    # VPC服务
├── security_group.go         # 安全组服务
└── disk.go                   # 磁盘服务

internal/tree/provider/
├── huawei_provider.go        # 华为云Provider实现
└── factory.go                # Provider工厂（需要更新）

test/huawei/
├── sdk_test.go               # SDK测试
├── ecs_test.go               # ECS服务测试
├── vpc_test.go               # VPC服务测试
├── security_group_test.go    # 安全组服务测试
└── disk_test.go              # 磁盘服务测试

doc/huawei/
├── README.md                 # 本文档
├── API_REFERENCE.md          # API参考文档
├── EXAMPLES.md               # 使用示例
└── TROUBLESHOOTING.md        # 故障排除指南
```

## 功能特性

### 支持的服务

1. **ECS（弹性云服务器）**
   - 实例列表查询
   - 实例详情获取
   - 实例创建
   - 实例删除
   - 实例启动/停止/重启

2. **VPC（虚拟私有云）**
   - VPC列表查询
   - VPC详情获取
   - VPC创建
   - VPC删除

3. **安全组**
   - 安全组列表查询
   - 安全组详情获取
   - 安全组创建
   - 安全组删除

4. **磁盘（EVS）**
   - 磁盘列表查询
   - 磁盘详情获取
   - 磁盘创建
   - 磁盘删除
   - 磁盘挂载/卸载

### 认证方式

华为云集成使用AK/SK（Access Key/Secret Key）认证方式：

- **Access Key**: 访问密钥ID
- **Secret Key**: 访问密钥
- **Project ID**: 项目ID
- **Region**: 区域

### 配置要求

1. **环境变量**
   ```bash
   HUAWEI_ACCESS_KEY=your_access_key
   HUAWEI_SECRET_KEY=your_secret_key
   HUAWEI_REGION=cn-north-4
   HUAWEI_PROJECT_ID=your_project_id
   ```

2. **权限要求**
   - ECS相关权限
   - VPC相关权限
   - EVS相关权限

## 快速开始

### 1. 安装依赖

```bash
go get github.com/huaweicloud/huaweicloud-sdk-go-v3
```

### 2. 创建SDK客户端

```go
package main

import (
    "github.com/GoSimplicity/AI-CloudOps/pkg/huawei"
    "go.uber.org/zap"
)

func main() {
    logger, _ := zap.NewDevelopment()
    
    // 创建SDK客户端
    sdk, err := huawei.NewSDK(
        "your_access_key",
        "your_secret_key", 
        "cn-north-4",
        "your_project_id",
        logger,
    )
    if err != nil {
        panic(err)
    }
    
    // 创建ECS服务
    ecsService := huawei.NewEcsService(sdk)
    
    // 使用服务...
}
```

### 3. 使用服务

```go
// 获取ECS实例列表
req := &huawei.ListInstancesRequest{
    Region: "cn-north-4",
    Page:   1,
    Size:   10,
}

instances, err := ecsService.ListInstances(context.Background(), req)
if err != nil {
    log.Printf("Failed to list instances: %v", err)
    return
}

for _, instance := range instances {
    log.Printf("Instance: %s, Status: %s", instance.Name, instance.Status)
}
```

## 开发指南

### 添加新的API

1. 在 `types.go` 中定义请求/响应类型
2. 在对应的服务文件中实现API方法
3. 添加单元测试
4. 更新文档

### 错误处理

所有API方法都返回标准的Go错误，建议使用以下模式：

```go
result, err := service.SomeMethod(ctx, req)
if err != nil {
    return fmt.Errorf("failed to call SomeMethod: %w", err)
}
```

### 日志记录

SDK客户端支持结构化日志记录，使用zap日志库：

```go
logger, _ := zap.NewDevelopment()
sdk, err := huawei.NewSDK(accessKey, secretKey, region, projectId, logger)
```

## 测试

### 运行单元测试

```bash
# 运行所有华为云相关测试
go test ./test/huawei/...

# 运行特定测试
go test ./test/huawei/ -run TestNewSDK
```

### 集成测试

```bash
# 运行集成测试
go test ./test/integration/...
```

## 故障排除

### 常见问题

1. **认证失败**
   - 检查Access Key和Secret Key是否正确
   - 确认Project ID是否正确
   - 验证区域设置

2. **API调用失败**
   - 检查网络连接
   - 确认API权限
   - 查看错误响应详情

3. **依赖问题**
   - 运行 `go mod tidy` 更新依赖
   - 检查Go版本兼容性

### 调试模式

启用调试日志：

```go
logger, _ := zap.NewDevelopment()
sdk, err := huawei.NewSDK(accessKey, secretKey, region, projectId, logger)
```

## 贡献指南

1. Fork项目
2. 创建功能分支
3. 提交更改
4. 创建Pull Request

## 许可证

本项目采用MIT许可证，详见LICENSE文件。 