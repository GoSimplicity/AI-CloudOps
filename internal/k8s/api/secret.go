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
	"errors"

	"github.com/GoSimplicity/AI-CloudOps/internal/constants"
	"github.com/GoSimplicity/AI-CloudOps/internal/k8s/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/base"
	"github.com/gin-gonic/gin"
)

// K8sSecretHandler Secret处理器
type K8sSecretHandler struct {
	secretService service.SecretService
}

func NewK8sSecretHandler(secretService service.SecretService) *K8sSecretHandler {
	return &K8sSecretHandler{
		secretService: secretService,
	}
}

// RegisterRouters 注册路由
func (h *K8sSecretHandler) RegisterRouters(server *gin.Engine) {
	k8sGroup := server.Group("/api/k8s")
	{
		k8sGroup.GET("/secret/:cluster_id/list", h.GetSecretList)
		k8sGroup.GET("/secret/:cluster_id/:namespace/:name/detail", h.GetSecret)
		k8sGroup.GET("/secret/:cluster_id/:namespace/:name/detail/yaml", h.GetSecretYAML)
		k8sGroup.GET("/secret/:cluster_id/:namespace/:name/reveal", h.RevealSecret)
		k8sGroup.POST("/secret/:cluster_id/create", h.CreateSecret)
		k8sGroup.POST("/secret/:cluster_id/create/yaml", h.CreateSecretByYaml)
		k8sGroup.PUT("/secret/:cluster_id/:namespace/:name/update", h.UpdateSecret)
		k8sGroup.PUT("/secret/:cluster_id/:namespace/:name/update/yaml", h.UpdateSecretByYaml)
		k8sGroup.DELETE("/secret/:cluster_id/:namespace/:name/delete", h.DeleteSecret)
	}
}

func (h *K8sSecretHandler) GetSecretList(ctx *gin.Context) {
	var req model.GetSecretListReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.secretService.GetSecretList(ctx.Request.Context(), &req)
	})
}

func (h *K8sSecretHandler) GetSecret(ctx *gin.Context) {
	var req model.GetSecretDetailsReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.secretService.GetSecret(ctx.Request.Context(), &req)
	})
}

func (h *K8sSecretHandler) CreateSecret(ctx *gin.Context) {
	var req model.CreateSecretReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.CreateSecret(ctx.Request.Context(), &req)
	})
}

func (h *K8sSecretHandler) UpdateSecret(ctx *gin.Context) {
	var req model.UpdateSecretReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.UpdateSecret(ctx.Request.Context(), &req)
	})
}

func (h *K8sSecretHandler) DeleteSecret(ctx *gin.Context) {
	var req model.DeleteSecretReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.DeleteSecret(ctx.Request.Context(), &req)
	})
}

func (h *K8sSecretHandler) GetSecretYAML(ctx *gin.Context) {
	var req model.GetSecretYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.secretService.GetSecretYAML(ctx.Request.Context(), &req)
	})
}

func (h *K8sSecretHandler) CreateSecretByYaml(ctx *gin.Context) {
	var req model.CreateSecretByYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.CreateSecretByYaml(ctx.Request.Context(), &req)
	})
}

func (h *K8sSecretHandler) UpdateSecretByYaml(ctx *gin.Context) {
	var req model.UpdateSecretByYamlReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	base.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, h.secretService.UpdateSecretByYaml(ctx.Request.Context(), &req)
	})
}

// RevealSecret 返回 Secret 明文（SEC-25）。list/detail/yaml 默认脱敏，明文只能经此接口获取，
// 每次调用都会写一条审计（只含操作者、集群、命名空间、名称与键名）。
func (h *K8sSecretHandler) RevealSecret(ctx *gin.Context) {
	var req model.RevealSecretReq

	clusterID, err := base.GetCustomParamID(ctx, "cluster_id")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	namespace, err := base.GetParamCustomName(ctx, "namespace")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	name, err := base.GetParamCustomName(ctx, "name")
	if err != nil {
		base.BadRequestError(ctx, err.Error())
		return
	}

	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	user := base.GetUserInfoFromGinContext(ctx)
	actor := model.SecretRevealActor{
		UserID:    user.UserID,
		Username:  user.Username,
		IP:        user.IP,
		UserAgent: user.UserAgent,
		TraceID:   ctx.GetHeader("X-Trace-ID"),
		Endpoint:  ctx.Request.URL.Path,
	}

	// 明文响应禁止被浏览器或中间代理缓存
	ctx.Header("Cache-Control", "no-store")
	ctx.Header("Pragma", "no-cache")

	handleAuditedRequest(ctx, &req, func() (interface{}, error) {
		return h.secretService.RevealSecret(ctx.Request.Context(), &req, actor)
	})
}

// handleAuditedRequest 与 base.HandleRequest 相同，但同步审计写入失败（service.ErrAuditWriteFailed）时
// 返回 HTTP 500 而不是 200+code=1，使网关/监控与审计告警能看到真实的失败状态；
// service 返回的参数错误（BusinessError{constants.ErrInvalidParam}）返回 HTTP 400。响应中不含任何明文。
func handleAuditedRequest(ctx *gin.Context, req interface{}, action func() (interface{}, error)) {
	if req != nil {
		if err := base.BindRequest(ctx, req); err != nil {
			base.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
			return
		}
	}
	data, err := action()
	if err != nil {
		if errors.Is(err, service.ErrAuditWriteFailed) {
			base.InternalServerErrorWithDetails(ctx, map[string]interface{}{}, err.Error())
			return
		}
		respondBusinessError(ctx, err)
		return
	}
	base.SuccessWithData(ctx, data)
}

// handleMappedRequest 与 base.HandleRequest 相同，但按 respondBusinessError 把带类型的业务错误映射为对应 HTTP 状态码
func handleMappedRequest(ctx *gin.Context, req interface{}, action func() (interface{}, error)) {
	if req != nil {
		if err := base.BindRequest(ctx, req); err != nil {
			base.BadRequestWithDetails(ctx, err.Error(), "绑定数据失败")
			return
		}
	}
	data, err := action()
	if err != nil {
		respondBusinessError(ctx, err)
		return
	}
	if data != nil {
		base.SuccessWithData(ctx, data)
	} else {
		base.Success(ctx)
	}
}

// respondBusinessError 业务错误映射：BusinessError{constants.ErrInvalidParam} → HTTP 400；
// BusinessError{constants.ErrK8sResourceConflict}（乐观锁冲突）→ HTTP 409；其余保持项目约定的 HTTP 200 + code=1
func respondBusinessError(ctx *gin.Context, err error) {
	var bizErr *base.BusinessError
	if errors.As(err, &bizErr) {
		switch {
		case errors.Is(bizErr.Code, constants.ErrInvalidParam):
			// 参数错误（如 TokenRequest 有效期越界）按项目约定返回 HTTP 400 + code=1
			base.BadRequestError(ctx, err.Error())
			return
		case errors.Is(bizErr.Code, constants.ErrK8sResourceConflict):
			base.ConflictError(ctx, err.Error())
			return
		}
	}
	base.ErrorWithMessage(ctx, err.Error())
}
