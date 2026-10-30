package robot

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type tokenPrivacyTransport func(*http.Request) (*http.Response, error)

func (transport tokenPrivacyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

type tokenPrivateRead struct{}

func (*tokenPrivateRead) Read([]byte) (int, error) { return 0, errors.New("TOKEN_PRIVATE") }
func (*tokenPrivateRead) Close() error             { return nil }

func TestTokenFailureChainIsPrivate(t *testing.T) {
	const canary = "TOKEN_PRIVATE"
	for _, entrypoint := range []string{"direct", "refresh", "tenant"} {
		t.Run(entrypoint, func(t *testing.T) {
			for _, mode := range []string{"build", "transport", "read", "http", "decode", "provider", "empty-token", "blank-token", "zero-expiry", "negative-expiry", "overflow-expiry"} {
				t.Run(mode, func(t *testing.T) {
					viper.Reset()
					t.Cleanup(viper.Reset)
					viper.Set("webhook.im_feishu.private_robot_app_id", canary)
					viper.Set("webhook.im_feishu.private_robot_app_secret", canary)
					endpoint := "https://" + canary + ":" + canary + "@fixture.invalid/token?private=" + canary
					if mode == "build" {
						endpoint = "https://fixture.invalid/%zz"
					}
					viper.Set("webhook.im_feishu.tenant_access_token_api", endpoint)
					core, logs := observer.New(zap.DebugLevel)
					calls := 0
					client := &http.Client{Timeout: 7 * time.Second, Transport: tokenPrivacyTransport(func(request *http.Request) (*http.Response, error) {
						calls++
						if request.Method != "POST" {
							t.Fatal("token HTTP method changed")
						}
						deadline, ok := request.Context().Deadline()
						if !ok || time.Until(deadline) > 7*time.Second {
							t.Fatal("token timeout was lost")
						}
						if mode == "transport" {
							return nil, errors.New(canary)
						}
						status := 200
						body := `{"code":0,"tenant_access_token":"` + canary + `","expire":7200}`
						switch mode {
						case "http":
							status = 503
							body = canary
						case "decode":
							body = canary
						case "provider":
							body = `{"code":777,"msg":"` + canary + `"}`
						case "empty-token":
							body = `{"code":0,"tenant_access_token":"","expire":7200}`
						case "blank-token":
							body = `{"code":0,"tenant_access_token":"  ","expire":7200}`
						case "zero-expiry":
							body = `{"code":0,"tenant_access_token":"` + canary + `","expire":0}`
						case "negative-expiry":
							body = `{"code":0,"tenant_access_token":"` + canary + `","expire":-1}`
						case "overflow-expiry":
							body = `{"code":0,"tenant_access_token":"` + canary + `","expire":9223372036854775807}`
						}
						var reader io.ReadCloser = io.NopCloser(strings.NewReader(body))
						if mode == "read" {
							reader = &tokenPrivateRead{}
						}
						return &http.Response{StatusCode: status, Status: canary, Body: reader, Header: make(http.Header), Request: request}, nil
					})}
					robot := &webhookRobot{logger: zap.New(core), httpClient: client, privateRobotToken: "previous-private", tenantToken: "previous-tenant", tenantExpireAt: time.Unix(1, 0)}
					var err error
					switch entrypoint {
					case "direct":
						_, _, err = robot.getTokenFromAPI(context.Background())
					case "refresh":
						robot.RefreshPrivateRobotToken(context.Background())
					case "tenant":
						_, err = robot.GetTenantAccessToken(context.Background())
					}
					if entrypoint != "refresh" && (err == nil || strings.Contains(err.Error(), canary)) {
						t.Fatal("private token error escaped or failure was accepted")
					}
					expected := 1
					if mode == "build" {
						expected = 0
					}
					if calls != expected {
						t.Fatal("token failure retried")
					}
					if robot.privateRobotToken != "previous-private" || robot.tenantToken != "previous-tenant" || !robot.tenantExpireAt.Equal(time.Unix(1, 0)) {
						t.Fatal("unusable provider result entered the token cache")
					}
					for _, entry := range logs.All() {
						if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), canary) {
							t.Fatal("token, secret, provider message or URL leaked")
						}
					}
				})
			}
		})
	}
}

func TestConcurrentTenantTokenCacheRemainsSingleFetch(t *testing.T) {
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("webhook.im_feishu.private_robot_app_id", "fixture-id")
	viper.Set("webhook.im_feishu.private_robot_app_secret", "fixture-secret")
	viper.Set("webhook.im_feishu.tenant_access_token_api", "https://fixture.invalid/token")
	var calls atomic.Int32
	robot := &webhookRobot{logger: zap.NewNop(), httpClient: &http.Client{Transport: tokenPrivacyTransport(func(request *http.Request) (*http.Response, error) {
		calls.Add(1)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0,"tenant_access_token":"fixture-token","expire":7200}`)), Header: make(http.Header), Request: request}, nil
	})}}
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 16; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			token, err := robot.GetTenantAccessToken(context.Background())
			if err != nil || token != "fixture-token" {
				t.Error("concurrent token cache failed")
			}
		}()
	}
	close(start)
	wg.Wait()
	if calls.Load() != 1 {
		t.Fatal("tenant cache performed duplicate concurrent fetches")
	}
	for i := 0; i < 4; i++ {
		robot.RefreshPrivateRobotToken(context.Background())
	}
	if calls.Load() != 5 || robot.GetPrivateRobotToken() != "fixture-token" {
		t.Fatal("valid private polling/cache contract changed")
	}
}
