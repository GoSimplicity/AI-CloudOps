package cache

import "testing"

// 第三轮 J-6：scheme 判断不区分大小写（RFC 3986 scheme 大小写不敏感），HTTPS://h/x 不能拼成 http://HTTPS://…
func TestBuildWebhookURLSchemeCaseInsensitive(t *testing.T) {
	for _, c := range []struct{ name, addr, want string }{
		{"upper-https", "HTTPS://h/x", "HTTPS://h/x/api/v1/alerts/receive?send_group_id=3"},
		{"mixed-http-full-path", "Http://h:8889/api/v1/alerts/receive", "Http://h:8889/api/v1/alerts/receive?send_group_id=3"},
		{"lower-https-trailing-slash", "https://h/x/", "https://h/x/api/v1/alerts/receive?send_group_id=3"},
		{"host-port", "h:8889", "http://h:8889/api/v1/alerts/receive?send_group_id=3"},
		{"short-addr", "HTTP", "http://HTTP/api/v1/alerts/receive?send_group_id=3"},
		{"long-host-port-no-scheme", "alertwebhook.example:8889", "http://alertwebhook.example:8889/api/v1/alerts/receive?send_group_id=3"},
	} {
		t.Run(c.name, func(t *testing.T) {
			a := &alertManagerConfigCache{alertWebhookAddr: c.addr}
			if got := a.buildWebhookURL(3); got != c.want {
				t.Fatalf("got %q want %q", got, c.want)
			}
		})
	}
}
