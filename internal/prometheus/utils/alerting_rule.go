package utils

import (
	"errors"
	"strings"
	"unicode/utf8"

	prommodel "github.com/prometheus/common/model"
	"github.com/prometheus/prometheus/model/rulefmt"
	"github.com/prometheus/prometheus/promql/parser"
)

// BuildAlertingRule is shared by persistence validation and generated configuration.
// Alert names are label values, so UTF-8 and braces remain valid; recording-rule
// metric-name restrictions must not be applied here.
func BuildAlertingRule(name, expression, forTime string, labels, annotations []string) (rulefmt.Rule, error) {
	if name == "" || !utf8.ValidString(name) {
		return rulefmt.Rule{}, errors.New("告警规则名称无效")
	}
	parsed, err := parser.NewParser(parser.Options{}).ParseExpr(expression)
	if err != nil {
		return rulefmt.Rule{}, errors.New("告警规则表达式不是有效的PromQL")
	}
	if parsed.Type() != parser.ValueTypeVector && parsed.Type() != parser.ValueTypeScalar {
		return rulefmt.Rule{}, errors.New("告警规则表达式必须返回即时向量或标量")
	}
	duration, err := prommodel.ParseDuration(forTime)
	if err != nil || forTime == "" || duration < 0 {
		return rulefmt.Rule{}, errors.New("告警规则持续时间无效")
	}
	labelValues, err := parseAlertRuleValues(labels)
	if err != nil {
		return rulefmt.Rule{}, errors.New("告警规则标签无效，请使用合法的key=value")
	}
	if _, reserved := labelValues[prommodel.MetricNameLabel]; reserved {
		return rulefmt.Rule{}, errors.New("告警规则标签不能覆盖指标名称")
	}
	annotationValues, err := parseAlertRuleValues(annotations)
	if err != nil {
		return rulefmt.Rule{}, errors.New("告警规则注解无效，请使用合法的key=value")
	}
	rule := rulefmt.Rule{Alert: name, Expr: expression, For: duration, Labels: labelValues, Annotations: annotationValues}
	// This invokes the fixed Prometheus implementation, including its template
	// functions and predefined $labels/$value variables, without executing queries.
	if problems := rule.Validate(rulefmt.RuleNode{}, prommodel.UTF8Validation, parser.NewParser(parser.Options{})); len(problems) > 0 {
		return rulefmt.Rule{}, errors.New("告警规则标签或注解模板无效")
	}
	return rule, nil
}
func parseAlertRuleValues(entries []string) (map[string]string, error) {
	if len(entries) == 0 {
		return nil, nil
	}
	values := make(map[string]string, len(entries))
	for _, entry := range entries {
		key, value, found := strings.Cut(entry, "=")
		if !found || !prommodel.UTF8Validation.IsValidLabelName(key) || !prommodel.LabelValue(value).IsValid() {
			return nil, errors.New("无效的key=value条目")
		}
		values[key] = value
	}
	return values, nil
}
