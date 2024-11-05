package admin

type ConfigMapDAO interface {
}

type configMapDAO struct {
}

func NewConfigMapDAO() ConfigMapDAO {
	return &configMapDAO{}
}
