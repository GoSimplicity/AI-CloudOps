package event

type PrometheusEventService interface {
}

type prometheusEventService struct {
}

func NewPrometheusEventService() PrometheusEventService {
	return &prometheusEventService{}
}
