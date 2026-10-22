package api

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/workorder/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type categoryRecorder struct {
	service.CategoryGroupService
	created *model.CreateWorkorderCategoryReq
	updated *model.UpdateWorkorderCategoryReq
}

func (s *categoryRecorder) CreateCategory(_ context.Context, r *model.CreateWorkorderCategoryReq) error {
	s.created = r
	return nil
}
func (s *categoryRecorder) UpdateCategory(_ context.Context, r *model.UpdateWorkorderCategoryReq) error {
	s.updated = r
	return nil
}

func TestCategoryTrustedIdentity(t *testing.T) {
	for _, tt := range []struct{ method, path, body string }{
		{"POST", "/api/workorder/category/create", `{"name":"ops","status":1,"operator_id":999,"operator_name":"forged"}`},
		{"PUT", "/api/workorder/category/update/12", `{"id":999,"name":"ops","status":1}`},
	} {
		t.Run(tt.method, func(t *testing.T) {
			recorder := &categoryRecorder{}
			engine := gin.New()
			engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "alice"}); c.Next() })
			NewCategoryGroupHandler(recorder).RegisterRouters(engine)
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			req.Header.Set("Content-Type", "application/json")
			response := httptest.NewRecorder()
			engine.ServeHTTP(response, req)
			if response.Code != 200 {
				t.Fatalf("response %s", response.Body.String())
			}
			if tt.method == "POST" && (recorder.created == nil || recorder.created.OperatorID != 7 || recorder.created.OperatorName != "alice") {
				t.Fatalf("forged creator reached service: %+v", recorder.created)
			}
			if tt.method == "PUT" && (recorder.updated == nil || recorder.updated.ID != 12) {
				t.Fatalf("forged resource ID reached service: %+v", recorder.updated)
			}
		})
	}
}
