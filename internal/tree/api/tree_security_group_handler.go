package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/gin-gonic/gin"
)

type TreeSecurityGroupHandler struct {
	securityGroupService service.TreeSecurityGroupService
}

func NewTreeSecurityGroupHandler(securityGroupService service.TreeSecurityGroupService) *TreeSecurityGroupHandler {
	return &TreeSecurityGroupHandler{
		securityGroupService: securityGroupService,
	}
}

func (h *TreeSecurityGroupHandler) RegisterRouters(server *gin.Engine) {
	securityGroupGroup := server.Group("/security_group")
	{
		securityGroupGroup.POST("/create", h.CreateSecurityGroup)
		securityGroupGroup.DELETE("/delete", h.DeleteSecurityGroup)
		securityGroupGroup.POST("/list", h.ListSecurityGroups)
		securityGroupGroup.POST("/detail", h.GetSecurityGroupDetail)
	}
}

func (h *TreeSecurityGroupHandler) CreateSecurityGroup(ctx *gin.Context) {

}

func (h *TreeSecurityGroupHandler) DeleteSecurityGroup(ctx *gin.Context) {

}

func (h *TreeSecurityGroupHandler) ListSecurityGroups(ctx *gin.Context) {

}

func (h *TreeSecurityGroupHandler) GetSecurityGroupDetail(ctx *gin.Context) {

}
