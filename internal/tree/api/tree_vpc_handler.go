package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/gin-gonic/gin"
)

type TreeVpcHandler struct {
	vpcService service.TreeVpcService
}

func NewTreeVpcHandler(vpcService service.TreeVpcService) *TreeVpcHandler {
	return &TreeVpcHandler{
		vpcService: vpcService,
	}
}

func (h *TreeVpcHandler) RegisterRouters(server *gin.Engine) {
	vpcGroup := server.Group("/vpc")
	{
		vpcGroup.POST("/detail", h.GetVpcDetail)
		vpcGroup.POST("/create", h.CreateVpcResource)
		vpcGroup.DELETE("/delete", h.DeleteVpc)
		vpcGroup.POST("/list", h.ListVpcResources)
	}
}

func (h *TreeVpcHandler) GetVpcDetail(ctx *gin.Context) {

}

func (h *TreeVpcHandler) CreateVpcResource(ctx *gin.Context) {

}

func (h *TreeVpcHandler) DeleteVpc(ctx *gin.Context) {

}

func (h *TreeVpcHandler) ListVpcResources(ctx *gin.Context) {

}
