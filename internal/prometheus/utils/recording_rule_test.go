package utils

import (
	"maps"
	"strings"
	"testing"
)

func TestBuildRecordingRule(t *testing.T) {
	cases := []struct {
		name, metric, expression string
		labels                   []string
		want                     map[string]string
		invalid                  bool
	}{
		{name: "vector", metric: "job:up:sum", expression: "sum(up)"},
		{name: "scalar", metric: "scalar", expression: "42"},
		{name: "UTF8", metric: "服务:up", expression: "up"},
		{name: "UTF8 hyphen", metric: "bad-name", expression: "up"},
		{name: "delimiter in UTF8 name", metric: "a|up|b", expression: "up"},
		{name: "preserve raw name", metric: " service:up ", expression: "up"},
		{name: "long legacy stored name", metric: strings.Repeat("a", 100), expression: "up"},
		{name: "labels", metric: "up:sum", expression: "up", labels: []string{"team=ops", "url=a=b", "empty=", "raw= leading = trailing ", "团队=服务"}, want: map[string]string{"team": "ops", "url": "a=b", "empty": "", "raw": " leading = trailing ", "团队": "服务"}},
		{name: "last duplicate wins", metric: "up:sum", expression: "up", labels: []string{"team=before", "team=after"}, want: map[string]string{"team": "after"}},
		{name: "empty name", expression: "up", invalid: true},
		{name: "blank name", metric: " \t ", expression: "up", invalid: true},
		{name: "opening brace", metric: "bad{rule", expression: "up", invalid: true},
		{name: "closing brace", metric: "bad}rule", expression: "up", invalid: true},
		{name: "invalid UTF8 name", metric: string([]byte{0xff}), expression: "up", invalid: true},
		{name: "empty expression", metric: "up:sum", invalid: true},
		{name: "syntax", metric: "up:sum", expression: "sum(", invalid: true},
		{name: "string expression", metric: "up:sum", expression: `"hello"`, invalid: true},
		{name: "parenthesized string", metric: "up:sum", expression: `("hello")`, invalid: true},
		{name: "matrix expression", metric: "up:sum", expression: "up[5m]", invalid: true},
		{name: "missing delimiter", metric: "up:sum", expression: "up", labels: []string{"team"}, invalid: true},
		{name: "empty key", metric: "up:sum", expression: "up", labels: []string{"=ops"}, invalid: true},
		{name: "reserved metric label", metric: "up:sum", expression: "up", labels: []string{"__name__=override"}, invalid: true},
		{name: "invalid UTF8 key", metric: "up:sum", expression: "up", labels: []string{string([]byte{0xff}) + "=value"}, invalid: true},
		{name: "invalid UTF8 value", metric: "up:sum", expression: "up", labels: []string{"team=" + string([]byte{0xff})}, invalid: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := BuildRecordingRule(tc.metric, tc.expression, tc.labels)
			if tc.invalid {
				if err == nil {
					t.Fatal("invalid configuration accepted")
				}
				return
			}
			if err != nil {
				t.Fatalf("valid configuration rejected: %v", err)
			}
			if got.Record != tc.metric || got.Expr != tc.expression || !maps.Equal(got.Labels, tc.want) {
				t.Fatal("record, expression, or labels changed")
			}
		})
	}
}

func TestBuildRecordingRuleErrorsDoNotEchoInputs(t *testing.T) {
	const canary = "opaque-private-canary"
	for _, tc := range []struct {
		name, metric, expression string
		labels                   []string
	}{
		{name: "name", metric: canary + "{x}", expression: "up"},
		{name: "expression", metric: "up:sum", expression: "sum(" + canary},
		{name: "label", metric: "up:sum", expression: "up", labels: []string{"__name__=" + canary}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := BuildRecordingRule(tc.metric, tc.expression, tc.labels)
			if err == nil || strings.Contains(err.Error(), canary) {
				t.Fatal("validation accepted invalid data or echoed its contents")
			}
		})
	}
}
