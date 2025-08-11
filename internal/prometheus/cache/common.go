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

package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
)

const (
	// 日志模块标识
	LogModuleMonitor = "[监控模块]"

	// 配置文件名模板
	ConfigNameAlertManager = "alertmanager_pool_%d_%s.yaml"
	ConfigNamePrometheus   = "prometheus_scrape_pool_%d_%s.yaml"
	ConfigNameAlertRule    = "prometheus_alert_rule_%d_%s.yaml"
	ConfigNameRecordRule   = "prometheus_record_rule_%d_%s.yaml"
)

// calculateConfigHash 计算配置内容的哈希值
func calculateConfigHash(content string) string {
	hash := sha256.Sum256([]byte(content))
	return hex.EncodeToString(hash[:])
}

// validateYAMLConfig 验证YAML配置格式是否正确
func validateYAMLConfig(content string) error {
	var temp interface{}
	if err := yaml.Unmarshal([]byte(content), &temp); err != nil {
		return fmt.Errorf("YAML格式验证失败: %w", err)
	}
	return nil
}

// batchSaveConfigsToDatabase 批量保存配置到数据库的优化版本
func batchSaveConfigsToDatabase(
	ctx context.Context,
	batchManager *BatchConfigManager,
	configMap map[string]ConfigData,
) error {
	if len(configMap) == 0 {
		return nil
	}

	return batchManager.BatchSaveConfigs(ctx, configMap)
}

// logCacheOperation 统一的缓存操作日志记录
func logCacheOperation(logger *zap.Logger, operation string, poolName string, startTime time.Time, err error) {
	duration := time.Since(startTime)

	if err != nil {
		logger.Error(LogModuleMonitor+operation+"失败",
			zap.String("pool_name", poolName),
			zap.Duration("duration", duration),
			zap.Error(err))
	} else {
		logger.Info(LogModuleMonitor+operation+"成功",
			zap.String("pool_name", poolName),
			zap.Duration("duration", duration))
	}
}

// logBatchOperation 批量操作日志记录
func logBatchOperation(logger *zap.Logger, operation string, processed, total int, startTime time.Time) {
	logger.Info(LogModuleMonitor+operation+"批量处理完成",
		zap.Int("processed", processed),
		zap.Int("total", total),
		zap.Duration("duration", time.Since(startTime)))
}

// validateInstanceIPs 验证实例IP列表
func validateInstanceIPs(ips []string) error {
	if len(ips) == 0 {
		return fmt.Errorf("实例IP列表不能为空")
	}

	for _, ip := range ips {
		if strings.TrimSpace(ip) == "" {
			return fmt.Errorf("实例IP不能为空")
		}
	}

	return nil
}

// cleanupInvalidIPs 清理无效的IP配置
func cleanupInvalidIPs(configMap map[string]string, validIPs map[string]struct{}, logger *zap.Logger) {
	for ip := range configMap {
		if _, ok := validIPs[ip]; !ok {
			delete(configMap, ip)
			logger.Debug(LogModuleMonitor+"删除无效IP配置", zap.String("ip", ip))
		}
	}
}
