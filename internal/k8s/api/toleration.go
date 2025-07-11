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
	"strconv"

	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service/admin"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

type K8sTolerationHandler struct {
	tolerationService admin.TolerationService
	logger            *zap.Logger
}

func NewK8sTolerationHandler(logger *zap.Logger, tolerationService admin.TolerationService) *K8sTolerationHandler {
	return &K8sTolerationHandler{
		logger:            logger,
		tolerationService: tolerationService,
	}
}

func (k *K8sTolerationHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")

	tolerations := k8sGroup.Group("/tolerations")
	{
		tolerations.POST("/add", k.AddTolerations)
		tolerations.POST("/update", k.UpdateTolerations)
		tolerations.DELETE("/delete", k.DeleteTolerations)
		tolerations.POST("/validate", k.ValidateTolerations)
		tolerations.GET("/list", k.ListTolerations)
		tolerations.POST("/time/config", k.ConfigTolerationTime)
		tolerations.POST("/time/validate", k.ValidateTolerationTime)
		tolerations.POST("/batch", k.BatchUpdateTolerations)
		tolerations.POST("/template", k.CreateTolerationTemplate)
		tolerations.GET("/template/:name", k.GetTolerationTemplate)
		tolerations.DELETE("/template/:name", k.DeleteTolerationTemplate)
	}

	taintEffects := k8sGroup.Group("/taint-effects")
	{
		taintEffects.POST("/manage", k.ManageTaintEffects)
		taintEffects.POST("/transition", k.TransitionTaintEffect)
		taintEffects.POST("/validate", k.ValidateTaintEffects)
		taintEffects.GET("/status", k.GetTaintEffectStatus)
		taintEffects.POST("/batch", k.BatchManageTaintEffects)
	}
}

func (k *K8sTolerationHandler) AddTolerations(ctx *gin.Context) {
	var req model.K8sTaintTolerationRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.AddTolerations(ctx, &req)
	})
}

func (k *K8sTolerationHandler) UpdateTolerations(ctx *gin.Context) {
	var req model.K8sTaintTolerationRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.UpdateTolerations(ctx, &req)
	})
}

func (k *K8sTolerationHandler) DeleteTolerations(ctx *gin.Context) {
	var req model.K8sTaintTolerationRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.tolerationService.DeleteTolerations(ctx, &req)
	})
}

func (k *K8sTolerationHandler) ValidateTolerations(ctx *gin.Context) {
	var req model.K8sTaintTolerationValidationRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.ValidateTolerations(ctx, &req)
	})
}

func (k *K8sTolerationHandler) ListTolerations(ctx *gin.Context) {
	var req model.K8sTaintTolerationRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.ListTolerations(ctx, &req)
	})
}

func (k *K8sTolerationHandler) ConfigTolerationTime(ctx *gin.Context) {
	var req model.K8sTolerationTimeRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.ConfigTolerationTime(ctx, &req)
	})
}

func (k *K8sTolerationHandler) ValidateTolerationTime(ctx *gin.Context) {
	var req model.K8sTolerationTimeRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.ValidateTolerationTime(ctx, &req)
	})
}

func (k *K8sTolerationHandler) BatchUpdateTolerations(ctx *gin.Context) {
	var req model.K8sTaintTolerationRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.BatchUpdateTolerations(ctx, &req)
	})
}

func (k *K8sTolerationHandler) CreateTolerationTemplate(ctx *gin.Context) {
	var req model.K8sTolerationConfigRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.CreateTolerationTemplate(ctx, &req)
	})
}

func (k *K8sTolerationHandler) GetTolerationTemplate(ctx *gin.Context) {
	templateName := ctx.Param("name")
	clusterId := ctx.Query("cluster_id")

	clusterID, err := strconv.Atoi(clusterId)
	if err != nil {
		utils.BadRequestWithDetails(ctx, "Invalid cluster_id", "cluster_id must be a valid integer")
		return
	}

	req := model.K8sTolerationConfigRequest{
		ClusterID: clusterID,
		TolerationTemplate: model.K8sTolerationTemplate{
			Name: templateName,
		},
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.GetTolerationTemplate(ctx, &req)
	})
}

func (k *K8sTolerationHandler) DeleteTolerationTemplate(ctx *gin.Context) {
	templateName := ctx.Param("name")
	clusterId := ctx.Query("cluster_id")

	clusterID, err := strconv.Atoi(clusterId)
	if err != nil {
		utils.BadRequestWithDetails(ctx, "Invalid cluster_id", "cluster_id must be a valid integer")
		return
	}

	req := model.K8sTolerationConfigRequest{
		ClusterID: clusterID,
		TolerationTemplate: model.K8sTolerationTemplate{
			Name: templateName,
		},
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, k.tolerationService.DeleteTolerationTemplate(ctx, &req)
	})
}

func (k *K8sTolerationHandler) ManageTaintEffects(ctx *gin.Context) {
	var req model.K8sTaintEffectManagementRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.ManageTaintEffects(ctx, &req)
	})
}

func (k *K8sTolerationHandler) TransitionTaintEffect(ctx *gin.Context) {
	var req model.K8sTaintEffectManagementRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.TransitionTaintEffect(ctx, &req)
	})
}

func (k *K8sTolerationHandler) ValidateTaintEffects(ctx *gin.Context) {
	var req model.K8sTaintEffectManagementRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.ValidateTaintEffects(ctx, &req)
	})
}

func (k *K8sTolerationHandler) GetTaintEffectStatus(ctx *gin.Context) {
	var req model.K8sTaintEffectManagementRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.GetTaintEffectStatus(ctx, &req)
	})
}

func (k *K8sTolerationHandler) BatchManageTaintEffects(ctx *gin.Context) {
	var req model.K8sTaintEffectManagementRequest

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return k.tolerationService.BatchManageTaintEffects(ctx, &req)
	})
}
