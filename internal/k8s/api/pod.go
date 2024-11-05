package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sPodHandler struct {
	l          *zap.Logger
	podService admin.PodService
}

func NewK8sPodHandler(l *zap.Logger, podService admin.PodService) *K8sPodHandler {
	return &K8sPodHandler{
		l:          l,
		podService: podService,
	}
}

func (k *K8sPodHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// Pod 相关路由
	pods := k8sGroup.Group("/pods")
	{
		pods.GET("/", k.GetPodListByNamespace)                               // 根据命名空间获取 Pods 列表
		pods.GET("/:podName/containers", k.GetPodContainers)                 // 获取指定 Pod 的容器列表
		pods.GET("/:podName/containers/:container/logs", k.GetContainerLogs) // 获取指定容器的日志
		pods.GET("/:podName/yaml", k.GetPodYaml)                             // 获取指定 Pod 的 YAML 配置
		pods.POST("/", k.CreatePod)                                          // 创建新的 Pod
		pods.PUT("/", k.UpdatePod)                                           // 更新指定名称的 Pod
		pods.DELETE("/", k.DeletePod)                                        // 删除指定名称的 Pod
	}
}

// GetPodListByNamespace 根据命名空间获取 Pods 列表
func (k *K8sPodHandler) GetPodListByNamespace(ctx *gin.Context) {
	idStr := ctx.Query("id")
	if idStr == "" {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	clusterID, err := strconv.Atoi(idStr)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 参数必须为整数")
		return
	}

	// 可选参数：按 Pod 名称过滤
	podName := ctx.Query("podName") // 例如，?podName=my-pod

	pods, err := k.podService.GetPodsByNamespace(ctx.Request.Context(), clusterID, namespace, podName)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, pods)
}

// GetPodContainers 获取 Pod 的容器列表
func (k *K8sPodHandler) GetPodContainers(ctx *gin.Context) {
	idStr := ctx.Query("id")
	if idStr == "" {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	clusterID, err := strconv.Atoi(idStr)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 参数必须为整数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	podName := ctx.Param("podName")
	if podName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'podName' 参数")
		return
	}

	containers, err := k.podService.GetContainersByPod(ctx.Request.Context(), clusterID, namespace, podName)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, containers)
}

// GetPodsListByNodeName 获取指定节点上的 Pods 列表
func (k *K8sPodHandler) GetPodsListByNodeName(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		apiresponse.BadRequestError(ctx, "'id' 参数缺失")
		return
	}

	clusterID, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 非整数")
		return
	}

	name := ctx.Param("name")
	if name == "" {
		apiresponse.BadRequestError(ctx, "'name' 参数缺失")
		return
	}

	node, err := k.podService.GetPodsByNodeName(ctx, clusterID, name)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, node)
}

// GetContainerLogs 获取容器日志
func (k *K8sPodHandler) GetContainerLogs(ctx *gin.Context) {
	idStr := ctx.Query("id")
	if idStr == "" {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	namespace := ctx.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	clusterID, err := strconv.Atoi(idStr)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 参数必须为整数")
		return
	}

	podName := ctx.Param("podName")
	if podName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'podName' 参数")
		return
	}

	containerName := ctx.Param("container")
	if containerName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'container' 参数")
		return
	}

	logs, err := k.podService.GetContainerLogs(ctx.Request.Context(), clusterID, namespace, podName, containerName)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, logs)
}

// GetPodYaml 获取 Pod 的 YAML 配置
func (k *K8sPodHandler) GetPodYaml(c *gin.Context) {
	idStr := c.Query("id")
	if idStr == "" {
		apiresponse.BadRequestError(c, "缺少 'id' 参数")
		return
	}

	clusterID, err := strconv.Atoi(idStr)
	if err != nil {
		apiresponse.BadRequestError(c, "'id' 参数必须为整数")
		return
	}

	namespace := c.Query("namespace")
	if namespace == "" {
		apiresponse.BadRequestError(c, "缺少 'namespace' 参数")
		return
	}

	podName := c.Param("podName")
	if podName == "" {
		apiresponse.BadRequestError(c, "缺少 'podName' 参数")
		return
	}

	pod, err := k.podService.GetPodYaml(c.Request.Context(), clusterID, namespace, podName)
	if err != nil {
		apiresponse.InternalServerError(c, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(c, pod)
}

// CreatePod 创建新的 Pod
func (k *K8sPodHandler) CreatePod(c *gin.Context) {
	var req model.K8sPodRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(c, "绑定数据失败", err.Error())
		return
	}

	if err := k.podService.CreatePod(c.Request.Context(), &req); err != nil {
		apiresponse.InternalServerError(c, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(c)
}

// UpdatePod 更新指定名称的 Pod
func (k *K8sPodHandler) UpdatePod(c *gin.Context) {
	var req model.K8sPodRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(c, "绑定数据失败", err.Error())
		return
	}

	if err := k.podService.UpdatePod(c.Request.Context(), &req); err != nil {
		apiresponse.InternalServerError(c, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(c)
}

// DeletePod 删除指定名称的 Pod
func (k *K8sPodHandler) DeletePod(c *gin.Context) {
	var req model.K8sPodRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(c, "绑定数据失败", err.Error())
		return
	}

	if err := k.podService.DeletePod(c.Request.Context(), req.ClusterName, req.Pod.Namespace, req.Pod.Name); err != nil {
		apiresponse.InternalServerError(c, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(c)
}
