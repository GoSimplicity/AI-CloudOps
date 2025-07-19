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

// K8sRBACHandler RBAC（基于角色的访问控制）API处理器
// 提供 Kubernetes RBAC 资源管理的 HTTP API 接口
type K8sRBACHandler struct {
	logger      *zap.Logger       // 日志记录器
	rbacService admin.RBACService // RBAC服务
}

// NewK8sRBACHandler 创建新的RBAC API处理器实例
// @param logger 日志记录器
// @param rbacService RBAC服务实例
// @return *K8sRBACHandler RBAC API处理器实例
func NewK8sRBACHandler(logger *zap.Logger, rbacService admin.RBACService) *K8sRBACHandler {
	return &K8sRBACHandler{
		logger:      logger,
		rbacService: rbacService,
	}
}

// RegisterRouters 注册RBAC相关的路由
// @param router Gin路由引擎实例
func (h *K8sRBACHandler) RegisterRouters(router *gin.Engine) {
	// 创建API分组
	k8sGroup := router.Group("/api/k8s")
	rbacGroup := k8sGroup.Group("/rbac")

	// Role 管理路由
	roleGroup := rbacGroup.Group("/roles")
	{
		// GET /api/k8s/rbac/roles/:cluster_id/:namespace - 获取指定命名空间的所有Role
		roleGroup.GET("/:cluster_id/:namespace", h.GetRolesByNamespace)
		// GET /api/k8s/rbac/roles/:cluster_id/:namespace/:name - 获取指定Role详情
		roleGroup.GET("/:cluster_id/:namespace/:name", h.GetRole)
		// POST /api/k8s/rbac/roles/:cluster_id/:namespace - 创建Role
		roleGroup.POST("/:cluster_id/:namespace", h.CreateRole)
		// PUT /api/k8s/rbac/roles/:cluster_id/:namespace/:name - 更新Role
		roleGroup.PUT("/:cluster_id/:namespace/:name", h.UpdateRole)
		// DELETE /api/k8s/rbac/roles/:cluster_id/:namespace/:name - 删除Role
		roleGroup.DELETE("/:cluster_id/:namespace/:name", h.DeleteRole)
	}

	// ClusterRole 管理路由
	clusterRoleGroup := rbacGroup.Group("/cluster-roles")
	{
		// GET /api/k8s/rbac/cluster-roles/:cluster_id - 获取集群所有ClusterRole
		clusterRoleGroup.GET("/:cluster_id", h.GetClusterRoles)
		// GET /api/k8s/rbac/cluster-roles/:cluster_id/:name - 获取指定ClusterRole详情
		clusterRoleGroup.GET("/:cluster_id/:name", h.GetClusterRole)
		// POST /api/k8s/rbac/cluster-roles/:cluster_id - 创建ClusterRole
		clusterRoleGroup.POST("/:cluster_id", h.CreateClusterRole)
		// PUT /api/k8s/rbac/cluster-roles/:cluster_id/:name - 更新ClusterRole
		clusterRoleGroup.PUT("/:cluster_id/:name", h.UpdateClusterRole)
		// DELETE /api/k8s/rbac/cluster-roles/:cluster_id/:name - 删除ClusterRole
		clusterRoleGroup.DELETE("/:cluster_id/:name", h.DeleteClusterRole)
	}

	// RoleBinding 管理路由
	roleBindingGroup := rbacGroup.Group("/role-bindings")
	{
		// GET /api/k8s/rbac/role-bindings/:cluster_id/:namespace - 获取指定命名空间的所有RoleBinding
		roleBindingGroup.GET("/:cluster_id/:namespace", h.GetRoleBindingsByNamespace)
		// GET /api/k8s/rbac/role-bindings/:cluster_id/:namespace/:name - 获取指定RoleBinding详情
		roleBindingGroup.GET("/:cluster_id/:namespace/:name", h.GetRoleBinding)
		// POST /api/k8s/rbac/role-bindings/:cluster_id/:namespace - 创建RoleBinding
		roleBindingGroup.POST("/:cluster_id/:namespace", h.CreateRoleBinding)
		// PUT /api/k8s/rbac/role-bindings/:cluster_id/:namespace/:name - 更新RoleBinding
		roleBindingGroup.PUT("/:cluster_id/:namespace/:name", h.UpdateRoleBinding)
		// DELETE /api/k8s/rbac/role-bindings/:cluster_id/:namespace/:name - 删除RoleBinding
		roleBindingGroup.DELETE("/:cluster_id/:namespace/:name", h.DeleteRoleBinding)
	}

	// ClusterRoleBinding 管理路由
	clusterRoleBindingGroup := rbacGroup.Group("/cluster-role-bindings")
	{
		// GET /api/k8s/rbac/cluster-role-bindings/:cluster_id - 获取集群所有ClusterRoleBinding
		clusterRoleBindingGroup.GET("/:cluster_id", h.GetClusterRoleBindings)
		// GET /api/k8s/rbac/cluster-role-bindings/:cluster_id/:name - 获取指定ClusterRoleBinding详情
		clusterRoleBindingGroup.GET("/:cluster_id/:name", h.GetClusterRoleBinding)
		// POST /api/k8s/rbac/cluster-role-bindings/:cluster_id - 创建ClusterRoleBinding
		clusterRoleBindingGroup.POST("/:cluster_id", h.CreateClusterRoleBinding)
		// PUT /api/k8s/rbac/cluster-role-bindings/:cluster_id/:name - 更新ClusterRoleBinding
		clusterRoleBindingGroup.PUT("/:cluster_id/:name", h.UpdateClusterRoleBinding)
		// DELETE /api/k8s/rbac/cluster-role-bindings/:cluster_id/:name - 删除ClusterRoleBinding
		clusterRoleBindingGroup.DELETE("/:cluster_id/:name", h.DeleteClusterRoleBinding)
	}
}

// ========== Role 管理 API ==========

// GetRolesByNamespace 获取指定命名空间的所有Role
// @Summary 获取Role列表
// @Description 根据集群ID和命名空间获取所有Role资源
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Success 200 {object} utils.Response{data=[]model.K8sRole} "成功返回Role列表"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/roles/{cluster_id}/{namespace} [get]
func (h *K8sRBACHandler) GetRolesByNamespace(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	namespace := c.Param("namespace")
	if namespace == "" {
		utils.BadRequestError(c, "命名空间名称不能为空")
		return
	}

	// 调用服务层获取Role列表
	roles, err := h.rbacService.GetRolesByNamespace(c.Request.Context(), clusterID, namespace)
	if err != nil {
		h.logger.Error("获取Role列表失败", zap.Error(err), zap.Int("cluster_id", clusterID), zap.String("namespace", namespace))
		utils.ErrorWithMessage(c, "获取Role列表失败: "+err.Error())
		return
	}

	utils.SuccessWithData(c, roles)
}

// GetRole 获取指定Role详情
// @Summary 获取Role详情
// @Description 根据集群ID、命名空间和Role名称获取Role详细信息
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Param name path string true "Role名称"
// @Success 200 {object} utils.Response{data=model.K8sRole} "成功返回Role详情"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 404 {object} utils.Response "Role未找到"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/roles/{cluster_id}/{namespace}/{name} [get]
func (h *K8sRBACHandler) GetRole(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	if namespace == "" || name == "" {
		utils.BadRequestError(c, "命名空间名称和Role名称不能为空")
		return
	}

	// 调用服务层获取Role详情
	role, err := h.rbacService.GetRole(c.Request.Context(), clusterID, namespace, name)
	if err != nil {
		h.logger.Error("获取Role详情失败", zap.Error(err), zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		utils.ErrorWithMessage(c, "获取Role详情失败: "+err.Error())
		return
	}

	utils.SuccessWithData(c, role)
}

// CreateRole 创建Role
// @Summary 创建Role
// @Description 在指定集群和命名空间中创建新的Role
// @Tags RBAC
// @Accept json
// @Produce json
// @Param request body model.CreateK8sRoleRequest true "创建Role请求参数"
// @Success 200 {object} utils.Response "成功创建Role"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/roles/{cluster_id}/{namespace} [post]
func (h *K8sRBACHandler) CreateRole(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	namespace := c.Param("namespace")
	if namespace == "" {
		utils.BadRequestError(c, "命名空间名称不能为空")
		return
	}

	var req model.CreateK8sRoleRequest

	// 绑定JSON参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 从路径参数设置cluster_id和namespace
	req.ClusterID = clusterID
	req.Namespace = namespace

	// 调用服务层创建Role
	if err := h.rbacService.CreateRole(c.Request.Context(), req); err != nil {
		h.logger.Error("创建Role失败", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		utils.ErrorWithMessage(c, "创建Role失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "Role创建成功")
}

// UpdateRole 更新Role
// @Summary 更新Role
// @Description 更新指定Role的配置信息
// @Tags RBAC
// @Accept json
// @Produce json
// @Param request body model.UpdateK8sRoleRequest true "更新Role请求参数"
// @Success 200 {object} utils.Response "成功更新Role"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/roles/{cluster_id}/{namespace}/{name} [put]
func (h *K8sRBACHandler) UpdateRole(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	if namespace == "" || name == "" {
		utils.BadRequestError(c, "命名空间名称和Role名称不能为空")
		return
	}

	var req model.UpdateK8sRoleRequest

	// 绑定JSON参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 从路径参数设置cluster_id、namespace和name
	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	// 调用服务层更新Role
	if err := h.rbacService.UpdateRole(c.Request.Context(), req); err != nil {
		h.logger.Error("更新Role失败", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		utils.ErrorWithMessage(c, "更新Role失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "Role更新成功")
}

// DeleteRole 删除Role
// @Summary 删除Role
// @Description 删除指定的Role资源
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Param name path string true "Role名称"
// @Success 200 {object} utils.Response "成功删除Role"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/roles/{cluster_id}/{namespace}/{name} [delete]
func (h *K8sRBACHandler) DeleteRole(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	if namespace == "" || name == "" {
		utils.BadRequestError(c, "命名空间名称和Role名称不能为空")
		return
	}

	// 构建删除请求
	req := model.DeleteK8sRoleRequest{
		ClusterID: clusterID,
		Namespace: namespace,
		Name:      name,
	}

	// 调用服务层删除Role
	if err := h.rbacService.DeleteRole(c.Request.Context(), req); err != nil {
		h.logger.Error("删除Role失败", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		utils.ErrorWithMessage(c, "删除Role失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "Role删除成功")
}

// ========== ClusterRole 管理 API ==========

// GetClusterRoles 获取集群所有ClusterRole
// @Summary 获取ClusterRole列表
// @Description 根据集群ID获取所有ClusterRole资源
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Success 200 {object} utils.Response{data=[]model.K8sClusterRole} "成功返回ClusterRole列表"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/cluster-roles/{cluster_id} [get]
func (h *K8sRBACHandler) GetClusterRoles(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	// 调用服务层获取ClusterRole列表
	clusterRoles, err := h.rbacService.GetClusterRoles(c.Request.Context(), clusterID)
	if err != nil {
		h.logger.Error("获取ClusterRole列表失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		utils.ErrorWithMessage(c, "获取ClusterRole列表失败: "+err.Error())
		return
	}

	utils.SuccessWithData(c, clusterRoles)
}

// GetClusterRole 获取指定ClusterRole详情
// @Summary 获取ClusterRole详情
// @Description 根据集群ID和ClusterRole名称获取ClusterRole详细信息
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "ClusterRole名称"
// @Success 200 {object} utils.Response{data=model.K8sClusterRole} "成功返回ClusterRole详情"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 404 {object} utils.Response "ClusterRole未找到"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/cluster-roles/{cluster_id}/{name} [get]
func (h *K8sRBACHandler) GetClusterRole(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	name := c.Param("name")
	if name == "" {
		utils.BadRequestError(c, "ClusterRole名称不能为空")
		return
	}

	// 调用服务层获取ClusterRole详情
	clusterRole, err := h.rbacService.GetClusterRole(c.Request.Context(), clusterID, name)
	if err != nil {
		h.logger.Error("获取ClusterRole详情失败", zap.Error(err), zap.Int("cluster_id", clusterID), zap.String("name", name))
		utils.ErrorWithMessage(c, "获取ClusterRole详情失败: "+err.Error())
		return
	}

	utils.SuccessWithData(c, clusterRole)
}

// CreateClusterRole 创建ClusterRole
// @Summary 创建ClusterRole
// @Description 在指定集群中创建新的ClusterRole
// @Tags RBAC
// @Accept json
// @Produce json
// @Param request body model.CreateClusterRoleRequest true "创建ClusterRole请求参数"
// @Success 200 {object} utils.Response "成功创建ClusterRole"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/cluster-roles/{cluster_id} [post]
func (h *K8sRBACHandler) CreateClusterRole(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	var req model.CreateClusterRoleRequest

	// 绑定JSON参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 从路径参数设置cluster_id
	req.ClusterID = clusterID

	// 调用服务层创建ClusterRole
	if err := h.rbacService.CreateClusterRole(c.Request.Context(), req); err != nil {
		h.logger.Error("创建ClusterRole失败", zap.Error(err), zap.String("name", req.Name))
		utils.ErrorWithMessage(c, "创建ClusterRole失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "ClusterRole创建成功")
}

// UpdateClusterRole 更新ClusterRole
// @Summary 更新ClusterRole
// @Description 更新指定ClusterRole的配置信息
// @Tags RBAC
// @Accept json
// @Produce json
// @Param request body model.UpdateClusterRoleRequest true "更新ClusterRole请求参数"
// @Success 200 {object} utils.Response "成功更新ClusterRole"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/cluster-roles/{cluster_id}/{name} [put]
func (h *K8sRBACHandler) UpdateClusterRole(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	name := c.Param("name")
	if name == "" {
		utils.BadRequestError(c, "ClusterRole名称不能为空")
		return
	}

	var req model.UpdateClusterRoleRequest

	// 绑定JSON参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 从路径参数设置cluster_id和name
	req.ClusterID = clusterID
	req.Name = name

	// 调用服务层更新ClusterRole
	if err := h.rbacService.UpdateClusterRole(c.Request.Context(), req); err != nil {
		h.logger.Error("更新ClusterRole失败", zap.Error(err), zap.String("name", req.Name))
		utils.ErrorWithMessage(c, "更新ClusterRole失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "ClusterRole更新成功")
}

// DeleteClusterRole 删除ClusterRole
// @Summary 删除ClusterRole
// @Description 删除指定的ClusterRole资源
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "ClusterRole名称"
// @Success 200 {object} utils.Response "成功删除ClusterRole"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/cluster-roles/{cluster_id}/{name} [delete]
func (h *K8sRBACHandler) DeleteClusterRole(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	name := c.Param("name")
	if name == "" {
		utils.BadRequestError(c, "ClusterRole名称不能为空")
		return
	}

	// 构建删除请求
	req := model.DeleteClusterRoleRequest{
		ClusterID: clusterID,
		Name:      name,
	}

	// 调用服务层删除ClusterRole
	if err := h.rbacService.DeleteClusterRole(c.Request.Context(), req); err != nil {
		h.logger.Error("删除ClusterRole失败", zap.Error(err), zap.String("name", req.Name))
		utils.ErrorWithMessage(c, "删除ClusterRole失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "ClusterRole删除成功")
}

// ========== RoleBinding 管理 API ==========

// GetRoleBindingsByNamespace 获取指定命名空间的所有RoleBinding
// @Summary 获取RoleBinding列表
// @Description 根据集群ID和命名空间获取所有RoleBinding资源
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Success 200 {object} utils.Response{data=[]model.K8sRoleBinding} "成功返回RoleBinding列表"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/role-bindings/{cluster_id}/{namespace} [get]
func (h *K8sRBACHandler) GetRoleBindingsByNamespace(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	namespace := c.Param("namespace")
	if namespace == "" {
		utils.BadRequestError(c, "命名空间名称不能为空")
		return
	}

	// 调用服务层获取RoleBinding列表
	roleBindings, err := h.rbacService.GetRoleBindingsByNamespace(c.Request.Context(), clusterID, namespace)
	if err != nil {
		h.logger.Error("获取RoleBinding列表失败", zap.Error(err), zap.Int("cluster_id", clusterID), zap.String("namespace", namespace))
		utils.ErrorWithMessage(c, "获取RoleBinding列表失败: "+err.Error())
		return
	}

	utils.SuccessWithData(c, roleBindings)
}

// GetRoleBinding 获取指定RoleBinding详情
// @Summary 获取RoleBinding详情
// @Description 根据集群ID、命名空间和RoleBinding名称获取RoleBinding详细信息
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Param name path string true "RoleBinding名称"
// @Success 200 {object} utils.Response{data=model.K8sRoleBinding} "成功返回RoleBinding详情"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 404 {object} utils.Response "RoleBinding未找到"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/role-bindings/{cluster_id}/{namespace}/{name} [get]
func (h *K8sRBACHandler) GetRoleBinding(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	if namespace == "" || name == "" {
		utils.BadRequestError(c, "命名空间名称和RoleBinding名称不能为空")
		return
	}

	// 调用服务层获取RoleBinding详情
	roleBinding, err := h.rbacService.GetRoleBinding(c.Request.Context(), clusterID, namespace, name)
	if err != nil {
		h.logger.Error("获取RoleBinding详情失败", zap.Error(err), zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		utils.ErrorWithMessage(c, "获取RoleBinding详情失败: "+err.Error())
		return
	}

	utils.SuccessWithData(c, roleBinding)
}

// CreateRoleBinding 创建RoleBinding
// @Summary 创建RoleBinding
// @Description 在指定集群和命名空间中创建新的RoleBinding
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Param request body model.CreateRoleBindingRequest true "创建RoleBinding请求参数"
// @Success 200 {object} utils.Response "成功创建RoleBinding"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/role-bindings/{cluster_id}/{namespace} [post]
func (h *K8sRBACHandler) CreateRoleBinding(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	namespace := c.Param("namespace")
	if namespace == "" {
		utils.BadRequestError(c, "命名空间名称不能为空")
		return
	}

	var req model.CreateRoleBindingRequest

	// 绑定JSON参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 从路径参数设置cluster_id和namespace
	req.ClusterID = clusterID
	req.Namespace = namespace

	// 调用服务层创建RoleBinding
	if err := h.rbacService.CreateRoleBinding(c.Request.Context(), req); err != nil {
		h.logger.Error("创建RoleBinding失败", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		utils.ErrorWithMessage(c, "创建RoleBinding失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "RoleBinding创建成功")
}

// UpdateRoleBinding 更新RoleBinding
// @Summary 更新RoleBinding
// @Description 更新指定RoleBinding的配置信息
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Param name path string true "RoleBinding名称"
// @Param request body model.UpdateRoleBindingRequest true "更新RoleBinding请求参数"
// @Success 200 {object} utils.Response "成功更新RoleBinding"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/role-bindings/{cluster_id}/{namespace}/{name} [put]
func (h *K8sRBACHandler) UpdateRoleBinding(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	if namespace == "" || name == "" {
		utils.BadRequestError(c, "命名空间名称和RoleBinding名称不能为空")
		return
	}

	var req model.UpdateRoleBindingRequest

	// 绑定JSON参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 从路径参数设置cluster_id、namespace和name
	req.ClusterID = clusterID
	req.Namespace = namespace
	req.Name = name

	// 调用服务层更新RoleBinding
	if err := h.rbacService.UpdateRoleBinding(c.Request.Context(), req); err != nil {
		h.logger.Error("更新RoleBinding失败", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		utils.ErrorWithMessage(c, "更新RoleBinding失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "RoleBinding更新成功")
}

// DeleteRoleBinding 删除RoleBinding
// @Summary 删除RoleBinding
// @Description 删除指定的RoleBinding资源
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Param name path string true "RoleBinding名称"
// @Success 200 {object} utils.Response "成功删除RoleBinding"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/role-bindings/{cluster_id}/{namespace}/{name} [delete]
func (h *K8sRBACHandler) DeleteRoleBinding(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	namespace := c.Param("namespace")
	name := c.Param("name")
	if namespace == "" || name == "" {
		utils.BadRequestError(c, "命名空间名称和RoleBinding名称不能为空")
		return
	}

	// 构建删除请求
	req := model.DeleteRoleBindingRequest{
		ClusterID: clusterID,
		Namespace: namespace,
		Name:      name,
	}

	// 调用服务层删除RoleBinding
	if err := h.rbacService.DeleteRoleBinding(c.Request.Context(), req); err != nil {
		h.logger.Error("删除RoleBinding失败", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		utils.ErrorWithMessage(c, "删除RoleBinding失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "RoleBinding删除成功")
}

// ========== ClusterRoleBinding 管理 API ==========

// GetClusterRoleBindings 获取集群所有ClusterRoleBinding
// @Summary 获取ClusterRoleBinding列表
// @Description 根据集群ID获取所有ClusterRoleBinding资源
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Success 200 {object} utils.Response{data=[]model.K8sClusterRoleBinding} "成功返回ClusterRoleBinding列表"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/cluster-role-bindings/{cluster_id} [get]
func (h *K8sRBACHandler) GetClusterRoleBindings(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	// 调用服务层获取ClusterRoleBinding列表
	clusterRoleBindings, err := h.rbacService.GetClusterRoleBindings(c.Request.Context(), clusterID)
	if err != nil {
		h.logger.Error("获取ClusterRoleBinding列表失败", zap.Error(err), zap.Int("cluster_id", clusterID))
		utils.ErrorWithMessage(c, "获取ClusterRoleBinding列表失败: "+err.Error())
		return
	}

	utils.SuccessWithData(c, clusterRoleBindings)
}

// GetClusterRoleBinding 获取指定ClusterRoleBinding详情
// @Summary 获取ClusterRoleBinding详情
// @Description 根据集群ID和ClusterRoleBinding名称获取ClusterRoleBinding详细信息
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "ClusterRoleBinding名称"
// @Success 200 {object} utils.Response{data=model.K8sClusterRoleBinding} "成功返回ClusterRoleBinding详情"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 404 {object} utils.Response "ClusterRoleBinding未找到"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/cluster-role-bindings/{cluster_id}/{name} [get]
func (h *K8sRBACHandler) GetClusterRoleBinding(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	name := c.Param("name")
	if name == "" {
		utils.BadRequestError(c, "ClusterRoleBinding名称不能为空")
		return
	}

	// 调用服务层获取ClusterRoleBinding详情
	clusterRoleBinding, err := h.rbacService.GetClusterRoleBinding(c.Request.Context(), clusterID, name)
	if err != nil {
		h.logger.Error("获取ClusterRoleBinding详情失败", zap.Error(err), zap.Int("cluster_id", clusterID), zap.String("name", name))
		utils.ErrorWithMessage(c, "获取ClusterRoleBinding详情失败: "+err.Error())
		return
	}

	utils.SuccessWithData(c, clusterRoleBinding)
}

// CreateClusterRoleBinding 创建ClusterRoleBinding
// @Summary 创建ClusterRoleBinding
// @Description 在指定集群中创建新的ClusterRoleBinding
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param request body model.CreateClusterRoleBindingRequest true "创建ClusterRoleBinding请求参数"
// @Success 200 {object} utils.Response "成功创建ClusterRoleBinding"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/cluster-role-bindings/{cluster_id} [post]
func (h *K8sRBACHandler) CreateClusterRoleBinding(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	var req model.CreateClusterRoleBindingRequest

	// 绑定JSON参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 从路径参数设置cluster_id
	req.ClusterID = clusterID

	// 调用服务层创建ClusterRoleBinding
	if err := h.rbacService.CreateClusterRoleBinding(c.Request.Context(), req); err != nil {
		h.logger.Error("创建ClusterRoleBinding失败", zap.Error(err), zap.String("name", req.Name))
		utils.ErrorWithMessage(c, "创建ClusterRoleBinding失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "ClusterRoleBinding创建成功")
}

// UpdateClusterRoleBinding 更新ClusterRoleBinding
// @Summary 更新ClusterRoleBinding
// @Description 更新指定ClusterRoleBinding的配置信息
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "ClusterRoleBinding名称"
// @Param request body model.UpdateClusterRoleBindingRequest true "更新ClusterRoleBinding请求参数"
// @Success 200 {object} utils.Response "成功更新ClusterRoleBinding"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/cluster-role-bindings/{cluster_id}/{name} [put]
func (h *K8sRBACHandler) UpdateClusterRoleBinding(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	name := c.Param("name")
	if name == "" {
		utils.BadRequestError(c, "ClusterRoleBinding名称不能为空")
		return
	}

	var req model.UpdateClusterRoleBindingRequest

	// 绑定JSON参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 从路径参数设置cluster_id和name
	req.ClusterID = clusterID
	req.Name = name

	// 调用服务层更新ClusterRoleBinding
	if err := h.rbacService.UpdateClusterRoleBinding(c.Request.Context(), req); err != nil {
		h.logger.Error("更新ClusterRoleBinding失败", zap.Error(err), zap.String("name", req.Name))
		utils.ErrorWithMessage(c, "更新ClusterRoleBinding失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "ClusterRoleBinding更新成功")
}

// DeleteClusterRoleBinding 删除ClusterRoleBinding
// @Summary 删除ClusterRoleBinding
// @Description 删除指定的ClusterRoleBinding资源
// @Tags RBAC
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param name path string true "ClusterRoleBinding名称"
// @Success 200 {object} utils.Response "成功删除ClusterRoleBinding"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/rbac/cluster-role-bindings/{cluster_id}/{name} [delete]
func (h *K8sRBACHandler) DeleteClusterRoleBinding(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	name := c.Param("name")
	if name == "" {
		utils.BadRequestError(c, "ClusterRoleBinding名称不能为空")
		return
	}

	// 构建删除请求
	req := model.DeleteClusterRoleBindingRequest{
		ClusterID: clusterID,
		Name:      name,
	}

	// 调用服务层删除ClusterRoleBinding
	if err := h.rbacService.DeleteClusterRoleBinding(c.Request.Context(), req); err != nil {
		h.logger.Error("删除ClusterRoleBinding失败", zap.Error(err), zap.String("name", req.Name))
		utils.ErrorWithMessage(c, "删除ClusterRoleBinding失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "ClusterRoleBinding删除成功")
}
