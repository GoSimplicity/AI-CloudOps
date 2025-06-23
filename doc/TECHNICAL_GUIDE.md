# 华为云实现技术指南

## 🔧 技术实现细节

### 1. 华为云SDK依赖

#### 添加依赖包
```bash
# 添加华为云官方SDK
go get github.com/huaweicloud/huaweicloud-sdk-go-v3
```

#### 更新go.mod
```go
require (
    github.com/huaweicloud/huaweicloud-sdk-go-v3 v0.1.0
    // ... 其他依赖
)
```

### 2. SDK基础配置实现

#### 2.1 SDK结构体定义
```go
// pkg/huawei/sdk.go
package huawei

import (
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/auth/basic"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/core/config"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/evs/v3"
    "go.uber.org/zap"
)

type SDK struct {
    logger          *zap.Logger
    accessKeyId     string
    accessKeySecret string
    region          string
}

func NewSDK(logger *zap.Logger, accessKeyId, accessKeySecret string) *SDK {
    return &SDK{
        logger:          logger,
        accessKeyId:     accessKeyId,
        accessKeySecret: accessKeySecret,
    }
}
```

#### 2.2 客户端创建方法
```go
// 创建ECS客户端
func (s *SDK) CreateEcsClient(region string) (*ecs.EcsClient, error) {
    auth := basic.NewCredentialsBuilder().
        WithAk(s.accessKeyId).
        WithSk(s.accessKeySecret).
        Build()

    client := ecs.NewEcsClient(
        ecs.EcsClientBuilder().
            WithRegion(region).
            WithCredential(auth).
            Build())
    
    return client, nil
}

// 创建VPC客户端
func (s *SDK) CreateVpcClient(region string) (*vpc.VpcClient, error) {
    auth := basic.NewCredentialsBuilder().
        WithAk(s.accessKeyId).
        WithSk(s.accessKeySecret).
        Build()

    client := vpc.NewVpcClient(
        vpc.VpcClientBuilder().
            WithRegion(region).
            WithCredential(auth).
            Build())
    
    return client, nil
}

// 创建磁盘客户端
func (s *SDK) CreateEvsClient(region string) (*evs.EvsClient, error) {
    auth := basic.NewCredentialsBuilder().
        WithAk(s.accessKeyId).
        WithSk(s.accessKeySecret).
        Build()

    client := evs.NewEvsClient(
        evs.EvsClientBuilder().
            WithRegion(region).
            WithCredential(auth).
            Build())
    
    return client, nil
}
```

### 3. ECS服务实现

#### 3.1 ECS服务结构体
```go
// pkg/huawei/ecs.go
package huawei

import (
    "context"
    "fmt"
    "strconv"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/ecs/v2/model"
    "go.uber.org/zap"
)

type EcsService struct {
    sdk *SDK
}

func NewEcsService(sdk *SDK) *EcsService {
    return &EcsService{sdk: sdk}
}
```

#### 3.2 实例创建功能
```go
type CreateInstanceRequest struct {
    Region             string
    ZoneId             string
    ImageId            string
    InstanceType       string
    SecurityGroupIds   []string
    SubnetId           string
    InstanceName       string
    Hostname           string
    Password           string
    Description        string
    Amount             int
    SystemDiskCategory string
    SystemDiskSize     int
    DataDiskCategory   string
    DataDiskSize       int
}

type CreateInstanceResponseBody struct {
    InstanceIds []string
}

func (e *EcsService) CreateInstance(ctx context.Context, req *CreateInstanceRequest) (*CreateInstanceResponseBody, error) {
    client, err := e.sdk.CreateEcsClient(req.Region)
    if err != nil {
        e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
        return nil, err
    }

    // 构建系统盘配置
    systemDisk := &model.PrePaidServerRootVolume{
        Volumetype: req.SystemDiskCategory,
        Size:       int32(req.SystemDiskSize),
    }

    // 构建数据盘配置
    var dataVolumes []model.PrePaidServerDataVolume
    if req.DataDiskCategory != "" && req.DataDiskSize > 0 {
        dataVolumes = []model.PrePaidServerDataVolume{
            {
                Volumetype: req.DataDiskCategory,
                Size:       int32(req.DataDiskSize),
            },
        }
    }

    // 构建网络配置
    nics := []model.PrePaidServerNic{
        {
            SubnetId: req.SubnetId,
        },
    }

    // 构建安全组配置
    var securityGroups []model.PrePaidServerSecurityGroup
    for _, sgId := range req.SecurityGroupIds {
        securityGroups = append(securityGroups, model.PrePaidServerSecurityGroup{
            Id: sgId,
        })
    }

    request := &model.CreatePostPaidServersRequest{
        Body: &model.CreatePostPaidServersRequestBody{
            Server: &model.PostPaidServer{
                Name:               req.InstanceName,
                ImageRef:           req.ImageId,
                FlavorRef:          req.InstanceType,
                AvailabilityZone:   req.ZoneId,
                RootVolume:         systemDisk,
                DataVolumes:        &dataVolumes,
                Nics:               nics,
                SecurityGroups:     &securityGroups,
                AdminPass:          req.Password,
                Description:        req.Description,
                Count:              int32(req.Amount),
            },
        },
    }

    e.sdk.logger.Info("开始创建ECS实例", zap.String("region", req.Region), zap.Any("request", req))
    response, err := client.CreatePostPaidServers(request)
    if err != nil {
        e.sdk.logger.Error("创建ECS实例失败", zap.Error(err))
        return nil, err
    }

    // 提取实例ID
    var instanceIds []string
    if response.ServerIds != nil {
        for _, id := range *response.ServerIds {
            instanceIds = append(instanceIds, id)
        }
    }

    e.sdk.logger.Info("创建ECS实例成功", zap.Strings("instanceIds", instanceIds))

    return &CreateInstanceResponseBody{
        InstanceIds: instanceIds,
    }, nil
}
```

#### 3.3 实例生命周期管理
```go
// 启动实例
func (e *EcsService) StartInstance(ctx context.Context, region string, instanceID string) error {
    client, err := e.sdk.CreateEcsClient(region)
    if err != nil {
        e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
        return err
    }

    request := &model.StartServerRequest{
        ServerId: instanceID,
    }

    e.sdk.logger.Info("开始启动ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
    _, err = client.StartServer(request)
    if err != nil {
        e.sdk.logger.Error("启动ECS实例失败", zap.Error(err))
        return err
    }

    e.sdk.logger.Info("启动ECS实例成功", zap.String("instanceID", instanceID))
    return nil
}

// 停止实例
func (e *EcsService) StopInstance(ctx context.Context, region string, instanceID string, forceStop bool) error {
    client, err := e.sdk.CreateEcsClient(region)
    if err != nil {
        e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
        return err
    }

    request := &model.StopServerRequest{
        ServerId: instanceID,
        ForceStop: &forceStop,
    }

    e.sdk.logger.Info("开始停止ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
    _, err = client.StopServer(request)
    if err != nil {
        e.sdk.logger.Error("停止ECS实例失败", zap.Error(err))
        return err
    }

    e.sdk.logger.Info("停止ECS实例成功", zap.String("instanceID", instanceID))
    return nil
}

// 重启实例
func (e *EcsService) RestartInstance(ctx context.Context, region string, instanceID string) error {
    client, err := e.sdk.CreateEcsClient(region)
    if err != nil {
        e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
        return err
    }

    request := &model.RebootServerRequest{
        ServerId: instanceID,
    }

    e.sdk.logger.Info("开始重启ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
    _, err = client.RebootServer(request)
    if err != nil {
        e.sdk.logger.Error("重启ECS实例失败", zap.Error(err))
        return err
    }

    e.sdk.logger.Info("重启ECS实例成功", zap.String("instanceID", instanceID))
    return nil
}

// 删除实例
func (e *EcsService) DeleteInstance(ctx context.Context, region string, instanceID string, force bool) error {
    client, err := e.sdk.CreateEcsClient(region)
    if err != nil {
        e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
        return err
    }

    request := &model.DeleteServerRequest{
        ServerId: instanceID,
        DeleteVolume: &force,
    }

    e.sdk.logger.Info("开始删除ECS实例", zap.String("region", region), zap.String("instanceID", instanceID))
    _, err = client.DeleteServer(request)
    if err != nil {
        e.sdk.logger.Error("删除ECS实例失败", zap.Error(err))
        return err
    }

    e.sdk.logger.Info("删除ECS实例成功", zap.String("instanceID", instanceID))
    return nil
}
```

#### 3.4 实例查询功能
```go
type ListInstancesRequest struct {
    Region string
    Page   int
    Size   int
}

type ListInstancesResponseBody struct {
    Instances []*model.ServerDetail
    Total     int64
}

func (e *EcsService) ListInstances(ctx context.Context, req *ListInstancesRequest) (*ListInstancesResponseBody, error) {
    client, err := e.sdk.CreateEcsClient(req.Region)
    if err != nil {
        e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
        return nil, err
    }

    request := &model.ListServersDetailsRequest{
        Limit:  int32(req.Size),
        Offset: int32((req.Page - 1) * req.Size),
    }

    response, err := client.ListServersDetails(request)
    if err != nil {
        e.sdk.logger.Error("获取ECS实例列表失败", zap.Error(err))
        return nil, err
    }

    return &ListInstancesResponseBody{
        Instances: response.Servers,
        Total:     int64(response.Count),
    }, nil
}

func (e *EcsService) GetInstanceDetail(ctx context.Context, region string, instanceID string) (*model.ServerDetail, error) {
    client, err := e.sdk.CreateEcsClient(region)
    if err != nil {
        e.sdk.logger.Error("创建ECS客户端失败", zap.Error(err))
        return nil, err
    }

    request := &model.ShowServerRequest{
        ServerId: instanceID,
    }

    response, err := client.ShowServer(request)
    if err != nil {
        e.sdk.logger.Error("获取ECS实例详情失败", zap.Error(err))
        return nil, err
    }

    return response.Server, nil
}
```

### 4. VPC服务实现

#### 4.1 VPC服务结构体
```go
// pkg/huawei/vpc.go
package huawei

import (
    "context"
    
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3"
    "github.com/huaweicloud/huaweicloud-sdk-go-v3/services/vpc/v3/model"
    "go.uber.org/zap"
)

type VpcService struct {
    sdk *SDK
}

func NewVpcService(sdk *SDK) *VpcService {
    return &VpcService{sdk: sdk}
}
```

#### 4.2 VPC管理功能
```go
func (v *VpcService) CreateVpc(ctx context.Context, region string, name, cidr string) (*model.Vpc, error) {
    client, err := v.sdk.CreateVpcClient(region)
    if err != nil {
        v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
        return nil, err
    }

    request := &model.CreateVpcRequest{
        Body: &model.CreateVpcRequestBody{
            Vpc: &model.CreateVpcOption{
                Name: name,
                Cidr: cidr,
            },
        },
    }

    response, err := client.CreateVpc(request)
    if err != nil {
        v.sdk.logger.Error("创建VPC失败", zap.Error(err))
        return nil, err
    }

    return response.Vpc, nil
}

func (v *VpcService) DeleteVpc(ctx context.Context, region string, vpcID string) error {
    client, err := v.sdk.CreateVpcClient(region)
    if err != nil {
        v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
        return err
    }

    request := &model.DeleteVpcRequest{
        VpcId: vpcID,
    }

    _, err = client.DeleteVpc(request)
    if err != nil {
        v.sdk.logger.Error("删除VPC失败", zap.Error(err))
        return err
    }

    return nil
}

func (v *VpcService) ListVpcs(ctx context.Context, region string, limit, offset int) ([]*model.Vpc, error) {
    client, err := v.sdk.CreateVpcClient(region)
    if err != nil {
        v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
        return nil, err
    }

    request := &model.ListVpcsRequest{
        Limit:  int32(limit),
        Offset: int32(offset),
    }

    response, err := client.ListVpcs(request)
    if err != nil {
        v.sdk.logger.Error("获取VPC列表失败", zap.Error(err))
        return nil, err
    }

    return response.Vpcs, nil
}

func (v *VpcService) GetVpcDetail(ctx context.Context, region string, vpcID string) (*model.Vpc, error) {
    client, err := v.sdk.CreateVpcClient(region)
    if err != nil {
        v.sdk.logger.Error("创建VPC客户端失败", zap.Error(err))
        return nil, err
    }

    request := &model.ShowVpcRequest{
        VpcId: vpcID,
    }

    response, err := client.ShowVpc(request)
    if err != nil {
        v.sdk.logger.Error("获取VPC详情失败", zap.Error(err))
        return nil, err
    }

    return response.Vpc, nil
}
```

### 5. 华为云Provider实现

#### 5.1 Provider结构体
```go
// internal/tree/provider/huawei_provider.go
package provider

import (
    "context"
    "fmt"
    "time"

    "github.com/GoSimplicity/AI-CloudOps/internal/model"
    "github.com/GoSimplicity/AI-CloudOps/pkg/huawei"
    "go.uber.org/zap"
)

type HuaweiProviderImpl struct {
    logger               *zap.Logger
    sdk                  *huawei.SDK
    ecsService           *huawei.EcsService
    vpcService           *huawei.VpcService
    diskService          *huawei.DiskService
    securityGroupService *huawei.SecurityGroupService
}

func NewHuaweiProvider(logger *zap.Logger) *HuaweiProviderImpl {
    accessKeyId := os.Getenv("HUAWEI_ACCESS_KEY_ID")
    accessKeySecret := os.Getenv("HUAWEI_ACCESS_KEY_SECRET")

    if accessKeyId == "" || accessKeySecret == "" {
        logger.Error("HUAWEI_ACCESS_KEY_ID and HUAWEI_ACCESS_KEY_SECRET environment variables are required")
        return nil
    }

    sdk := huawei.NewSDK(logger, accessKeyId, accessKeySecret)

    return &HuaweiProviderImpl{
        logger:               logger,
        sdk:                  sdk,
        ecsService:           huawei.NewEcsService(sdk),
        vpcService:           huawei.NewVpcService(sdk),
        diskService:          huawei.NewDiskService(sdk),
        securityGroupService: huawei.NewSecurityGroupService(sdk),
    }
}
```

#### 5.2 基础服务方法实现
```go
func (h *HuaweiProviderImpl) SyncResources(ctx context.Context, region string) error {
    if region == "" {
        return fmt.Errorf("region cannot be empty")
    }

    h.logger.Info("starting resource sync", zap.String("region", region))

    // TODO: 实现具体的资源同步逻辑
    // 可以包括同步ECS实例、VPC、安全组等资源

    h.logger.Info("resource sync completed", zap.String("region", region))
    return nil
}

func (h *HuaweiProviderImpl) ListRegions(ctx context.Context) ([]*model.RegionResp, error) {
    // 华为云预定义区域列表
    regions := []*model.RegionResp{
        {
            RegionId:       "cn-north-4",
            LocalName:      "华北-北京四",
            RegionEndpoint: "ecs.cn-north-4.myhuaweicloud.com",
        },
        {
            RegionId:       "cn-east-3",
            LocalName:      "华东-上海一",
            RegionEndpoint: "ecs.cn-east-3.myhuaweicloud.com",
        },
        {
            RegionId:       "cn-south-1",
            LocalName:      "华南-广州",
            RegionEndpoint: "ecs.cn-south-1.myhuaweicloud.com",
        },
    }

    return regions, nil
}
```

#### 5.3 ECS管理方法实现
```go
func (h *HuaweiProviderImpl) ListInstances(ctx context.Context, region string, page, size int) ([]*model.ResourceEcs, int64, error) {
    if region == "" {
        return nil, 0, fmt.Errorf("region cannot be empty")
    }
    if page <= 0 || size <= 0 {
        return nil, 0, fmt.Errorf("page and size must be positive integers")
    }

    req := &huawei.ListInstancesRequest{
        Region: region,
        Page:   page,
        Size:   size,
    }

    resp, err := h.ecsService.ListInstances(ctx, req)
    if err != nil {
        h.logger.Error("failed to list instances", zap.Error(err), zap.String("region", region))
        return nil, 0, fmt.Errorf("list instances failed: %w", err)
    }

    if resp == nil || len(resp.Instances) == 0 {
        return nil, 0, nil
    }

    result := make([]*model.ResourceEcs, 0, len(resp.Instances))
    for _, instance := range resp.Instances {
        if instance == nil {
            continue
        }
        result = append(result, h.convertToResourceEcsFromListInstance(instance))
    }

    return result, resp.Total, nil
}

func (h *HuaweiProviderImpl) GetInstance(ctx context.Context, region string, instanceID string) (*model.ResourceEcs, error) {
    if region == "" || instanceID == "" {
        return nil, fmt.Errorf("region and instanceID cannot be empty")
    }

    instance, err := h.ecsService.GetInstanceDetail(ctx, region, instanceID)
    if err != nil {
        h.logger.Error("failed to get instance detail", zap.Error(err), zap.String("instanceID", instanceID))
        return nil, fmt.Errorf("get instance detail failed: %w", err)
    }

    if instance == nil {
        return nil, fmt.Errorf("instance not found")
    }

    return h.convertToResourceEcsFromInstanceDetail(instance), nil
}

func (h *HuaweiProviderImpl) CreateInstance(ctx context.Context, region string, config *model.CreateEcsResourceReq) error {
    if region == "" {
        return fmt.Errorf("region cannot be empty")
    }

    req := &huawei.CreateInstanceRequest{
        Region:             region,
        ZoneId:             config.ZoneId,
        ImageId:            config.ImageId,
        InstanceType:       config.InstanceType,
        SecurityGroupIds:   config.SecurityGroupIds,
        SubnetId:           config.SubnetId,
        InstanceName:       config.InstanceName,
        Hostname:           config.Hostname,
        Password:           config.Password,
        Description:        config.Description,
        Amount:             config.Amount,
        SystemDiskCategory: config.SystemDiskCategory,
        SystemDiskSize:     config.SystemDiskSize,
        DataDiskCategory:   config.DataDiskCategory,
        DataDiskSize:       config.DataDiskSize,
    }

    _, err := h.ecsService.CreateInstance(ctx, req)
    if err != nil {
        h.logger.Error("failed to create instance", zap.Error(err))
        return fmt.Errorf("create instance failed: %w", err)
    }

    return nil
}

func (h *HuaweiProviderImpl) DeleteInstance(ctx context.Context, region string, instanceID string) error {
    if region == "" || instanceID == "" {
        return fmt.Errorf("region and instanceID cannot be empty")
    }

    err := h.ecsService.DeleteInstance(ctx, region, instanceID, true)
    if err != nil {
        h.logger.Error("failed to delete instance", zap.Error(err))
        return fmt.Errorf("delete instance failed: %w", err)
    }

    return nil
}

func (h *HuaweiProviderImpl) StartInstance(ctx context.Context, region string, instanceID string) error {
    if region == "" || instanceID == "" {
        return fmt.Errorf("region and instanceID cannot be empty")
    }

    err := h.ecsService.StartInstance(ctx, region, instanceID)
    if err != nil {
        h.logger.Error("failed to start instance", zap.Error(err))
        return fmt.Errorf("start instance failed: %w", err)
    }

    return nil
}

func (h *HuaweiProviderImpl) StopInstance(ctx context.Context, region string, instanceID string) error {
    if region == "" || instanceID == "" {
        return fmt.Errorf("region and instanceID cannot be empty")
    }

    err := h.ecsService.StopInstance(ctx, region, instanceID, false)
    if err != nil {
        h.logger.Error("failed to stop instance", zap.Error(err))
        return fmt.Errorf("stop instance failed: %w", err)
    }

    return nil
}

func (h *HuaweiProviderImpl) RestartInstance(ctx context.Context, region string, instanceID string) error {
    if region == "" || instanceID == "" {
        return fmt.Errorf("region and instanceID cannot be empty")
    }

    err := h.ecsService.RestartInstance(ctx, region, instanceID)
    if err != nil {
        h.logger.Error("failed to restart instance", zap.Error(err))
        return fmt.Errorf("restart instance failed: %w", err)
    }

    return nil
}
```

#### 5.4 资源转换方法
```go
func (h *HuaweiProviderImpl) convertToResourceEcsFromListInstance(instance *model.ServerDetail) *model.ResourceEcs {
    if instance == nil {
        return nil
    }

    var securityGroupIds []string
    if instance.SecurityGroups != nil {
        for _, sg := range *instance.SecurityGroups {
            securityGroupIds = append(securityGroupIds, sg.Id)
        }
    }

    var privateIPs []string
    if instance.Addresses != nil {
        for _, addr := range *instance.Addresses {
            if addr.Type == "fixed" {
                for _, ip := range addr.Addr {
                    privateIPs = append(privateIPs, ip)
                }
            }
        }
    }

    var publicIPs []string
    if instance.Addresses != nil {
        for _, addr := range *instance.Addresses {
            if addr.Type == "floating" {
                for _, ip := range addr.Addr {
                    publicIPs = append(publicIPs, ip)
                }
            }
        }
    }

    var vpcId string
    if instance.Metadata != nil {
        if vpc, ok := (*instance.Metadata)["vpc_id"]; ok {
            vpcId = vpc
        }
    }

    // 计算内存，华为云返回的是MB，转换为GB
    memory := int(instance.Flavor.Ram) / 1024
    if memory == 0 && instance.Flavor.Ram > 0 {
        memory = 1 // 如果小于1GB但大于0，设为1GB
    }

    var tags []string
    if instance.Tags != nil {
        for _, tag := range *instance.Tags {
            tags = append(tags, fmt.Sprintf("%s=%s", tag.Key, tag.Value))
        }
    }

    lastSyncTime := time.Now()

    return &model.ResourceEcs{
        InstanceName:       instance.Name,
        InstanceId:         instance.Id,
        Provider:           model.CloudProviderHuawei,
        RegionId:           instance.Metadata["region_id"],
        ZoneId:             instance.AvailabilityZone,
        VpcId:              vpcId,
        Status:             instance.Status,
        CreationTime:       instance.Created,
        InstanceChargeType: "PostPaid", // 华为云默认为按需付费
        Description:        instance.Description,
        SecurityGroupIds:   model.StringList(securityGroupIds),
        PrivateIpAddress:   model.StringList(privateIPs),
        PublicIpAddress:    model.StringList(publicIPs),
        LastSyncTime:       &lastSyncTime,
        Tags:               model.StringList(tags),
        Cpu:                int(instance.Flavor.Vcpus),
        Memory:             memory,
        InstanceType:       instance.Flavor.Id,
        ImageId:            instance.Image.Id,
        HostName:           instance.Name,
        IpAddr:             getFirstIP(privateIPs),
    }
}

func (h *HuaweiProviderImpl) convertToResourceEcsFromInstanceDetail(instance *model.ServerDetail) *model.ResourceEcs {
    // 与convertToResourceEcsFromListInstance类似，但处理更详细的实例信息
    return h.convertToResourceEcsFromListInstance(instance)
}
```

### 6. 工厂模式更新

#### 6.1 更新工厂模式
```go
// internal/tree/provider/factory.go
func NewProviderFactory(
    aliyun *AliyunProviderImpl,
    huawei *HuaweiProviderImpl,
) *ProviderFactory {
    return &ProviderFactory{
        providers: map[model.CloudProvider]Provider{
            model.CloudProviderAliyun: aliyun,
            model.CloudProviderHuawei: huawei,
        },
    }
}
```

## 🔑 关键技术要点

### 1. 华为云认证
- 使用AK/SK认证方式
- 支持环境变量配置
- 支持配置文件配置

### 2. 区域管理
- 华为云区域ID格式：`cn-north-4`
- 支持多区域操作
- 区域端点自动配置

### 3. 错误处理
- 华为云特定错误码处理
- 统一错误信息格式
- 详细日志记录

### 4. 资源转换
- 华为云API响应转换为统一模型
- 保持与阿里云实现一致
- 处理数据类型差异

## 🧪 测试指南

### 单元测试
```go
func TestHuaweiProvider_ListInstances(t *testing.T) {
    // 测试华为云Provider的ListInstances方法
}

func TestHuaweiProvider_CreateInstance(t *testing.T) {
    // 测试华为云Provider的CreateInstance方法
}
```

### 集成测试
```go
func TestHuaweiProvider_Integration(t *testing.T) {
    // 测试华为云Provider的完整功能
}
```

## 📝 最佳实践

1. **错误处理**: 始终检查API调用的错误返回值
2. **日志记录**: 记录关键操作的开始和结束
3. **资源清理**: 确保测试后清理创建的资源
4. **并发安全**: 确保Provider实现是线程安全的
5. **性能优化**: 合理使用连接池和缓存

---

**最后更新**: 2025-06-20  
**版本**: 1.0.0  
**状态**: 草稿 