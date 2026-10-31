package content

import (
	"context"
	"net/http"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"go.uber.org/zap"
)

// 第三轮 P3-7/J-2：群消息地址带首尾空白时先去空白再去尾斜杠，不能拼出 hook/ /<token>
func TestSentFeishuGroupURLTrimsSpace(t *testing.T) {
	const want = "https://open.feishu.cn/open-apis/bot/v2/hook/tok123"
	for _, c := range []struct{ name, api string }{
		{"trailing-slash-space", "https://open.feishu.cn/open-apis/bot/v2/hook/ "},
		{"tab-and-newline", "\thttps://open.feishu.cn/open-apis/bot/v2/hook/\n"},
	} {
		t.Run(c.name, func(t *testing.T) {
			cfg := &config.WebhookConfig{}
			cfg.ImFeishu.GroupMessageAPI = c.api
			rt := &recordTransport{}
			wc := &webhookContent{frontDomain: cfg.FrontDomain, backendDomain: cfg.BackendDomain, groupMessageAPI: cfg.ImFeishu.GroupMessageAPI, privateMessageAPI: cfg.ImFeishu.PrivateMessageAPI, l: zap.NewNop(), client: &http.Client{Transport: rt}}
			if err := wc.SentFeishuGroup(context.Background(), "{}", "tok123"); err != nil {
				t.Fatalf("不应出错: %v", err)
			}
			if len(rt.urls) != 1 || rt.urls[0] != want {
				t.Fatalf("got %v, want %s", rt.urls, want)
			}
		})
	}
}
