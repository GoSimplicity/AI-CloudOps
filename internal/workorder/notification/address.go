package notification

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode"
)

// NormalizeAddress validates an explicit channel without inferring one from text.
func NormalizeAddress(channel, address string) (string, error) {
	address = strings.TrimSpace(address)
	if address == "" || strings.IndexFunc(address, unicode.IsControl) >= 0 {
		return "", fmt.Errorf("接收地址为空或含控制字符")
	}
	switch channel {
	case "email":
		parsed, err := mail.ParseAddress(address)
		if err != nil || parsed.Name != "" || parsed.Address != address || len(address) > 254 || !strings.Contains(address, "@") {
			return "", fmt.Errorf("邮箱地址格式不合法")
		}
		// Addresses are deduplicated case-insensitively to avoid repeat delivery.
		return strings.ToLower(address), nil
	case "feishu":
		if len(address) > 100 || !(chatIDPattern.MatchString(address) || openIDPattern.MatchString(address) || unionIDPattern.MatchString(address)) {
			return "", fmt.Errorf("飞书地址必须为ou_/on_/oc_格式ID")
		}
		return address, nil
	default:
		return "", fmt.Errorf("不支持的通知渠道: %s", channel)
	}
}
