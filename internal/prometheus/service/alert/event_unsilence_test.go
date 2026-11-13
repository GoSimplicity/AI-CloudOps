package alert

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	"go.uber.org/zap"
)

type unsilenceMemoryDAO struct {
	alertdao.AlertManagerEventDAO
	event   *model.MonitorAlertEvent
	err     error
	updates int
}

func (d *unsilenceMemoryDAO) GetAlertEventByID(context.Context, int) (*model.MonitorAlertEvent, error) {
	if d.err != nil {
		return nil, d.err
	}
	if d.event == nil {
		return nil, errors.New("未找到告警事件")
	}
	event := *d.event
	return &event, nil
}

func (d *unsilenceMemoryDAO) UpdateAlertEvent(context.Context, *model.MonitorAlertEvent) error {
	d.updates++
	return nil
}

func TestEventAlertUnSilenceRefusesUnprovenOwnership(t *testing.T) {
	original := http.DefaultTransport
	http.DefaultTransport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		t.Fatalf("remote call %s %s", request.Method, request.URL)
		return nil, errors.New("remote call")
	})
	t.Cleanup(func() { http.DefaultTransport = original })
	for _, tt := range []struct {
		name, silenceID, label string
	}{
		{"unknown ownership", "am-silence-old", "instance=db-1"},
		{"shared alertname matcher", "am-silence-shared", "alertname=DiskFull"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			dao := &unsilenceMemoryDAO{event: &model.MonitorAlertEvent{
				Model: model.Model{ID: 7}, Status: model.MonitorAlertEventStatusSilenced,
				SilenceID: tt.silenceID, Labels: model.StringList{tt.label},
			}}
			svc := NewAlertManagerEventService(dao, nil, zap.NewNop(), nil, nil)
			err := svc.EventAlertUnSilence(context.Background(), &model.EventAlertUnSilenceReq{ID: 7, UserID: 3})
			if err == nil || err.Error() != "未能取消，静默关联已保留" {
				t.Fatalf("error = %v", err)
			}
			if dao.updates != 0 || dao.event.Status != model.MonitorAlertEventStatusSilenced || dao.event.SilenceID != tt.silenceID {
				t.Fatalf("local association changed: updates=%d status=%d id=%s", dao.updates, dao.event.Status, dao.event.SilenceID)
			}
		})
	}
}

func TestEventAlertUnSilenceDoesNotClearReplacementID(t *testing.T) {
	dao := &unsilenceMemoryDAO{event: &model.MonitorAlertEvent{
		Model: model.Model{ID: 7}, Status: model.MonitorAlertEventStatusSilenced, SilenceID: "newer-id",
	}}
	svc := NewAlertManagerEventService(dao, nil, zap.NewNop(), nil, nil)
	if err := svc.EventAlertUnSilence(context.Background(), &model.EventAlertUnSilenceReq{ID: 7, UserID: 3}); err == nil {
		t.Fatal("replacement silence was accepted")
	}
	if dao.updates != 0 || dao.event.SilenceID != "newer-id" || dao.event.Status != model.MonitorAlertEventStatusSilenced {
		t.Fatalf("replacement silence cleared: updates=%d id=%s status=%d", dao.updates, dao.event.SilenceID, dao.event.Status)
	}
}

func TestEventAlertUnSilenceKeepsOriginalAbsenceErrors(t *testing.T) {
	svc := NewAlertManagerEventService(&unsilenceMemoryDAO{}, nil, zap.NewNop(), nil, nil)
	if err := svc.EventAlertUnSilence(context.Background(), &model.EventAlertUnSilenceReq{ID: 0, UserID: 3}); err == nil || err.Error() == "未能取消，静默关联已保留" {
		t.Fatalf("invalid id = %v", err)
	}
	missing := NewAlertManagerEventService(&unsilenceMemoryDAO{err: errors.New("未找到ID为 9 的告警事件")}, nil, zap.NewNop(), nil, nil)
	if err := missing.EventAlertUnSilence(context.Background(), &model.EventAlertUnSilenceReq{ID: 9, UserID: 3}); err == nil || err.Error() == "未能取消，静默关联已保留" || !strings.Contains(err.Error(), "获取告警事件失败") {
		t.Fatalf("missing event = %v", err)
	}
	empty := NewAlertManagerEventService(&unsilenceMemoryDAO{event: &model.MonitorAlertEvent{Model: model.Model{ID: 7}, Status: model.MonitorAlertEventStatusFiring}}, nil, zap.NewNop(), nil, nil)
	if err := empty.EventAlertUnSilence(context.Background(), &model.EventAlertUnSilenceReq{ID: 7, UserID: 3}); err == nil || err.Error() != "该告警事件未被静默" {
		t.Fatalf("empty association = %v", err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
