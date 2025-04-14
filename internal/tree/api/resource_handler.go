package api

import (
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type ResourceHandler struct {
	resourceService service.ResourceService
	ecsService      service.EcsService
	vpcService      service.VpcService
	elbService      service.ElbService
	rdsService      service.RdsService
	cloudService    service.CloudService
}

func NewResourceHandler(resourceService service.ResourceService, ecsService service.EcsService, vpcService service.VpcService, elbService service.ElbService, rdsService service.RdsService, cloudService service.CloudService) *ResourceHandler {
	return &ResourceHandler{
		resourceService: resourceService,
		ecsService:      ecsService,
		vpcService:      vpcService,
		elbService:      elbService,
		rdsService:      rdsService,
		cloudService:    cloudService,
	}
}

func (h *ResourceHandler) RegisterRouters(server *gin.Engine) {
	resourceGroup := server.Group("/api/resource")
	{
		// 通用资源接口
		resourceGroup.POST("/sync", h.SyncResources)
		
		// ECS相关接口
		ecsGroup := resourceGroup.Group("/ecs")
		{
			ecsGroup.GET("/list", h.ListEcsResources)
			ecsGroup.GET("/detail/:id", h.GetEcsDetail)
			ecsGroup.POST("/create", h.CreateEcsResource)
			ecsGroup.POST("/start/:id", h.StartEcs)
			ecsGroup.POST("/stop/:id", h.StopEcs)
			ecsGroup.POST("/restart/:id", h.RestartEcs)
			ecsGroup.POST("/delete/:id", h.DeleteEcs)
		}

		// VPC相关接口
		vpcGroup := resourceGroup.Group("/vpc")
		{
			vpcGroup.GET("/detail/:id", h.GetVpcDetail)
			vpcGroup.POST("/create", h.CreateVpcResource)
			vpcGroup.POST("/delete/:id", h.DeleteVpc)
		}

		// ELB相关接口
		elbGroup := resourceGroup.Group("/elb")
		{
			elbGroup.GET("/list", h.ListElbResources)
			elbGroup.GET("/detail/:id", h.GetElbDetail)
			elbGroup.POST("/create", h.CreateElbResource)
			elbGroup.POST("/delete/:id", h.DeleteElb)
		}
		
		// RDS相关接口
		rdsGroup := resourceGroup.Group("/rds")
		{
			rdsGroup.GET("/list", h.ListRdsResources)
			rdsGroup.GET("/detail/:id", h.GetRdsDetail)
			rdsGroup.POST("/create", h.CreateRdsResource)
			rdsGroup.POST("/start/:id", h.StartRds)
			rdsGroup.POST("/stop/:id", h.StopRds)
			rdsGroup.POST("/restart/:id", h.RestartRds)
			rdsGroup.POST("/delete/:id", h.DeleteRds)
		}
		
		// 云厂商相关接口
		cloudGroup := resourceGroup.Group("/cloud")
		{
			cloudGroup.GET("/providers", h.ListCloudProviders)
			cloudGroup.GET("/regions/:provider", h.ListRegions)
			cloudGroup.GET("/zones/:provider/:region", h.ListZones)
			cloudGroup.GET("/instance_types/:provider/:region", h.ListInstanceTypes)
			cloudGroup.GET("/images/:provider/:region", h.ListImages)
			cloudGroup.GET("/vpcs/:provider/:region", h.ListVpcs)
			cloudGroup.GET("/security_groups/:provider/:region", h.ListSecurityGroups)
		}
	}
}

// SyncResources 同步资源
func (h *ResourceHandler) SyncResources(ctx *gin.Context) {
	var req model.SyncResourcesReq
	
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.resourceService.SyncResources(ctx, req.Provider, req.Region, req.PageSize, req.PageNumber)
	})
}

// ECS资源相关接口

// ListEcsResources 获取ECS资源列表
func (h *ResourceHandler) ListEcsResources(ctx *gin.Context) {
	var req model.ListEcsResourcesReq
	
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.ecsService.ListEcsResources(ctx, &req)
	})
}

// GetEcsDetail 获取ECS资源详情
func (h *ResourceHandler) GetEcsDetail(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.ecsService.GetEcsResourceById(ctx, id)
	})
}

// CreateEcsResource 创建ECS资源
func (h *ResourceHandler) CreateEcsResource(ctx *gin.Context) {
	var req model.EcsCreationParams
	
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.CreateEcsResource(ctx, &req)
	})
}

// StartEcs 启动ECS实例
func (h *ResourceHandler) StartEcs(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.resourceService.StartResource(ctx, "ecs", id)
	})
}

// StopEcs 停止ECS实例
func (h *ResourceHandler) StopEcs(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.resourceService.StopResource(ctx, "ecs", id)
	})
}

// RestartEcs 重启ECS实例
func (h *ResourceHandler) RestartEcs(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.resourceService.RestartResource(ctx, "ecs", id)
	})
}

// DeleteEcs 删除ECS实例
func (h *ResourceHandler) DeleteEcs(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.resourceService.DeleteResource(ctx, "ecs", id)
	})
}

// VPC资源相关接口

// GetVpcDetail 获取VPC资源详情
func (h *ResourceHandler) GetVpcDetail(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.vpcService.GetVpcResourceById(ctx, id)
	})
}

// CreateVpcResource 创建VPC资源
func (h *ResourceHandler) CreateVpcResource(ctx *gin.Context) {
	var req model.VpcCreationParams
	
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.vpcService.CreateVpcResource(ctx, &req)
	})
}

// DeleteVpc 删除VPC资源
func (h *ResourceHandler) DeleteVpc(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.resourceService.DeleteResource(ctx, "vpc", id)
	})
}

// ELB资源相关接口

// ListElbResources 获取ELB资源列表
func (h *ResourceHandler) ListElbResources(ctx *gin.Context) {
	var req model.ListElbResourcesReq
	
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.elbService.ListElbResources(ctx, &req)
	})
}

// GetElbDetail 获取ELB资源详情
func (h *ResourceHandler) GetElbDetail(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.elbService.GetElbResourceById(ctx, id)
	})
}

// CreateElbResource 创建ELB资源
func (h *ResourceHandler) CreateElbResource(ctx *gin.Context) {
	var req model.ElbCreationParams
	
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.elbService.CreateElbResource(ctx, &req)
	})
}

// DeleteElb 删除ELB实例
func (h *ResourceHandler) DeleteElb(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.resourceService.DeleteResource(ctx, "elb", id)
	})
}

// RDS资源相关接口

// ListRdsResources 获取RDS资源列表
func (h *ResourceHandler) ListRdsResources(ctx *gin.Context) {
	var req model.ListRdsResourcesReq
	
	if err := ctx.ShouldBindQuery(&req); err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.rdsService.ListRdsResources(ctx, &req)
	})
}

// GetRdsDetail 获取RDS资源详情
func (h *ResourceHandler) GetRdsDetail(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.rdsService.GetRdsResourceById(ctx, id)
	})
}

// CreateRdsResource 创建RDS资源
func (h *ResourceHandler) CreateRdsResource(ctx *gin.Context) {
	var req model.RdsCreationParams
	
	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.CreateRdsResource(ctx, &req)
	})
}

// StartRds 启动RDS实例
func (h *ResourceHandler) StartRds(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.resourceService.StartResource(ctx, "rds", id)
	})
}

// StopRds 停止RDS实例
func (h *ResourceHandler) StopRds(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.resourceService.StopResource(ctx, "rds", id)
	})
}

// RestartRds 重启RDS实例
func (h *ResourceHandler) RestartRds(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.resourceService.RestartResource(ctx, "rds", id)
	})
}

// DeleteRds 删除RDS实例
func (h *ResourceHandler) DeleteRds(ctx *gin.Context) {
	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return nil, h.resourceService.DeleteResource(ctx, "rds", id)
	})
}

// 云厂商相关接口

// ListCloudProviders 获取支持的云厂商列表
func (h *ResourceHandler) ListCloudProviders(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.cloudService.ListCloudProviders(ctx)
	})
}

// ListRegions 获取指定云厂商的区域列表
func (h *ResourceHandler) ListRegions(ctx *gin.Context) {
	provider := model.CloudProvider(ctx.Param("provider"))
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.cloudService.ListRegions(ctx, provider)
	})
}

// ListZones 获取指定区域的可用区列表
func (h *ResourceHandler) ListZones(ctx *gin.Context) {
	provider, err := utils.GetQueryParam[model.CloudProvider](ctx, "provider")
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	region, err := utils.GetQueryParam[string](ctx, "region")
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.cloudService.ListZones(ctx, provider, region)
	})
}

// ListInstanceTypes 获取实例类型列表
func (h *ResourceHandler) ListInstanceTypes(ctx *gin.Context) {
	provider, err := utils.GetQueryParam[model.CloudProvider](ctx, "provider")
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	region, err := utils.GetQueryParam[string](ctx, "region")
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.cloudService.ListInstanceTypes(ctx, provider, region)
	})
}

// ListImages 获取镜像列表
func (h *ResourceHandler) ListImages(ctx *gin.Context) {
	provider, err := utils.GetQueryParam[model.CloudProvider](ctx, "provider")
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	region, err := utils.GetQueryParam[string](ctx, "region")
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.cloudService.ListImages(ctx, provider, region)
	})
}

// ListVpcs 获取VPC列表
func (h *ResourceHandler) ListVpcs(ctx *gin.Context) {
	provider, err := utils.GetQueryParam[model.CloudProvider](ctx, "provider")
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	region, err := utils.GetQueryParam[string](ctx, "region")
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.cloudService.ListVpcs(ctx, provider, region)
	})
}

// ListSecurityGroups 获取安全组列表
func (h *ResourceHandler) ListSecurityGroups(ctx *gin.Context) {
	provider, err := utils.GetQueryParam[model.CloudProvider](ctx, "provider")
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}

	region, err := utils.GetQueryParam[string](ctx, "region")
	if err != nil {
		utils.ErrorWithMessage(ctx, err.Error())
		return
	}
	
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.cloudService.ListSecurityGroups(ctx, provider, region)
	})
}