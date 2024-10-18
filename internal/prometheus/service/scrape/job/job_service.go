package job

type PrometheusScrapeService interface {
}

type prometheusScrapeService struct {
}

func NewPrometheusScrapeService() PrometheusScrapeService {
	return &prometheusScrapeService{}
}
