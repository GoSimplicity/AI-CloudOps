package prometheus

import (
	"fmt"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	promModel "github.com/prometheus/common/model"
	"strings"
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
