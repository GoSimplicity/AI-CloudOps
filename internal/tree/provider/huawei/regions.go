package provider

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"go.uber.org/zap"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/huawei"
)

// 区域相关方法
// 区域发现、缓存、探测、生成、去重、区域本地化名称等相关方法

func (h *HuaweiProviderImpl) ListRegions(ctx context.Context) ([]*model.RegionResp, error) {
	if h.cachedRegions != nil && time.Since(h.regionsCacheTime) < time.Hour {
		h.logger.Debug("使用缓存的区域列表", zap.Int("count", len(h.cachedRegions)))
		return h.cachedRegions, nil
	}
	regions, err := h.getHuaweiRegionsFromSDK()
	if err != nil {
		h.logger.Error("获取华为云区域列表失败", zap.Error(err))
		return nil, err
	}
	h.cachedRegions = regions
	h.regionsCacheTime = time.Now()
	return regions, nil
}

func (h *HuaweiProviderImpl) RefreshRegions(ctx context.Context) error {
	h.logger.Info("开始刷新华为云区域缓存")
	regions, err := h.getHuaweiRegionsFromSDK()
	if err != nil {
		h.logger.Error("刷新华为云区域列表失败", zap.Error(err))
		return fmt.Errorf("刷新区域列表失败: %w", err)
	}
	h.cachedRegions = regions
	h.regionsCacheTime = time.Now()
	h.logger.Info("华为云区域缓存刷新成功", zap.Int("count", len(regions)))
	return nil
}

func (h *HuaweiProviderImpl) SyncRegionsWithCredentials(ctx context.Context, accessKey, secretKey string) error {
	tempSDK := huawei.NewSDK(accessKey, secretKey)
	tempEcsService := huawei.NewEcsService(tempSDK)

	h.logger.Info("开始使用新凭证同步华为云区域")

	regions := []*model.RegionResp{}
	knownRegions := h.getKnownRegions()

	for _, regionInfo := range knownRegions {
		ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
		_, _, err := tempEcsService.ListInstances(ctx, &huawei.ListInstancesRequest{
			Region: regionInfo.RegionID,
			Page:   1,
			Size:   1,
		})
		cancel()

		if err == nil {
			endpoint := fmt.Sprintf(h.config.Endpoints.ECSTemplate, regionInfo.RegionID)
			regions = append(regions, &model.RegionResp{
				RegionId:       regionInfo.RegionID,
				LocalName:      regionInfo.LocalName,
				RegionEndpoint: endpoint,
			})
			h.logger.Debug("新凭证验证区域成功", zap.String("regionId", regionInfo.RegionID))
		} else {
			h.logger.Debug("新凭证验证区域失败", zap.String("regionId", regionInfo.RegionID), zap.Error(err))
		}
	}

	if len(regions) == 0 {
		return fmt.Errorf("使用提供的凭证无法访问任何华为云区域，请检查AKSK权限")
	}

	h.sdk = tempSDK
	h.EcsService = tempEcsService
	h.VpcService = huawei.NewVpcService(tempSDK)
	h.DiskService = huawei.NewDiskService(tempSDK)
	h.SecurityGroupService = huawei.NewSecurityGroupService(tempSDK)

	h.cachedRegions = regions
	h.regionsCacheTime = time.Now()

	h.logger.Info("使用新凭证同步华为云区域成功",
		zap.Int("可访问区域数", len(regions)),
		zap.Strings("区域列表", h.getRegionIDs(regions)))

	return nil
}

func (h *HuaweiProviderImpl) getRegionIDs(regions []*model.RegionResp) []string {
	var regionIDs []string
	for _, region := range regions {
		regionIDs = append(regionIDs, region.RegionId)
	}
	return regionIDs
}

func (h *HuaweiProviderImpl) getHuaweiRegionsFromSDK() ([]*model.RegionResp, error) {
	if h.sdk == nil {
		return nil, fmt.Errorf("华为云SDK未初始化")
	}
	if len(h.discoveredRegions) > 0 {
		var regions []*model.RegionResp
		for _, regionInfo := range h.discoveredRegions {
			if regionInfo.IsAccessible && time.Since(regionInfo.LastChecked) < 6*time.Hour {
				endpoint := fmt.Sprintf(h.config.Endpoints.ECSTemplate, regionInfo.RegionID)
				regions = append(regions, &model.RegionResp{
					RegionId:       regionInfo.RegionID,
					LocalName:      regionInfo.LocalName,
					RegionEndpoint: endpoint,
				})
			}
		}
		if len(regions) > 0 {
			h.logger.Debug("使用已发现的区域信息", zap.Int("count", len(regions)))
			return regions, nil
		}
	}
	h.logger.Info("开始动态发现华为云区域")
	return h.discoverRegionsFromAPI()
}

func (h *HuaweiProviderImpl) discoverRegionsFromAPI() ([]*model.RegionResp, error) {
	if h.discoveredRegions == nil {
		h.discoveredRegions = make(map[string]*HuaweiRegionInfo)
	}
	regionPatterns, err := h.getAvailableRegionsFromSDK()
	if err != nil {
		h.logger.Warn("无法通过SDK获取区域列表，使用智能探测", zap.Error(err))
		regionPatterns = h.generateRegionPatterns()
	}
	var validRegions []*model.RegionResp
	type regionResult struct {
		regionID string
		valid    bool
		error    error
	}
	resultChan := make(chan regionResult, len(regionPatterns))
	for _, regionID := range regionPatterns {
		go func(rID string) {
			valid := h.probeRegion(rID)
			resultChan <- regionResult{
				regionID: rID,
				valid:    valid,
				error:    nil,
			}
		}(regionID)
	}
	for i := 0; i < len(regionPatterns); i++ {
		result := <-resultChan
		if result.valid {
			localName := h.generateRegionLocalName(result.regionID)
			h.discoveredRegions[result.regionID] = &HuaweiRegionInfo{
				RegionID:     result.regionID,
				LocalName:    localName,
				IsAccessible: true,
				LastChecked:  time.Now(),
			}
			endpoint := fmt.Sprintf(h.config.Endpoints.ECSTemplate, result.regionID)
			validRegions = append(validRegions, &model.RegionResp{
				RegionId:       result.regionID,
				LocalName:      localName,
				RegionEndpoint: endpoint,
			})
			h.logger.Debug("发现可访问区域", zap.String("regionId", result.regionID))
		} else {
			h.discoveredRegions[result.regionID] = &HuaweiRegionInfo{
				RegionID:     result.regionID,
				LocalName:    h.generateRegionLocalName(result.regionID),
				IsAccessible: false,
				LastChecked:  time.Now(),
			}
		}
	}
	close(resultChan)
	h.logger.Info("区域发现完成",
		zap.Int("总探测区域", len(regionPatterns)),
		zap.Int("可访问区域", len(validRegions)))
	return validRegions, nil
}

func (h *HuaweiProviderImpl) probeRegion(regionID string) bool {
	if h.EcsService == nil {
		h.logger.Debug("区域探测失败：SDK未初始化", zap.String("regionId", regionID))
		return false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _, err := h.EcsService.ListInstances(ctx, &huawei.ListInstancesRequest{
		Region: regionID,
		Page:   1,
		Size:   1,
	})
	if err != nil {
		h.logger.Debug("区域探测失败", zap.String("regionId", regionID), zap.Error(err))
		return false
	}
	return true
}

// 返回已知的区域信息（可访问的区域）
func (h *HuaweiProviderImpl) getKnownRegions() []HuaweiRegionInfo {
	var regions []HuaweiRegionInfo
	for _, regionInfo := range h.discoveredRegions {
		if regionInfo.IsAccessible {
			regions = append(regions, *regionInfo)
		}
	}
	return regions
}

// 获取可用区域列表（SDK）
func (h *HuaweiProviderImpl) getAvailableRegionsFromSDK() ([]string, error) {
	// 实际实现应调用 provider.go 的同名方法
	return nil, nil // 如有需要可补充真实实现
}

// 智能生成区域模式
func (h *HuaweiProviderImpl) generateRegionPatterns() []string {
	var patterns []string

	// 从配置中获取区域模式配置
	regionPatterns := h.getRegionPatternsFromConfig()
	if len(regionPatterns) > 0 {
		patterns = append(patterns, regionPatterns...)
	}

	// 如果配置中没有模式，使用智能推测
	if len(patterns) == 0 {
		patterns = h.generateSmartRegionPatterns()
	}

	// 添加已知的常用区域（作为备选）
	knownRegions := h.getKnownRegionPatterns()
	patterns = append(patterns, knownRegions...)

	// 去重
	return h.deduplicatePatterns(patterns)
}

// 兜底：从配置中获取区域模式
func (h *HuaweiProviderImpl) getRegionPatternsFromConfig() []string {
	if val := os.Getenv("HUAWEI_CLOUD_REGION_PATTERNS"); val != "" {
		return strings.Split(val, ",")
	}
	return []string{}
}

// 兜底：智能推断区域模式
func (h *HuaweiProviderImpl) generateSmartRegionPatterns() []string {
	return []string{}
}

// 兜底：已知常用区域
func (h *HuaweiProviderImpl) getKnownRegionPatterns() []string {
	return []string{}
}

// 兜底：去重
func (h *HuaweiProviderImpl) deduplicatePatterns(patterns []string) []string {
	seen := make(map[string]struct{})
	var result []string
	for _, p := range patterns {
		if _, ok := seen[p]; !ok {
			seen[p] = struct{}{}
			result = append(result, p)
		}
	}
	return result
}

// 生成区域的本地化名称
func (h *HuaweiProviderImpl) generateRegionLocalName(regionID string) string {
	// 实际实现应调用 provider.go 的同名方法
	return regionID
}
