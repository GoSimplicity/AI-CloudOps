package websocket

import "strings"

const Protocol = "cloudops-v1"
const CredentialProtocolPrefix = "cloudops-auth."

// TokenFromProtocols 仅提取唯一认证子协议，避免令牌出现在请求 URL
func TokenFromProtocols(header string) string {
	token := ""
	found := false
	for _, value := range strings.Split(header, ",") {
		value = strings.TrimSpace(value)
		if !strings.HasPrefix(value, CredentialProtocolPrefix) {
			continue
		}
		if found {
			return ""
		}
		found = true
		token = strings.TrimPrefix(value, CredentialProtocolPrefix)
	}
	return token
}
