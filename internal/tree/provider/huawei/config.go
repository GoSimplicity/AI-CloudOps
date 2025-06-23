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
type HuaweiCloudConfig struct {
	Regions   map[string]HuaweiRegionConfig `json:"regions"`
	Defaults  HuaweiDefaultConfig           `json:"defaults"`
	Endpoints HuaweiEndpointConfig          `json:"endpoints"`
	Discovery HuaweiDiscoveryConfig         `json:"discovery"`
}

type HuaweiRegionConfig struct {
	RegionID     string                 `json:"region_id"`
	LocalName    string                 `json:"local_name"`
	CityName     string                 `json:"city_name"`
	Enabled      bool                   `json:"enabled"`
	ZonePrefix   string                 `json:"zone_prefix"`
	IsAccessible bool                   `json:"is_accessible"`
	LastChecked  *time.Time             `json:"last_checked,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type HuaweiDefaultConfig struct {
	InstanceChargeType string `json:"instance_charge_type"`
	ForceDelete        bool   `json:"force_delete"`
	ForceStop          bool   `json:"force_stop"`
	PageSize           int    `json:"page_size"`
	MaxRetries         int    `json:"max_retries"`
	TimeoutSeconds     int    `json:"timeout_seconds"`
	ConcurrentLimit    int    `json:"concurrent_limit"`
}

type HuaweiEndpointConfig struct {
	ECSTemplate         string            `json:"ecs_template"`
	VPCTemplate         string            `json:"vpc_template"`
	EVSTemplate         string            `json:"evs_template"`
	IAMTemplate         string            `json:"iam_template"`
	CustomEndpoints     map[string]string `json:"custom_endpoints,omitempty"`
	GlobalEndpoint      string            `json:"global_endpoint"`
	InternationalSuffix string            `json:"international_suffix"`
}

type HuaweiDiscoveryConfig struct {
	EnableAutoDiscovery bool          `json:"enable_auto_discovery"`
	CacheTimeout        time.Duration `json:"cache_timeout"`
	ProbeTimeout        time.Duration `json:"probe_timeout"`
	MaxConcurrent       int           `json:"max_concurrent"`
	RetryAttempts       int           `json:"retry_attempts"`
	RetryDelay          time.Duration `json:"retry_delay"`
}

type HuaweiRegionInfo struct {
	RegionID     string    `json:"region_id"`
	LocalName    string    `json:"local_name"`
	IsAccessible bool      `json:"is_accessible"`
	LastChecked  time.Time `json:"last_checked"`
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
