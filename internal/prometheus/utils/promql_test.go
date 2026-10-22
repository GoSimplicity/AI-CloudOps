package utils

import "testing"

func TestPromqlExprCheck(t *testing.T) {
	for _, tt := range []struct {
		expr  string
		valid bool
	}{
		{`up`, true},
		{`sum(rate(http_requests_total{job="api"}[5m])) by (status)`, true},
		{`histogram_quantile(0.95, sum by (le) (rate(request_duration_seconds_bucket[5m])))`, true},
		{``, false},
		{`sum(`, false},
		{`unknown_function(up)`, false},
	} {
		t.Run(tt.expr, func(t *testing.T) {
			valid, err := PromqlExprCheck(tt.expr)
			if valid != tt.valid || (err == nil) != tt.valid {
				t.Fatalf("valid=%v err=%v", valid, err)
			}
		})
	}
}
