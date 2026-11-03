package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type pathIDCalls struct {
	count int
	id    int
	actor int
}

type pathIDCommentService struct {
	service.InstanceCommentService
	calls *pathIDCalls
}

func (s *pathIDCommentService) UpdateInstanceComment(_ context.Context, req *model.UpdateWorkorderInstanceCommentReq, actor int) error {
	s.calls.count++
	s.calls.id, s.calls.actor = req.ID, actor
	return nil
}

func (s *pathIDCommentService) DeleteInstanceComment(_ context.Context, id, actor int) error {
	s.calls.count++
	s.calls.id, s.calls.actor = id, actor
	return nil
}

func (s *pathIDCommentService) GetInstanceComment(_ context.Context, id int) (*model.WorkorderInstanceComment, error) {
	s.calls.count++
	s.calls.id = id
	return nil, nil
}

func (s *pathIDCommentService) GetInstanceCommentsTree(_ context.Context, id int) ([]*model.WorkorderInstanceComment, error) {
	s.calls.count++
	s.calls.id = id
	return nil, nil
}

type pathIDTimelineService struct {
	service.WorkorderInstanceTimeLineService
	calls *pathIDCalls
}

func (s *pathIDTimelineService) GetInstanceTimeLine(_ context.Context, id int) (*model.WorkorderInstanceTimeline, error) {
	s.calls.count++
	s.calls.id = id
	return nil, nil
}

func pathIDEngine(calls *pathIDCalls) *gin.Engine {
	engine := gin.New()
	engine.Use(func(ctx *gin.Context) {
		ctx.Set("user", jwt.UserClaims{Uid: 7, Username: "fixture-operator"})
		ctx.Next()
	})
	NewInstanceCommentHandler(&pathIDCommentService{calls: calls}).RegisterRouters(engine)
	NewInstanceTimeLineHandler(&pathIDTimelineService{calls: calls}).RegisterRouters(engine)
	return engine
}

func checkPathIDErrors(t *testing.T, method, prefix, message string, wantsActor bool) {
	t.Helper()
	for _, tc := range []struct {
		name string
		id   string
	}{
		{name: "zero", id: "0"},
		{name: "negative", id: "-1"},
		{name: "not numeric", id: "private-path-id-canary"},
		{name: "overflow", id: "99999999999999999999999999999"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := &pathIDCalls{}
			recorder := httptest.NewRecorder()
			req := httptest.NewRequest(method, prefix+tc.id+"?id=99", strings.NewReader(`{"id":99,"content":"fixture comment"}`))
			req.Header.Set("Content-Type", "application/json")
			pathIDEngine(calls).ServeHTTP(recorder, req)
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("invalid path must be HTTP 400, got %d: %s", recorder.Code, recorder.Body.String())
			}
			var response struct {
				Code    int             `json:"code"`
				Data    json.RawMessage `json:"data"`
				Message string          `json:"message"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("invalid API error envelope: %v", err)
			}
			if response.Code != base.StatusError || response.Message != message || string(response.Data) != "{}" {
				t.Fatalf("unexpected error envelope: %s", recorder.Body.String())
			}
			if !strings.HasPrefix(recorder.Header().Get("Content-Type"), "application/json") {
				t.Fatal("error response must be JSON")
			}
			if calls.count != 0 {
				t.Fatalf("invalid path reached business service %d times", calls.count)
			}
		})
	}
	t.Run("positive path remains authoritative", func(t *testing.T) {
		calls := &pathIDCalls{}
		recorder := httptest.NewRecorder()
		req := httptest.NewRequest(method, prefix+"17?id=99", strings.NewReader(`{"id":99,"content":"fixture comment"}`))
		req.Header.Set("Content-Type", "application/json")
		pathIDEngine(calls).ServeHTTP(recorder, req)
		var response struct {
			Code int `json:"code"`
		}
		if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
			t.Fatalf("invalid positive response: %v", err)
		}
		if recorder.Code != http.StatusOK || response.Code != base.StatusSuccess || calls.count != 1 || calls.id != 17 {
			t.Fatalf("positive path or service dispatch changed: %d %s calls=%+v", recorder.Code, recorder.Body.String(), calls)
		}
		if wantsActor && calls.actor != 7 {
			t.Fatalf("authenticated actor changed: %d", calls.actor)
		}
	})
}

func TestUpdateInstanceCommentPathIDErrors(t *testing.T) {
	checkPathIDErrors(t, http.MethodPut, "/api/workorder/instance/comment/update/", "评论ID不合法", true)
}

func TestDeleteInstanceCommentPathIDErrors(t *testing.T) {
	checkPathIDErrors(t, http.MethodDelete, "/api/workorder/instance/comment/delete/", "评论ID不合法", true)
}

func TestGetInstanceCommentPathIDErrors(t *testing.T) {
	checkPathIDErrors(t, http.MethodGet, "/api/workorder/instance/comment/detail/", "评论ID不合法", false)
}

func TestGetInstanceCommentsTreePathIDErrors(t *testing.T) {
	checkPathIDErrors(t, http.MethodGet, "/api/workorder/instance/comment/tree/", "工单ID不合法", false)
}

func TestDetailInstanceTimelinePathIDErrors(t *testing.T) {
	checkPathIDErrors(t, http.MethodGet, "/api/workorder/instance/timeline/detail/", "时间线ID不合法", false)
}
