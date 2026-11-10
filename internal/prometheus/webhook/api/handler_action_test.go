package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/action"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

const fixtureActionKey = "fixture-action-hmac"
const fixtureReceiveToken = "fixture-receive-auth"

func signActionRequest(t *testing.T, method, actionName, fingerprint, hour string) *http.Request {
	t.Helper()
	return signActionRequestAt(t, method, actionName, fingerprint, hour, time.Now().Add(time.Hour))
}

func signActionRequestAt(t *testing.T, method, actionName, fingerprint, hour string, expires time.Time) *http.Request {
	t.Helper()
	extra := ""
	if actionName == "silence" {
		extra = hour
	}
	sig, err := action.Sign(fixtureActionKey, actionName, fingerprint, extra, expires)
	if err != nil {
		t.Fatal(err)
	}
	query := url.Values{}
	query.Set("fingerprint", fingerprint)
	if hour != "" {
		query.Set("hour", hour)
	}
	query.Set("exp", strconv.FormatInt(expires.Unix(), 10))
	query.Set("sig", sig)
	return httptest.NewRequest(method, "/api/v1/alerts/"+actionName+"?"+query.Encode(), nil)
}

func withActionKey(cfg *config.WebhookConfig) *config.WebhookConfig {
	if cfg.ActionSigningKey == "" {
		cfg.ActionSigningKey = fixtureActionKey
	}
	return cfg
}

type countingDAO struct {
	dao.WebhookDao
	lookups int
}

func (counting *countingDAO) GetMonitorAlertEventByFingerprintId(context.Context, string) (*model.MonitorAlertEvent, error) {
	counting.lookups++
	event := model.MonitorAlertEvent{Model: model.Model{ID: 3}, Status: model.MonitorAlertEventStatusFiring, Labels: model.StringList{"alertname=fixture"}}
	return &event, nil
}

func TestSignedGetDoesNotMutateAndPostDoes(t *testing.T) {
	memory := &silenceMemoryDAO{event: model.MonitorAlertEvent{Model: model.Model{ID: 4}, Status: model.MonitorAlertEventStatusFiring, Labels: model.StringList{"alertname=fixture", "team=a=b"}}}
	handler, err := NewWebHookHandler(withActionKey(&config.WebhookConfig{AlertManagerAPI: "https://typed-alertmanager.invalid", ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}), zap.NewNop(), memory, nil)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	var posted []byte
	useSilenceTransport(t, func(request *http.Request) (*http.Response, error) {
		calls++
		posted, _ = io.ReadAll(request.Body)
		return fixtureResponse(request, http.StatusOK, `{"silenceID":"fixture-silence"}`), nil
	})
	engine := gin.New()
	handler.RegisterRouters(engine)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, signActionRequest(t, http.MethodGet, "silence", "fixture", "1"))
	if calls != 0 || memory.updates != 0 {
		t.Fatal("GET silence mutated state")
	}
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "<form") || !strings.Contains(recorder.Body.String(), `method="post"`) {
		t.Fatal("GET silence did not render a confirmation form")
	}
	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, signActionRequest(t, http.MethodPost, "silence", "fixture", "1"))
	if calls != 1 || memory.updates != 1 || memory.event.Status != model.MonitorAlertEventStatusSilenced {
		t.Fatal("POST silence did not apply")
	}
	if !strings.Contains(string(posted), `"Value":"a=b"`) && !strings.Contains(string(posted), `"value":"a=b"`) {
		t.Fatal("label value containing '=' was dropped")
	}
}

func TestInvalidOrExpiredSignatureIsRejectedWithoutLookup(t *testing.T) {
	counting := &countingDAO{}
	handler, err := NewWebHookHandler(withActionKey(&config.WebhookConfig{ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}), zap.NewNop(), counting, nil)
	if err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	handler.RegisterRouters(engine)
	expired := signActionRequestAt(t, http.MethodPost, "unsilence", "fixture", "", time.Now().Add(-time.Minute))
	tampered := signActionRequest(t, http.MethodGet, "renling", "fixture", "")
	tampered.URL.RawQuery = strings.Replace(tampered.URL.RawQuery, "sig=", "sig=00", 1)
	for _, request := range []*http.Request{expired, tampered, httptest.NewRequest(http.MethodPost, "/api/v1/alerts/silence?fingerprint=fixture&hour=1", nil)} {
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, request)
		if recorder.Code != http.StatusForbidden || counting.lookups != 0 {
			t.Fatalf("status %d lookups %d", recorder.Code, counting.lookups)
		}
	}
}

func TestMissingFingerprintStaysJSON200WithoutLookup(t *testing.T) {
	counting := &countingDAO{}
	handler, err := NewWebHookHandler(withActionKey(&config.WebhookConfig{ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}), zap.NewNop(), counting, nil)
	if err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	handler.RegisterRouters(engine)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/alerts/silence", nil))
	if recorder.Code != http.StatusOK || counting.lookups != 0 || !strings.Contains(recorder.Body.String(), "fingerprint") {
		t.Fatal("health probe contract changed")
	}
}

func TestBrokenLabelRejectsSilenceBeforeUpstream(t *testing.T) {
	memory := &silenceMemoryDAO{event: model.MonitorAlertEvent{Status: model.MonitorAlertEventStatusFiring, Labels: model.StringList{"alertname=fixture", "broken"}}}
	handler, err := NewWebHookHandler(withActionKey(&config.WebhookConfig{AlertManagerAPI: "https://typed-alertmanager.invalid", ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}), zap.NewNop(), memory, nil)
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	useSilenceTransport(t, func(request *http.Request) (*http.Response, error) {
		calls++
		return fixtureResponse(request, http.StatusOK, `{"silenceID":"fixture-silence"}`), nil
	})
	engine := gin.New()
	handler.RegisterRouters(engine)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, signActionRequest(t, http.MethodPost, "silence", "fixture", "1"))
	var body struct{ Code int }
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if calls != 0 || memory.updates != 0 || body.Code == 0 {
		t.Fatal("invalid label was silenced")
	}
}

func TestClaimUpdatesExistingStatusOnly(t *testing.T) {
	for _, initial := range []model.MonitorAlertEventStatus{model.MonitorAlertEventStatusFiring, model.MonitorAlertEventStatusUpgraded, model.MonitorAlertEventStatusClaimed, model.MonitorAlertEventStatusResolved, model.MonitorAlertEventStatusSilenced} {
		memory := &silenceMemoryDAO{event: model.MonitorAlertEvent{Model: model.Model{ID: 9}, Status: initial, SilenceID: "keep", RenLingUserID: 0, Labels: model.StringList{"alertname=fixture"}}}
		core, logs := observer.New(zap.DebugLevel)
		handler, err := NewWebHookHandler(withActionKey(&config.WebhookConfig{ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}), zap.New(core), memory, nil)
		if err != nil {
			t.Fatal(err)
		}
		engine := gin.New()
		handler.RegisterRouters(engine)
		recorder := httptest.NewRecorder()
		engine.ServeHTTP(recorder, signActionRequest(t, http.MethodGet, "renling", "fixture", ""))
		if memory.updates != 0 || !strings.Contains(recorder.Body.String(), "<form") {
			t.Fatal("GET claim mutated or skipped the confirmation page")
		}
		recorder = httptest.NewRecorder()
		engine.ServeHTTP(recorder, signActionRequest(t, http.MethodPost, "renling", "fixture", ""))
		var body struct {
			Code    int
			Message string
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
			t.Fatal(err)
		}
		switch initial {
		case model.MonitorAlertEventStatusFiring, model.MonitorAlertEventStatusUpgraded:
			if body.Code != 0 || memory.event.Status != model.MonitorAlertEventStatusClaimed || memory.event.RenLingUserID != 0 || memory.event.SilenceID != "keep" {
				t.Fatalf("claim did not keep the existing row: %+v code %d", memory.event, body.Code)
			}
		case model.MonitorAlertEventStatusClaimed:
			if body.Code != 0 || memory.updates != 0 {
				t.Fatal("repeat claim wrote again")
			}
		default:
			if body.Code == 0 || memory.updates != 0 || memory.event.Status != initial {
				t.Fatal("claim changed a status it should leave alone")
			}
		}
		for _, entry := range logs.All() {
			blob := entry.Message
			for key, value := range entry.ContextMap() {
				blob += key + stringify(value)
			}
			if strings.Contains(blob, fixtureActionKey) {
				t.Fatal("log contained the signing key")
			}
		}
	}
}

func stringify(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return typed
	default:
		return ""
	}
}

func TestReceiveBearerToken(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	handler, err := NewWebHookHandler(&config.WebhookConfig{AlertmanagerBearerToken: fixtureReceiveToken, ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}, zap.New(core), nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	engine := gin.New()
	handler.RegisterRouters(engine)
	denied := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/v1/alerts/receive", strings.NewReader(`{"alerts":[]}`))
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(denied, request)
	if denied.Code != http.StatusUnauthorized || strings.Contains(denied.Body.String(), fixtureReceiveToken) {
		t.Fatal("missing bearer was accepted or echoed the token")
	}
	allowed := httptest.NewRecorder()
	request = httptest.NewRequest(http.MethodPost, "/api/v1/alerts/receive", strings.NewReader(`not-json`))
	request.Header.Set("Authorization", "Bearer "+fixtureReceiveToken)
	request.Header.Set("Content-Type", "application/json")
	engine.ServeHTTP(allowed, request)
	if allowed.Code == http.StatusUnauthorized {
		t.Fatal("matching bearer was rejected")
	}
	for _, entry := range logs.All() {
		blob := entry.Message
		for _, value := range entry.ContextMap() {
			if text, ok := value.(string); ok {
				blob += text
			}
		}
		if strings.Contains(blob, fixtureReceiveToken) {
			t.Fatal("log contained the bearer token")
		}
	}
}

func TestStartupWarnsWhenActionSecretsUnset(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	if _, err := NewWebHookHandler(&config.WebhookConfig{ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}, zap.New(core), nil, nil); err != nil {
		t.Fatal(err)
	}
	var sawSign, sawBearer bool
	for _, entry := range logs.All() {
		if strings.Contains(entry.Message, "WEBHOOK_ACTION_SIGNING_KEY") {
			sawSign = true
		}
		if strings.Contains(entry.Message, "WEBHOOK_ALERTMANAGER_BEARER_TOKEN") {
			sawBearer = true
		}
	}
	if !sawSign || !sawBearer {
		t.Fatal("missing startup warnings")
	}
}
