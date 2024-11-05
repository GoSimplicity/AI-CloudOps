package admin

type YamlDAO interface {
}

type yamlDAO struct {
}

func NewYamlDAO() YamlDAO {
	return &yamlDAO{}
}
