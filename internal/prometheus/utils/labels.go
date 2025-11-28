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

package utils

import (
	"fmt"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/prometheus/alertmanager/pkg/labels"
	promModel "github.com/prometheus/common/model"
	"go.uber.org/zap"
)

// ParseTags 将 ECS 的 Tags 切片解析为 Prometheus 的标签映射
func ParseTags(tags []string) (map[promModel.LabelName]promModel.LabelValue, error) {
	labelMap := make(map[promModel.LabelName]promModel.LabelValue)

	for i := 0; i < len(tags); i += 2 {
		key := strings.TrimSpace(tags[i])
		if key == "" {
			return nil, fmt.Errorf("标签键不能为空")
		}

		if i+1 >= len(tags) {
			return nil, fmt.Errorf("标签值缺失，键: '%s' 无对应值", key)
		}

		value := strings.TrimSpace(tags[i+1])
		labelMap[promModel.LabelName(key)] = promModel.LabelValue(value)
	}

	return labelMap, nil
}

// ParseExternalLabels 解析外部标签
func ParseExternalLabels(labelsList []string) []string {
	var parsed []string

	for _, label := range labelsList {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) == 2 {
			parsed = append(parsed, parts[0], parts[1])
		}
	}

	return parsed
}

func BuildMatchers(alertEvent *model.MonitorAlertEvent, l *zap.Logger, useName int8) ([]*labels.Matcher, error) {
	var matchers []*labels.Matcher
	if useName == 1 {
		alertName, exists := alertEvent.LabelsMap["alertname"]
		if !exists {
			l.Error("EventAlertSilence failed: alertname missing in LabelsMatcher", zap.Int("id", alertEvent.ID))
			return nil, fmt.Errorf("alertname missing in LabelsMatcher")
		}
		matchers = []*labels.Matcher{
			{
				Type:  labels.MatchEqual,
				Name:  "alertname",
				Value: alertName,
			},
		}
	} else {
		for key, val := range alertEvent.LabelsMap {
			matcher := &labels.Matcher{
				Type:  labels.MatchEqual,
				Name:  key,
				Value: val,
			}
			matchers = append(matchers, matcher)
		}
	}
	return matchers, nil
}

func FromSliceTuMap(kvs []string) map[string]string {
	labelsMap := make(map[string]string)
	for _, i := range kvs {
		parts := strings.Split(i, "=")
		if len(parts) != 2 {
			continue
		}
		labelsMap[parts[0]] = parts[1]
	}
	return labelsMap
}
