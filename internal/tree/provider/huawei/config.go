package provider

import (
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
)

// HuaweiCloudConfig 及相关结构体、getDefaultHuaweiConfig、validateConfig、ExportConfig、ImportConfig 等

// HuaweiCloudConfig 华为云配置
// 该配置结构体用于管理华为云提供商的动态配置信息
// 支持区域自动发现、端点配置、默认参数设置等功能
//
// 字段说明：
//
//	Regions   - 区域配置映射，key为regionId，value为区域详细配置
//	Defaults  - 默认参数配置，影响所有API调用的全局行为
//	Endpoints - 各服务端点模板及自定义端点
//	Discovery - 区域自动发现与探测相关配置
type HuaweiCloudConfig struct {
	Regions   map[string]HuaweiRegionConfig `json:"regions"`   // 区域ID到区域配置的映射
	Defaults  HuaweiDefaultConfig           `json:"defaults"`  // 默认参数配置
	Endpoints HuaweiEndpointConfig          `json:"endpoints"` // 服务端点配置
	Discovery HuaweiDiscoveryConfig         `json:"discovery"` // 区域发现相关配置
}

// HuaweiRegionConfig 区域详细配置
//
//	RegionID     - 区域ID（如 cn-north-4）
//	LocalName    - 区域本地化名称（如 华北-北京四）
//	CityName     - 城市名
//	Enabled      - 是否启用该区域
//	ZonePrefix   - 可用区前缀（如 cn-north-4a）
//	IsAccessible - 探测到该区域是否可访问
//	LastChecked  - 上次探测时间
//	Metadata     - 其他扩展元数据
type HuaweiRegionConfig struct {
	RegionID     string                 `json:"region_id"`              // 区域ID
	LocalName    string                 `json:"local_name"`             // 区域本地化名称
	CityName     string                 `json:"city_name"`              // 城市名
	Enabled      bool                   `json:"enabled"`                // 是否启用
	ZonePrefix   string                 `json:"zone_prefix"`            // 可用区前缀
	IsAccessible bool                   `json:"is_accessible"`          // 是否可访问
	LastChecked  *time.Time             `json:"last_checked,omitempty"` // 上次探测时间
	Metadata     map[string]interface{} `json:"metadata,omitempty"`     // 扩展元数据
}

// HuaweiDefaultConfig 默认参数配置
//
//	InstanceChargeType - 实例计费类型（如 PostPaid/PrePaid/Spot），默认 PostPaid
//	ForceDelete        - 删除资源时是否强制，默认 false
//	ForceStop          - 停止实例时是否强制，默认 false
//	PageSize           - 分页大小，默认 50，范围 1-1000
//	MaxRetries         - 最大重试次数，默认 3，范围 0-10
//	TimeoutSeconds     - API超时时间（秒），默认 300，范围 1-3600
//	ConcurrentLimit    - 并发请求上限，默认 10
type HuaweiDefaultConfig struct {
	InstanceChargeType string `json:"instance_charge_type"` // 实例计费类型
	ForceDelete        bool   `json:"force_delete"`         // 是否强制删除
	ForceStop          bool   `json:"force_stop"`           // 是否强制停止
	PageSize           int    `json:"page_size"`            // 分页大小
	MaxRetries         int    `json:"max_retries"`          // 最大重试次数
	TimeoutSeconds     int    `json:"timeout_seconds"`      // API超时时间（秒）
	ConcurrentLimit    int    `json:"concurrent_limit"`     // 并发请求上限
}

// HuaweiEndpointConfig 服务端点配置
//
//	ECSTemplate         - ECS服务端点模板（如 ecs.%s.myhuaweicloud.com）
//	VPCTemplate         - VPC服务端点模板
//	EVSTemplate         - EVS服务端点模板
//	IAMTemplate         - IAM服务端点模板
//	CustomEndpoints     - 自定义服务端点（service->endpoint）
//	GlobalEndpoint      - 全局端点后缀
//	InternationalSuffix - 国际站端点后缀
type HuaweiEndpointConfig struct {
	ECSTemplate         string            `json:"ecs_template"`               // ECS服务端点模板
	VPCTemplate         string            `json:"vpc_template"`               // VPC服务端点模板
	EVSTemplate         string            `json:"evs_template"`               // EVS服务端点模板
	IAMTemplate         string            `json:"iam_template"`               // IAM服务端点模板
	CustomEndpoints     map[string]string `json:"custom_endpoints,omitempty"` // 自定义端点
	GlobalEndpoint      string            `json:"global_endpoint"`            // 全局端点后缀
	InternationalSuffix string            `json:"international_suffix"`       // 国际站端点后缀
}

// HuaweiDiscoveryConfig 区域自动发现与探测相关配置
//
//	EnableAutoDiscovery - 是否启用自动发现，默认 true
//	CacheTimeout        - 区域缓存有效期，默认 6h
//	ProbeTimeout        - 区域探测超时时间，默认 5s
//	MaxConcurrent       - 区域探测最大并发数，默认 20
//	RetryAttempts       - 区域探测重试次数，默认 2
//	RetryDelay          - 探测重试间隔，默认 1s
type HuaweiDiscoveryConfig struct {
	EnableAutoDiscovery bool          `json:"enable_auto_discovery"` // 是否启用自动发现
	CacheTimeout        time.Duration `json:"cache_timeout"`         // 区域缓存有效期
	ProbeTimeout        time.Duration `json:"probe_timeout"`         // 区域探测超时时间
	MaxConcurrent       int           `json:"max_concurrent"`        // 区域探测最大并发数
	RetryAttempts       int           `json:"retry_attempts"`        // 探测重试次数
	RetryDelay          time.Duration `json:"retry_delay"`           // 探测重试间隔
}

// HuaweiRegionInfo 区域探测结果信息
//
//	RegionID     - 区域ID
//	LocalName    - 区域本地化名称
//	IsAccessible - 是否可访问
//	LastChecked  - 上次探测时间
type HuaweiRegionInfo struct {
	RegionID     string    `json:"region_id"`     // 区域ID
	LocalName    string    `json:"local_name"`    // 区域本地化名称
	IsAccessible bool      `json:"is_accessible"` // 是否可访问
	LastChecked  time.Time `json:"last_checked"`  // 上次探测时间
}

// getDefaultHuaweiConfig 获取默认华为云配置（完全动态）
func getDefaultHuaweiConfig() *HuaweiCloudConfig {
	return &HuaweiCloudConfig{
		Regions: make(map[string]HuaweiRegionConfig),
		Defaults: HuaweiDefaultConfig{
			InstanceChargeType: "PostPaid",
			ForceDelete:        false,
			ForceStop:          false,
			PageSize:           50,
			MaxRetries:         3,
			TimeoutSeconds:     300,
			ConcurrentLimit:    10,
		},
		Endpoints: HuaweiEndpointConfig{
			ECSTemplate:         "ecs.%s.myhuaweicloud.com",
			VPCTemplate:         "vpc.%s.myhuaweicloud.com",
			EVSTemplate:         "evs.%s.myhuaweicloud.com",
			IAMTemplate:         "iam.%s.myhuaweicloud.com",
			CustomEndpoints:     make(map[string]string),
			GlobalEndpoint:      "myhuaweicloud.com",
			InternationalSuffix: "myhuaweicloud.com",
		},
		Discovery: HuaweiDiscoveryConfig{
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
func (h *HuaweiProviderImpl) UpdateConfig(config *HuaweiCloudConfig) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}
	if err := h.validateConfig(config); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}
	h.config = config
	h.logger.Info("华为云配置更新成功")
	return nil
}

// validateConfig 验证配置的合理性
func (h *HuaweiProviderImpl) validateConfig(config *HuaweiCloudConfig) error {
	if config.Defaults.PageSize <= 0 || config.Defaults.PageSize > 1000 {
		return fmt.Errorf("分页大小必须在1-1000之间")
	}
	if config.Defaults.MaxRetries < 0 || config.Defaults.MaxRetries > 10 {
		return fmt.Errorf("重试次数必须在0-10之间")
	}
	if config.Defaults.TimeoutSeconds <= 0 || config.Defaults.TimeoutSeconds > 3600 {
		return fmt.Errorf("超时时间必须在1-3600秒之间")
	}
	if config.Endpoints.ECSTemplate == "" {
		return fmt.Errorf("ECS端点模板不能为空")
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
func (h *HuaweiProviderImpl) GetConfig() *HuaweiCloudConfig {
	return h.config
}

// EnableRegion 启用指定区域
func (h *HuaweiProviderImpl) EnableRegion(regionID string) error {
	if regionID == "" {
		return fmt.Errorf("区域ID不能为空")
	}
	if regionConfig, exists := h.config.Regions[regionID]; exists {
		regionConfig.Enabled = true
		h.config.Regions[regionID] = regionConfig
		h.logger.Info("区域已启用", zap.String("regionId", regionID))
		return nil
	}
	return fmt.Errorf("区域 %s 不存在", regionID)
}

// DisableRegion 禁用指定区域
func (h *HuaweiProviderImpl) DisableRegion(regionID string) error {
	if regionID == "" {
		return fmt.Errorf("区域ID不能为空")
	}
	if regionConfig, exists := h.config.Regions[regionID]; exists {
		regionConfig.Enabled = false
		h.config.Regions[regionID] = regionConfig
		h.logger.Info("区域已禁用", zap.String("regionId", regionID))
		return nil
	}
	return fmt.Errorf("区域 %s 不存在", regionID)
}

// SetCustomEndpoint 设置自定义端点
func (h *HuaweiProviderImpl) SetCustomEndpoint(service, endpoint string) error {
	if service == "" || endpoint == "" {
		return fmt.Errorf("服务名和端点不能为空")
	}
	if h.config.Endpoints.CustomEndpoints == nil {
		h.config.Endpoints.CustomEndpoints = make(map[string]string)
	}
	h.config.Endpoints.CustomEndpoints[service] = endpoint
	h.logger.Info("自定义端点已设置", zap.String("service", service), zap.String("endpoint", endpoint))
	return nil
}

// GetServiceEndpoint 获取服务端点
func (h *HuaweiProviderImpl) GetServiceEndpoint(service, region string) (string, error) {
	if service == "" || region == "" {
		return "", fmt.Errorf("服务名和区域不能为空")
	}
	if customEndpoint, exists := h.config.Endpoints.CustomEndpoints[service]; exists {
		return customEndpoint, nil
	}
	var template string
	switch service {
	case "ecs":
		template = h.config.Endpoints.ECSTemplate
	case "vpc":
		template = h.config.Endpoints.VPCTemplate
	case "evs":
		template = h.config.Endpoints.EVSTemplate
	case "iam":
		template = h.config.Endpoints.IAMTemplate
	default:
		return "", fmt.Errorf("不支持的服务: %s", service)
	}
	if template == "" {
		return "", fmt.Errorf("服务 %s 的端点模板未配置", service)
	}
	return fmt.Sprintf(template, region), nil
}

// ResetToDefaults 重置为默认配置
func (h *HuaweiProviderImpl) ResetToDefaults() {
	h.config = getDefaultHuaweiConfig()
	h.logger.Info("配置已重置为默认值")
}

// ExportConfig 导出配置（用于持久化）
func (h *HuaweiProviderImpl) ExportConfig() ([]byte, error) {
	return json.Marshal(h.config)
}

// ImportConfig 导入配置（从持久化存储加载）
func (h *HuaweiProviderImpl) ImportConfig(data []byte) error {
	var config HuaweiCloudConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("配置解析失败: %w", err)
	}
	if err := h.validateConfig(&config); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}
	h.config = &config
	h.logger.Info("配置导入成功")
	return nil
}
