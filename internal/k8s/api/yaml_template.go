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

type K8sYamlTemplateHandler struct {
	l                   *zap.Logger
	yamlTemplateService admin.YamlTemplateService
}

func NewK8sYamlTemplateHandler(l *zap.Logger, yamlTemplateService admin.YamlTemplateService) *K8sYamlTemplateHandler {
	return &K8sYamlTemplateHandler{
		l:                   l,
		yamlTemplateService: yamlTemplateService,
	}
}

// RegisterRouters 注册所有 Kubernetes 相关的路由
func (k *K8sYamlTemplateHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	// YAML 模板相关路由
	yamlTemplates := k8sGroup.Group("/yaml-templates")
	{
		yamlTemplates.GET("/", k.GetYamlTemplateList)      // 获取 YAML 模板列表
		yamlTemplates.POST("/", k.CreateYamlTemplate)      // 创建新的 YAML 模板
		yamlTemplates.PUT("/:id", k.UpdateYamlTemplate)    // 更新指定 ID 的 YAML 模板
		yamlTemplates.DELETE("/:id", k.DeleteYamlTemplate) // 删除指定 ID 的 YAML 模板
	}
}

// GetYamlTemplateList 获取 YAML 模板列表
func (k *K8sYamlTemplateHandler) GetYamlTemplateList(ctx *gin.Context) {
	list, err := k.yamlTemplateService.GetYamlTemplateList(ctx)
	if err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.SuccessWithData(ctx, list)
}

// CreateYamlTemplate 创建新的 YAML 模板
func (k *K8sYamlTemplateHandler) CreateYamlTemplate(ctx *gin.Context) {
	var req model.K8sYamlTemplate

	if err := ctx.ShouldBind(&req); err != nil {
		apiresponse.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
		return
	}

	uc := ctx.MustGet("user").(ijwt.UserClaims)
	req.UserID = uc.Uid

	if err := k.yamlTemplateService.CreateYamlTemplate(ctx, &req); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// UpdateYamlTemplate 更新指定 ID 的 YAML 模板
func (k *K8sYamlTemplateHandler) UpdateYamlTemplate(ctx *gin.Context) {
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

	if err := k.yamlTemplateService.UpdateYamlTemplate(ctx, &req); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}

// DeleteYamlTemplate 删除指定 ID 的 YAML 模板
func (k *K8sYamlTemplateHandler) DeleteYamlTemplate(ctx *gin.Context) {
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

	if err := k.yamlTemplateService.DeleteYamlTemplate(ctx, yamlId); err != nil {
		apiresponse.InternalServerErrorWithDetails(ctx, err.Error(), "服务器内部错误")
		return
	}

	apiresponse.Success(ctx)
}
