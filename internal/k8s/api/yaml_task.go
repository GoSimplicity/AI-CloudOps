package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	ijwt "github.com/GoSimplicity/AI-CloudOps/pkg/utils/jwt"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sYamlTaskHandler struct {
	l               *zap.Logger
	yamlTaskService admin.YamlTaskService
}

func NewK8sYamlTaskHandler(l *zap.Logger, yamlTaskService admin.YamlTaskService) *K8sYamlTaskHandler {
	return &K8sYamlTaskHandler{
		l:               l,
		yamlTaskService: yamlTaskService,
	}
}

// RegisterRouters 注册所有 Kubernetes 相关的路由
func (k *K8sYamlTaskHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// task 任务相关路由
	yamlTasks := k8sGroup.Group("/yaml_tasks")
	{
		yamlTasks.GET("/", k.GetYamlTaskList)         // 获取 YAML 任务列表
		yamlTasks.POST("/", k.CreateYamlTask)         // 创建新的 YAML 任务
		yamlTasks.PUT("/:id", k.UpdateYamlTask)       // 更新指定 ID 的 YAML 任务
		yamlTasks.POST("/:id/apply", k.ApplyYamlTask) // 应用指定 ID 的 YAML 任务
		yamlTasks.DELETE("/:id", k.DeleteYamlTask)    // 删除指定 ID 的 YAML 任务
	}
}

// GetYamlTaskList 获取 YAML 任务列表
func (k *K8sYamlTaskHandler) GetYamlTaskList(ctx *gin.Context) {
	list, err := k.yamlTaskService.GetYamlTaskList(ctx)
	if err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateYamlTask 创建新的 YAML 任务
func (k *K8sYamlTaskHandler) CreateYamlTask(ctx *gin.Context) {
	var req model.K8sYamlTask

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid

	if err := k.yamlTaskService.CreateYamlTask(ctx, &req); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateYamlTask 更新指定 ID 的 YAML 任务
func (k *K8sYamlTaskHandler) UpdateYamlTask(ctx *gin.Context) {
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

	if err := k.yamlTaskService.UpdateYamlTask(ctx, &req); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// ApplyYamlTask 应用指定 ID 的 YAML 任务
func (k *K8sYamlTaskHandler) ApplyYamlTask(ctx *gin.Context) {
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

	if err := k.yamlTaskService.ApplyYamlTask(ctx, taskID); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteYamlTask 删除指定 ID 的 YAML 任务
func (k *K8sYamlTaskHandler) DeleteYamlTask(ctx *gin.Context) {
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

	if err := k.yamlTaskService.DeleteYamlTask(ctx, taskID); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}
