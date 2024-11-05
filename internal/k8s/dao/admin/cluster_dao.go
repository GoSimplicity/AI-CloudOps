package admin

type ClusterDAO interface {
}

type clusterDAO struct {
}

func NewClusterDAO() ClusterDAO {
	return &clusterDAO{}
}
