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

	"github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/discovery/targetgroup"
	"go.uber.org/zap"
)

type NotAuthService interface {
	BuildPrometheusServiceDiscovery(ctx context.Context, ipAddress string) ([]*targetgroup.Group, error)
}

type notAuthService struct {
	l *zap.Logger
}

func NewNotAuthService(l *zap.Logger) NotAuthService {
	return &notAuthService{
		l: l,
	}
}

// BuildPrometheusServiceDiscovery 构建 Prometheus 服务发现的目标组
func (n *notAuthService) BuildPrometheusServiceDiscovery(ctx context.Context, ipAddress string) ([]*targetgroup.Group, error) {
	if ipAddress == "" {
		n.l.Warn("IP 地址为空")
		return nil, fmt.Errorf("IP 地址不能为空")
	}

	// 创建目标组
	tg := &targetgroup.Group{
		Targets: []model.LabelSet{
			{
				model.AddressLabel: model.LabelValue(ipAddress),
			},
		},
		Labels: model.LabelSet{
			"instance": model.LabelValue(ipAddress),
		},
	}

	return []*targetgroup.Group{tg}, nil
}
