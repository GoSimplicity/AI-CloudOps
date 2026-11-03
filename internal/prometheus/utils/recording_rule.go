package utils

import (
	"errors"
	"strings"

	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
)

// BuildRecordingRule applies the same contract at save and configuration generation.
// Prometheus defaults to UTF-8 names; a legacy ASCII-only regex rejects valid names.
func BuildRecordingRule(name, expression string, labels []string) (rulefmt.Rule, error) {
	if strings.TrimSpace(name) == "" || !prommodel.UTF8Validation.IsValidMetricName(name) || strings.ContainsAny(name, "{}") {
		return rulefmt.Rule{}, errors.New("记录规则名称无效")
	}
	parsed, err := parser.NewParser(parser.Options{}).ParseExpr(expression)
	if err != nil {
		return rulefmt.Rule{}, errors.New("记录规则表达式不是有效的PromQL")
	}
	if parsed.Type() != parser.ValueTypeVector && parsed.Type() != parser.ValueTypeScalar {
		return rulefmt.Rule{}, errors.New("记录规则表达式必须返回即时向量或标量")
	}
	values := make(map[string]string, len(labels))
	for _, entry := range labels {
		key, value, found := strings.Cut(entry, "=")
		if !found || key == prommodel.MetricNameLabel || !prommodel.UTF8Validation.IsValidLabelName(key) || !prommodel.LabelValue(value).IsValid() {
			return rulefmt.Rule{}, errors.New("记录规则标签无效，请使用合法的key=value")
		}
		values[key] = value
	}
	return rulefmt.Rule{Record: name, Expr: expression, Labels: values}, nil
}
