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

// K8sServiceAccountHandler ServiceAccount（服务账户）API处理器
// 提供 Kubernetes ServiceAccount 资源管理的 HTTP API 接口
type K8sServiceAccountHandler struct {
	logger                *zap.Logger                 // 日志记录器
	serviceAccountService admin.ServiceAccountService // ServiceAccount服务
}

// NewK8sServiceAccountHandler 创建新的ServiceAccount API处理器实例
// @param logger 日志记录器
// @param serviceAccountService ServiceAccount服务实例
// @return *K8sServiceAccountHandler ServiceAccount API处理器实例
func NewK8sServiceAccountHandler(logger *zap.Logger, serviceAccountService admin.ServiceAccountService) *K8sServiceAccountHandler {
	return &K8sServiceAccountHandler{
		logger:                logger,
		serviceAccountService: serviceAccountService,
	}
}

// RegisterRouters 注册ServiceAccount相关的路由
// @param router Gin路由引擎实例
func (h *K8sServiceAccountHandler) RegisterRouters(router *gin.Engine) {
	// 创建API分组
	k8sGroup := router.Group("/api/k8s")
	saGroup := k8sGroup.Group("/service-accounts")

	// ServiceAccount 基本管理路由
	{
		// GET /api/k8s/service-accounts/:cluster_id/:namespace - 获取指定命名空间的所有ServiceAccount
		saGroup.GET("/:cluster_id/:namespace", h.GetServiceAccountsByNamespace)
		// GET /api/k8s/service-accounts/:cluster_id/:namespace/:name - 获取指定ServiceAccount详情
		saGroup.GET("/:cluster_id/:namespace/:name", h.GetServiceAccount)
		// POST /api/k8s/service-accounts/:cluster_id/:namespace - 创建ServiceAccount
		saGroup.POST("/:cluster_id/:namespace", h.CreateServiceAccount)
		// PUT /api/k8s/service-accounts/:cluster_id/:namespace/:name - 更新ServiceAccount
		saGroup.PUT("/:cluster_id/:namespace/:name", h.UpdateServiceAccount)
		// DELETE /api/k8s/service-accounts/:cluster_id/:namespace/:name - 删除ServiceAccount
		saGroup.DELETE("/:cluster_id/:namespace/:name", h.DeleteServiceAccount)
	}

	// ServiceAccount Token 管理路由
	tokenGroup := saGroup.Group("/tokens")
	{
		// POST /api/k8s/service-accounts/tokens/:cluster_id/:namespace/:service_account_name - 创建ServiceAccount Token
		tokenGroup.POST("/:cluster_id/:namespace/:service_account_name", h.CreateServiceAccountToken)
	}

	// ServiceAccount 权限管理路由
	permissionGroup := saGroup.Group("/permissions")
	{
		// GET /api/k8s/service-accounts/permissions/:cluster_id/:namespace/:name - 获取ServiceAccount权限信息
		permissionGroup.GET("/:cluster_id/:namespace/:name", h.GetServiceAccountPermissions)
		// POST /api/k8s/service-accounts/permissions/:cluster_id/:namespace/bind-role - 绑定Role到ServiceAccount
		permissionGroup.POST("/:cluster_id/:namespace/bind-role", h.BindRoleToServiceAccount)
		// POST /api/k8s/service-accounts/permissions/:cluster_id/:namespace/bind-cluster-role - 绑定ClusterRole到ServiceAccount
		permissionGroup.POST("/:cluster_id/:namespace/bind-cluster-role", h.BindClusterRoleToServiceAccount)
		// DELETE /api/k8s/service-accounts/permissions/:cluster_id/:namespace/unbind-role/:role_binding_name - 解绑Role从ServiceAccount
		permissionGroup.DELETE("/:cluster_id/:namespace/unbind-role/:role_binding_name", h.UnbindRoleFromServiceAccount)
		// DELETE /api/k8s/service-accounts/permissions/:cluster_id/unbind-cluster-role/:cluster_role_binding_name - 解绑ClusterRole从ServiceAccount
		permissionGroup.DELETE("/:cluster_id/unbind-cluster-role/:cluster_role_binding_name", h.UnbindClusterRoleFromServiceAccount)
	}
}

// ========== ServiceAccount 基本管理 API ==========

// GetServiceAccountsByNamespace 获取指定命名空间的所有ServiceAccount
// @Summary 获取ServiceAccount列表
// @Description 根据集群ID和命名空间获取所有ServiceAccount资源
// @Tags ServiceAccount
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Success 200 {object} utils.Response{data=[]model.K8sServiceAccount} "成功返回ServiceAccount列表"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/service-accounts/{cluster_id}/{namespace} [get]
func (h *K8sServiceAccountHandler) GetServiceAccountsByNamespace(c *gin.Context) {
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

	// 调用服务层获取ServiceAccount列表
	serviceAccounts, err := h.serviceAccountService.GetServiceAccountsByNamespace(c.Request.Context(), clusterID, namespace)
	if err != nil {
		h.logger.Error("获取ServiceAccount列表失败", zap.Error(err), zap.Int("cluster_id", clusterID), zap.String("namespace", namespace))
		utils.ErrorWithMessage(c, "获取ServiceAccount列表失败: "+err.Error())
		return
	}

	utils.SuccessWithData(c, serviceAccounts)
}

// GetServiceAccount 获取指定ServiceAccount详情
// @Summary 获取ServiceAccount详情
// @Description 根据集群ID、命名空间和ServiceAccount名称获取ServiceAccount详细信息
// @Tags ServiceAccount
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Param name path string true "ServiceAccount名称"
// @Success 200 {object} utils.Response{data=model.K8sServiceAccount} "成功返回ServiceAccount详情"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 404 {object} utils.Response "ServiceAccount未找到"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/service-accounts/{cluster_id}/{namespace}/{name} [get]
func (h *K8sServiceAccountHandler) GetServiceAccount(c *gin.Context) {
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
		utils.BadRequestError(c, "命名空间名称和ServiceAccount名称不能为空")
		return
	}

	// 调用服务层获取ServiceAccount详情
	serviceAccount, err := h.serviceAccountService.GetServiceAccount(c.Request.Context(), clusterID, namespace, name)
	if err != nil {
		h.logger.Error("获取ServiceAccount详情失败", zap.Error(err), zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("name", name))
		utils.ErrorWithMessage(c, "获取ServiceAccount详情失败: "+err.Error())
		return
	}

	utils.SuccessWithData(c, serviceAccount)
}

// CreateServiceAccount 创建ServiceAccount
// @Summary 创建ServiceAccount
// @Description 在指定集群和命名空间中创建新的ServiceAccount
// @Tags ServiceAccount
// @Accept json
// @Produce json
// @Param request body model.CreateServiceAccountRequest true "创建ServiceAccount请求参数"
// @Success 200 {object} utils.Response "成功创建ServiceAccount"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/service-accounts/{cluster_id}/{namespace} [post]
func (h *K8sServiceAccountHandler) CreateServiceAccount(c *gin.Context) {
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

	var req model.CreateServiceAccountRequest

	// 绑定JSON参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 从路径参数设置cluster_id和namespace
	req.ClusterID = clusterID
	req.Namespace = namespace

	// 调用服务层创建ServiceAccount
	if err := h.serviceAccountService.CreateServiceAccount(c.Request.Context(), req); err != nil {
		h.logger.Error("创建ServiceAccount失败", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		utils.ErrorWithMessage(c, "创建ServiceAccount失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "ServiceAccount创建成功")
}

// UpdateServiceAccount 更新ServiceAccount
// @Summary 更新ServiceAccount
// @Description 更新指定ServiceAccount的配置信息
// @Tags ServiceAccount
// @Accept json
// @Produce json
// @Param request body model.UpdateServiceAccountRequest true "更新ServiceAccount请求参数"
// @Success 200 {object} utils.Response "成功更新ServiceAccount"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/service-accounts/{cluster_id}/{namespace}/{name} [put]
func (h *K8sServiceAccountHandler) UpdateServiceAccount(c *gin.Context) {
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
		utils.BadRequestError(c, "命名空间名称和ServiceAccount名称不能为空")
		return
	}

	var req model.UpdateServiceAccountRequest

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

	// 调用服务层更新ServiceAccount
	if err := h.serviceAccountService.UpdateServiceAccount(c.Request.Context(), req); err != nil {
		h.logger.Error("更新ServiceAccount失败", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		utils.ErrorWithMessage(c, "更新ServiceAccount失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "ServiceAccount更新成功")
}

// DeleteServiceAccount 删除ServiceAccount
// @Summary 删除ServiceAccount
// @Description 删除指定的ServiceAccount资源
// @Tags ServiceAccount
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Param name path string true "ServiceAccount名称"
// @Success 200 {object} utils.Response "成功删除ServiceAccount"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/service-accounts/{cluster_id}/{namespace}/{name} [delete]
func (h *K8sServiceAccountHandler) DeleteServiceAccount(c *gin.Context) {
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
		utils.BadRequestError(c, "命名空间名称和ServiceAccount名称不能为空")
		return
	}

	// 构建删除请求
	req := model.DeleteServiceAccountRequest{
		ClusterID: clusterID,
		Namespace: namespace,
		Name:      name,
	}

	// 调用服务层删除ServiceAccount
	if err := h.serviceAccountService.DeleteServiceAccount(c.Request.Context(), req); err != nil {
		h.logger.Error("删除ServiceAccount失败", zap.Error(err), zap.String("name", req.Name), zap.String("namespace", req.Namespace))
		utils.ErrorWithMessage(c, "删除ServiceAccount失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "ServiceAccount删除成功")
}

// ========== ServiceAccount Token 管理 API ==========

// CreateServiceAccountToken 创建ServiceAccount Token
// @Summary 创建ServiceAccount Token
// @Description 为指定的ServiceAccount创建访问令牌
// @Tags ServiceAccount
// @Accept json
// @Produce json
// @Param request body model.ServiceAccountTokenRequest true "创建Token请求参数"
// @Success 200 {object} utils.Response{data=model.ServiceAccountToken} "成功创建Token"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/service-accounts/tokens/{cluster_id}/{namespace}/{service_account_name} [post]
func (h *K8sServiceAccountHandler) CreateServiceAccountToken(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	namespace := c.Param("namespace")
	serviceAccountName := c.Param("service_account_name")
	if namespace == "" || serviceAccountName == "" {
		utils.BadRequestError(c, "命名空间名称和ServiceAccount名称不能为空")
		return
	}

	var req model.ServiceAccountTokenRequest

	// 绑定JSON参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 从路径参数设置参数
	req.ClusterID = clusterID
	req.Namespace = namespace
	req.ServiceAccountName = serviceAccountName

	// 调用服务层创建Token
	token, err := h.serviceAccountService.CreateServiceAccountToken(c.Request.Context(), req)
	if err != nil {
		h.logger.Error("创建ServiceAccount Token失败", zap.Error(err), zap.String("service_account", req.ServiceAccountName), zap.String("namespace", req.Namespace))
		utils.ErrorWithMessage(c, "创建ServiceAccount Token失败: "+err.Error())
		return
	}

	utils.SuccessWithData(c, token)
}

// ========== ServiceAccount 权限管理 API ==========

// GetServiceAccountPermissions 获取ServiceAccount权限信息
// @Summary 获取ServiceAccount权限信息
// @Description 获取指定ServiceAccount的所有权限绑定信息，包括Role和ClusterRole
// @Tags ServiceAccount
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Param name path string true "ServiceAccount名称"
// @Success 200 {object} utils.Response{data=model.ServiceAccountPermissions} "成功返回权限信息"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/service-accounts/permissions/{cluster_id}/{namespace}/{name} [get]
func (h *K8sServiceAccountHandler) GetServiceAccountPermissions(c *gin.Context) {
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
		utils.BadRequestError(c, "命名空间名称和ServiceAccount名称不能为空")
		return
	}

	// 调用服务层获取权限信息
	permissions, err := h.serviceAccountService.GetServiceAccountPermissions(c.Request.Context(), clusterID, namespace, name)
	if err != nil {
		h.logger.Error("获取ServiceAccount权限信息失败", zap.Error(err), zap.Int("cluster_id", clusterID), zap.String("namespace", namespace), zap.String("service_account", name))
		utils.ErrorWithMessage(c, "获取ServiceAccount权限信息失败: "+err.Error())
		return
	}

	utils.SuccessWithData(c, permissions)
}

// BindRoleToServiceAccount 绑定Role到ServiceAccount
// @Summary 绑定Role到ServiceAccount
// @Description 通过创建RoleBinding将Role权限绑定到ServiceAccount
// @Tags ServiceAccount
// @Accept json
// @Produce json
// @Param request body model.BindRoleToServiceAccountRequest true "绑定Role请求参数"
// @Success 200 {object} utils.Response "成功绑定Role"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/service-accounts/permissions/{cluster_id}/{namespace}/bind-role [post]
func (h *K8sServiceAccountHandler) BindRoleToServiceAccount(c *gin.Context) {
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

	var req model.BindRoleToServiceAccountRequest

	// 绑定JSON参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 从路径参数设置参数
	req.ClusterID = clusterID
	req.Namespace = namespace

	// 调用服务层绑定Role
	if err := h.serviceAccountService.BindRoleToServiceAccount(c.Request.Context(), req); err != nil {
		h.logger.Error("绑定Role到ServiceAccount失败", zap.Error(err), zap.String("service_account", req.ServiceAccountName), zap.String("role", req.RoleName))
		utils.ErrorWithMessage(c, "绑定Role到ServiceAccount失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "Role绑定成功")
}

// BindClusterRoleToServiceAccount 绑定ClusterRole到ServiceAccount
// @Summary 绑定ClusterRole到ServiceAccount
// @Description 通过创建ClusterRoleBinding将ClusterRole权限绑定到ServiceAccount
// @Tags ServiceAccount
// @Accept json
// @Produce json
// @Param request body model.BindClusterRoleToServiceAccountRequest true "绑定ClusterRole请求参数"
// @Success 200 {object} utils.Response "成功绑定ClusterRole"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/service-accounts/permissions/{cluster_id}/{namespace}/bind-cluster-role [post]
func (h *K8sServiceAccountHandler) BindClusterRoleToServiceAccount(c *gin.Context) {
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

	var req model.BindClusterRoleToServiceAccountRequest

	// 绑定JSON参数
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("绑定请求参数失败", zap.Error(err))
		utils.BadRequestError(c, "请求参数格式错误: "+err.Error())
		return
	}

	// 从路径参数设置参数
	req.ClusterID = clusterID
	req.Namespace = namespace

	// 调用服务层绑定ClusterRole
	if err := h.serviceAccountService.BindClusterRoleToServiceAccount(c.Request.Context(), req); err != nil {
		h.logger.Error("绑定ClusterRole到ServiceAccount失败", zap.Error(err), zap.String("service_account", req.ServiceAccountName), zap.String("cluster_role", req.ClusterRoleName))
		utils.ErrorWithMessage(c, "绑定ClusterRole到ServiceAccount失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "ClusterRole绑定成功")
}

// UnbindRoleFromServiceAccount 解绑Role从ServiceAccount
// @Summary 解绑Role从ServiceAccount
// @Description 通过删除RoleBinding解除Role权限绑定
// @Tags ServiceAccount
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param namespace path string true "命名空间名称"
// @Param role_binding_name path string true "RoleBinding名称"
// @Success 200 {object} utils.Response "成功解绑Role"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/service-accounts/permissions/{cluster_id}/{namespace}/unbind-role/{role_binding_name} [delete]
func (h *K8sServiceAccountHandler) UnbindRoleFromServiceAccount(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	namespace := c.Param("namespace")
	roleBindingName := c.Param("role_binding_name")
	if namespace == "" || roleBindingName == "" {
		utils.BadRequestError(c, "命名空间名称和RoleBinding名称不能为空")
		return
	}

	// 构建解绑请求
	req := model.UnbindRoleFromServiceAccountRequest{
		ClusterID:       clusterID,
		Namespace:       namespace,
		RoleBindingName: roleBindingName,
	}

	// 调用服务层解绑Role
	if err := h.serviceAccountService.UnbindRoleFromServiceAccount(c.Request.Context(), req); err != nil {
		h.logger.Error("解绑Role从ServiceAccount失败", zap.Error(err), zap.String("service_account", req.ServiceAccountName), zap.String("role_binding", req.RoleBindingName))
		utils.ErrorWithMessage(c, "解绑Role从ServiceAccount失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "Role解绑成功")
}

// UnbindClusterRoleFromServiceAccount 解绑ClusterRole从ServiceAccount
// @Summary 解绑ClusterRole从ServiceAccount
// @Description 通过删除ClusterRoleBinding解除ClusterRole权限绑定
// @Tags ServiceAccount
// @Accept json
// @Produce json
// @Param cluster_id path int true "集群ID"
// @Param cluster_role_binding_name path string true "ClusterRoleBinding名称"
// @Success 200 {object} utils.Response "成功解绑ClusterRole"
// @Failure 400 {object} utils.Response "请求参数错误"
// @Failure 500 {object} utils.Response "服务器内部错误"
// @Router /api/k8s/service-accounts/permissions/{cluster_id}/unbind-cluster-role/{cluster_role_binding_name} [delete]
func (h *K8sServiceAccountHandler) UnbindClusterRoleFromServiceAccount(c *gin.Context) {
	// 解析路径参数
	clusterID, err := strconv.Atoi(c.Param("cluster_id"))
	if err != nil {
		h.logger.Error("解析集群ID失败", zap.Error(err))
		utils.BadRequestError(c, "无效的集群ID")
		return
	}

	clusterRoleBindingName := c.Param("cluster_role_binding_name")
	if clusterRoleBindingName == "" {
		utils.BadRequestError(c, "ClusterRoleBinding名称不能为空")
		return
	}

	// 构建解绑请求
	req := model.UnbindClusterRoleFromServiceAccountRequest{
		ClusterID:              clusterID,
		ClusterRoleBindingName: clusterRoleBindingName,
	}

	// 调用服务层解绑ClusterRole
	if err := h.serviceAccountService.UnbindClusterRoleFromServiceAccount(c.Request.Context(), req); err != nil {
		h.logger.Error("解绑ClusterRole从ServiceAccount失败", zap.Error(err), zap.String("cluster_role_binding", req.ClusterRoleBindingName))
		utils.ErrorWithMessage(c, "解绑ClusterRole从ServiceAccount失败: "+err.Error())
		return
	}

	utils.SuccessWithMessage(c, "ClusterRole解绑成功")
}
