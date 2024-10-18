package rule

type PrometheusRuleService interface {
}

type prometheusRuleService struct {
}

func NewPrometheusRuleService() PrometheusRuleService {
	return &prometheusRuleService{}
}
