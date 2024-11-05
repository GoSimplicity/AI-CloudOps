package admin

type SvcDAO interface {
}

type svcDAO struct {
}

func NewSvcDAO() SvcDAO {
	return &svcDAO{}
}
