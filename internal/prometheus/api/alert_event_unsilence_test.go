package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/middleware"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	alertdao "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/dao/alert"
	alertsvc "github.com/GoSimplicity/AI-CloudOps/internal/prometheus/service/alert"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type consoleUnsilenceDAO struct {
	alertdao.AlertManagerEventDAO
	event   *model.MonitorAlertEvent
	updates int
}

func (d *consoleUnsilenceDAO) GetAlertEventByID(context.Context, int) (*model.MonitorAlertEvent, error) {
	event := *d.event
	return &event, nil
}

func (d *consoleUnsilenceDAO) UpdateAlertEvent(context.Context, *model.MonitorAlertEvent) error {
	d.updates++
	return nil
}

func TestConsoleUnsilenceEnvelopeKeepsAssociation(t *testing.T) {
	dao := &consoleUnsilenceDAO{event: &model.MonitorAlertEvent{Model: model.Model{ID: 7}, Status: model.MonitorAlertEventStatusSilenced, SilenceID: "kept-id"}}
	engine := gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", ijwt.UserClaims{Uid: 4, Username: "operator"}); c.Next() })
	NewAlertEventHandler(alertsvc.NewAlertManagerEventService(dao, nil, zap.NewNop(), nil, nil)).RegisterRouters(engine)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/monitor/alert_events/unsilence/7", nil))
	var body struct {
		Code    int
		Data    json.RawMessage
		Message string
	}
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if recorder.Code != http.StatusOK || body.Code != 1 || string(body.Data) != "{}" || body.Message != "未能取消，静默关联已保留" {
		t.Fatalf("status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	if dao.updates != 0 || dao.event.SilenceID != "kept-id" || dao.event.Status != model.MonitorAlertEventStatusSilenced {
		t.Fatal("console refusal wrote the local association")
	}
}

type unsilenceGateService struct {
	alertsvc.AlertManagerEventService
	calls int
}

func (s *unsilenceGateService) EventAlertUnSilence(context.Context, *model.EventAlertUnSilenceReq) error {
	s.calls++
	return nil
}

type unsilenceGateRoles struct{ service.RoleService }

func (unsilenceGateRoles) GetUserRoles(context.Context, int) (*model.ListResp[*model.Role], error) {
	return &model.ListResp[*model.Role]{Items: []*model.Role{{Status: 1, Apis: []*model.Api{{Path: "/api/item/:id", Method: 1}}}}}, nil
}

func TestConsoleUnsilenceKeepsLoginAndRoleGates(t *testing.T) {
	jwtHandler, err := ijwt.NewJWTHandler(nil, ijwt.Config{Issuer: "test-issuer"})
	if err != nil {
		t.Fatal(err)
	}
	anonymous := &unsilenceGateService{}
	engine := gin.New()
	engine.Use(middleware.NewJWTMiddleware(jwtHandler, nil).CheckLogin(), middleware.NewAuthMiddleware(unsilenceGateRoles{}).CheckAuth())
	NewAlertEventHandler(anonymous).RegisterRouters(engine)
	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/monitor/alert_events/unsilence/7", nil))
	if recorder.Code != http.StatusUnauthorized || anonymous.calls != 0 {
		t.Fatalf("anonymous status=%d calls=%d", recorder.Code, anonymous.calls)
	}

	denied := &unsilenceGateService{}
	engine = gin.New()
	engine.Use(func(c *gin.Context) { c.Set("user", ijwt.UserClaims{Uid: 7, Username: "reader"}); c.Next() })
	engine.Use(middleware.NewAuthMiddleware(unsilenceGateRoles{}).CheckAuth())
	NewAlertEventHandler(denied).RegisterRouters(engine)
	recorder = httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/monitor/alert_events/unsilence/7", nil))
	if recorder.Code != http.StatusForbidden || denied.calls != 0 {
		t.Fatalf("unauthorized role status=%d calls=%d", recorder.Code, denied.calls)
	}
}
