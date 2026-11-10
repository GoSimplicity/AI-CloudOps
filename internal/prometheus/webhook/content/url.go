package content

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/action"
)

// joinPublicURL 把已配置的对外地址和路径段拼在一起。
// 配置值应包含 scheme 和前缀。没有 scheme 时只补一次 http://，已经带 scheme 时不再加，避免 http://http://。
func joinPublicURL(base, elem string) (string, error) {
	base = strings.TrimSpace(base)
	if base == "" {
		return "", fmt.Errorf("public url is empty")
	}
	lower := strings.ToLower(base)
	if !strings.HasPrefix(lower, "http://") && !strings.HasPrefix(lower, "https://") {
		base = "http://" + strings.TrimLeft(base, "/")
	}
	joined, err := url.JoinPath(base, elem)
	if err != nil {
		return "", err
	}
	folded := strings.ToLower(joined)
	for _, duplicated := range []string{"http://http://", "http://https://", "https://http://", "https://https://"} {
		if strings.Contains(folded, duplicated) {
			return "", fmt.Errorf("public url duplicated scheme")
		}
	}
	return joined, nil
}

func withQuery(raw string, query url.Values) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func (wc *webhookContent) signedActionURL(actionName, fingerprint, hour string, now time.Time) (string, error) {
	raw, err := joinPublicURL(wc.backendDomain, actionName)
	if err != nil {
		return "", err
	}
	query := url.Values{}
	query.Set("fingerprint", fingerprint)
	extra := ""
	if actionName == "silence" {
		extra = hour
		query.Set("hour", hour)
	}
	if strings.TrimSpace(wc.signingKey) != "" {
		expires := now.Add(action.LinkTTL)
		sig, err := action.Sign(wc.signingKey, actionName, fingerprint, extra, expires)
		if err != nil {
			return "", err
		}
		query.Set("exp", strconv.FormatInt(expires.Unix(), 10))
		query.Set("sig", sig)
	}
	return withQuery(raw, query)
}

func (wc *webhookContent) frontURL(elem string, query url.Values) (string, error) {
	raw, err := joinPublicURL(wc.frontDomain, elem)
	if err != nil {
		return "", err
	}
	return withQuery(raw, query)
}
