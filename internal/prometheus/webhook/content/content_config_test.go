package content

import (
	"context"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func TestContentUsesTypedConfigurationSnapshot(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("webhook.front_domain", "https://wrong.invalid")
	viper.Set("webhook.backend_domain", "https://wrong.invalid")
	viper.Set("webhook.default_upgrade_minutes", 99)
	viper.Set("webhook.im_feishu.group_message_api", "https://wrong.invalid/group")
	viper.Set("webhook.im_feishu.private_message_api", "https://wrong.invalid/private")
	cfg := &config.WebhookConfig{FrontDomain: "https://fixture.invalid/front", BackendDomain: "https://fixture.invalid/back", DefaultUpgradeMinutes: 17, ImFeishu: config.ImFeishuConfig{GroupMessageAPI: "https://fixture.invalid/group/ ", PrivateMessageAPI: "https://fixture.invalid/private"}}
	client := NewWebhookContent(cfg, zap.NewNop(), nil, fixturePrivateRobot{}).(*webhookContent)
	*cfg = config.WebhookConfig{}
	if client.frontDomain != "https://fixture.invalid/front" || client.backendDomain != "https://fixture.invalid/back" || client.defaultUpgradeMinutes != 17 {
		t.Fatal("typed card settings drifted")
	}
	if client.client.Timeout != 10*time.Second {
		t.Fatal("existing message timeout changed")
	}
	transport := &recordTransport{}
	client.client.Transport = transport
	if err := client.SentFeishuGroup(context.Background(), "{}", "fixture-group"); err != nil {
		t.Fatal(err)
	}
	if err := client.SentFeishuPrivate(context.Background(), "{}", map[string]string{"fixture-user": ""}); err != nil {
		t.Fatal(err)
	}
	if len(transport.urls) != 2 || transport.urls[0] != "https://fixture.invalid/group/fixture-group" || transport.urls[1] != "https://fixture.invalid/private?receive_id_type=user_id" {
		t.Fatal("typed message endpoints drifted")
	}
}
