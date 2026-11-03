package base

import (
	"math"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestResourceParamIDRequiresPositive(t *testing.T) {
	for _, tc := range []struct {
		name string
		raw  string
		want int
	}{
		{"missing", "", 0},
		{"zero", "0", 0},
		{"negative-zero", "-0", 0},
		{"padded-zero", "000", 0},
		{"negative", "-1", 0},
		{"non-integer", "1.5", 0},
		{"overflow", strconv.Itoa(math.MaxInt) + "0", 0},
		{"positive", "7", 7},
		{"legacy-padded-positive", "0007", 7},
		{"largest-positive", strconv.Itoa(math.MaxInt), math.MaxInt},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, _ := gin.CreateTestContext(httptest.NewRecorder())
			ctx.Params = gin.Params{{Key: "id", Value: tc.raw}}
			id, err := GetParamID(ctx)
			if tc.want == 0 {
				if err == nil || id != 0 {
					t.Fatal("invalid entity path ID must fail before request binding")
				}
				return
			}
			if err != nil || id != tc.want {
				t.Fatal("valid positive entity ID must retain its existing numeric value")
			}
		})
	}
}
