package config

import (
	"crypto/sha256"
	"crypto/subtle"
	"fmt"
	"net"
	"strings"
	"unicode"

	"github.com/GoSimplicity/AI-CloudOps/pkg/envmode"
)

// maxHTTPServiceDiscoveryTokenLength 限制写入 Authorization 头和 Prometheus 配置的令牌长度。
const maxHTTPServiceDiscoveryTokenLength = 512

// validateHTTPServiceDiscoveryToken 空值只允许开发环境（development/dev）。
// 生产别名以及未设置、空白、test 等无法识别的取值都按 envmode.IsProduction 拒绝启动。
// 非空值在所有环境都拒绝空白、控制字符和超长值。错误信息只含变量名。
func validateHTTPServiceDiscoveryToken(value string) error {
	if value == "" {
		if envmode.IsProduction() {
			return fmt.Errorf("PROMETHEUS_HTTP_SD_TOKEN 未配置：生产环境及无法识别的环境拒绝启动")
		}
		return nil
	}
	if strings.IndexFunc(value, unicode.IsSpace) >= 0 || hasControlChar(value) {
		return fmt.Errorf("PROMETHEUS_HTTP_SD_TOKEN 不能包含空白、控制字符或不可见格式字符")
	}
	if len(value) > maxHTTPServiceDiscoveryTokenLength {
		return fmt.Errorf("PROMETHEUS_HTTP_SD_TOKEN 长度不能超过%d字节", maxHTTPServiceDiscoveryTokenLength)
	}
	return nil
}

// UsableHTTPServiceDiscoveryToken 返回可以参与校验和配置生成的令牌。
// 空值或未通过校验的值都视为未配置，调用方不得把原始值写入请求头、日志或抓取配置。
func (c PrometheusConfig) UsableHTTPServiceDiscoveryToken() string {
	if validateHTTPServiceDiscoveryToken(c.HttpSdToken) != nil {
		return ""
	}
	return c.HttpSdToken
}

// HTTPServiceDiscoveryStartup 返回启动日志。warn 为 true 时表示仍处于仅本机回环的模式。
// 文案不包含令牌本身。
func (c PrometheusConfig) HTTPServiceDiscoveryStartup() (warn bool, message string) {
	if c.HttpSdToken == "" {
		if envmode.IsProduction() {
			return true, "[SECURITY] PROMETHEUS_HTTP_SD_TOKEN 未配置，生产环境及无法识别的环境拒绝启动。令牌缺失时 HTTP-SD 只接受本机回环直连。升级后必须重新生成一次 Prometheus 配置，否则 http_sd 会返回 401。"
		}
		return true, "[SECURITY] PROMETHEUS_HTTP_SD_TOKEN 未配置，当前为开发环境，HTTP-SD 仅接受本机回环地址的直连；经反向代理或其他主机的请求会被拒绝。升级后必须重新生成一次 Prometheus 配置，否则 http_sd 会返回 401。"
	}
	if validateHTTPServiceDiscoveryToken(c.HttpSdToken) != nil {
		return true, "[SECURITY] PROMETHEUS_HTTP_SD_TOKEN 无效，已按未配置处理，HTTP-SD 仅接受本机回环地址的直连。请改为不含空白和控制字符、且不超过512字节的令牌。"
	}
	return false, "HTTP-SD 已启用 bearer token 校验"
}

// HTTPServiceDiscoveryRequestAllowed 判断 HTTP-SD 请求是否可以继续。
// 令牌为空时只看 TCP 对端是否为回环地址，不读取 X-Forwarded-For 或 X-Real-IP。
// 令牌已配置时，回环地址同样必须携带正确的 bearer token。
func HTTPServiceDiscoveryRequestAllowed(token, remoteAddr, authorizationHeader string) bool {
	if token == "" {
		return loopbackPeer(remoteAddr)
	}
	return bearerEquals(token, authorizationHeader)
}

func loopbackPeer(remoteAddr string) bool {
	host := remoteAddr
	if parsed, _, err := net.SplitHostPort(remoteAddr); err == nil {
		host = parsed
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}

func bearerCredential(header string) (string, bool) {
	if len(header) < 7 || len(header) > maxHTTPServiceDiscoveryTokenLength+16 || !strings.EqualFold(header[:6], "bearer") || header[6] != ' ' {
		return "", false
	}
	credential := strings.TrimSpace(header[7:])
	if credential == "" || len(credential) > maxHTTPServiceDiscoveryTokenLength || strings.TrimSpace(credential) != credential || hasControlChar(credential) {
		return "", false
	}
	return credential, true
}

func bearerEquals(configured, header string) bool {
	provided, ok := bearerCredential(header)
	if !ok {
		// 仍然做一次等长摘要比较，避免缺少令牌和错误令牌在比较步骤上提前返回。
		provided = configured + "\x00"
	}
	match := tokenEquals(configured, provided)
	if !ok {
		return false
	}
	return match
}

func tokenEquals(configured, provided string) bool {
	sumConfigured := sha256.Sum256([]byte(configured))
	sumProvided := sha256.Sum256([]byte(provided))
	return subtle.ConstantTimeCompare(sumConfigured[:], sumProvided[:]) == 1
}
