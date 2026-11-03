package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/gin-gonic/gin"
)

type pathIdentityDaemonSetService struct {
	service.DaemonSetService
	calls     int
	clusterID int
}

func (s *pathIdentityDaemonSetService) RollbackDaemonSet(_ context.Context, req *model.RollbackDaemonSetReq) error {
	s.calls++
	s.clusterID = req.ClusterID
	return nil
}

func TestClusterPathIDCannotBeReplaced(t *testing.T) {
	for _, tc := range []struct {
		path    string
		allowed bool
	}{
		{"0", false}, {"-0", false}, {"000", false}, {"-1", false},
		{"bad", false}, {"7", true}, {"0007", true},
	} {
		t.Run(tc.path, func(t *testing.T) {
			svc := &pathIdentityDaemonSetService{}
			engine := gin.New()
			NewK8sDaemonSetHandler(svc).RegisterRouters(engine)
			request := httptest.NewRequest(http.MethodPost, "/api/k8s/daemonset/"+tc.path+"/default/fixture/rollback", strings.NewReader(`{"cluster_id":999,"revision":1}`))
			request.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, request)
			var envelope struct{ Code int }
			if err := json.Unmarshal(response.Body.Bytes(), &envelope); err != nil {
				t.Fatal("handler must return the normal response envelope")
			}
			if tc.allowed {
				if svc.calls != 1 || svc.clusterID != 7 || envelope.Code != 0 {
					t.Fatal("the positive path cluster ID must remain authoritative over a forged body ID")
				}
			} else if svc.calls != 0 || envelope.Code == 0 {
				t.Fatal("invalid cluster path ID must not borrow a body ID or issue a rollback")
			}
		})
	}
}
