package cache

import (
	"fmt"
	pc "github.com/prometheus/prometheus/config"
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
	return yaml.Marshal(&document)
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
