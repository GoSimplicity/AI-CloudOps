/*
 * MIT License
 *
 * Copyright (c) 2024 Bamboo
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE.
 *
 */

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
			ecsGroup.POST("/list", h.ListEcsResources)
			ecsGroup.POST("/detail", h.GetEcsDetail)
			ecsGroup.POST("/create", h.CreateEcsResource)
			ecsGroup.POST("/start", h.StartEcs)
			ecsGroup.POST("/stop", h.StopEcs)
			ecsGroup.POST("/restart", h.RestartEcs)
			ecsGroup.DELETE("/delete", h.DeleteEcs)
			ecsGroup.POST("/instance_options", h.ListInstanceOptions)
		}

		// VPC相关接口
		vpcGroup := resourceGroup.Group("/vpc")
		{
			vpcGroup.POST("/detail", h.GetVpcDetail)
			vpcGroup.POST("/create", h.CreateVpcResource)
			vpcGroup.POST("/delete", h.DeleteVpc)
		}

		// ELB相关接口
		elbGroup := resourceGroup.Group("/elb")
		{
			elbGroup.POST("/list", h.ListElbResources)
			elbGroup.POST("/detail", h.GetElbDetail)
			elbGroup.POST("/create", h.CreateElbResource)
			elbGroup.POST("/delete", h.DeleteElb)
		}

		// RDS相关接口
		rdsGroup := resourceGroup.Group("/rds")
		{
			rdsGroup.POST("/list", h.ListRdsResources)
			rdsGroup.POST("/detail", h.GetRdsDetail)
			rdsGroup.POST("/create", h.CreateRdsResource)
			rdsGroup.POST("/start", h.StartRds)
			rdsGroup.POST("/stop", h.StopRds)
			rdsGroup.POST("/restart", h.RestartRds)
			rdsGroup.POST("/delete", h.DeleteRds)
		}

		// 云厂商相关接口
		cloudGroup := resourceGroup.Group("/cloud")
		{
			cloudGroup.POST("/providers", h.ListCloudProviders)
			cloudGroup.POST("/regions", h.ListRegions)
			cloudGroup.POST("/zones", h.ListZones)
			cloudGroup.POST("/instance_types", h.ListInstanceTypes)
			cloudGroup.POST("/images", h.ListImages)
			cloudGroup.POST("/vpcs", h.ListVpcs)
			cloudGroup.POST("/security_groups", h.ListSecurityGroups)
		}
	}
}

// SyncResources 同步资源
func (h *ResourceHandler) SyncResources(ctx *gin.Context) {
	var req model.SyncResourcesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.resourceService.SyncResources(ctx, req.Provider, req.Region)
	})
}

// ECS资源相关接口

// ListEcsResources 获取ECS资源列表
func (h *ResourceHandler) ListEcsResources(ctx *gin.Context) {
	var req model.ListEcsResourcesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.ecsService.ListEcsResources(ctx, &req)
	})
}

// GetEcsDetail 获取ECS资源详情
func (h *ResourceHandler) GetEcsDetail(ctx *gin.Context) {
	var req model.GetEcsDetailReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.ecsService.GetEcsResourceById(ctx, &req)
	})
}

// CreateEcsResource 创建ECS资源
func (h *ResourceHandler) CreateEcsResource(ctx *gin.Context) {
	var req model.CreateEcsResourceReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.CreateEcsResource(ctx, &req)
	})
}

// StartEcs 启动ECS实例
func (h *ResourceHandler) StartEcs(ctx *gin.Context) {
	var req model.StartEcsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.StartEcsResource(ctx, &req)
	})
}

// StopEcs 停止ECS实例
func (h *ResourceHandler) StopEcs(ctx *gin.Context) {
	var req model.StopEcsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.StopEcsResource(ctx, &req)
	})
}

// RestartEcs 重启ECS实例
func (h *ResourceHandler) RestartEcs(ctx *gin.Context) {
	var req model.RestartEcsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.RestartEcsResource(ctx, &req)
	})
}

// DeleteEcs 删除ECS实例
func (h *ResourceHandler) DeleteEcs(ctx *gin.Context) {
	var req model.DeleteEcsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.ecsService.DeleteEcsResource(ctx, &req)
	})
}

// VPC资源相关接口

// GetVpcDetail 获取VPC资源详情
func (h *ResourceHandler) GetVpcDetail(ctx *gin.Context) {
	var req model.GetVpcDetailReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.vpcService.GetVpcResourceById(ctx, &req)
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
	var req model.DeleteVpcReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.vpcService.DeleteVpcResource(ctx, &req)
	})
}

// ELB资源相关接口

// ListElbResources 获取ELB资源列表
func (h *ResourceHandler) ListElbResources(ctx *gin.Context) {
	var req model.ListElbResourcesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.elbService.ListElbResources(ctx, &req)
	})
}

// GetElbDetail 获取ELB资源详情
func (h *ResourceHandler) GetElbDetail(ctx *gin.Context) {
	var req model.GetElbDetailReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.elbService.GetElbResourceById(ctx, &req)
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
	var req model.DeleteElbReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.elbService.DeleteElbResource(ctx, &req)
	})
}

// RDS资源相关接口

// ListRdsResources 获取RDS资源列表
func (h *ResourceHandler) ListRdsResources(ctx *gin.Context) {
	var req model.ListRdsResourcesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.rdsService.ListRdsResources(ctx, &req)
	})
}

// GetRdsDetail 获取RDS资源详情
func (h *ResourceHandler) GetRdsDetail(ctx *gin.Context) {
	var req model.GetRdsDetailReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.rdsService.GetRdsResourceById(ctx, &req)
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
	var req model.StartRdsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.StartRdsResource(ctx, &req)
	})
}

// StopRds 停止RDS实例
func (h *ResourceHandler) StopRds(ctx *gin.Context) {
	var req model.StopRdsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.StopRdsResource(ctx, &req)
	})
}

// RestartRds 重启RDS实例
func (h *ResourceHandler) RestartRds(ctx *gin.Context) {
	var req model.RestartRdsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.RestartRdsResource(ctx, &req)
	})
}

// DeleteRds 删除RDS实例
func (h *ResourceHandler) DeleteRds(ctx *gin.Context) {
	var req model.DeleteRdsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.rdsService.DeleteRdsResource(ctx, &req)
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
	var req model.ListRegionsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cloudService.ListRegions(ctx, &req)
	})
}

// ListZones 获取指定区域的可用区列表
func (h *ResourceHandler) ListZones(ctx *gin.Context) {
	var req model.ListZonesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cloudService.ListZones(ctx, &req)
	})
}

// ListInstanceTypes 获取实例类型列表
func (h *ResourceHandler) ListInstanceTypes(ctx *gin.Context) {
	var req model.ListInstanceTypesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cloudService.ListInstanceTypes(ctx, &req)
	})
}

// ListImages 获取镜像列表
func (h *ResourceHandler) ListImages(ctx *gin.Context) {
	var req model.ListImagesReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cloudService.ListImages(ctx, &req)
	})
}

// ListVpcs 获取VPC列表
func (h *ResourceHandler) ListVpcs(ctx *gin.Context) {
	var req model.ListVpcsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cloudService.ListVpcs(ctx, &req)
	})
}

// ListSecurityGroups 获取安全组列表
func (h *ResourceHandler) ListSecurityGroups(ctx *gin.Context) {
	var req model.ListSecurityGroupsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cloudService.ListSecurityGroups(ctx, &req)
	})
}

// ListInstanceOptions 获取实例选项
func (h *ResourceHandler) ListInstanceOptions(ctx *gin.Context) {
	var req model.ListInstanceOptionsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.ecsService.ListInstanceOptions(ctx, &req)
	})
}
