package utils

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	pcc "github.com/prometheus/common/config"
)

// ScrapeHTTPClientConfig 仅配置指标目标，服务发现仍使用自身的身份
func ScrapeHTTPClientConfig(job *model.MonitorScrapeJob) (pcc.HTTPClientConfig, error) {
	cfg := pcc.DefaultHTTPClientConfig
	cfg.BearerToken = pcc.Secret(job.BearerToken)
	cfg.BearerTokenFile = job.BearerTokenFile
	cfg.TLSConfig.CA = job.TlsCaContent
	cfg.TLSConfig.CAFile = job.TlsCaFilePath
	if err := cfg.Validate(); err != nil {
		return cfg, err
	}
	if err := cfg.TLSConfig.Validate(); err != nil {
		return cfg, err
	}
	return cfg, nil
}
