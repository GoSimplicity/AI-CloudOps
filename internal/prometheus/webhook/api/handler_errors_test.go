package api

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/prometheus/webhook/dao"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type chainDAO struct {
	dao.WebhookDao
	event                  model.MonitorAlertEvent
	lookupError, saveError error
	updates                int
}

func (memory *chainDAO) GetMonitorAlertEventByFingerprintId(context.Context, string) (*model.MonitorAlertEvent, error) {
	if memory.lookupError != nil {
		return nil, memory.lookupError
	}
	event := memory.event
	return &event, nil
}
func (memory *chainDAO) UpdateMonitorAlertEvent(_ context.Context, event *model.MonitorAlertEvent) error {
	memory.updates++
	if memory.saveError != nil {
		return memory.saveError
	}
	memory.event = *event
	return nil
}

type chainPrivateBody struct{ sent bool }

func (body *chainPrivateBody) Read(buffer []byte) (int, error) {
	if !body.sent {
		body.sent = true
		return copy(buffer, "PRIVATE_CANARY"), nil
	}
	return 0, errors.New("PRIVATE_CANARY read error")
}
func (*chainPrivateBody) Close() error { return nil }

func TestSilenceErrorChainDoesNotLeakOrFalselyConfirm(t *testing.T) {
	const canary = "PRIVATE_CANARY"
	for _, action := range []string{"silence", "unsilence"} {
		t.Run(action, func(t *testing.T) {
			for _, mode := range []string{"success", "lookup", "transport", "status", "read", "save", "invalid_id", "decode"} {
				if action == "unsilence" && mode == "decode" {
					continue
				}
				t.Run(mode, func(t *testing.T) {
					core, logs := observer.New(zap.DebugLevel)
					logger := zap.New(core)
					initial := model.MonitorAlertEventStatusFiring
					if action == "unsilence" {
						initial = model.MonitorAlertEventStatusSilenced
					}
					memory := &chainDAO{event: model.MonitorAlertEvent{Model: model.Model{ID: 7}, Status: initial, SilenceID: canary, Fingerprint: canary, Labels: model.StringList{"private=" + canary, "invalid=" + canary + "=private"}}}
					if mode == "lookup" {
						memory.lookupError = errors.New(canary)
					}
					if mode == "save" {
						memory.saveError = errors.New(canary)
					}
					if action == "unsilence" && mode == "invalid_id" {
						memory.event.SilenceID = "../" + canary
					}
					handler, err := NewWebHookHandler(&config.WebhookConfig{AlertManagerAPI: "https://" + canary + ":" + canary + "@upstream.invalid", ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}, logger, memory, nil)
					if err != nil {
						t.Fatal(err)
					}
					calls := 0
					useSilenceTransport(t, func(request *http.Request) (*http.Response, error) {
						calls++
						if memory.updates != 0 {
							t.Fatal("local save occurred before upstream request")
						}
						if mode == "transport" {
							return nil, errors.New(canary)
						}
						response := fixtureResponse(request, 200, `{"silenceID":"`+canary+`"}`)
						switch mode {
						case "status":
							response.StatusCode = 503
							response.Status = canary
							response.Body = io.NopCloser(strings.NewReader(canary))
						case "read":
							response.Body = &chainPrivateBody{}
						case "invalid_id":
							response.Body = io.NopCloser(strings.NewReader(`{"silenceID":"../` + canary + `"}`))
						case "decode":
							response.Body = io.NopCloser(strings.NewReader(canary))
						}
						return response, nil
					})
					engine := gin.New()
					handler.RegisterRouters(engine)
					recorder := httptest.NewRecorder()
					path := "/api/v1/alerts/" + action + "?fingerprint=" + canary
					if action == "silence" {
						path += "&hour=1"
					}
					engine.ServeHTTP(recorder, httptest.NewRequest("GET", path, nil))
					expectedCalls := 1
					if mode == "lookup" || (mode == "invalid_id" && action == "unsilence") {
						expectedCalls = 0
					}
					if calls != expectedCalls {
						t.Fatal("failed operation retried or reached an unsafe upstream path")
					}
					expectedUpdates := 0
					if mode == "success" || mode == "save" {
						expectedUpdates = 1
					}
					if memory.updates != expectedUpdates {
						t.Fatal("local save count changed")
					}
					var business struct {
						Code    int
						Message string
					}
					if err := json.Unmarshal(recorder.Body.Bytes(), &business); err != nil {
						t.Fatal(err)
					}
					if recorder.Code != http.StatusOK {
						t.Fatal("existing business envelope HTTP status changed")
					}
					if (business.Code == 0) != (mode == "success") {
						t.Fatal("failed or unconfirmed operation returned business success")
					}
					if mode == "save" && !strings.Contains(business.Message, "未确认") {
						t.Fatal("DAO failure omitted manual reconciliation status")
					}
					if mode != "success" && memory.event.Status != initial {
						t.Fatal("failure changed persisted status")
					}
					if strings.Contains(recorder.Body.String(), canary) {
						t.Fatal("private upstream/request/DAO value leaked in business response")
					}
					for _, entry := range logs.All() {
						if strings.Contains(entry.Message+fmt.Sprint(entry.ContextMap()), canary) {
							t.Fatal("private value leaked into helper or handler log")
						}
					}
				})
			}
		})
	}
}

func TestSilenceIdentifierValidation(t *testing.T) {
	for _, test := range []struct {
		name, id string
		valid    bool
	}{
		{"existing ID", "fixture-silence", true}, {"underscores", "fixture_id_1", true}, {"bound", strings.Repeat("a", 100), true},
		{"empty", "", false}, {"too long", strings.Repeat("a", 101), false}, {"dot path", "..", false}, {"slash", "id/path", false}, {"backslash", "id\\path", false}, {"query", "id?private", false}, {"fragment", "id#private", false}, {"encoded path", "id%2fpath", false}, {"control", "id\nprivate", false}, {"NUL", "id\x00private", false}, {"space", " id ", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if validSilenceID(test.id) != test.valid {
				t.Fatal("identifier grammar changed")
			}
		})
	}
}

func (memory *chainDAO) UpdateMonitorAlertEventSilence(ctx context.Context, id int, expected, desired dao.AlertEventSilenceState) error {
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
