package content

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	robotAPI "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/robot"
	"github.com/prometheus/alertmanager/template"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type messagePrivacyTransport func(*http.Request) (*http.Response, error)

func (transport messagePrivacyTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}

type messagePrivateRead struct{}

func (*messagePrivateRead) Read([]byte) (int, error) { return 0, errors.New("MESSAGE_PRIVATE") }
func (*messagePrivateRead) Close() error             { return nil }

type privateTokenFixture struct {
	robotAPI.WebhookRobot
	calls int
	token string
}

func (robot *privateTokenFixture) GetPrivateRobotToken() string { robot.calls++; return robot.token }

func TestGroupFailureLogsAndErrorsArePrivate(t *testing.T) {
	const canary = "MESSAGE_PRIVATE"
	for _, mode := range []string{"missing-token", "transport", "http", "read", "provider", "decode", "missing-code", "success"} {
		t.Run(mode, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("webhook.im_feishu.group_message_api", "https://"+canary+":"+canary+"@fixture.invalid/hook")
			core, logs := observer.New(zap.DebugLevel)
			calls := 0
			client := &http.Client{Timeout: 7 * time.Second, Transport: messagePrivacyTransport(func(request *http.Request) (*http.Response, error) {
				calls++
				if request.Method != "POST" {
					t.Fatal("message method changed")
				}
				if mode == "transport" {
					return nil, errors.New(canary)
				}
				body := `{"code":0}`
				status := 200
				switch mode {
				case "http":
					status = 503
					body = canary
				case "provider":
					body = `{"code":777,"msg":"` + canary + `"}`
				case "decode":
					body = canary
				case "missing-code":
					body = `{}`
				}
				var reader io.ReadCloser = io.NopCloser(strings.NewReader(body))
				if mode == "read" {
					reader = &messagePrivateRead{}
				}
				return &http.Response{StatusCode: status, Status: canary, Body: reader, Header: make(http.Header), Request: request}, nil
			})}
			content := &webhookContent{l: zap.New(core), client: client}
			token := canary
			if mode == "missing-token" {
				token = " \t"
			}
			err := content.SentFeishuGroup(context.Background(), canary, token)
			expectedCalls := 1
			if mode == "missing-token" {
				expectedCalls = 0
			}
			if calls != expectedCalls {
				t.Fatal("group failure retried or unconfigured group called HTTP")
			}
			if (err == nil) != (mode == "success") || (err != nil && strings.Contains(err.Error(), canary)) {
				t.Fatal("group provider failure accepted or leaked private data")
			}
			for _, entry := range logs.All() {
				if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), canary) {
					t.Fatal("group log exposed token, body, URL or provider message")
				}
			}
		})
	}
}

func TestPrivateDeliveryAttemptsAllRecipientsAndReturnsPartial(t *testing.T) {
	const canary = "MESSAGE_PRIVATE"
	for _, mode := range []string{"transport", "http", "provider", "decode"} {
		t.Run(mode, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("webhook.im_feishu.private_message_api", "https://"+canary+":"+canary+"@fixture.invalid/messages?private="+canary)
			core, logs := observer.New(zap.DebugLevel)
			seen := map[string]int{}
			token := &privateTokenFixture{token: canary}
			content := &webhookContent{l: zap.New(core), robot: token, client: &http.Client{Transport: messagePrivacyTransport(func(request *http.Request) (*http.Response, error) {
				var payload struct {
					ReceiveID string `json:"receive_id"`
				}
				if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				seen[payload.ReceiveID]++
				if request.Method != "POST" || request.Header.Get("Authorization") != "Bearer "+canary || request.URL.Query().Get("receive_id_type") != "user_id" {
					t.Fatal("private request contract changed")
				}
				status := 200
				body := `{"code":0}`
				if payload.ReceiveID == canary+"2" {
					switch mode {
					case "transport":
						return nil, errors.New(canary)
					case "http":
						status = 503
						body = canary
					case "provider":
						body = `{"code":777,"msg":"` + canary + `"}`
					case "decode":
						body = canary
					}
				}
				return &http.Response{StatusCode: status, Status: canary, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}, nil
			})}}
			err := content.SentFeishuPrivate(context.Background(), canary, map[string]string{canary + "1": "", canary + "2": "", canary + "3": ""})
			if !errors.Is(err, ErrPartialDelivery) || strings.Contains(err.Error(), canary) {
				t.Fatal("partial private failure was hidden or leaked")
			}
			if len(seen) != 3 || token.calls != 3 {
				t.Fatal("private failure skipped another recipient or changed per-recipient cached token reads")
			}
			for _, count := range seen {
				if count != 1 {
					t.Fatal("private recipient was retried")
				}
			}
			for _, entry := range logs.All() {
				if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), canary) {
					t.Fatal("private notification log leaked data")
				}
			}
		})
	}
}

func TestEmptyPrivateRecipientsDoNotReadToken(t *testing.T) {
	content := &webhookContent{l: zap.NewNop()}
	if err := content.SentFeishuPrivate(context.Background(), "{}", nil); err != nil {
		t.Fatal(err)
	}
}

type notificationDutyFixture struct {
	dao.WebhookDao
	withUser bool
}

func (fixture notificationDutyFixture) GetOnDutyGroupById(context.Context, int) (*model.MonitorOnDutyGroup, error) {
	group := &model.MonitorOnDutyGroup{Model: model.Model{ID: 1}, Name: "fixture"}
	if fixture.withUser {
		group.TodayDutyUser = &model.User{RealName: "fixture", FeiShuUserId: "MESSAGE_PRIVATE"}
	}
	return group, nil
}
func (notificationDutyFixture) FillTodayOnDutyUser(_ context.Context, group *model.MonitorOnDutyGroup) (*model.MonitorOnDutyGroup, error) {
	return group, nil
}

func TestNotificationChannelsAreIndependentAndAggregateFailures(t *testing.T) {
	const canary = "MESSAGE_PRIVATE"
	for _, test := range []struct {
		name                                   string
		private, group, privateFail, groupFail bool
	}{{"all success", true, true, false, false}, {"private fails group still attempted", true, true, true, false}, {"group fails after private", true, true, false, true}, {"both fail", true, true, true, true}, {"group only", false, true, false, false}, {"private only", true, false, false, false}, {"no configured channel", false, false, false, false}} {
		t.Run(test.name, func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("webhook.front_domain", "https://fixture.invalid")
			viper.Set("webhook.backend_domain", "https://fixture.invalid")
			viper.Set("webhook.im_feishu.private_message_api", "https://fixture.invalid/private")
			viper.Set("webhook.im_feishu.group_message_api", "https://fixture.invalid/group")
			core, logs := observer.New(zap.DebugLevel)
			token := &privateTokenFixture{token: canary}
			privateCalls, groupCalls := 0, 0
			content := &webhookContent{l: zap.New(core), dao: notificationDutyFixture{withUser: test.private}, robot: token, client: &http.Client{Transport: messagePrivacyTransport(func(request *http.Request) (*http.Response, error) {
				failure := false
				if request.URL.Path == "/private" {
					privateCalls++
					if groupCalls != 0 {
						t.Fatal("successful channel order changed")
					}
					failure = test.privateFail
				} else {
					groupCalls++
					failure = test.groupFail
				}
				body := `{"code":0}`
				if failure {
					body = `{"code":777,"msg":"` + canary + `"}`
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}, nil
			})}}
			groupToken := ""
			if test.group {
				groupToken = canary
			}
			err := content.GenerateFeishuCardContentOneAlert(context.Background(), template.Alert{Status: "firing", StartsAt: time.Now(), Labels: template.KV{"alertname": "fixture", "severity": "critical"}}, &model.MonitorAlertEvent{EventTimes: 1, Status: model.MonitorAlertEventStatusFiring}, nil, &model.MonitorSendGroup{Model: model.Model{ID: 3}, Name: "fixture", OnDutyGroupID: 1, FeiShuQunRobotToken: groupToken})
			wantPrivate, wantGroup := 0, 0
			if test.private {
				wantPrivate = 1
			}
			if test.group {
				wantGroup = 1
			}
			if privateCalls != wantPrivate || groupCalls != wantGroup {
				t.Fatal("partial failure skipped or retried a configured channel")
			}
			if (err != nil) != (test.privateFail || test.groupFail) {
				t.Fatal("channel failure aggregation changed")
			}
			if !test.private && token.calls != 0 {
				t.Fatal("group-only alert read private robot token")
			}
			if err != nil && strings.Contains(err.Error(), canary) {
				t.Fatal("notification error exposed a token")
			}
			for _, entry := range logs.All() {
				if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), canary) {
					t.Fatal("outer notification log exposed token/user/body")
				}
			}
		})
	}
}
