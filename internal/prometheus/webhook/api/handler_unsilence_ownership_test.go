package api

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

func TestWebhookUnsilenceRefusesUnprovenOwnershipWithoutDelete(t *testing.T) {
	for _, tt := range []struct {
		name, silenceID, label string
	}{
		{"unknown ownership", "am-silence-old", "instance=db-1"},
		{"shared alertname matcher", "am-silence-shared", "alertname=DiskFull"},
		{"replacement id", "newer-id", "instance=db-1"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			memory := &silenceMemoryDAO{event: model.MonitorAlertEvent{
				Model: model.Model{ID: 7}, Status: model.MonitorAlertEventStatusSilenced,
				SilenceID: tt.silenceID, Fingerprint: "fixture", Labels: model.StringList{tt.label},
			}}
			handler, err := NewWebHookHandler(withActionKey(&config.WebhookConfig{AlertManagerAPI: "https://typed-alertmanager.invalid", ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}), zap.NewNop(), memory, nil)
			if err != nil {
				t.Fatal(err)
			}
			deletes := 0
			useSilenceTransport(t, func(request *http.Request) (*http.Response, error) {
				if request.Method == http.MethodDelete {
					deletes++
				}
				t.Fatalf("remote call %s %s", request.Method, request.URL)
				return nil, io.EOF
			})
			engine := gin.New()
			handler.RegisterRouters(engine)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, signActionRequest(t, http.MethodPost, "unsilence", "fixture", ""))
			var body struct {
				Code    int
				Data    map[string]any
				Message string
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if recorder.Code != http.StatusOK || body.Code != 1 || body.Message != "未能取消，静默关联已保留" || len(body.Data) != 0 {
				t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
			}
			if deletes != 0 || memory.updates != 0 || memory.event.SilenceID != tt.silenceID || memory.event.Status != model.MonitorAlertEventStatusSilenced {
				t.Fatalf("deletes=%d updates=%d id=%s status=%d", deletes, memory.updates, memory.event.SilenceID, memory.event.Status)
			}
		})
	}
}

func TestWebhookUnsilenceAbsenceIsNotDescribedAsRetained(t *testing.T) {
	for _, tt := range []struct {
		name   string
		dao    *ownershipAbsenceDAO
		needle string
	}{
		{"missing event", &ownershipAbsenceDAO{}, "未找到对应的告警事件"},
		{"not silenced", &ownershipAbsenceDAO{event: &model.MonitorAlertEvent{Status: model.MonitorAlertEventStatusFiring, Fingerprint: "fixture"}}, "该告警未处于静默状态"},
		{"invalid id", &ownershipAbsenceDAO{event: &model.MonitorAlertEvent{Status: model.MonitorAlertEventStatusSilenced, SilenceID: "../secret", Fingerprint: "fixture"}}, "本地静默标识无效，请人工核对"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			handler, err := NewWebHookHandler(withActionKey(&config.WebhookConfig{AlertManagerAPI: "https://typed-alertmanager.invalid", ImFeishu: config.ImFeishuConfig{RequestTimeoutSeconds: 7}}), zap.NewNop(), tt.dao, nil)
			if err != nil {
				t.Fatal(err)
			}
			useSilenceTransport(t, func(request *http.Request) (*http.Response, error) {
				t.Fatalf("remote call %s", request.Method)
				return nil, io.EOF
			})
			engine := gin.New()
			handler.RegisterRouters(engine)
			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, signActionRequest(t, http.MethodPost, "unsilence", "fixture", ""))
			if !strings.Contains(recorder.Body.String(), tt.needle) || strings.Contains(recorder.Body.String(), "未能取消，静默关联已保留") {
				t.Fatalf("body=%s", recorder.Body.String())
			}
		})
	}
}

type ownershipAbsenceDAO struct {
	silenceMemoryDAO
	event *model.MonitorAlertEvent
}

func (d *ownershipAbsenceDAO) GetMonitorAlertEventByFingerprintId(context.Context, string) (*model.MonitorAlertEvent, error) {
	if d.event == nil {
		return nil, nil
	}
	event := *d.event
	return &event, nil
}
