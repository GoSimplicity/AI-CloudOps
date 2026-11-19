package api

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/gin-gonic/gin"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type silenceMemoryDAO struct {
	dao.WebhookDao
	event   model.MonitorAlertEvent
	updates int
}

func (memory *silenceMemoryDAO) GetMonitorAlertEventByFingerprintId(context.Context, string) (*model.MonitorAlertEvent, error) {
	event := memory.event
	return &event, nil
}
func (memory *silenceMemoryDAO) UpdateMonitorAlertEvent(_ context.Context, event *model.MonitorAlertEvent) error {
	memory.updates++
	memory.event = *event
	return nil
}

type silenceTransport func(*http.Request) (*http.Response, error)

func (transport silenceTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	return transport(request)
}
func useSilenceTransport(t *testing.T, transport silenceTransport) {
	t.Helper()
	original := http.DefaultTransport
	http.DefaultTransport = transport
	t.Cleanup(func() { http.DefaultTransport = original })
}
func fixtureResponse(request *http.Request, status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header), Request: request}
}

func TestSilenceHandlersUseTypedConfiguration(t *testing.T) {
	for _, globalTimeout := range []int{0, 99} {
		t.Run("global-timeout-"+strconv.Itoa(globalTimeout), func(t *testing.T) {
			viper.Reset()
			t.Cleanup(viper.Reset)
			viper.Set("webhook.alert_manager_api", "https://global-wrong.invalid")
			viper.Set("webhook.im_feishu.request_timeout_seconds", globalTimeout)
			cfg := withActionKey(&config.WebhookConfig{AlertManagerAPI: "https://typed-alertmanager.invalid/base", ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}})
			memory := &silenceMemoryDAO{event: model.MonitorAlertEvent{Status: model.MonitorAlertEventStatusFiring, Labels: model.StringList{"alertname=fixture"}}}
			handler, err := NewWebHookHandler(cfg, zap.NewNop(), memory, nil)
			if err != nil {
				t.Fatal(err)
			}
			engine := gin.New()
			handler.RegisterRouters(engine)
			calls := 0
			useSilenceTransport(t, func(request *http.Request) (*http.Response, error) {
				calls++
				deadline, exists := request.Context().Deadline()
				remaining := time.Until(deadline)
				if !exists || remaining <= 4*time.Second || remaining > 7*time.Second {
					t.Fatal("outbound request did not receive the typed seven-second deadline")
				}
				switch calls {
				case 1:
					if request.Method != "POST" || request.URL.String() != "https://typed-alertmanager.invalid/base/api/v2/silences" {
						t.Fatal("POST ignored typed endpoint")
					}
					if memory.updates != 0 {
						t.Fatal("DAO updated before upstream silence succeeded")
					}
					var payload struct {
						Matchers         []struct{ Name, Value string }
						StartsAt, EndsAt time.Time
					}
					if err := json.NewDecoder(request.Body).Decode(&payload); err != nil {
						t.Fatal(err)
					}
					if len(payload.Matchers) != 1 || payload.Matchers[0].Name != "alertname" || payload.Matchers[0].Value != "fixture" {
						t.Fatal("silence matcher payload changed")
					}
					if duration := payload.EndsAt.Sub(payload.StartsAt); duration < time.Hour || duration > time.Hour+time.Second {
						t.Fatal("requested silence duration changed")
					}
					return fixtureResponse(request, http.StatusOK, `{"silenceID":"fixture-silence"}`), nil
				default:
					t.Fatal("unexpected outbound request")
					return nil, errors.New("unexpected request")
				}
			})
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, signActionRequest(t, http.MethodPost, "silence", "fixture", "1"))
			if memory.updates != 1 || memory.event.Status != model.MonitorAlertEventStatusSilenced || memory.event.SilenceID != "fixture-silence" {
				t.Fatal("successful silence state was not persisted")
			}
			// Neither configuration source may redirect an already constructed handler
			// or remove its deadline before the subsequent public unsilence request.
			cfg.AlertManagerAPI = "https://later-typed.invalid"
			cfg.ImFeishu.RequestTimeoutSeconds = 0
			viper.Set("webhook.alert_manager_api", "https://later-global.invalid")
			if globalTimeout == 0 {
				viper.Set("webhook.im_feishu.request_timeout_seconds", 99)
			} else {
				viper.Set("webhook.im_feishu.request_timeout_seconds", 0)
			}
			recorder = httptest.NewRecorder()
			engine.ServeHTTP(recorder, signActionRequest(t, http.MethodPost, "unsilence", "fixture", ""))
			if calls != 1 || memory.updates != 1 || memory.event.Status != model.MonitorAlertEventStatusSilenced || memory.event.SilenceID != "fixture-silence" {
				t.Fatal("unsilence deleted the remote silence or cleared the local association")
			}
			var refused struct {
				Code    int
				Message string
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &refused); err != nil || recorder.Code != http.StatusOK || refused.Code != 1 || refused.Message != "未能取消，静默关联已保留" {
				t.Fatalf("unsilence response = %d %s", recorder.Code, recorder.Body.String())
			}
		})
	}
}

func TestSilenceHandlersDoNotUpdateAfterUpstreamFailure(t *testing.T) {
	for _, test := range []struct {
		name, path     string
		status         int
		body           string
		transportError error
	}{
		{"POST transport failure", "silence?fingerprint=fixture&hour=1", 0, "", errors.New("fixture transport unavailable")},
		{"POST HTTP failure", "silence?fingerprint=fixture&hour=1", http.StatusInternalServerError, `{}`, nil},
		{"POST malformed response", "silence?fingerprint=fixture&hour=1", http.StatusOK, `invalid-json`, nil},
		{"DELETE transport failure", "unsilence?fingerprint=fixture", 0, "", errors.New("fixture transport unavailable")},
		{"DELETE HTTP failure", "unsilence?fingerprint=fixture", http.StatusInternalServerError, `{}`, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			initial := model.MonitorAlertEventStatusFiring
			if strings.HasPrefix(test.path, "unsilence") {
				initial = model.MonitorAlertEventStatusSilenced
			}
			memory := &silenceMemoryDAO{event: model.MonitorAlertEvent{Status: initial, SilenceID: "fixture-old", Labels: model.StringList{"alertname=fixture"}}}
			handler, err := NewWebHookHandler(withActionKey(&config.WebhookConfig{AlertManagerAPI: "https://typed-alertmanager.invalid", ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}), zap.NewNop(), memory, nil)
			if err != nil {
				t.Fatal(err)
			}
			calls := 0
			useSilenceTransport(t, func(request *http.Request) (*http.Response, error) {
				calls++
				if memory.updates != 0 {
					t.Fatal("DAO update preceded HTTP request")
				}
				if test.transportError != nil {
					return nil, test.transportError
				}
				return fixtureResponse(request, test.status, test.body), nil
			})
			engine := gin.New()
			handler.RegisterRouters(engine)
			recorder := httptest.NewRecorder()
			actionName, hour := "silence", "1"
			if strings.HasPrefix(test.path, "unsilence") {
				actionName, hour = "unsilence", ""
			}
			engine.ServeHTTP(recorder, signActionRequest(t, http.MethodPost, actionName, "fixture", hour))
			wantCalls := 1
			if strings.HasPrefix(test.path, "unsilence") {
				wantCalls = 0
			}
			if calls != wantCalls || memory.updates != 0 || memory.event.Status != initial || memory.event.SilenceID != "fixture-old" {
				t.Fatal("failed upstream request changed persisted event")
			}
			if strings.HasPrefix(test.path, "unsilence") && !strings.Contains(recorder.Body.String(), "未能取消，静默关联已保留") {
				t.Fatalf("unsilence body = %s", recorder.Body.String())
			}
		})
	}
}

func TestWebhookHandlerRejectsUnboundedTimeout(t *testing.T) {
	for _, test := range []struct {
		name           string
		timeout        int
		missing, valid bool
	}{
		{"nil config", 0, true, false}, {"zero", 0, false, false}, {"negative", -1, false, false}, {"above validated maximum", 601, false, false}, {"minimum", 1, false, true}, {"maximum", 600, false, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg := &config.WebhookConfig{ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: test.timeout}}
			if test.missing {
				cfg = nil
			}
			handler, err := NewWebHookHandler(cfg, zap.NewNop(), nil, nil)
			if (err == nil) != test.valid {
				t.Fatal("constructor timeout validation changed")
			}
			if !test.valid && handler != nil {
				t.Fatal("invalid configuration returned usable handler")
			}
		})
	}
}

func (memory *silenceMemoryDAO) ClaimMonitorAlertEvent(_ context.Context, id int, expected model.MonitorAlertEventStatus) error {
	if id <= 0 || (expected != model.MonitorAlertEventStatusFiring && expected != model.MonitorAlertEventStatusUpgraded) {
		return dao.ErrAlertClaimConflict
	}
	if memory.event.ID == id && memory.event.Status == model.MonitorAlertEventStatusClaimed {
		return nil
	}
	if memory.event.ID != id || memory.event.Status != expected {
		return dao.ErrAlertClaimConflict
	}
	event := memory.event
	event.Status = model.MonitorAlertEventStatusClaimed
	return memory.UpdateMonitorAlertEvent(context.Background(), &event)
}

func (memory *silenceMemoryDAO) UpdateMonitorAlertEventSilence(ctx context.Context, id int, expected, desired dao.AlertEventSilenceState) error {
	current := dao.AlertEventSilenceState{Status: memory.event.Status, SilenceID: memory.event.SilenceID}
	if current != expected {
		if current == desired {
			return nil
		}
		return dao.ErrAlertSilenceConflict
	}
	event := memory.event
	event.Status = desired.Status
	event.SilenceID = desired.SilenceID
	return memory.UpdateMonitorAlertEvent(ctx, &event)
}
