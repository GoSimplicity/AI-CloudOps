package pool

type PrometheusPoolService interface {
}

type prometheusPoolService struct {
}

func NewPrometheusPoolService() PrometheusPoolService {
	return &prometheusPoolService{}
}
