package base

import (
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAuthoritativeBinding(t *testing.T) {
	type request struct {
		ID         int    `json:"id" form:"id" binding:"min=1"`
		OperatorID int    `json:"operator_id" form:"operator_id" binding:"min=1"`
		Name       string `json:"name" form:"name" binding:"required"`
		Page       int    `json:"page" form:"page"`
	}
	for _, tt := range []struct {
		name, method, path, body string
		wantErr                  bool
	}{
		{"JSON cannot overwrite identity", "POST", "/", `{"id":99,"operator_id":98,"name":"client","page":3}`, false},
		{"invalid forged identity is ignored", "POST", "/", `{"id":-1,"operator_id":0,"name":"client","page":3}`, false},
		{"query cannot overwrite identity", "GET", "/?id=99&operator_id=98&name=client&page=3", "", false},
		{"invalid client field rejected", "POST", "/", `{"name":"","page":3}`, true},
		{"malformed body rejected", "POST", "/", `{`, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Request = httptest.NewRequest(tt.method, tt.path, strings.NewReader(tt.body))
			if tt.method == "POST" {
				ctx.Request.Header.Set("Content-Type", "application/json")
			}
			req := request{ID: 7, OperatorID: 8, Page: 1}
			err := BindRequest(ctx, &req)
			if (err != nil) != tt.wantErr {
				t.Fatalf("unexpected error %v", err)
			}
			if req.ID != 7 || req.OperatorID != 8 {
				t.Fatalf("identity changed: %+v", req)
			}
			if !tt.wantErr && (req.Name != "client" || req.Page != 3) {
				t.Fatalf("client fields failed to bind: %+v", req)
			}
		})
	}
}
