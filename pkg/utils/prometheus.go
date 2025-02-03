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
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/prometheus/alertmanager/pkg/labels"
	pcc "github.com/prometheus/common/config"
	promModel "github.com/prometheus/common/model"
	pc "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/prometheus/prometheus/promql/parser"
	"go.uber.org/zap"
)

func CheckPoolIpExists(pools []*model.MonitorScrapePool, req *model.MonitorScrapePool) error {
	var ipPrometheus []string
	var ipAlertManager []string

	for _, pool := range pools {
		if pool.ID == req.ID {
			continue
		}

		ipPrometheus = append(ipPrometheus, pool.PrometheusInstances...)
		ipAlertManager = append(ipAlertManager, pool.AlertManagerInstances...)
	}

	for _, ip := range req.PrometheusInstances {
		if slices.Contains(ipPrometheus, ip) {
			return fmt.Errorf("PrometheusInstances %v 已存在", ip)
		}
	}

	for _, ip := range req.AlertManagerInstances {
		if slices.Contains(ipAlertManager, ip) {
			return fmt.Errorf("AlertManagerInstances %v 已存在", ip)
		}
	}

	return nil
}

func CheckAlertsIpExists(req *model.MonitorAlertManagerPool, rules []*model.MonitorAlertManagerPool) bool {
	ips := make(map[string]string)

	for _, rule := range rules {
		if req.ID == rule.ID {
			continue
		}

		for _, ip := range rule.AlertManagerInstances {
			ips[ip] = rule.Name
		}
	}

	for _, ip := range req.AlertManagerInstances {
		if req.ID != 0 && ips[ip] == req.Name {
			continue
		}

		if _, ok := ips[ip]; ok {
			return true
		}
	}

	return false
}

func CheckAlertIpExists(req *model.MonitorAlertManagerPool, pools []*model.MonitorAlertManagerPool) error {
	var ips []string

	for _, pool := range pools {
		if pool.ID == req.ID {
			continue
		}

		ips = append(ips, pool.AlertManagerInstances...)
	}

	for _, ip := range req.AlertManagerInstances {
		if slices.Contains(ips, ip) {
			return fmt.Errorf("AlertManagerInstances %v 已存在", ip)
		}
	}

	return nil
}

// ParseTags 将 ECS 的 Tags 切片解析为 Prometheus 的标签映射
func ParseTags(tags []string) (map[promModel.LabelName]promModel.LabelValue, error) {
	labels := make(map[promModel.LabelName]promModel.LabelValue)

	// 遍历 tags 切片，每两个元素构成一个键值对
	for i := 0; i < len(tags); i += 2 {
		key := strings.TrimSpace(tags[i])
		if key == "" {
			return nil, fmt.Errorf("标签键不能为空")
		}

		// 确保有对应的值
		if i+1 >= len(tags) {
			return nil, fmt.Errorf("标签值缺失，键: '%s' 无对应值", key)
		}

		value := strings.TrimSpace(tags[i+1])
		labels[promModel.LabelName(key)] = promModel.LabelValue(value)
	}

	return labels, nil
}

// ParseExternalLabels 解析外部标签
func ParseExternalLabels(labelsList []string) []string {
	var parsed []string

	// 示例：["key1=value1", "key2=value2"]
	for _, label := range labelsList {
		// 根据 "=" 分割字符串
		parts := strings.SplitN(label, "=", 2)
		if len(parts) == 2 {
			parsed = append(parsed, parts[0], parts[1])
		}
	}

	// 返回的格式为 ["key1", "value1", "key2", "value2"]
	return parsed
}

// ParseURL 解析字符串为URL，返回错误而非 panic
func ParseURL(u string) (*pcc.URL, error) {
	parsed, err := url.Parse(u)
	if err != nil {
		return nil, fmt.Errorf("无效的URL: %s", u)
	}

	return &pcc.URL{URL: parsed}, nil
}

// GenPromDuration 转换秒为Prometheus Duration
func GenPromDuration(seconds int) promModel.Duration {
	if seconds <= 0 {
		return promModel.Duration(5 * time.Second)
	}
	return promModel.Duration(time.Duration(seconds) * time.Second)
}

// DeepCopyScrapeConfig 深度拷贝 ScrapeConfig
func DeepCopyScrapeConfig(sc *pc.ScrapeConfig) *pc.ScrapeConfig {
	copySc := *sc

	// 深度拷贝 RelabelConfigs
	if sc.RelabelConfigs != nil {
		copySc.RelabelConfigs = make([]*relabel.Config, len(sc.RelabelConfigs))
		for i, rc := range sc.RelabelConfigs {
			copyRC := *rc
			copySc.RelabelConfigs[i] = &copyRC
		}
	}

	// 深度拷贝 ServiceDiscoveryConfigs
	if sc.ServiceDiscoveryConfigs != nil {
		copySc.ServiceDiscoveryConfigs = make(discovery.Configs, len(sc.ServiceDiscoveryConfigs))
		copy(copySc.ServiceDiscoveryConfigs, sc.ServiceDiscoveryConfigs)
	}

	return &copySc
}

func PromqlExprCheck(expr string) (bool, error) {
	if expr == "" {
		return false, fmt.Errorf("expression cannot be empty")
	}

	// 解析 PromQL 表达式
	_, err := parser.ParseExpr(expr)
	if err != nil {
		return false, fmt.Errorf("invalid PromQL expression: %v", err)
	}

	return true, nil
}

func BuildMatchers(alertEvent *model.MonitorAlertEvent, l *zap.Logger, useName bool) ([]*labels.Matcher, error) {
	var matchers []*labels.Matcher
	if useName {
		// 如果 useName 为 true，仅使用 alertname 匹配器
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
		// 否则，使用所有标签匹配器
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

func SendSilenceRequest(ctx context.Context, l *zap.Logger, url string, data []byte) (string, error) {
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(data))
	if err != nil {
		l.Error("sendSilenceRequest failed: create HTTP request error", zap.Error(err))
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		l.Error("sendSilenceRequest failed: send HTTP request error", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		l.Error("sendSilenceRequest failed: AlertManager response error", zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
		return "", fmt.Errorf("AlertManager request failed, status: %d, response: %s", resp.StatusCode, string(body))
	}

	// 解析响应
	var result struct {
		Status string `json:"status"`
		Data   struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		l.Error("sendSilenceRequest failed: decode response error", zap.Error(err))
		return "", err
	}

	if result.Status != "success" {
		l.Error("sendSilenceRequest failed: AlertManager status not success", zap.String("status", result.Status))
		return "", fmt.Errorf("AlertManager status not success, status: %s", result.Status)
	}

	return result.Data.ID, nil
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

// PostWithJson 发送带有JSON字符串的POST请求
func PostWithJson(ctx context.Context, client *http.Client, l *zap.Logger, url string, jsonStr string, params map[string]string, headers map[string]string) ([]byte, error) {
	// 创建 HTTP 请求
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer([]byte(jsonStr)))
	if err != nil {
		l.Error("创建 HTTP 请求失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}

	// 设置查询参数
	q := req.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}

	req.URL.RawQuery = q.Encode()

	// 设置请求头
	for key, value := range headers {
		req.Header.Set(key, value)
	}

	// 设置默认 Content-Type
	if _, exists := headers["Content-Type"]; !exists {
		req.Header.Set("Content-Type", "application/json")
	}

	// 发送请求
	resp, err := client.Do(req)
	if err != nil {
		l.Error("发送 HTTP 请求失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}
	defer resp.Body.Close()

	// 读取响应体
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		l.Error("读取响应体失败",
			zap.Error(err),
			zap.String("url", url),
		)
		return nil, err
	}

	// 检查 HTTP 状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		l.Error("服务器返回非2xx状态码",
			zap.String("url", url),
			zap.Int("statusCode", resp.StatusCode),
			zap.String("responseBody", string(bodyBytes)),
		)
		return bodyBytes, fmt.Errorf("server returned HTTP status %s", resp.Status)
	}

	return bodyBytes, nil
}

// CloneMap 克隆一个字符串到字符串的映射
func CloneMap(original map[string]string) map[string]string {
	if original == nil {
		return nil
	}
	cloned := make(map[string]string, len(original))
	for k, v := range original {
		cloned[k] = v
	}
	return cloned
}

// FormatMap 将 map[string]string 格式化为字符串，每个键值对占一行
func FormatMap(m map[string]string) string {
	var builder strings.Builder
	for k, v := range m {
		builder.WriteString(fmt.Sprintf("%s=%s ", k, v))
	}
	return strings.TrimSpace(builder.String())
}

type promHashPayload struct {
	Name                  string   `json:"name"`
	PrometheusInstances   []string `json:"prometheus_instances"`
	AlertManagerInstances []string `json:"alert_manager_instances"`
	ScrapeInterval        int      `json:"scrape_interval"`
	ScrapeTimeout         int      `json:"scrape_timeout"`
	RemoteTimeoutSeconds  int      `json:"remote_timeout_seconds"`
	SupportAlert          bool     `json:"support_alert"`
	SupportRecord         bool     `json:"support_record"`
	ExternalLabels        []string `json:"external_labels"`
	RemoteWriteUrl        string   `json:"remote_write_url"`
	RemoteReadUrl         string   `json:"remote_read_url"`
	AlertManagerUrl       string   `json:"alert_manager_url"`
	RecordFilePath        string   `json:"record_file_path"`
}

// CalculateHash 计算哈希值
func CalculatePromHash(pool *model.MonitorScrapePool) string {
	// 排序列表字段确保顺序不影响哈希
	sort.Strings(pool.PrometheusInstances)
	sort.Strings(pool.AlertManagerInstances)
	sort.Strings(pool.ExternalLabels)

	payload := promHashPayload{
		Name:                  pool.Name,
		PrometheusInstances:   pool.PrometheusInstances,
		AlertManagerInstances: pool.AlertManagerInstances,
		ScrapeInterval:        pool.ScrapeInterval,
		ScrapeTimeout:         pool.ScrapeTimeout,
		RemoteTimeoutSeconds:  pool.RemoteTimeoutSeconds,
		SupportAlert:          pool.SupportAlert,
		SupportRecord:         pool.SupportRecord,
		ExternalLabels:        pool.ExternalLabels,
		RemoteWriteUrl:        pool.RemoteWriteUrl,
		RemoteReadUrl:         pool.RemoteReadUrl,
		AlertManagerUrl:       pool.AlertManagerUrl,
		RecordFilePath:        pool.RecordFilePath,
	}

	// 稳定序列化
	data, _ := json.Marshal(payload)

	// 计算哈希
	hash := sha256.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}

type alertHashPayload struct {
	Name                  string   `json:"name"`
	AlertManagerInstances []string `json:"alert_manager_instances"`
	ResolveTimeout        string   `json:"resolve_timeout"`
	GroupWait             string   `json:"group_wait"`
	GroupInterval         string   `json:"group_interval"`
	RepeatInterval        string   `json:"repeat_interval"`
	GroupBy               []string `json:"group_by"`
	Receiver              string   `json:"receiver"`
}

func CalculateAlertHash(pool *model.MonitorAlertManagerPool) string {
	// 排序列表字段确保顺序不影响哈希
	sort.Strings(pool.AlertManagerInstances)
	sort.Strings(pool.GroupBy)

	payload := alertHashPayload{
		Name:                  pool.Name,
		AlertManagerInstances: pool.AlertManagerInstances,
		ResolveTimeout:        pool.ResolveTimeout,
		GroupWait:             pool.GroupWait,
		GroupInterval:         pool.GroupInterval,
		RepeatInterval:        pool.RepeatInterval,
		GroupBy:               pool.GroupBy,
		Receiver:              pool.Receiver,
	}

	// 稳定序列化
	data, _ := json.Marshal(payload)

	// 计算哈希
	hash := sha256.New()
	hash.Write(data)
	return hex.EncodeToString(hash.Sum(nil))
}
