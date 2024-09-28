package prometheus

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
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
		if _, ok := ips[ip]; ok {
			return true
		}
	}

	return false
}
