package utils

import (
	"reflect"
	"testing"

	common "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
	"gopkg.in/yaml.v3"
)

func TestBuildAlertingRuleValidContract(t *testing.T) {
	for _, test := range []struct {
		name, expression, duration string
		labels, annotations        []string
	}{
		{name: "服务{告警}", expression: "up", duration: "0s"},
		{name: "scalar", expression: "1 + 2", duration: "1m", labels: []string{"team=ops", "url=https://demo.invalid/?q=a=b,c", "empty=", "标签=正常"}, annotations: []string{"summary={{ $labels.team }}=正常,ready", "description=a=b,c"}},
		{name: "literal,name", expression: "up", duration: "0", labels: []string{"team=first", "team=last"}, annotations: []string{"__name__=annotation value", "empty="}},
	} {
		t.Run(test.name, func(t *testing.T) {
			beforeLabels := append([]string(nil), test.labels...)
			beforeAnnotations := append([]string(nil), test.annotations...)
			rule, err := BuildAlertingRule(test.name, test.expression, test.duration, test.labels, test.annotations)
			if err != nil {
				t.Fatal(err)
			}
			if rule.Alert != test.name || rule.Expr != test.expression {
				t.Fatal("valid name or expression was normalized")
			}
			if !reflect.DeepEqual(beforeLabels, test.labels) || !reflect.DeepEqual(beforeAnnotations, test.annotations) {
				t.Fatal("inputs mutated")
			}
			data, err := yaml.Marshal(&rulefmt.RuleGroups{Groups: []rulefmt.RuleGroup{{Name: "fixture", Rules: []rulefmt.Rule{rule}}}})
			if err != nil {
				t.Fatal(err)
			}
			parsed, errs := rulefmt.Parse(data, false, common.UTF8Validation, parser.NewParser(parser.Options{}))
			if len(errs) > 0 {
				t.Fatalf("fixed Prometheus parser rejected valid construction: %v", errs)
			}
			if len(parsed.Groups) != 1 {
				t.Fatal("group missing")
			}
			if test.name == "scalar" {
				if rule.Labels["url"] != "https://demo.invalid/?q=a=b,c" || rule.Annotations["description"] != "a=b,c" {
					t.Fatal("value after first separator was changed")
				}
				if _, ok := rule.Labels["empty"]; !ok {
					t.Fatal("empty label value omitted")
				}
			}
			if test.name == "literal,name" && rule.Labels["team"] != "last" {
				t.Fatal("existing last-value-wins contract changed")
			}
		})
	}
}
func TestBuildAlertingRuleRejectsMalformedConstruction(t *testing.T) {
	for _, test := range []struct {
		name, alert, expression, duration string
		labels, annotations               []string
	}{
		{name: "empty name", expression: "up", duration: "1m"},
		{name: "invalid UTF8 name", alert: string([]byte{255}), expression: "up", duration: "1m"},
		{name: "empty expr", alert: "valid", duration: "1m"},
		{name: "syntax", alert: "valid", expression: "sum(", duration: "1m"},
		{name: "matrix", alert: "valid", expression: "up[5m]", duration: "1m"},
		{name: "string", alert: "valid", expression: `"hello"`, duration: "1m"},
		{name: "empty duration", alert: "valid", expression: "up"},
		{name: "invalid duration", alert: "valid", expression: "up", duration: "later"},
		{name: "negative duration", alert: "valid", expression: "up", duration: "-1m"},
		{name: "label missing separator", alert: "valid", expression: "up", duration: "1m", labels: []string{"team,ops"}},
		{name: "empty label key", alert: "valid", expression: "up", duration: "1m", labels: []string{"=ops"}},
		{name: "reserved label", alert: "valid", expression: "up", duration: "1m", labels: []string{"__name__=override"}},
		{name: "invalid UTF8 label", alert: "valid", expression: "up", duration: "1m", labels: []string{"team=" + string([]byte{255})}},
		{name: "label template", alert: "valid", expression: "up", duration: "1m", labels: []string{"team={{"}},
		{name: "annotation missing separator", alert: "valid", expression: "up", duration: "1m", annotations: []string{"summary,legacy"}},
		{name: "empty annotation key", alert: "valid", expression: "up", duration: "1m", annotations: []string{"=oops"}},
		{name: "invalid UTF8 annotation", alert: "valid", expression: "up", duration: "1m", annotations: []string{"summary=" + string([]byte{255})}},
		{name: "annotation template", alert: "valid", expression: "up", duration: "1m", annotations: []string{"summary={{"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := BuildAlertingRule(test.alert, test.expression, test.duration, test.labels, test.annotations); err == nil {
				t.Fatal("invalid construction accepted")
			}
		})
	}
}
