package record

type PrometheusRecordService interface {
}

type prometheusRecordService struct {
}

func NewPrometheusRecordService() PrometheusRecordService {
	return &prometheusRecordService{}
}
