package config

import (
	"fmt"
	"net"
	"strings"
)

// defaultTrustedProxies 是直连对端默认可信的来源：回环和私有网段，覆盖 Compose 内网。
var defaultTrustedProxies = []string{
	"127.0.0.0/8",
	"::1/128",
	"10.0.0.0/8",
	"172.16.0.0/12",
	"192.168.0.0/16",
	"fc00::/7",
}

// DefaultTrustedProxies 返回默认可信代理网段的副本。
func DefaultTrustedProxies() []string {
	out := make([]string, len(defaultTrustedProxies))
	copy(out, defaultTrustedProxies)
	return out
}

// TrustedProxyList 解析逗号分隔的 IP 或 CIDR。空值使用默认网段。
func TrustedProxyList(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return DefaultTrustedProxies(), nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if net.ParseIP(part) == nil {
			if _, _, err := net.ParseCIDR(part); err != nil {
				return nil, fmt.Errorf("server.trusted_proxies 含无效网段 %q", part)
			}
		}
		out = append(out, part)
	}
	if len(out) == 0 {
		return DefaultTrustedProxies(), nil
	}
	return out, nil
}
