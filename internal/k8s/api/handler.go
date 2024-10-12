package api

import (
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sHandler struct {
	service service.K8sService
	l       *zap.Logger
}

// NewK8sHandler 创建一个新的 K8sHandler
func NewK8sHandler(service service.K8sService, l *zap.Logger) *K8sHandler {
	return &K8sHandler{
		service: service,
		l:       l,
	}
}

// RegisterRouters 注册所有 Kubernetes 相关的路由
func (k *K8sHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// 集群相关路由
	clusters := k8sGroup.Group("/clusters")
	{
		clusters.GET("/", k.GetClusterList)                // 获取集群列表
		clusters.GET("/select", k.GetClusterListForSelect) // 获取用于选择的集群列表
		clusters.POST("/", k.CreateCluster)                // 创建新的集群
		clusters.PUT("/:id", k.UpdateCluster)              // 更新指定 ID 的集群
		clusters.DELETE("/:id", k.DeleteCluster)           // 删除指定 ID 的集群
	}

	// 节点相关路由
	nodes := k8sGroup.Group("/nodes")
	{
		nodes.GET("/", k.GetNodeList)                     // 获取节点列表
		nodes.GET("/:name", k.GetNodeDetail)              // 获取指定名称的节点详情
		nodes.GET("/:name/pods", k.GetPodsListByNodeName) // 获取指定节点上的 Pods 列表

		nodes.POST("/taint_check", k.TaintYamlCheck)              // 检查节点 Taint 的 YAML 配置
		nodes.POST("/enable_switch", k.ScheduleEnableSwitchNodes) // 启用或切换节点调度
		nodes.POST("/labels/add", k.AddLabelNodes)                // 为节点添加标签
		nodes.DELETE("/labels/delete", k.DeleteLabelNodes)        // 删除节点标签
		nodes.POST("/taints/add", k.AddTaintsNodes)               // 为节点添加 Taint
		nodes.DELETE("/taints/delete", k.DeleteTaintsNodes)       // 删除节点 Taint
		nodes.POST("/drain", k.DrainPods)                         // 清空节点上的 Pods
	}

	// YAML 模板相关路由
	yamlTemplates := k8sGroup.Group("/yaml-templates")
	{
		yamlTemplates.GET("/", k.GetYamlTemplateList)      // 获取 YAML 模板列表
		yamlTemplates.POST("/", k.CreateYamlTemplate)      // 创建新的 YAML 模板
		yamlTemplates.PUT("/:id", k.UpdateYamlTemplate)    // 更新指定 ID 的 YAML 模板
		yamlTemplates.DELETE("/:id", k.DeleteYamlTemplate) // 删除指定 ID 的 YAML 模板
	}

	// YAML 任务相关路由
	yamlTasks := k8sGroup.Group("/yaml-tasks")
	{
		yamlTasks.GET("/", k.GetYamlTaskList)         // 获取 YAML 任务列表
		yamlTasks.POST("/", k.CreateYamlTask)         // 创建新的 YAML 任务
		yamlTasks.PUT("/:id", k.UpdateYamlTask)       // 更新指定 ID 的 YAML 任务
		yamlTasks.POST("/:id/apply", k.ApplyYamlTask) // 应用指定 ID 的 YAML 任务
		yamlTasks.DELETE("/:id", k.DeleteYamlTask)    // 删除指定 ID 的 YAML 任务
	}

	// 命名空间相关路由
	namespaces := k8sGroup.Group("/namespaces")
	{
		namespaces.GET("/cascade", k.GetClusterNamespacesForCascade) // 获取级联选择的命名空间列表
		namespaces.GET("/select", k.GetClusterNamespacesForSelect)   // 获取用于选择的命名空间列表
	}

	// Pod 相关路由
	pods := k8sGroup.Group("/pods")
	{
		pods.GET("/", k.GetPodListByNamespace)                               // 根据命名空间获取 Pods 列表
		pods.GET("/:podName/containers", k.GetPodContainers)                 // 获取指定 Pod 的容器列表
		pods.GET("/:podName/containers/:container/logs", k.GetContainerLogs) // 获取指定容器的日志
		pods.GET("/:podName/yaml", k.GetPodYaml)                             // 获取指定 Pod 的 YAML 配置

		pods.POST("/", k.CreatePod)   // 创建新的 Pod
		pods.PUT("/", k.UpdatePod)    // 更新指定名称的 Pod
		pods.DELETE("/", k.DeletePod) // 删除指定名称的 Pod
	}

	// Deployment 相关路由
	deployments := k8sGroup.Group("/deployments")
	{
		deployments.GET("/", k.GetDeployListByNamespace) // 根据命名空间获取部署列表
		deployments.POST("/", k.CreateDeployment)        // 创建新的部署
		deployments.PUT("/", k.UpdateDeployment)         // 更新指定 deploymentName 的部署
		deployments.DELETE("/", k.DeleteDeployment)      // 删除指定 deploymentName 的部署

		deployments.PUT("/:name/image", k.SetDeploymentContainerImage) // 设置部署中容器的镜像
		deployments.POST("/:name/scale", k.ScaleDeployment)            // 扩缩指定 ID 的部署
		deployments.POST("/restart", k.BatchRestartDeployments)        // 批量重启部署
		deployments.GET("/:name/yaml", k.GetDeployYaml)                // 获取指定部署的 YAML 配置
	}

	// ConfigMap 相关路由
	configMaps := k8sGroup.Group("/configmaps")
	{
		configMaps.GET("/", k.GetConfigMapListByNamespace) // 根据命名空间获取 ConfigMap 列表
		configMaps.POST("/", k.CreateConfigMap)            // 创建新的 ConfigMap
		configMaps.PUT("/:id", k.UpdateConfigMap)          // 更新指定 ID 的 ConfigMap
		configMaps.PUT("/:id/data", k.UpdateConfigMapData) // 更新指定 ID 的 ConfigMap 数据
		configMaps.GET("/:id/yaml", k.GetConfigMapYaml)    // 获取指定 ConfigMap 的 YAML 配置
		configMaps.DELETE("/:id", k.DeleteConfigMap)       // 删除指定 ID 的 ConfigMap
		configMaps.DELETE("/", k.BatchDeleteConfigMaps)    // 批量删除 ConfigMap
	}

	// Service 相关路由
	services := k8sGroup.Group("/services")
	{
		services.GET("/", k.GetServiceListByNamespace) // 根据命名空间获取 Service 列表
		services.GET("/:id/yaml", k.GetServiceYaml)    // 获取指定 Service 的 YAML 配置
		services.POST("/", k.CreateOrUpdateService)    // 创建或更新 Service
		services.PUT("/:id", k.UpdateService)          // 更新指定 ID 的 Service
		services.DELETE("/:id", k.DeleteService)       // 删除指定 ID 的 Service
		services.DELETE("/", k.BatchDeleteServices)    // 批量删除 Service
	}

	// 普通运维相关路由（K8sApp）
	k8sAppApiGroup := k8sGroup.Group("/k8sApp")
	{
		// 命名空间
		k8sAppApiGroup.GET("/namespaces/unique", k.GetClusterNamespacesUnique) // 获取唯一的命名空间列表

		// 实例
		instances := k8sAppApiGroup.Group("/instances")
		{
			instances.POST("/", k.CreateK8sInstanceOne)           // 创建单个 Kubernetes 实例
			instances.PUT("/", k.UpdateK8sInstanceOne)            // 更新单个 Kubernetes 实例
			instances.DELETE("/", k.BatchDeleteK8sInstance)       // 批量删除 Kubernetes 实例
			instances.POST("/restart", k.BatchRestartK8sInstance) // 批量重启 Kubernetes 实例
			instances.GET("/by-app", k.GetK8sInstanceByApp)       // 根据应用获取 Kubernetes 实例
			instances.GET("/", k.GetK8sInstanceList)              // 获取 Kubernetes 实例列表
			instances.GET("/:id", k.GetK8sInstanceOne)            // 获取单个 Kubernetes 实例
		}

		// 应用 Deployment 和 Service 的抽象
		apps := k8sAppApiGroup.Group("/apps")
		{
			apps.GET("/", k.GetK8sAppList)                 // 获取 Kubernetes 应用列表
			apps.POST("/", k.CreateK8sAppOne)              // 创建单个 Kubernetes 应用
			apps.PUT("/:id", k.UpdateK8sAppOne)            // 更新单个 Kubernetes 应用
			apps.DELETE("/:id", k.DeleteK8sAppOne)         // 删除单个 Kubernetes 应用
			apps.GET("/:id", k.GetK8sAppOne)               // 获取单个 Kubernetes 应用
			apps.GET("/:id/pods", k.GetK8sPodListByDeploy) // 根据部署获取 Kubernetes Pod 列表
			apps.GET("/select", k.GetK8sAppListForSelect)  // 获取用于选择的 Kubernetes 应用列表
		}

		// 项目
		projects := k8sAppApiGroup.Group("/projects")
		{
			projects.GET("/", k.GetK8sProjectList)                // 获取 Kubernetes 项目列表
			projects.GET("/select", k.GetK8sProjectListForSelect) // 获取用于选择的 Kubernetes 项目列表
			projects.POST("/", k.CreateK8sProject)                // 创建 Kubernetes 项目
			projects.PUT("/", k.UpdateK8sProject)                 // 更新 Kubernetes 项目
			projects.DELETE("/:id", k.DeleteK8sProjectOne)        // 删除单个 Kubernetes 项目
		}

		// CronJob
		cronJobs := k8sAppApiGroup.Group("/cronJobs")
		{
			cronJobs.GET("/", k.GetK8sCronjobList)                // 获取 CronJob 列表
			cronJobs.POST("/", k.CreateK8sCronjobOne)             // 创建单个 CronJob
			cronJobs.PUT("/:id", k.UpdateK8sCronjobOne)           // 更新单个 CronJob
			cronJobs.GET("/:id", k.GetK8sCronjobOne)              // 获取单个 CronJob
			cronJobs.GET("/:id/last-pod", k.GetK8sCronjobLastPod) // 获取 CronJob 最近的 Pod
			cronJobs.DELETE("/", k.BatchDeleteK8sCronjob)         // 批量删除 CronJob
		}
	}
}

// GetClusterList 获取集群列表
func (k *K8sHandler) GetClusterList(ctx *gin.Context) {
	clusters, err := k.service.ListAllClusters(ctx)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, clusters)
}

// GetClusterListForSelect 获取用于选择的集群列表
func (k *K8sHandler) GetClusterListForSelect(ctx *gin.Context) {
	// TODO: 实现获取用于选择的集群列表的逻辑
}

// CreateCluster 创建新的集群
func (k *K8sHandler) CreateCluster(ctx *gin.Context) {
	var cluster *model.K8sCluster

	uc := ctx.MustGet("user").(ijwt.UserClaims)

	err := ctx.ShouldBind(&cluster)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "参数错误")
		return
	}

	cluster.UserID = uc.Uid

	err = k.service.CreateCluster(ctx, cluster)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateCluster 更新指定 ID 的集群
func (k *K8sHandler) UpdateCluster(ctx *gin.Context) {
	// TODO: 实现更新集群的逻辑
}

// DeleteCluster 删除指定 ID 的集群
func (k *K8sHandler) DeleteCluster(ctx *gin.Context) {
	// TODO: 实现删除集群的逻辑
}

// GetNodeList 获取节点列表
func (k *K8sHandler) GetNodeList(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	clusterID, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 非整数")
		return
	}

	nodes, err := k.service.ListAllNodes(ctx, clusterID)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, nodes)
}

// GetNodeDetail 获取指定名称的节点详情
func (k *K8sHandler) GetNodeDetail(ctx *gin.Context) {
	id := ctx.Query("id")
	if id == "" {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	clusterID, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 非整数")
		return
	}

	name := ctx.Param("name")
	if name == "" {
		apiresponse.BadRequestError(ctx, "缺少 'name' 参数")
		return
	}

	node, err := k.service.GetNodeByName(ctx, clusterID, name)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, node)
}

// GetPodsListByNodeName 获取指定节点上的 Pods 列表
func (k *K8sHandler) GetPodsListByNodeName(ctx *gin.Context) {
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

	node, err := k.service.GetPodsByNodeName(ctx, clusterID, name)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, node)
}

// TaintYamlCheck 检查节点 Taint 的 YAML 配置
func (k *K8sHandler) TaintYamlCheck(ctx *gin.Context) {
	var req model.TaintK8sNodesRequest

	if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.service.CheckTaintYaml(ctx, &req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "Taint YAML 配置检查失败")
		return
	}

	apiresponse.Success(ctx)
}

// ScheduleEnableSwitchNodes 启用或切换节点调度
func (k *K8sHandler) ScheduleEnableSwitchNodes(ctx *gin.Context) {
	var req model.ScheduleK8sNodesRequest

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.service.BatchEnableSwitchNodes(ctx, &req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "启用或切换节点调度失败")
		return
	}

	apiresponse.Success(ctx)
}

// AddLabelNodes 为节点添加标签
func (k *K8sHandler) AddLabelNodes(ctx *gin.Context) {
	var label *model.LabelK8sNodesRequest

	err := ctx.ShouldBind(&label)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.service.UpdateNodeLabel(ctx, label); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

// DeleteLabelNodes 删除节点标签
func (k *K8sHandler) DeleteLabelNodes(ctx *gin.Context) {
	var label *model.LabelK8sNodesRequest

	err := ctx.ShouldBind(&label)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.service.UpdateNodeLabel(ctx, label); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

// AddTaintsNodes 为节点添加 Taint
func (k *K8sHandler) AddTaintsNodes(ctx *gin.Context) {
	var taint *model.TaintK8sNodesRequest

	err := ctx.ShouldBind(&taint)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.service.UpdateNodeTaint(ctx, taint); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

// DeleteTaintsNodes 删除节点 Taint
func (k *K8sHandler) DeleteTaintsNodes(ctx *gin.Context) {
	var taint *model.TaintK8sNodesRequest

	err := ctx.ShouldBind(&taint)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.service.UpdateNodeTaint(ctx, taint); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

// DrainPods 清空节点上的 Pods
func (k *K8sHandler) DrainPods(ctx *gin.Context) {
	var req model.K8sClusterNodesRequest

	err := ctx.ShouldBind(&req)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.service.DrainPods(ctx, &req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

// GetYamlTemplateList 获取 YAML 模板列表
func (k *K8sHandler) GetYamlTemplateList(ctx *gin.Context) {
	list, err := k.service.GetYamlTemplateList(ctx)
	if err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateYamlTemplate 创建新的 YAML 模板
func (k *K8sHandler) CreateYamlTemplate(ctx *gin.Context) {
	var req model.K8sYamlTemplate

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid

	if err := k.service.CreateYamlTemplate(ctx, &req); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateYamlTemplate 更新指定 ID 的 YAML 模板
func (k *K8sHandler) UpdateYamlTemplate(ctx *gin.Context) {
	var req model.K8sYamlTemplate

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	id := ctx.Param("id")
	if id == "" {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	yamlId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 非整数")
		return
	}

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.ID = yamlId
	req.UserID = uc.Uid

	if err := k.service.UpdateYamlTemplate(ctx, &req); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteYamlTemplate 删除指定 ID 的 YAML 模板
func (k *K8sHandler) DeleteYamlTemplate(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	yamlId, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 非整数")
		return
	}

	if err := k.service.DeleteYamlTemplate(ctx, yamlId); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetYamlTaskList 获取 YAML 任务列表
func (k *K8sHandler) GetYamlTaskList(ctx *gin.Context) {
	list, err := k.service.GetYamlTaskList(ctx)
	if err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateYamlTask 创建新的 YAML 任务
func (k *K8sHandler) CreateYamlTask(ctx *gin.Context) {
	var req model.K8sYamlTask

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid

	if err := k.service.CreateYamlTask(ctx, &req); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateYamlTask 更新指定 ID 的 YAML 任务
func (k *K8sHandler) UpdateYamlTask(ctx *gin.Context) {
	var req model.K8sYamlTask

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	id := ctx.Param("id")
	if id == "" {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	taskID, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 非整数")
		return
	}

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.ID = taskID
	req.UserID = uc.Uid

	if err := k.service.UpdateYamlTask(ctx, &req); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// ApplyYamlTask 应用指定 ID 的 YAML 任务
func (k *K8sHandler) ApplyYamlTask(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	taskID, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 非整数")
		return
	}

	if err := k.service.ApplyYamlTask(ctx, taskID); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteYamlTask 删除指定 ID 的 YAML 任务
func (k *K8sHandler) DeleteYamlTask(ctx *gin.Context) {
	id := ctx.Param("id")
	if id == "" {
		apiresponse.BadRequestError(ctx, "缺少 'id' 参数")
		return
	}

	taskID, err := strconv.Atoi(id)
	if err != nil {
		apiresponse.BadRequestError(ctx, "'id' 非整数")
		return
	}

	if err := k.service.DeleteYamlTask(ctx, taskID); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetClusterNamespacesForCascade 获取级联选择的命名空间列表
func (k *K8sHandler) GetClusterNamespacesForCascade(ctx *gin.Context) {
	namespaces, err := k.service.GetClusterNamespacesList(ctx)
	if err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, namespaces)
}

// GetClusterNamespacesForSelect 获取用于选择的命名空间列表
func (k *K8sHandler) GetClusterNamespacesForSelect(ctx *gin.Context) {
	namespace := ctx.Query("namespace")

	if namespace == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	namespaces, err := k.service.GetClusterNamespacesByName(ctx, namespace)
	if err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, namespaces)
}

// GetPodListByNamespace 根据命名空间获取 Pods 列表
func (k *K8sHandler) GetPodListByNamespace(ctx *gin.Context) {
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

	pods, err := k.service.GetPodsByNamespace(ctx.Request.Context(), clusterID, namespace, podName)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, pods)
}

// GetPodContainers 获取 Pod 的容器列表
func (k *K8sHandler) GetPodContainers(ctx *gin.Context) {
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

	containers, err := k.service.GetContainersByPod(ctx.Request.Context(), clusterID, namespace, podName)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, containers)
}

// GetContainerLogs 获取容器日志
func (k *K8sHandler) GetContainerLogs(ctx *gin.Context) {
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

	logs, err := k.service.GetContainerLogs(ctx.Request.Context(), clusterID, namespace, podName, containerName)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, logs)
}

// GetPodYaml 获取 Pod 的 YAML 配置
func (k *K8sHandler) GetPodYaml(c *gin.Context) {
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

	pod, err := k.service.GetPodYaml(c.Request.Context(), clusterID, namespace, podName)
	if err != nil {
		apiresponse.InternalServerError(c, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(c, pod)
}

// CreatePod 创建新的 Pod
func (k *K8sHandler) CreatePod(c *gin.Context) {
	var req model.K8sPodRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(c, "绑定数据失败", err.Error())
		return
	}

	if err := k.service.CreatePod(c.Request.Context(), &req); err != nil {
		apiresponse.InternalServerError(c, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(c)
}

// UpdatePod 更新指定名称的 Pod
func (k *K8sHandler) UpdatePod(c *gin.Context) {
	var req model.K8sPodRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(c, "绑定数据失败", err.Error())
		return
	}

	if err := k.service.UpdatePod(c.Request.Context(), &req); err != nil {
		apiresponse.InternalServerError(c, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(c)
}

// DeletePod 删除指定名称的 Pod
func (k *K8sHandler) DeletePod(c *gin.Context) {
	var req model.K8sPodRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(c, "绑定数据失败", err.Error())
		return
	}

	if err := k.service.DeletePod(c.Request.Context(), req.ClusterName, req.Pod.Namespace, req.Pod.Name); err != nil {
		apiresponse.InternalServerError(c, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(c)
}

// GetDeployListByNamespace 根据命名空间获取部署列表
func (k *K8sHandler) GetDeployListByNamespace(ctx *gin.Context) {
	namesapce := ctx.Query("namespace")
	if namesapce == "" {
		apiresponse.BadRequestError(ctx, "缺少 'namespace' 参数")
		return
	}

	clusterName := ctx.Query("cluster_name")
	if clusterName == "" {
		apiresponse.BadRequestError(ctx, "缺少 'cluster_name' 参数")
		return
	}

	deploys, err := k.service.GetDeploymentsByNamespace(ctx, clusterName, namesapce)
	if err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, deploys)
}

// CreateDeployment 创建新的部署
func (k *K8sHandler) CreateDeployment(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.service.CreateDeployment(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateDeployment 更新指定 Name 的部署
func (k *K8sHandler) UpdateDeployment(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.service.UpdateDeployment(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteDeployment 删除指定 Name 的部署
func (k *K8sHandler) DeleteDeployment(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.service.DeleteDeployment(ctx, req.ClusterName, req.Namespace, req.DeploymentNames[0]); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// SetDeploymentContainerImage 设置部署中容器的镜像
func (k *K8sHandler) SetDeploymentContainerImage(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.service.UpdateDeployment(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// ScaleDeployment 扩缩部署
func (k *K8sHandler) ScaleDeployment(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.service.UpdateDeployment(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// BatchRestartDeployments 批量重启部署
func (k *K8sHandler) BatchRestartDeployments(ctx *gin.Context) {
	var req model.K8sDeploymentRequest

	if err := ctx.ShouldBindJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, "绑定数据失败", err.Error())
		return
	}

	if err := k.service.BatchRestartDeployments(ctx, &req); err != nil {
		apiresponse.InternalServerError(ctx, 500, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// GetDeployYaml 获取部署的 YAML 配置
func (k *K8sHandler) GetDeployYaml(ctx *gin.Context) {
	// TODO: 实现获取部署的 YAML 配置的逻辑
}

// GetConfigMapListByNamespace 根据命名空间获取 ConfigMap 列表
func (k *K8sHandler) GetConfigMapListByNamespace(ctx *gin.Context) {
	// TODO: 实现根据命名空间获取 ConfigMap 列表的逻辑
}

// CreateConfigMap 创建新的 ConfigMap
func (k *K8sHandler) CreateConfigMap(ctx *gin.Context) {
	// TODO: 实现创建 ConfigMap 的逻辑
}

// UpdateConfigMap 更新指定 ID 的 ConfigMap
func (k *K8sHandler) UpdateConfigMap(ctx *gin.Context) {
	// TODO: 实现更新 ConfigMap 的逻辑
}

// UpdateConfigMapData 更新 ConfigMap 数据
func (k *K8sHandler) UpdateConfigMapData(ctx *gin.Context) {
	// TODO: 实现更新 ConfigMap 数据的逻辑
}

// GetConfigMapYaml 获取 ConfigMap 的 YAML 配置
func (k *K8sHandler) GetConfigMapYaml(ctx *gin.Context) {
	// TODO: 实现获取 ConfigMap 的 YAML 配置的逻辑
}

// DeleteConfigMap 删除指定 ID 的 ConfigMap
func (k *K8sHandler) DeleteConfigMap(ctx *gin.Context) {
	// TODO: 实现删除 ConfigMap 的逻辑
}

// BatchDeleteConfigMaps 批量删除 ConfigMap
func (k *K8sHandler) BatchDeleteConfigMaps(ctx *gin.Context) {
	// TODO: 实现批量删除 ConfigMap 的逻辑
}

// GetServiceListByNamespace 根据命名空间获取 Service 列表
func (k *K8sHandler) GetServiceListByNamespace(ctx *gin.Context) {
	// TODO: 实现根据命名空间获取 Service 列表的逻辑
}

// GetServiceYaml 获取 Service 的 YAML 配置
func (k *K8sHandler) GetServiceYaml(ctx *gin.Context) {
	// TODO: 实现获取 Service 的 YAML 配置的逻辑
}

// CreateOrUpdateService 创建或更新 Service
func (k *K8sHandler) CreateOrUpdateService(ctx *gin.Context) {
	// TODO: 实现创建或更新 Service 的逻辑
}

// UpdateService 更新指定 ID 的 Service
func (k *K8sHandler) UpdateService(ctx *gin.Context) {
	// TODO: 实现更新 Service 的逻辑
}

// DeleteService 删除指定 ID 的 Service
func (k *K8sHandler) DeleteService(ctx *gin.Context) {
	// TODO: 实现删除 Service 的逻辑
}

// BatchDeleteServices 批量删除 Service
func (k *K8sHandler) BatchDeleteServices(ctx *gin.Context) {
	// TODO: 实现批量删除 Service 的逻辑
}

// GetClusterNamespacesUnique 获取唯一的命名空间列表
func (k *K8sHandler) GetClusterNamespacesUnique(ctx *gin.Context) {
	// TODO: 实现获取唯一命名空间列表的逻辑
}

// CreateK8sInstanceOne 创建单个 Kubernetes 实例
func (k *K8sHandler) CreateK8sInstanceOne(ctx *gin.Context) {
	// TODO: 实现创建单个 Kubernetes 实例的逻辑
}

// UpdateK8sInstanceOne 更新单个 Kubernetes 实例
func (k *K8sHandler) UpdateK8sInstanceOne(ctx *gin.Context) {
	// TODO: 实现更新单个 Kubernetes 实例的逻辑
}

// BatchDeleteK8sInstance 批量删除 Kubernetes 实例
func (k *K8sHandler) BatchDeleteK8sInstance(ctx *gin.Context) {
	// TODO: 实现批量删除 Kubernetes 实例的逻辑
}

// BatchRestartK8sInstance 批量重启 Kubernetes 实例
func (k *K8sHandler) BatchRestartK8sInstance(ctx *gin.Context) {
	// TODO: 实现批量重启 Kubernetes 实例的逻辑
}

// GetK8sInstanceByApp 根据应用获取 Kubernetes 实例
func (k *K8sHandler) GetK8sInstanceByApp(ctx *gin.Context) {
	// TODO: 实现根据应用获取 Kubernetes 实例的逻辑
}

// GetK8sInstanceList 获取 Kubernetes 实例列表
func (k *K8sHandler) GetK8sInstanceList(ctx *gin.Context) {
	// TODO: 实现获取 Kubernetes 实例列表的逻辑
}

// GetK8sInstanceOne 获取单个 Kubernetes 实例
func (k *K8sHandler) GetK8sInstanceOne(ctx *gin.Context) {
	// TODO: 实现获取单个 Kubernetes 实例的逻辑
}

// GetK8sAppList 获取 Kubernetes 应用列表
func (k *K8sHandler) GetK8sAppList(ctx *gin.Context) {
	// TODO: 实现获取 Kubernetes 应用列表的逻辑
}

// CreateK8sAppOne 创建单个 Kubernetes 应用
func (k *K8sHandler) CreateK8sAppOne(ctx *gin.Context) {
	// TODO: 实现创建单个 Kubernetes 应用的逻辑
}

// UpdateK8sAppOne 更新单个 Kubernetes 应用
func (k *K8sHandler) UpdateK8sAppOne(ctx *gin.Context) {
	// TODO: 实现更新单个 Kubernetes 应用的逻辑
}

// DeleteK8sAppOne 删除单个 Kubernetes 应用
func (k *K8sHandler) DeleteK8sAppOne(ctx *gin.Context) {
	// TODO: 实现删除单个 Kubernetes 应用的逻辑
}

// GetK8sAppOne 获取单个 Kubernetes 应用
func (k *K8sHandler) GetK8sAppOne(ctx *gin.Context) {
	// TODO: 实现获取单个 Kubernetes 应用的逻辑
}

// GetK8sPodListByDeploy 根据部署获取 Kubernetes Pod 列表
func (k *K8sHandler) GetK8sPodListByDeploy(ctx *gin.Context) {
	// TODO: 实现根据部署获取 Kubernetes Pod 列表的逻辑
}

// GetK8sAppListForSelect 获取用于选择的 Kubernetes 应用列表
func (k *K8sHandler) GetK8sAppListForSelect(ctx *gin.Context) {
	// TODO: 实现获取用于选择的 Kubernetes 应用列表的逻辑
}

// GetK8sProjectList 获取 Kubernetes 项目列表
func (k *K8sHandler) GetK8sProjectList(ctx *gin.Context) {
	// TODO: 实现获取 Kubernetes 项目列表的逻辑
}

// GetK8sProjectListForSelect 获取用于选择的 Kubernetes 项目列表
func (k *K8sHandler) GetK8sProjectListForSelect(ctx *gin.Context) {
	// TODO: 实现获取用于选择的 Kubernetes 项目列表的逻辑
}

// CreateK8sProject 创建 Kubernetes 项目
func (k *K8sHandler) CreateK8sProject(ctx *gin.Context) {
	// TODO: 实现创建 Kubernetes 项目的逻辑
}

// UpdateK8sProject 更新 Kubernetes 项目
func (k *K8sHandler) UpdateK8sProject(ctx *gin.Context) {
	// TODO: 实现更新 Kubernetes 项目的逻辑
}

// DeleteK8sProjectOne 删除单个 Kubernetes 项目
func (k *K8sHandler) DeleteK8sProjectOne(ctx *gin.Context) {
	// TODO: 实现删除单个 Kubernetes 项目的逻辑
}

// GetK8sCronjobList 获取 CronJob 列表
func (k *K8sHandler) GetK8sCronjobList(ctx *gin.Context) {
	// TODO: 实现获取 CronJob 列表的逻辑
}

// CreateK8sCronjobOne 创建单个 CronJob
func (k *K8sHandler) CreateK8sCronjobOne(ctx *gin.Context) {
	// TODO: 实现创建单个 CronJob 的逻辑
}

// UpdateK8sCronjobOne 更新单个 CronJob
func (k *K8sHandler) UpdateK8sCronjobOne(ctx *gin.Context) {
	// TODO: 实现更新单个 CronJob 的逻辑
}

// GetK8sCronjobOne 获取单个 CronJob
func (k *K8sHandler) GetK8sCronjobOne(ctx *gin.Context) {
	// TODO: 实现获取单个 CronJob 的逻辑
}

// GetK8sCronjobLastPod 获取 CronJob 最近的 Pod
func (k *K8sHandler) GetK8sCronjobLastPod(ctx *gin.Context) {
	// TODO: 实现获取 CronJob 最近的 Pod 的逻辑
}

// BatchDeleteK8sCronjob 批量删除 CronJob
func (k *K8sHandler) BatchDeleteK8sCronjob(ctx *gin.Context) {
	// TODO: 实现批量删除 CronJob 的逻辑
}
