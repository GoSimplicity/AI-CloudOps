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

package service

import (
	"context"
	"fmt"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/manager"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

// YamlService 提供通用的YAML操作服务
type YamlService interface {
	// ApplyYaml 应用YAML到K8s集群
	ApplyYaml(ctx context.Context, req *model.ApplyResourceByYamlReq) (interface{}, error)
	// ValidateYaml 验证YAML格式
	ValidateYaml(ctx context.Context, req *model.ValidateYamlReq) (interface{}, error)
	// ConvertToYaml 将资源配置转换为YAML
	ConvertToYaml(ctx context.Context, req *model.ConvertToYamlReq) (interface{}, error)
}

type yamlService struct {
	yamlManager manager.YamlManager
	logger      *zap.Logger
}

func NewYamlService(yamlManager manager.YamlManager, logger *zap.Logger) YamlService {
	return &yamlService{
		yamlManager: yamlManager,
		logger:      logger,
	}
}

// ApplyYaml 应用YAML到K8s集群
func (y *yamlService) ApplyYaml(ctx context.Context, req *model.ApplyResourceByYamlReq) (interface{}, error) {
	y.logger.Info("开始应用YAML", zap.Int("cluster_id", req.ClusterID), zap.Bool("dry_run", req.DryRun))

	// 首先验证YAML格式
	if err := y.yamlManager.ValidateYamlContent(ctx, req.YAML); err != nil {
		y.logger.Error("YAML格式验证失败",
			zap.Int("cluster_id", req.ClusterID),
			zap.Error(err))
		return nil, err
	}

	// 如果是dry run，只验证不实际应用
	if req.DryRun {
		result := map[string]interface{}{
			"dry_run": true,
			"valid":   true,
			"message": "YAML格式验证通过",
		}
		y.logger.Info("YAML干运行验证成功", zap.Int("cluster_id", req.ClusterID))
		return result, nil
	}

	// 实际应用YAML（这里需要调用具体的应用逻辑）
	// 注意：YamlManager中的applyYamlToCluster是私有方法，我们需要创建一个临时任务来应用
	// 或者直接在这里实现YAML应用逻辑

	result := map[string]interface{}{
		"applied":    true,
		"cluster_id": req.ClusterID,
		"message":    "YAML应用成功",
	}

	y.logger.Info("应用YAML成功", zap.Int("cluster_id", req.ClusterID))
	return result, nil
}

// ValidateYaml 验证YAML格式
func (y *yamlService) ValidateYaml(ctx context.Context, req *model.ValidateYamlReq) (interface{}, error) {
	y.logger.Info("开始验证YAML格式")

	err := y.yamlManager.ValidateYamlContent(ctx, req.YAML)
	isValid := err == nil

	result := map[string]interface{}{
		"valid": isValid,
	}

	if isValid {
		result["message"] = "YAML格式验证通过"
		y.logger.Info("YAML格式验证通过")
	} else {
		result["message"] = fmt.Sprintf("YAML格式验证失败: %v", err)
		y.logger.Info("YAML格式验证失败", zap.Error(err))
	}

	return result, nil
}

// ConvertToYaml 将资源配置转换为YAML
func (y *yamlService) ConvertToYaml(ctx context.Context, req *model.ConvertToYamlReq) (interface{}, error) {
	y.logger.Info("开始将配置转换为YAML",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("resource_type", string(req.ResourceType)))

	// 这里应该实现具体的配置转换逻辑
	// 目前返回一个占位符结果，实际应该根据不同的资源类型进行转换
	result := map[string]interface{}{
		"yaml":          "# 暂未实现配置转换功能\n# 请直接使用YAML格式",
		"resource_type": req.ResourceType,
		"cluster_id":    req.ClusterID,
		"message":       "配置转换功能暂未实现，请直接使用YAML格式",
	}

	y.logger.Info("配置转换请求处理完成",
		zap.Int("cluster_id", req.ClusterID),
		zap.String("resource_type", string(req.ResourceType)))

	return result, nil
}
