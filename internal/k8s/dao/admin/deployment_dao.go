package admin

type DeploymentDAO interface {
}

type deploymentDAO struct {
}

func NewDeploymentDAO() DeploymentDAO {
	return &deploymentDAO{}
}
