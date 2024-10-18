package prometheus

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert/onduty"
	"github.com/prometheus/alertmanager/pkg/labels"
	pcc "github.com/prometheus/common/config"
	pm "github.com/prometheus/common/model"
	promModel "github.com/prometheus/common/model"
	pc "github.com/prometheus/prometheus/config"
	"github.com/prometheus/prometheus/discovery"
	"github.com/prometheus/prometheus/model/relabel"
	"github.com/prometheus/prometheus/promql/parser"
	"go.uber.org/zap"
	"gorm.io/gorm"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

func CheckPoolIpExists(req *model.MonitorScrapePool, pools []*model.MonitorScrapePool) bool {
	ips := make(map[string]string)

	// 遍历现有抓取池，将IP地址添加到映射中
	for _, pool := range pools {
		for _, ip := range pool.PrometheusInstances {
			ips[ip] = pool.Name
		}
	}

	// 遍历请求中的IP地址，检查是否存在于现有抓取池中
	for _, ip := range req.PrometheusInstances {
		if req.ID != 0 && ips[ip] == req.Name {
			continue
		}

		if _, ok := ips[ip]; ok {
			return true
		}
	}

	return false
}

func CheckAlertIpExists(req *model.MonitorAlertManagerPool, rules []*model.MonitorAlertManagerPool) bool {
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
func GenPromDuration(seconds int) pm.Duration {
	return pm.Duration(time.Duration(seconds) * time.Second)
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
		alertName, exists := alertEvent.LabelsMatcher["alertname"]
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
		for key, val := range alertEvent.LabelsMatcher {
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

// HandleList 处理搜索或获取所有记录
func HandleList[T any](ctx context.Context, search *string, searchFunc func(ctx context.Context, name string) ([]*T, error), listFunc func(ctx context.Context) ([]*T, error)) ([]*T, error) {
	if search != nil && *search != "" {
		return searchFunc(ctx, *search)
	}
	return listFunc(ctx)
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

// CalculateOnDutyUser 根据排班规则计算指定日期的值班人
func CalculateOnDutyUser(ctx context.Context, l *zap.Logger, dao onduty.AlertManagerOnDutyDAO, group *model.MonitorOnDutyGroup, dateStr string, todayStr string) *model.User {
	// 解析目标日期
	targetDate, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		l.Error("日期解析失败", zap.String("date", dateStr), zap.Error(err))
		return nil
	}

	// 解析今天的日期
	today, err := time.Parse("2006-01-02", todayStr)
	if err != nil {
		l.Error("解析今天日期失败", zap.String("today", todayStr), zap.Error(err))
		return nil
	}

	// 计算从今天开始的天数差
	daysDiff := int(targetDate.Sub(today).Hours()) / 24

	// 获取当前值班人的索引
	currentUserIndex := GetCurrentUserIndex(ctx, l, dao, group, todayStr)
	if currentUserIndex == -1 {
		l.Warn("未找到当前值班人", zap.String("today", todayStr))
		return nil
	}

	// 获取总成员数和每班轮值天数
	totalMembers := len(group.Members)
	shiftDays := group.ShiftDays

	// 检查轮班天数是否为零，避免除零错误
	if shiftDays == 0 || totalMembers == 0 {
		l.Error("轮班天数或成员数量无效", zap.Int("shiftDays", shiftDays), zap.Int("totalMembers", totalMembers))
		return nil
	}

	// 总的轮班周期长度
	totalShiftLength := totalMembers * shiftDays

	// 自定义取模函数，确保结果为非负数
	mod := func(a, b int) int {
		if b == 0 {
			l.Error("除零错误", zap.Int("b", b))
			return -1
		}
		return (a%b + b) % b
	}

	// 计算目标日期的值班人索引
	indexInShift := mod(currentUserIndex*shiftDays+daysDiff, totalShiftLength)
	if indexInShift == -1 {
		l.Error("取模运算出错，可能是因为轮班天数或成员数量无效", zap.Int("currentUserIndex", currentUserIndex), zap.Int("shiftDays", shiftDays), zap.Int("totalMembers", totalMembers))
		return nil
	}

	// 确定值班人的索引
	userIndex := indexInShift / shiftDays

	// 确认索引范围有效，返回对应的成员
	if userIndex >= 0 && userIndex < totalMembers {
		return group.Members[userIndex]
	}

	// 如果索引超出范围
	l.Error("计算出的值班人索引无效", zap.Int("userIndex", userIndex), zap.Int("totalMembers", totalMembers))

	return nil
}

// GetCurrentUserIndex 获取当前值班人在成员列表中的索引
func GetCurrentUserIndex(ctx context.Context, l *zap.Logger, dao onduty.AlertManagerOnDutyDAO, group *model.MonitorOnDutyGroup, todayStr string) int {
	// 尝试从历史记录中获取今天的值班人
	todayHistory, err := dao.GetMonitorOnDutyHistoryByGroupIdAndDay(ctx, group.ID, todayStr)
	if err == nil && todayHistory.OnDutyUserID > 0 {
		for index, member := range group.Members {
			if member.ID == todayHistory.OnDutyUserID {
				return index
			}
		}
	} else if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		// 如果查询发生了其他错误，记录日志
		l.Error("获取今天的值班历史记录失败", zap.Error(err))
	}

	// 如果没有历史记录，默认第一个成员为当前值班人
	return 0
}
