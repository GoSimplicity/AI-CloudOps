package admin

type PodDAO interface {
}

type podDAO struct {
}

func NewPodDAO() PodDAO {
	return &podDAO{}
}
