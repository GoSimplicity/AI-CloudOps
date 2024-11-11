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
	logger       *zap.Logger
}

func NewK8sTaintHandler(logger *zap.Logger, taintService admin.TaintService) *K8sTaintHandler {
	return &K8sTaintHandler{
		logger:       logger,
		taintService: taintService,
	}
}

func (k *K8sTaintHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// 节点相关路由组
	nodes := k8sGroup.Group("/taints")
	{
		nodes.POST("/add", k.AddTaintsNodes)                      // 为节点添加 Taint
		nodes.POST("/enable_switch", k.ScheduleEnableSwitchNodes) // 启用或切换节点调度
		nodes.POST("/taint_check", k.TaintYamlCheck)              // 检查节点 Taint 的 YAML 配置
		nodes.DELETE("/delete", k.DeleteTaintsNodes)              // 删除节点 Taint
		nodes.POST("/drain", k.DrainPods)                         // 清空节点上的 Pods
	}
}

// AddTaintsNodes 为节点添加 Taint
func (k *K8sTaintHandler) AddTaintsNodes(ctx *gin.Context) {
	var req model.TaintK8sNodesRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.AddOrUpdateNodeTaint(ctx, &req)
	})
}

// ScheduleEnableSwitchNodes 启用或切换节点调度
func (k *K8sTaintHandler) ScheduleEnableSwitchNodes(ctx *gin.Context) {
	var req model.ScheduleK8sNodesRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.BatchEnableSwitchNodes(ctx, &req)
	})
}

// TaintYamlCheck 检查节点 Taint 的 YAML 配置
func (k *K8sTaintHandler) TaintYamlCheck(ctx *gin.Context) {
	var req model.TaintK8sNodesRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.CheckTaintYaml(ctx, &req)
	})
}

// DeleteTaintsNodes 删除节点 Taint
func (k *K8sTaintHandler) DeleteTaintsNodes(ctx *gin.Context) {
	var req model.TaintK8sNodesRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.AddOrUpdateNodeTaint(ctx, &req)
	})
}

// DrainPods 清空节点上的 Pods
func (k *K8sTaintHandler) DrainPods(ctx *gin.Context) {
	var req model.K8sClusterNodesRequest

	apiresponse.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.taintService.DrainPods(ctx, &req)
	})
}
