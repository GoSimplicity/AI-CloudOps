package onduty

type PrometheusOnDutyService interface {
}

type prometheusOnDutyService struct {
}

func NewPrometheusOnDutyService() PrometheusOnDutyService {
	return &prometheusOnDutyService{}
}
