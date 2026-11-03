package alert_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
)

func TestAlertRuleSaveRejectsInvalidConstructionBeforePersistence(t *testing.T) {
	tests := []struct {
		name                           string
		nameValue, expression, forTime string
		labels, annotations            model.StringList
	}{
		{name: "empty name", nameValue: "", expression: "up", forTime: "1m"},
		{name: "empty duration", nameValue: "valid", expression: "up", forTime: ""},
		{name: "bad duration", nameValue: "valid", expression: "up", forTime: "later"},
		{name: "negative duration", nameValue: "valid", expression: "up", forTime: "-1m"},
		{name: "matrix", nameValue: "valid", expression: "up[5m]", forTime: "1m"},
		{name: "string", nameValue: "valid", expression: `"hello"`, forTime: "1m"},
		{name: "syntax", nameValue: "valid", expression: "sum(", forTime: "1m"},
		{name: "label separator", nameValue: "valid", expression: "up", forTime: "1m", labels: model.StringList{"team,ops"}},
		{name: "reserved label", nameValue: "valid", expression: "up", forTime: "1m", labels: model.StringList{"__name__=override"}},
		{name: "empty label name", nameValue: "valid", expression: "up", forTime: "1m", labels: model.StringList{"=ops"}},
		{name: "label template", nameValue: "valid", expression: "up", forTime: "1m", labels: model.StringList{"team={{"}},
		{name: "annotation separator", nameValue: "valid", expression: "up", forTime: "1m", annotations: model.StringList{"summary,legacy"}},
		{name: "annotation template", nameValue: "valid", expression: "up", forTime: "1m", annotations: model.StringList{"summary={{"}},
	}
	for _, mode := range []string{"create", "update"} {
		for _, test := range tests {
			t.Run(mode+"/"+test.name, func(t *testing.T) {
				service, store, pools := namespaceService()
				store.row = &model.MonitorAlertRule{Model: model.Model{ID: 31}, PoolID: 7}
				var err error
				if mode == "create" {
					req := namespaceCreate(7)
					req.Name = test.nameValue
					req.Expr = test.expression
					req.ForTime = test.forTime
					req.Labels = test.labels
					req.Annotations = test.annotations
					err = service.CreateMonitorAlertRule(context.Background(), req)
				} else {
					req := namespaceUpdate(7)
					req.Name = test.nameValue
					req.Expr = test.expression
					req.ForTime = test.forTime
					req.Labels = test.labels
					req.Annotations = test.annotations
					err = service.UpdateMonitorAlertRule(context.Background(), req)
				}
				if err == nil || store.creates+store.updates != 0 || len(pools.calls) > 0 {
					t.Fatal("invalid construction reached persistence or a foreign dependency")
				}
			})
		}
	}
}
func TestAlertRuleSavePreservesValidOptionalAndUTF8Fields(t *testing.T) {
	for _, labels := range []model.StringList{nil, {}, {"标签=正常", "url=https://demo.invalid/?q=a=b,c", "empty="}} {
		t.Run("create", func(t *testing.T) {
			service, store, _ := namespaceService()
			req := namespaceCreate(7)
			req.Name = "服务{告警}"
			req.ForTime = "0s"
			req.Labels = labels
			req.Annotations = nil
			before := *req
			if err := service.CreateMonitorAlertRule(context.Background(), req); err != nil {
				t.Fatal(err)
			}
			if store.row.Name != req.Name || store.row.ForTime != "0s" || !reflect.DeepEqual(store.row.Labels, labels) || !reflect.DeepEqual(*req, before) {
				t.Fatal("valid payload was normalized or mutated")
			}
		})
	}
}
func TestAlertRulePublicExpressionSyntaxCheckerKeepsItsContract(t *testing.T) {
	service, _, _ := namespaceService()
	for _, expression := range []string{"up", "1 + 2", "up[5m]", `"hello"`} {
		ok, err := service.PromqlExprCheck(context.Background(), &model.PromqlAlertRuleExprCheckReq{PromqlExpr: expression})
		if err != nil || !ok {
			t.Fatal("generic public PromQL syntax checker changed")
		}
	}
}
