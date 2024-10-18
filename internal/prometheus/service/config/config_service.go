package config

type PrometheusConfigService interface {
}

type prometheusConfigService struct {
}

func NewPrometheusConfigService() PrometheusConfigService {
	return &prometheusConfigService{}
}
