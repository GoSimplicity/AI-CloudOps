package admin

type NamespaceDAO interface {
}

type namespaceDAO struct {
}

func NewNamespaceDAO() NamespaceDAO {
	return &namespaceDAO{}
}
