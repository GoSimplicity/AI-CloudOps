package job

import "github.com/hibiken/asynq"

type Routes struct {
	CreateK8sCluster *CreateK8sClusterTask
	UpdateK8sCluster *UpdateK8sClusterTask
}

func NewRoutes(createK8sCluster *CreateK8sClusterTask, updateK8sCluster *UpdateK8sClusterTask) *Routes {
	return &Routes{
		CreateK8sCluster: createK8sCluster,
		UpdateK8sCluster: updateK8sCluster,
	}
}

func (r *Routes) RegisterHandlers() *asynq.ServeMux {
	mux := asynq.NewServeMux()

	mux.HandleFunc(DeferCreateK8sCluster, r.CreateK8sCluster.ProcessTask)
	mux.HandleFunc(DeferUpdateK8sCluster, r.UpdateK8sCluster.ProcessTask)

	return mux
}
