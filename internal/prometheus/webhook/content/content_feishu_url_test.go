package content

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// recordTransport 记录请求 URL 并返回 200，不发出任何网络请求
type recordTransport struct{ urls []string }

func (r *recordTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	r.urls = append(r.urls, req.URL.String())
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"code":0}`)), Header: http.Header{}, Request: req}, nil
}

// 第 18 点：群聊 URL 按 event_service.go 的 .../open-apis/bot/v2/hook/<token> 拼接，配置值带不带尾斜杠结果相同
func TestSentFeishuGroupURL(t *testing.T) {
	const want = "https://open.feishu.cn/open-apis/bot/v2/hook/tok123"
	for _, c := range []struct{ name, api string }{
		{"no-trailing-slash", "https://open.feishu.cn/open-apis/bot/v2/hook"},
		{"trailing-slash", "https://open.feishu.cn/open-apis/bot/v2/hook/"},
		{"double-trailing-slash", "https://open.feishu.cn/open-apis/bot/v2/hook//"},
	} {
		t.Run(c.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("webhook.im_feishu.group_message_api", c.api)
			rt := &recordTransport{}
			wc := &webhookContent{l: zap.NewNop(), client: &http.Client{Transport: rt}}
			if err := wc.SentFeishuGroup(context.Background(), "{}", "tok123"); err != nil {
				t.Fatal(err)
			}
			if len(rt.urls) != 1 || rt.urls[0] != want {
				t.Fatalf("got %v, want %s", rt.urls, want)
			}
		})
	}
}
