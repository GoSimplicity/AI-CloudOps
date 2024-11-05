package admin

type NodeDAO interface {
}

type nodeDAO struct {
}

func NewNodeDAO() NodeDAO {
	return &nodeDAO{}
}
