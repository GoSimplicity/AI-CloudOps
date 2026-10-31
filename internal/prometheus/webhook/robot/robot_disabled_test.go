package robot

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type forbiddenTokenTransport struct{ calls int }

func (transport *forbiddenTokenTransport) RoundTrip(*http.Request) (*http.Response, error) {
	transport.calls++
	return nil, errors.New("unexpected fixture HTTP")
}

func TestPrivateRobotMissingCredentialsDoNotRequestToken(t *testing.T) {
	for _, test := range []struct{ name, id, secret string }{{"both empty", "", ""}, {"ID only", "fixture-id", ""}, {"secret only", "", "fixture-secret"}, {"blank ID", " \t", "fixture-secret"}, {"blank secret", "fixture-id", " \n"}} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.WebhookConfig{}
			cfg.ImFeishu.PrivateRobotAppID = test.id
			cfg.ImFeishu.PrivateRobotAppSecret = test.secret
			transport := &forbiddenTokenTransport{}
			core, logs := observer.New(zap.DebugLevel)
			robot := &webhookRobot{appID: cfg.ImFeishu.PrivateRobotAppID, appSecret: cfg.ImFeishu.PrivateRobotAppSecret, tokenAPI: cfg.ImFeishu.TenantAccessTokenAPI, logger: zap.New(core), httpClient: &http.Client{Transport: transport}}
			if _, _, err := robot.getTokenFromAPI(context.Background()); !errors.Is(err, ErrPrivateRobotDisabled) {
				t.Fatal("missing credential configuration was not classified as disabled")
			}
			for i := 0; i < 3; i++ {
				robot.RefreshPrivateRobotToken(context.Background())
			}
			if transport.calls != 0 || logs.Len() != 0 {
				t.Fatal("disabled scheduled refresh attempted HTTP or emitted repeated errors")
			}
		})
	}
}
