package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sTaintHandler struct {
	taintService admin.TaintService
	l            *zap.Logger
}

func NewK8sTaintHandler(l *zap.Logger, taintService admin.TaintService) *K8sTaintHandler {
	return &K8sTaintHandler{
		l:            l,
		taintService: taintService,
	}
}

// RegisterRouters 注册所有 Kubernetes 相关的路由
func (k *K8sTaintHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// 节点相关路由
	nodes := k8sGroup.Group("/nodes")
	{
		nodes.POST("/taint_check", k.TaintYamlCheck)              // 检查节点 Taint 的 YAML 配置
		nodes.POST("/enable_switch", k.ScheduleEnableSwitchNodes) // 启用或切换节点调度
		nodes.POST("/taints/add", k.AddTaintsNodes)               // 为节点添加 Taint
		nodes.DELETE("/taints/delete", k.DeleteTaintsNodes)       // 删除节点 Taint
		nodes.POST("/drain", k.DrainPods)                         // 清空节点上的 Pods
	}
}

// AddTaintsNodes 为节点添加 Taint
func (k *K8sTaintHandler) AddTaintsNodes(ctx *gin.Context) {
	var taint *model.TaintK8sNodesRequest

	err := ctx.ShouldBind(&taint)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.taintService.UpdateNodeTaint(ctx, taint); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

// ScheduleEnableSwitchNodes 启用或切换节点调度
func (k *K8sTaintHandler) ScheduleEnableSwitchNodes(ctx *gin.Context) {
	var req model.ScheduleK8sNodesRequest

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.taintService.BatchEnableSwitchNodes(ctx, &req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "启用或切换节点调度失败")
		return
	}

	apiresponse.Success(ctx)
}

// TaintYamlCheck 检查节点 Taint 的 YAML 配置
func (k *K8sTaintHandler) TaintYamlCheck(ctx *gin.Context) {
	var req model.TaintK8sNodesRequest

	if err := ctx.ShouldBindBodyWithJSON(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.taintService.CheckTaintYaml(ctx, &req); err != nil {
		apiresponse.ErrorWithDetails(ctx, err.Error(), "Taint YAML 配置检查失败")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteTaintsNodes 删除节点 Taint
func (k *K8sTaintHandler) DeleteTaintsNodes(ctx *gin.Context) {
	var taint *model.TaintK8sNodesRequest

	err := ctx.ShouldBind(&taint)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.taintService.UpdateNodeTaint(ctx, taint); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}

// DrainPods 清空节点上的 Pods
func (k *K8sTaintHandler) DrainPods(ctx *gin.Context) {
	var req model.K8sClusterNodesRequest

	err := ctx.ShouldBind(&req)
	if err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	if err := k.taintService.DrainPods(ctx, &req); err != nil {
		apiresponse.ErrorWithMessage(ctx, err.Error())
		return
	}

	apiresponse.Success(ctx)
}
