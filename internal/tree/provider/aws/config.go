package provider

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/aws"
)

// AWSProviderImpl AWS云资源管理的核心Provider实现，负责EC2、VPC、EBS、安全组等资源的统一管理和服务聚合。
type AWSProviderImpl struct {
	logger            *zap.Logger
	accessKey         string
	secretKey         string
	config            *AWSCloudConfig
	cachedRegions     []*model.RegionResp       // 缓存的区域列表
	regionsCacheTime  time.Time                 // 区域缓存时间
	discoveredRegions map[string]*AWSRegionInfo // 动态发现的区域信息
	// AWS服务客户端
	EC2Service           *aws.EC2Service
	VpcService           *aws.VpcService
	SecurityGroupService *aws.SecurityGroupService
	EBSService           *aws.EBSService
	sdk                  *aws.SDK
}

// NewAWSProvider 创建一个基于账号信息的AWS Provider实例
func NewAWSProvider(logger *zap.Logger, account *model.CloudAccount) *AWSProviderImpl {
	if account == nil {
		logger.Error("CloudAccount 不能为空")
		return nil
	}
	if account.AccessKey == "" || account.EncryptedSecret == "" {
		logger.Error("AccessKey 和 SecretKey 不能为空")
		return nil
	}

	// 创建AWS SDK
	awsSDK := aws.NewSDK(account.AccessKey, account.EncryptedSecret)
	awsSDK.SetLogger(logger)

	return &AWSProviderImpl{
		logger:            logger,
		accessKey:         account.AccessKey,
		secretKey:         account.EncryptedSecret,
		config:            getDefaultAWSConfig(),
		discoveredRegions: make(map[string]*AWSRegionInfo),
		sdk:               awsSDK,
		EC2Service:        aws.NewEC2Service(awsSDK),
	}
}

// NewAWSProviderImpl 创建一个基本的AWS Provider实例用于依赖注入
func NewAWSProviderImpl(logger *zap.Logger) *AWSProviderImpl {
	return &AWSProviderImpl{
		logger:            logger,
		config:            getDefaultAWSConfig(),
		discoveredRegions: make(map[string]*AWSRegionInfo),
	}
}

// InitializeProvider 初始化Provider，注入AK/SK并完成SDK和各服务的初始化。
func (a *AWSProviderImpl) InitializeProvider(accessKey, secretKey string) error {
	if accessKey == "" || secretKey == "" {
		return fmt.Errorf("AWS访问密钥不能为空")
	}
	a.accessKey = accessKey
	a.secretKey = secretKey

	// 重新创建SDK和服务
	a.sdk = aws.NewSDK(accessKey, secretKey)
	a.sdk.SetLogger(a.logger)
	a.EC2Service = aws.NewEC2Service(a.sdk)
	a.VpcService = aws.NewVpcService(a.sdk)
	a.SecurityGroupService = aws.NewSecurityGroupService(a.sdk)
	a.EBSService = aws.NewEBSService(a.sdk)

	a.logger.Info("AWS提供商初始化成功")
	return nil
}

// AWSCloudConfig AWS云配置
// 该配置结构体用于管理AWS提供商的动态配置信息
// 支持区域自动发现、端点配置、默认参数设置等功能
//
// 字段说明：
//
//	Regions   - 区域配置映射，key为regionId，value为区域详细配置
//	Defaults  - 默认参数配置，影响所有API调用的全局行为
//	Endpoints - 各服务端点模板及自定义端点
//	Discovery - 区域自动发现与探测相关配置
type AWSCloudConfig struct {
	Regions   map[string]AWSRegionConfig `json:"regions"`   // 区域ID到区域配置的映射
	Defaults  AWSDefaultConfig           `json:"defaults"`  // 默认参数配置
	Endpoints AWSEndpointConfig          `json:"endpoints"` // 服务端点配置
	Discovery AWSDiscoveryConfig         `json:"discovery"` // 区域发现相关配置
}

// AWSRegionConfig 区域详细配置
//
//	RegionID     - 区域ID（如 us-east-1）
//	LocalName    - 区域本地化名称（如 US East (N. Virginia)）
//	CityName     - 城市名
//	Enabled      - 是否启用该区域
//	ZonePrefix   - 可用区前缀（如 us-east-1a）
//	IsAccessible - 探测到该区域是否可访问
//	LastChecked  - 上次探测时间
//	Metadata     - 其他扩展元数据
type AWSRegionConfig struct {
	RegionID     string                 `json:"region_id"`              // 区域ID
	LocalName    string                 `json:"local_name"`             // 区域本地化名称
	CityName     string                 `json:"city_name"`              // 城市名
	Enabled      bool                   `json:"enabled"`                // 是否启用
	ZonePrefix   string                 `json:"zone_prefix"`            // 可用区前缀
	IsAccessible bool                   `json:"is_accessible"`          // 是否可访问
	LastChecked  *time.Time             `json:"last_checked,omitempty"` // 上次探测时间
	Metadata     map[string]interface{} `json:"metadata,omitempty"`     // 扩展元数据
}

// AWSDefaultConfig 默认参数配置
//
//	InstanceType       - 默认实例类型（如 t3.micro），默认 t3.micro
//	ForceDelete        - 删除资源时是否强制，默认 false
//	ForceStop          - 停止实例时是否强制，默认 false
//	PageSize           - 分页大小，默认 50，范围 1-1000
//	MaxRetries         - 最大重试次数，默认 3，范围 0-10
//	TimeoutSeconds     - API超时时间（秒），默认 300，范围 1-3600
//	ConcurrentLimit    - 并发请求上限，默认 10
type AWSDefaultConfig struct {
	InstanceType    string `json:"instance_type"`    // 默认实例类型
	ForceDelete     bool   `json:"force_delete"`     // 是否强制删除
	ForceStop       bool   `json:"force_stop"`       // 是否强制停止
	PageSize        int    `json:"page_size"`        // 分页大小
	MaxRetries      int    `json:"max_retries"`      // 最大重试次数
	TimeoutSeconds  int    `json:"timeout_seconds"`  // API超时时间（秒）
	ConcurrentLimit int    `json:"concurrent_limit"` // 并发请求上限
}

// AWSEndpointConfig 服务端点配置
//
//	EC2Template         - EC2服务端点模板（如 ec2.%s.amazonaws.com）
//	VPCTemplate         - VPC服务端点模板
//	EBSTemplate         - EBS服务端点模板
//	IAMTemplate         - IAM服务端点模板
//	CustomEndpoints     - 自定义服务端点（service->endpoint）
//	GlobalEndpoint      - 全局端点后缀
//	ChinaSuffix         - 中国区端点后缀
type AWSEndpointConfig struct {
	EC2Template     string            `json:"ec2_template"`               // EC2服务端点模板
	VPCTemplate     string            `json:"vpc_template"`               // VPC服务端点模板
	EBSTemplate     string            `json:"ebs_template"`               // EBS服务端点模板
	IAMTemplate     string            `json:"iam_template"`               // IAM服务端点模板
	CustomEndpoints map[string]string `json:"custom_endpoints,omitempty"` // 自定义端点
	GlobalEndpoint  string            `json:"global_endpoint"`            // 全局端点后缀
	ChinaSuffix     string            `json:"china_suffix"`               // 中国区端点后缀
}

// AWSDiscoveryConfig 区域自动发现与探测相关配置
//
//	EnableAutoDiscovery - 是否启用自动发现，默认 true
//	CacheTimeout        - 区域缓存有效期，默认 6h
//	ProbeTimeout        - 区域探测超时时间，默认 5s
//	MaxConcurrent       - 区域探测最大并发数，默认 20
//	RetryAttempts       - 区域探测重试次数，默认 2
//	RetryDelay          - 探测重试间隔，默认 1s
type AWSDiscoveryConfig struct {
	EnableAutoDiscovery bool          `json:"enable_auto_discovery"` // 是否启用自动发现
	CacheTimeout        time.Duration `json:"cache_timeout"`         // 区域缓存有效期
	ProbeTimeout        time.Duration `json:"probe_timeout"`         // 区域探测超时时间
	MaxConcurrent       int           `json:"max_concurrent"`        // 区域探测最大并发数
	RetryAttempts       int           `json:"retry_attempts"`        // 探测重试次数
	RetryDelay          time.Duration `json:"retry_delay"`           // 探测重试间隔
}

// AWSRegionInfo 区域探测结果信息
//
//	RegionID     - 区域ID
//	LocalName    - 区域本地化名称
//	IsAccessible - 是否可访问
//	LastChecked  - 上次探测时间
type AWSRegionInfo struct {
	RegionID     string    `json:"region_id"`     // 区域ID
	LocalName    string    `json:"local_name"`    // 区域本地化名称
	IsAccessible bool      `json:"is_accessible"` // 是否可访问
	LastChecked  time.Time `json:"last_checked"`  // 上次探测时间
}

// getDefaultAWSConfig 获取默认AWS配置（完全动态）
func getDefaultAWSConfig() *AWSCloudConfig {
	return &AWSCloudConfig{
		Regions: make(map[string]AWSRegionConfig),
		Defaults: AWSDefaultConfig{
			InstanceType:    "t3.micro",
			ForceDelete:     false,
			ForceStop:       false,
			PageSize:        50,
			MaxRetries:      3,
			TimeoutSeconds:  300,
			ConcurrentLimit: 10,
		},
		Endpoints: AWSEndpointConfig{
			EC2Template:     "ec2.%s.amazonaws.com",
			VPCTemplate:     "ec2.%s.amazonaws.com", // VPC使用EC2端点
			EBSTemplate:     "ec2.%s.amazonaws.com", // EBS使用EC2端点
			IAMTemplate:     "iam.amazonaws.com",    // IAM是全局服务
			CustomEndpoints: make(map[string]string),
			GlobalEndpoint:  "amazonaws.com",
			ChinaSuffix:     "amazonaws.com.cn",
		},
		Discovery: AWSDiscoveryConfig{
			EnableAutoDiscovery: true,
			CacheTimeout:        6 * time.Hour,
			ProbeTimeout:        5 * time.Second,
			MaxConcurrent:       20,
			RetryAttempts:       2,
			RetryDelay:          1 * time.Second,
		},
	}
}

// UpdateConfig 更新配置
func (a *AWSProviderImpl) UpdateConfig(config *AWSCloudConfig) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}
	if err := a.validateConfig(config); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}
	a.config = config
	a.logger.Info("AWS配置更新成功")
	return nil
}

// validateConfig 验证配置的合理性
func (a *AWSProviderImpl) validateConfig(config *AWSCloudConfig) error {
	if config.Defaults.PageSize <= 0 || config.Defaults.PageSize > 1000 {
		return fmt.Errorf("分页大小必须在1-1000之间")
	}
	if config.Defaults.MaxRetries < 0 || config.Defaults.MaxRetries > 10 {
		return fmt.Errorf("重试次数必须在0-10之间")
	}
	if config.Defaults.TimeoutSeconds <= 0 || config.Defaults.TimeoutSeconds > 3600 {
		return fmt.Errorf("超时时间必须在1-3600秒之间")
	}
	if config.Endpoints.EC2Template == "" {
		return fmt.Errorf("EC2端点模板不能为空")
	}
	if config.Endpoints.VPCTemplate == "" {
		return fmt.Errorf("VPC端点模板不能为空")
	}
	if config.Discovery.CacheTimeout <= 0 {
		return fmt.Errorf("缓存超时时间必须大于0")
	}
	if config.Discovery.ProbeTimeout <= 0 {
		return fmt.Errorf("探测超时时间必须大于0")
	}
	if config.Discovery.MaxConcurrent <= 0 || config.Discovery.MaxConcurrent > 100 {
		return fmt.Errorf("最大并发数必须在1-100之间")
	}
	return nil
}

// GetConfig 获取当前配置
func (a *AWSProviderImpl) GetConfig() *AWSCloudConfig {
	return a.config
}

// EnableRegion 启用指定区域
func (a *AWSProviderImpl) EnableRegion(regionID string) error {
	if a.config == nil {
		a.config = getDefaultAWSConfig()
	}
	if region, exists := a.config.Regions[regionID]; exists {
		region.Enabled = true
		a.config.Regions[regionID] = region
		a.logger.Info("区域已启用", zap.String("region", regionID))
		return nil
	}
	return fmt.Errorf("区域 %s 不存在", regionID)
}

// DisableRegion 禁用指定区域
func (a *AWSProviderImpl) DisableRegion(regionID string) error {
	if a.config == nil {
		return fmt.Errorf("配置未初始化")
	}
	if region, exists := a.config.Regions[regionID]; exists {
		region.Enabled = false
		a.config.Regions[regionID] = region
		a.logger.Info("区域已禁用", zap.String("region", regionID))
		return nil
	}
	return fmt.Errorf("区域 %s 不存在", regionID)
}

// SetCustomEndpoint 设置自定义服务端点
func (a *AWSProviderImpl) SetCustomEndpoint(service, endpoint string) error {
	if a.config == nil {
		a.config = getDefaultAWSConfig()
	}
	if a.config.Endpoints.CustomEndpoints == nil {
		a.config.Endpoints.CustomEndpoints = make(map[string]string)
	}
	a.config.Endpoints.CustomEndpoints[service] = endpoint
	a.logger.Info("自定义端点已设置", zap.String("service", service), zap.String("endpoint", endpoint))
	return nil
}

// GetServiceEndpoint 获取服务端点
func (a *AWSProviderImpl) GetServiceEndpoint(service, region string) (string, error) {
	if a.config == nil {
		a.config = getDefaultAWSConfig()
	}

	// 检查自定义端点
	if endpoint, exists := a.config.Endpoints.CustomEndpoints[service]; exists {
		return endpoint, nil
	}

	// 根据服务类型返回对应端点
	switch service {
	case "ec2":
		return fmt.Sprintf(a.config.Endpoints.EC2Template, region), nil
	case "vpc":
		return fmt.Sprintf(a.config.Endpoints.VPCTemplate, region), nil
	case "ebs":
		return fmt.Sprintf(a.config.Endpoints.EBSTemplate, region), nil
	case "iam":
		return a.config.Endpoints.IAMTemplate, nil
	default:
		// 默认使用EC2端点模板
		return fmt.Sprintf(a.config.Endpoints.EC2Template, region), nil
	}
}

// ResetToDefaults 重置为默认配置
func (a *AWSProviderImpl) ResetToDefaults() {
	a.config = getDefaultAWSConfig()
	a.logger.Info("配置已重置为默认值")
}

// ExportConfig 导出配置为JSON
func (a *AWSProviderImpl) ExportConfig() ([]byte, error) {
	return json.MarshalIndent(a.config, "", "  ")
}

// ImportConfig 从JSON导入配置
func (a *AWSProviderImpl) ImportConfig(data []byte) error {
	var config AWSCloudConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}
	return a.UpdateConfig(&config)
}

// 转换函数

// convertToResourceEcsFromListInstance 将AWS Instance转换为ResourceEcs（列表模式）
func (a *AWSProviderImpl) convertToResourceEcsFromListInstance(instance types.Instance) *model.ResourceEcs {
	lastSyncTime := time.Now()

	// 提取安全组ID
	var securityGroupIds []string
	for _, sg := range instance.SecurityGroups {
		securityGroupIds = append(securityGroupIds, *sg.GroupId)
	}

	// 提取私有IP地址
	var privateIPs []string
	if instance.PrivateIpAddress != nil {
		privateIPs = append(privateIPs, *instance.PrivateIpAddress)
	}
	for _, ni := range instance.NetworkInterfaces {
		for _, pip := range ni.PrivateIpAddresses {
			if pip.PrivateIpAddress != nil && *pip.PrivateIpAddress != *instance.PrivateIpAddress {
				privateIPs = append(privateIPs, *pip.PrivateIpAddress)
			}
		}
	}

	// 提取公有IP地址
	var publicIPs []string
	if instance.PublicIpAddress != nil {
		publicIPs = append(publicIPs, *instance.PublicIpAddress)
	}
	for _, ni := range instance.NetworkInterfaces {
		if ni.Association != nil && ni.Association.PublicIp != nil && *ni.Association.PublicIp != *instance.PublicIpAddress {
			publicIPs = append(publicIPs, *ni.Association.PublicIp)
		}
	}

	// 提取标签
	var tags []string
	instanceName := ""
	description := ""
	for _, tag := range instance.Tags {
		if *tag.Key == "Name" {
			instanceName = *tag.Value
		} else if *tag.Key == "Description" {
			description = *tag.Value
		}
		tags = append(tags, fmt.Sprintf("%s=%s", *tag.Key, *tag.Value))
	}

	// 从实例类型推断CPU和内存
	cpu, memory := a.parseInstanceTypeResources(string(instance.InstanceType))

	// 计费类型映射
	instanceChargeType := "PostPaid" // AWS默认按需付费
	if instance.InstanceLifecycle == types.InstanceLifecycleTypeSpot {
		instanceChargeType = "Spot"
	}

	// 获取主机名
	hostName := ""
	for _, tag := range instance.Tags {
		if *tag.Key == "Hostname" {
			hostName = *tag.Value
			break
		}
	}

	return &model.ResourceEcs{
		InstanceName:       instanceName,
		InstanceId:         *instance.InstanceId,
		Provider:           model.CloudProviderAWS,
		RegionId:           a.extractRegionFromAZ(*instance.Placement.AvailabilityZone),
		ZoneId:             *instance.Placement.AvailabilityZone,
		VpcId:              *instance.VpcId,
		Status:             string(instance.State.Name),
		CreationTime:       instance.LaunchTime.Format(time.RFC3339),
		InstanceChargeType: instanceChargeType,
		Description:        description,
		SecurityGroupIds:   model.StringList(securityGroupIds),
		PrivateIpAddress:   model.StringList(privateIPs),
		PublicIpAddress:    model.StringList(publicIPs),
		LastSyncTime:       &lastSyncTime,
		Tags:               model.StringList(tags),
		Cpu:                cpu,
		Memory:             memory,
		InstanceType:       string(instance.InstanceType),
		ImageId:            *instance.ImageId,
		HostName:           hostName,
		IpAddr: func(ips []string) string {
			if len(ips) > 0 {
				return ips[0]
			}
			return ""
		}(privateIPs),
	}
}

// convertToResourceEcsFromInstanceDetail 将AWS Instance转换为ResourceEcs（详情模式）
func (a *AWSProviderImpl) convertToResourceEcsFromInstanceDetail(instance *types.Instance) *model.ResourceEcs {
	if instance == nil {
		return nil
	}
	return a.convertToResourceEcsFromListInstance(*instance)
}

// parseInstanceTypeResources 从实例类型解析CPU和内存配置（通过SDK查询）
func (a *AWSProviderImpl) parseInstanceTypeResources(instanceType string) (int, int) {
	if a.EC2Service == nil || a.config == nil {
		return 2, 4 // 默认值
	}
	ctx := context.Background()
	vcpu, memory, err := a.EC2Service.DescribeInstanceType(ctx, a.config.Defaults.InstanceType, instanceType)
	if err != nil || vcpu == 0 || memory == 0 {
		return 2, 4 // 查询失败时返回默认值
	}
	return vcpu, memory
}

// extractRegionFromAZ 从可用区名称提取区域ID
func (a *AWSProviderImpl) extractRegionFromAZ(availabilityZone string) string {
	if availabilityZone == "" {
		return ""
	}

	// AWS可用区格式：region-az，如 us-east-1a, eu-west-1b
	// 去掉最后一个字符（可用区字母）
	if len(availabilityZone) > 1 {
		return availabilityZone[:len(availabilityZone)-1]
	}

	return availabilityZone
}
