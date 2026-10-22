package middleware

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/system/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/jwt"
	"github.com/gin-gonic/gin"
)

type readOnlyRoles struct{ service.RoleService }

func (readOnlyRoles) GetUserRoles(context.Context, int) (*model.ListResp[*model.Role], error) {
	return &model.ListResp[*model.Role]{Items: []*model.Role{{Status: 1, Apis: []*model.Api{{Path: "/api/item/:id", Method: 1}}}}}, nil
}
func TestRegisteredReadPermissionDoesNotAllowWrites(t *testing.T) {
	for _, tt := range []struct {
		method string
		status int
	}{{"GET", 204}, {"POST", 403}} {
		engine := gin.New()
		engine.Use(func(c *gin.Context) { c.Set("user", jwt.UserClaims{Uid: 7, Username: "reader"}); c.Next() })
		engine.Use(NewAuthMiddleware(readOnlyRoles{}).CheckAuth())
		handler := func(c *gin.Context) { c.Status(204) }
		engine.GET("/api/item/:id", handler)
		engine.POST("/api/item/:id", handler)
		rec := httptest.NewRecorder()
		engine.ServeHTTP(rec, httptest.NewRequest(tt.method, "/api/item/42", nil))
		if rec.Code != tt.status {
			t.Fatalf("%s returned %d", tt.method, rec.Code)
		}
	}
}
