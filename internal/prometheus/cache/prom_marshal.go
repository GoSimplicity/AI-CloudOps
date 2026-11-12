package cache

import (
	"fmt"

	pc "github.com/prometheus/prometheus/config"
	httpsd "github.com/prometheus/prometheus/discovery/http"
	"gopkg.in/yaml.v3"
)

// marshalPrometheusConfig 导出可执行配置，不改变SDK全局Secret脱敏开关
func marshalPrometheusConfig(config pc.Config) ([]byte, error) {
	var document yaml.Node
	if err := document.Encode(config); err != nil {
		return nil, err
	}
	scrapes := yamlField(&document, "scrape_configs")
	for i, scrape := range config.ScrapeConfigs {
		if scrape.HTTPClientConfig.Authorization == nil || scrape.HTTPClientConfig.Authorization.Credentials == "" {
			continue
		}
		if scrapes == nil || i >= len(scrapes.Content) {
			return nil, fmt.Errorf("采集配置序列化结构无效")
		}
		authorization := yamlField(scrapes.Content[i], "authorization")
		credentials := yamlField(authorization, "credentials")
		if credentials == nil {
			return nil, fmt.Errorf("采集认证序列化结构无效")
		}
		credentials.Value = string(scrape.HTTPClientConfig.Authorization.Credentials)
		credentials.Tag = "!!str"
	}
	if err := restoreHTTPServiceDiscoveryCredentials(scrapes, config.ScrapeConfigs); err != nil {
		return nil, err
	}
	return yaml.Marshal(&document)
}

// restoreHTTPServiceDiscoveryCredentials 把 HTTP-SD 的 bearer token 写回可执行配置。
// SDK 序列化会把 Secret 脱敏成 <secret>，Prometheus 需要真实令牌才能访问 HTTP-SD。
func restoreHTTPServiceDiscoveryCredentials(scrapes *yaml.Node, configs []*pc.ScrapeConfig) error {
	if scrapes == nil {
		return nil
	}
	for i, scrape := range configs {
		if scrape == nil {
			continue
		}
		var httpConfigs []*httpsd.SDConfig
		restore := false
		for _, item := range scrape.ServiceDiscoveryConfigs {
			httpSD, ok := item.(*httpsd.SDConfig)
			if !ok || httpSD == nil {
				continue
			}
			httpConfigs = append(httpConfigs, httpSD)
			if httpSD.HTTPClientConfig.Authorization != nil && httpSD.HTTPClientConfig.Authorization.Credentials != "" {
				restore = true
			}
		}
		if !restore {
			continue
		}
		if i >= len(scrapes.Content) {
			return fmt.Errorf("HTTP SD 序列化结构无效")
		}
		nodes := yamlField(scrapes.Content[i], "http_sd_configs")
		if nodes == nil || nodes.Kind != yaml.SequenceNode || len(nodes.Content) < len(httpConfigs) {
			return fmt.Errorf("HTTP SD 序列化结构无效")
		}
		for n, httpSD := range httpConfigs {
			if httpSD.HTTPClientConfig.Authorization == nil || httpSD.HTTPClientConfig.Authorization.Credentials == "" {
				continue
			}
			credentials := yamlField(yamlField(nodes.Content[n], "authorization"), "credentials")
			if credentials == nil {
				return fmt.Errorf("HTTP SD 认证序列化结构无效")
			}
			credentials.Value = string(httpSD.HTTPClientConfig.Authorization.Credentials)
			credentials.Tag = "!!str"
		}
	}
	return nil
}

func yamlField(node *yaml.Node, key string) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) == 1 {
		node = node.Content[0]
	}
	if node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}
