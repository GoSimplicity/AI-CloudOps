package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/not_auth/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils/apiresponse"
	"github.com/gin-gonic/gin"
	"net/http"
	"strconv"
	"strings"
)

type NotAuthHandler struct {
	svc service.NotAuthService
}

func NewNotAuthHandler(svc service.NotAuthService) *NotAuthHandler {
	return &NotAuthHandler{
		svc: svc,
	}
}

func (n *NotAuthHandler) RegisterRouters(server *gin.Engine) {
	notAuthGroup := server.Group("/api/not_auth")
	notAuthGroup.GET("/getTreeNodeBindIps", n.GetTreeNodeBindIps)
}

func (n *NotAuthHandler) GetTreeNodeBindIps(ctx *gin.Context) {
	// 获取和验证 leafNodeIds
	leafNodeIds := ctx.DefaultQuery("leafNodeIds", "")
	if leafNodeIds == "" {
		apiresponse.BadRequestError(ctx, "leafNodeIds 参数不能为空")
		return
	}

	leafNodeIdList := strings.Split(leafNodeIds, ",")
	if len(leafNodeIdList) == 0 {
		apiresponse.BadRequestError(ctx, "leafNodeIds 参数格式无效")
		return
	}

	// 获取和验证 port
	port := ctx.DefaultQuery("port", "")
	if port == "" {
		apiresponse.BadRequestError(ctx, "port 参数不能为空")
		return
	}

	p, err := strconv.Atoi(port)
	if err != nil || p <= 0 {
		apiresponse.BadRequestError(ctx, "port 必须为正整数")
		return
	}

	// 调用服务逻辑构建 Prometheus 服务发现结果
	res, err := n.svc.BuildPrometheusServiceDiscovery(ctx, leafNodeIdList, p)
	if err != nil {
		apiresponse.ErrorWithMessage(ctx, "服务器内部错误")
		return
	}

	// 返回成功结果
	ctx.JSON(http.StatusOK, res)
}
