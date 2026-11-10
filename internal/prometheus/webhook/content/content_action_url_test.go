package content

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/action"
	"github.com/prometheus/alertmanager/template"
	"go.uber.org/zap"
)

func TestJoinPublicURLDoesNotDoubleScheme(t *testing.T) {
	cases := []struct{ base, elem, want string }{
		{"http://localhost:8889/api/v1/alerts", "silence", "http://localhost:8889/api/v1/alerts/silence"},
		{"http://localhost:8889/api/v1/alerts/", "renling", "http://localhost:8889/api/v1/alerts/renling"},
		{"https://alerts.example/hook/api/v1/alerts", "unsilence", "https://alerts.example/hook/api/v1/alerts/unsilence"},
		{"localhost:8889/api/v1/alerts", "silence", "http://localhost:8889/api/v1/alerts/silence"},
		{"http://localhost:3000", "monitor/onduty/detail", "http://localhost:3000/monitor/onduty/detail"},
	}
	for _, tc := range cases {
		got, err := joinPublicURL(tc.base, tc.elem)
		if err != nil {
			t.Fatal(err)
		}
		if got != tc.want || strings.Contains(got, "http://http://") || strings.Contains(got, "https://http://") {
			t.Fatalf("base %s -> %s", tc.base, got)
		}
	}
	if _, err := joinPublicURL("  ", "silence"); err == nil {
		t.Fatal("empty base accepted")
	}
}

func TestCardLinksUseConfiguredBaseAndSignature(t *testing.T) {
	const key = "fixture-action-hmac"
	wc := &webhookContent{
		l:             zap.NewNop(),
		frontDomain:   "http://localhost:3000",
		backendDomain: "http://localhost:8889/api/v1/alerts",
		signingKey:    key,
		dao:           notificationDutyFixture{},
		robot:         fixturePrivateRobot{},
		client:        &http.Client{Transport: &recordTransport{}},
	}
	issued := time.Now()
	raw, err := wc.signedActionURL("silence", "fp-1", "6", issued)
	if err != nil {
		t.Fatal(err)
	}
	parsed, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if parsed.Scheme != "http" || parsed.Host != "localhost:8889" || parsed.Path != "/api/v1/alerts/silence" {
		t.Fatalf("link %s", raw)
	}
	if strings.Contains(raw, "http://http://") {
		t.Fatal(raw)
	}
	if err := action.Verify(key, "silence", parsed.Query().Get("fingerprint"), parsed.Query().Get("hour"), parsed.Query().Get("exp"), parsed.Query().Get("sig"), issued.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}
	front, err := wc.frontURL("monitor/sendgroup/detail", url.Values{"id": {"3"}})
	if err != nil || front != "http://localhost:3000/monitor/sendgroup/detail?id=3" {
		t.Fatalf("front link %s %v", front, err)
	}

	var body string
	wc.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		payload, _ := io.ReadAll(request.Body)
		body = string(payload)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0}`)), Header: make(http.Header), Request: request}, nil
	})}
	err = wc.GenerateFeishuCardContentOneAlert(context.Background(), template.Alert{
		Fingerprint: "fp-1", Status: "firing", StartsAt: time.Now(),
		Labels: template.KV{"alertname": "fixture", "severity": "critical"},
	}, &model.MonitorAlertEvent{EventTimes: 1, Status: model.MonitorAlertEventStatusFiring}, nil, &model.MonitorSendGroup{Model: model.Model{ID: 3}, Name: "fixture", OnDutyGroupID: 1, FeiShuQunRobotToken: "fixture-group"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(body, "http://localhost:8889/api/v1/alerts/renling?") || strings.Contains(body, "http://http://") || !strings.Contains(body, "sig=") {
		t.Fatal("card links were not signed against the configured base")
	}
	if !strings.Contains(body, "http://localhost:3000/monitor/sendgroup/detail?id=3") {
		t.Fatal("front link dropped its scheme or doubled it")
	}
}

func TestCardWithoutSigningKeyOmitsActionButtons(t *testing.T) {
	wc := &webhookContent{
		l:             zap.NewNop(),
		frontDomain:   "http://localhost:3000",
		backendDomain: "http://localhost:8889/api/v1/alerts",
		dao:           notificationDutyFixture{},
		robot:         fixturePrivateRobot{},
		client:        &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) { return nil, nil })},
	}
	var body string
	wc.client = &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		payload, _ := io.ReadAll(request.Body)
		body = string(payload)
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"code":0}`)), Header: make(http.Header), Request: request}, nil
	})}
	err := wc.GenerateFeishuCardContentOneAlert(context.Background(), template.Alert{
		Fingerprint: "fp-1", Status: "firing", StartsAt: time.Now(),
		Labels: template.KV{"alertname": "fixture", "severity": "critical"},
	}, &model.MonitorAlertEvent{EventTimes: 1, Status: model.MonitorAlertEventStatusFiring}, nil, &model.MonitorSendGroup{Model: model.Model{ID: 3}, Name: "fixture", OnDutyGroupID: 1, FeiShuQunRobotToken: "fixture-group"})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(body, "认领告警") || strings.Contains(body, "屏蔽1小时") || strings.Contains(body, "/renling?") {
		t.Fatal("unsigned card still exposed action buttons")
	}
	if !strings.Contains(body, "未配置签名密钥") {
		t.Fatal("unsigned card did not explain why buttons are absent")
	}
}

func TestContentKeepsSigningKeySnapshot(t *testing.T) {
	cfg := &config.WebhookConfig{ActionSigningKey: " fixture-action-hmac ", FrontDomain: "http://localhost:3000", BackendDomain: "http://localhost:8889/api/v1/alerts", ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}
	client := NewWebhookContent(cfg, zap.NewNop(), nil, fixturePrivateRobot{}).(*webhookContent)
	cfg.ActionSigningKey = ""
	if client.signingKey != "fixture-action-hmac" {
		t.Fatal("signing key was not snapshotted")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return fn(request) }
