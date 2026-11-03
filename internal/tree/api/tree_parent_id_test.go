package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/gin-gonic/gin"
)

type parentIDTreeService struct {
	service.TreeNodeService
	calls int
	id    int
}

func (s *parentIDTreeService) GetChildNodes(_ context.Context, id int) ([]*model.TreeNode, error) {
	s.calls++
	s.id = id
	return []*model.TreeNode{}, nil
}

func TestTreeParentIDKeepsVirtualRoot(t *testing.T) {
	for _, tc := range []struct {
		path    string
		id      int
		allowed bool
	}{
		{"0", 0, true}, {"7", 7, true}, {"-1", 0, false}, {"bad", 0, false},
	} {
		t.Run(tc.path, func(t *testing.T) {
			svc := &parentIDTreeService{}
			engine := gin.New()
			NewTreeNodeHandler(svc).RegisterRouters(engine)
			request := httptest.NewRequest(http.MethodGet, "/api/tree/node/children/"+tc.path+"?id=999", nil)
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			var envelope struct{ Code int }
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal("handler must return the normal response envelope")
			}
			if tc.allowed {
				if svc.calls != 1 || svc.id != tc.id || envelope.Code != 0 {
					t.Fatal("tree children must retain explicit zero-root and positive-parent semantics")
				}
			} else if svc.calls != 0 || envelope.Code == 0 {
				t.Fatal("invalid parent ID must not reach the service")
			}
		})
	}
}
