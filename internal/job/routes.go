package job

import "github.com/hibiken/asynq"

type Routes struct {
	CreateK8sCluster *CreateK8sClusterTask
	UpdateK8sCluster *UpdateK8sClusterTask
	TimedTask        *TimedTask
}

func NewRoutes(createK8sCluster *CreateK8sClusterTask, updateK8sCluster *UpdateK8sClusterTask, timedTask *TimedTask) *Routes {
	return &Routes{
		CreateK8sCluster: createK8sCluster,
		UpdateK8sCluster: updateK8sCluster,
		TimedTask:        timedTask,
	}
}

func (r *Routes) RegisterHandlers() *asynq.ServeMux {
	mux := asynq.NewServeMux()

	// 注册任务
	mux.HandleFunc(DeferCreateK8sCluster, r.CreateK8sCluster.ProcessTask)
	mux.HandleFunc(DeferUpdateK8sCluster, r.UpdateK8sCluster.ProcessTask)
	mux.HandleFunc(DeferTimedTask, r.TimedTask.ProcessTask)

	return mux
}
