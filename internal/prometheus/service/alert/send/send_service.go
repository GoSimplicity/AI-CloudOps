package send

type PrometheusSendService interface {
}

type prometheusSendService struct {
}

func NewPrometheusSendService() PrometheusSendService {
	return &prometheusSendService{}
}
