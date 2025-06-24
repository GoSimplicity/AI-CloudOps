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
	"fmt"
	"strconv"
	"strings"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/service"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// TreeCloudHandler 云账号管理处理器
// @Summary 云账号管理
// @Description 提供云账号的创建、更新、删除、查询等管理功能
type TreeCloudHandler struct {
	cloudService service.TreeCloudService
	logger       *zap.Logger
}

// NewTreeCloudHandler 创建云账号处理器
func NewTreeCloudHandler(cloudService service.TreeCloudService, logger *zap.Logger) *TreeCloudHandler {
	return &TreeCloudHandler{
		cloudService: cloudService,
		logger:       logger,
	}
}

// RegisterRouters 注册路由
func (h *TreeCloudHandler) RegisterRouters(r gin.IRouter) {
	cloudGroup := r.Group("/api/tree/cloud")
	{
		// 云账号管理
		accounts := cloudGroup.Group("/accounts")
		{
			accounts.POST("/create", h.CreateCloudAccount)
			accounts.GET("/list", h.ListCloudAccounts)
			accounts.GET("/detail/:id", h.DetailCloudAccount)
			accounts.PUT("/update/:id", h.UpdateCloudAccount)
			accounts.DELETE("/delete/:id", h.DeleteCloudAccount)
			accounts.POST("/test/:id", h.TestCloudAccount)
			accounts.POST("/batch/delete", h.BatchDeleteCloudAccounts)
			accounts.POST("/batch/test", h.BatchTestCloudAccounts)
		}

		// 云资源同步
		cloudGroup.POST("/sync", h.SyncCloudResources)
		cloudGroup.POST("/sync/:id", h.SyncCloudAccountResources)

		// 云账号统计
		cloudGroup.GET("/statistics", h.GetCloudAccountStatistics)
	}
}

// CreateCloudAccount 创建云账号
// @Summary 创建云账号
// @Description 创建新的云服务账号，支持阿里云、华为云等主流云厂商
// @Tags 云账号管理
// @Accept json
// @Produce json
// @Param account body model.CreateCloudAccountReq true "云账号信息"
// @Success 200 {object} utils.ApiResponse{data=string} "创建成功"
// @Failure 400 {object} utils.ApiResponse{data=string} "参数错误"
// @Failure 401 {object} utils.ApiResponse{data=string} "未认证"
// @Failure 500 {object} utils.ApiResponse{data=string} "服务器内部错误"
// @Router /api/tree/cloud/accounts/create [post]
func (h *TreeCloudHandler) CreateCloudAccount(ctx *gin.Context) {
	var req model.CreateCloudAccountReq

	// 获取用户信息
	user := h.getUserFromContext(ctx)
	if user == nil {
		utils.UnauthorizedErrorWithDetails(ctx, nil, "用户未认证")
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		// 输入验证
		if err := h.validateCreateCloudAccountRequest(&req); err != nil {
			return nil, err
		}

		// 记录操作日志
		h.logger.Info("创建云账号",
			zap.String("username", user.Username),
			zap.String("account_name", req.Name),
			zap.String("provider", string(req.Provider)),
			zap.String("ip", user.IP),
		)

		return nil, h.cloudService.CreateCloudAccount(ctx, &req)
	})
}

// UpdateCloudAccount 更新云账号
// @Summary 更新云账号
// @Description 更新指定ID的云账号信息
// @Tags 云账号管理
// @Accept json
// @Produce json
// @Param id path int true "云账号ID"
// @Param account body model.UpdateCloudAccountReq true "更新的云账号信息"
// @Success 200 {object} utils.ApiResponse{data=string} "更新成功"
// @Failure 400 {object} utils.ApiResponse{data=string} "参数错误"
// @Failure 401 {object} utils.ApiResponse{data=string} "未认证"
// @Failure 404 {object} utils.ApiResponse{data=string} "账号不存在"
// @Failure 500 {object} utils.ApiResponse{data=string} "服务器内部错误"
// @Router /api/tree/cloud/accounts/update/{id} [put]
func (h *TreeCloudHandler) UpdateCloudAccount(ctx *gin.Context) {
	var req model.UpdateCloudAccountReq

	// 获取用户信息
	user := h.getUserFromContext(ctx)
	if user == nil {
		utils.UnauthorizedErrorWithDetails(ctx, nil, "用户未认证")
		return
	}

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "账号ID格式错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		// 输入验证
		if err := h.validateUpdateCloudAccountRequest(&req); err != nil {
			return nil, err
		}

		// 记录操作日志
		h.logger.Info("更新云账号",
			zap.String("username", user.Username),
			zap.Int("account_id", id),
			zap.String("ip", user.IP),
		)

		return nil, h.cloudService.UpdateCloudAccount(ctx, req.ID, &req)
	})
}

// DeleteCloudAccount 删除云账号
// @Summary 删除云账号
// @Description 删除指定ID的云账号
// @Tags 云账号管理
// @Accept json
// @Produce json
// @Param id path int true "云账号ID"
// @Success 200 {object} utils.ApiResponse{data=string} "删除成功"
// @Failure 400 {object} utils.ApiResponse{data=string} "参数错误"
// @Failure 401 {object} utils.ApiResponse{data=string} "未认证"
// @Failure 404 {object} utils.ApiResponse{data=string} "账号不存在"
// @Failure 500 {object} utils.ApiResponse{data=string} "服务器内部错误"
// @Router /api/tree/cloud/accounts/delete/{id} [delete]
func (h *TreeCloudHandler) DeleteCloudAccount(ctx *gin.Context) {
	// 获取用户信息
	user := h.getUserFromContext(ctx)
	if user == nil {
		utils.UnauthorizedErrorWithDetails(ctx, nil, "用户未认证")
		return
	}

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "账号ID格式错误")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		// 记录操作日志
		h.logger.Info("删除云账号",
			zap.String("username", user.Username),
			zap.Int("account_id", id),
			zap.String("ip", user.IP),
		)

		return nil, h.cloudService.DeleteCloudAccount(ctx, id)
	})
}

// BatchDeleteCloudAccounts 批量删除云账号
// @Summary 批量删除云账号
// @Description 批量删除多个云账号，单次最多删除50个
// @Tags 云账号管理
// @Accept json
// @Produce json
// @Param request body model.BatchDeleteCloudAccountsReq true "批量删除请求"
// @Success 200 {object} utils.ApiResponse{data=string} "删除成功"
// @Failure 400 {object} utils.ApiResponse{data=string} "参数错误"
// @Failure 401 {object} utils.ApiResponse{data=string} "未认证"
// @Failure 500 {object} utils.ApiResponse{data=string} "服务器内部错误"
// @Router /api/tree/cloud/accounts/batch/delete [post]
func (h *TreeCloudHandler) BatchDeleteCloudAccounts(ctx *gin.Context) {
	var req model.BatchDeleteCloudAccountsReq

	// 获取用户信息
	user := h.getUserFromContext(ctx)
	if user == nil {
		utils.UnauthorizedErrorWithDetails(ctx, nil, "用户未认证")
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		// 输入验证
		if err := h.validateBatchDeleteRequest(&req); err != nil {
			return nil, err
		}

		// 记录操作日志
		h.logger.Info("批量删除云账号",
			zap.String("username", user.Username),
			zap.Ints("account_ids", req.AccountIDs),
			zap.String("ip", user.IP),
		)

		return nil, h.cloudService.BatchDeleteCloudAccounts(ctx, req.AccountIDs)
	})
}

// DetailCloudAccount 获取云账号详情
// @Summary 获取云账号详情
// @Description 获取指定ID的云账号详细信息
// @Tags 云账号管理
// @Accept json
// @Produce json
// @Param id path int true "云账号ID"
// @Success 200 {object} utils.ApiResponse{data=model.CloudAccount} "获取成功"
// @Failure 400 {object} utils.ApiResponse{data=string} "参数错误"
// @Failure 401 {object} utils.ApiResponse{data=string} "未认证"
// @Failure 404 {object} utils.ApiResponse{data=string} "账号不存在"
// @Failure 500 {object} utils.ApiResponse{data=string} "服务器内部错误"
// @Router /api/tree/cloud/accounts/detail/{id} [get]
func (h *TreeCloudHandler) DetailCloudAccount(ctx *gin.Context) {
	var req model.GetCloudAccountReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "账号ID格式错误")
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cloudService.GetCloudAccount(ctx, req.ID)
	})
}

// ListCloudAccounts 获取云账号列表
// @Summary 获取云账号列表
// @Description 分页获取云账号列表，支持按名称、云厂商、状态等条件过滤
// @Tags 云账号管理
// @Accept json
// @Produce json
// @Param page query int false "页码，默认为1"
// @Param page_size query int false "每页大小，默认为10，最大100"
// @Param name query string false "账号名称，支持模糊匹配"
// @Param provider query string false "云厂商：aliyun、huawei、aws、tencent"
// @Param enabled query bool false "是否启用"
// @Success 200 {object} utils.ApiResponse{data=model.ListResp{items=[]model.CloudAccount}} "获取成功"
// @Failure 400 {object} utils.ApiResponse{data=string} "参数错误"
// @Failure 401 {object} utils.ApiResponse{data=string} "未认证"
// @Failure 500 {object} utils.ApiResponse{data=string} "服务器内部错误"
// @Router /api/tree/cloud/accounts/list [get]
func (h *TreeCloudHandler) ListCloudAccounts(ctx *gin.Context) {
	var req model.ListCloudAccountsReq

	// 从查询参数获取分页信息
	if pageStr := ctx.Query("page"); pageStr != "" {
		if page, err := strconv.Atoi(pageStr); err == nil && page > 0 {
			req.Page = page
		} else {
			utils.BadRequestError(ctx, "页码必须是正整数")
			return
		}
	}

	if pageSizeStr := ctx.Query("page_size"); pageSizeStr != "" {
		if pageSize, err := strconv.Atoi(pageSizeStr); err == nil && pageSize > 0 && pageSize <= 100 {
			req.PageSize = pageSize
		} else {
			utils.BadRequestError(ctx, "每页大小必须是1-100之间的正整数")
			return
		}
	}

	// 从查询参数获取过滤条件
	req.Name = strings.TrimSpace(ctx.Query("name"))
	if provider := strings.TrimSpace(ctx.Query("provider")); provider != "" {
		if !h.isValidProvider(provider) {
			utils.BadRequestError(ctx, "不支持的云服务提供商")
			return
		}
		req.Provider = model.CloudProvider(provider)
	}
	req.Enabled = ctx.Query("enabled") == "true"

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return h.cloudService.ListCloudAccounts(ctx, &req)
	})
}

// TestCloudAccount 测试云账号连接
// @Summary 测试云账号连接
// @Description 测试指定云账号的连接是否正常
// @Tags 云账号管理
// @Accept json
// @Produce json
// @Param id path int true "云账号ID"
// @Success 200 {object} utils.ApiResponse{data=string} "连接测试成功"
// @Failure 400 {object} utils.ApiResponse{data=string} "参数错误"
// @Failure 401 {object} utils.ApiResponse{data=string} "未认证"
// @Failure 404 {object} utils.ApiResponse{data=string} "账号不存在"
// @Failure 500 {object} utils.ApiResponse{data=string} "服务器内部错误"
// @Router /api/tree/cloud/accounts/test/{id} [post]
func (h *TreeCloudHandler) TestCloudAccount(ctx *gin.Context) {
	// 获取用户信息
	user := h.getUserFromContext(ctx)
	if user == nil {
		utils.UnauthorizedErrorWithDetails(ctx, nil, "用户未认证")
		return
	}

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "账号ID格式错误")
		return
	}

	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		// 记录操作日志
		h.logger.Info("测试云账号连接",
			zap.String("username", user.Username),
			zap.Int("account_id", id),
			zap.String("ip", user.IP),
		)

		return nil, h.cloudService.TestCloudAccount(ctx, id)
	})
}

// BatchTestCloudAccounts 批量测试云账号连接
// @Summary 批量测试云账号连接
// @Description 批量测试多个云账号的连接状态，单次最多测试20个
// @Tags 云账号管理
// @Accept json
// @Produce json
// @Param request body model.BatchTestCloudAccountsReq true "批量测试请求"
// @Success 200 {object} utils.ApiResponse{data=map[int]error} "测试结果，key为账号ID，value为错误信息"
// @Failure 400 {object} utils.ApiResponse{data=string} "参数错误"
// @Failure 401 {object} utils.ApiResponse{data=string} "未认证"
// @Failure 500 {object} utils.ApiResponse{data=string} "服务器内部错误"
// @Router /api/tree/cloud/accounts/batch/test [post]
func (h *TreeCloudHandler) BatchTestCloudAccounts(ctx *gin.Context) {
	var req model.BatchTestCloudAccountsReq

	// 获取用户信息
	user := h.getUserFromContext(ctx)
	if user == nil {
		utils.UnauthorizedErrorWithDetails(ctx, nil, "用户未认证")
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		// 输入验证
		if err := h.validateBatchTestRequest(&req); err != nil {
			return nil, err
		}

		// 记录操作日志
		h.logger.Info("批量测试云账号连接",
			zap.String("username", user.Username),
			zap.Ints("account_ids", req.AccountIDs),
			zap.String("ip", user.IP),
		)

		return h.cloudService.BatchTestCloudAccounts(ctx, req.AccountIDs)
	})
}

// SyncCloudResources 同步所有云资源
// @Summary 同步所有云资源
// @Description 同步所有启用的云账号资源
// @Tags 云资源同步
// @Accept json
// @Produce json
// @Param request body model.SyncCloudReq false "同步请求参数"
// @Success 200 {object} utils.ApiResponse{data=string} "同步成功"
// @Failure 400 {object} utils.ApiResponse{data=string} "参数错误"
// @Failure 401 {object} utils.ApiResponse{data=string} "未认证"
// @Failure 500 {object} utils.ApiResponse{data=string} "服务器内部错误"
// @Router /api/tree/cloud/sync [post]
func (h *TreeCloudHandler) SyncCloudResources(ctx *gin.Context) {
	var req model.SyncCloudReq

	// 获取用户信息
	user := h.getUserFromContext(ctx)
	if user == nil {
		utils.UnauthorizedErrorWithDetails(ctx, nil, "用户未认证")
		return
	}

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		// 记录操作日志
		h.logger.Info("同步所有云资源",
			zap.String("username", user.Username),
			zap.String("ip", user.IP),
		)

		return nil, h.cloudService.SyncCloudResources(ctx, &req)
	})
}

// SyncCloudAccountResources 同步指定云账号的资源
// @Summary 同步指定云账号资源
// @Description 同步指定云账号的资源信息
// @Tags 云资源同步
// @Accept json
// @Produce json
// @Param id path int true "云账号ID"
// @Param request body model.SyncCloudAccountResourcesReq false "同步请求参数"
// @Success 200 {object} utils.ApiResponse{data=string} "同步成功"
// @Failure 400 {object} utils.ApiResponse{data=string} "参数错误"
// @Failure 401 {object} utils.ApiResponse{data=string} "未认证"
// @Failure 404 {object} utils.ApiResponse{data=string} "账号不存在"
// @Failure 500 {object} utils.ApiResponse{data=string} "服务器内部错误"
// @Router /api/tree/cloud/sync/{id} [post]
func (h *TreeCloudHandler) SyncCloudAccountResources(ctx *gin.Context) {
	var req model.SyncCloudAccountResourcesReq

	// 获取用户信息
	user := h.getUserFromContext(ctx)
	if user == nil {
		utils.UnauthorizedErrorWithDetails(ctx, nil, "用户未认证")
		return
	}

	id, err := utils.GetParamID(ctx)
	if err != nil {
		utils.BadRequestError(ctx, "账号ID格式错误")
		return
	}

	req.AccountID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		// 记录操作日志
		h.logger.Info("同步云账号资源",
			zap.String("username", user.Username),
			zap.Int("account_id", id),
			zap.String("ip", user.IP),
		)

		return nil, h.cloudService.SyncCloudAccountResources(ctx, &req)
	})
}

// GetCloudAccountStatistics 获取云账号统计信息
// @Summary 获取云账号统计信息
// @Description 获取云账号的统计信息，包括总数、各云厂商分布等
// @Tags 云账号管理
// @Accept json
// @Produce json
// @Success 200 {object} utils.ApiResponse{data=model.CloudAccountStatistics} "获取成功"
// @Failure 401 {object} utils.ApiResponse{data=string} "未认证"
// @Failure 500 {object} utils.ApiResponse{data=string} "服务器内部错误"
// @Router /api/tree/cloud/statistics [get]
func (h *TreeCloudHandler) GetCloudAccountStatistics(ctx *gin.Context) {
	utils.HandleRequest(ctx, nil, func() (interface{}, error) {
		return h.cloudService.GetCloudAccountStatistics(ctx)
	})
}

// getUserFromContext 从上下文中获取用户信息
func (h *TreeCloudHandler) getUserFromContext(ctx *gin.Context) *utils.UserInfo {
	userInfo := utils.GetUserInfoFromContext(ctx)
	if userInfo.UserID == 0 {
		return nil
	}
	return userInfo
}

// validateCreateCloudAccountRequest 验证创建云账号请求
func (h *TreeCloudHandler) validateCreateCloudAccountRequest(req *model.CreateCloudAccountReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	// 验证必填字段
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("账号名称不能为空")
	}

	if len(strings.TrimSpace(req.Name)) > 100 {
		return fmt.Errorf("账号名称长度不能超过100个字符")
	}

	if strings.TrimSpace(string(req.Provider)) == "" {
		return fmt.Errorf("云服务提供商不能为空")
	}

	if !h.isValidProvider(string(req.Provider)) {
		return fmt.Errorf("不支持的云服务提供商: %s", req.Provider)
	}

	if strings.TrimSpace(req.AccountId) == "" {
		return fmt.Errorf("账户ID不能为空")
	}

	if len(strings.TrimSpace(req.AccountId)) > 100 {
		return fmt.Errorf("账户ID长度不能超过100个字符")
	}

	if strings.TrimSpace(req.AccessKey) == "" {
		return fmt.Errorf("访问密钥ID不能为空")
	}

	if len(strings.TrimSpace(req.AccessKey)) > 100 {
		return fmt.Errorf("访问密钥ID长度不能超过100个字符")
	}

	if strings.TrimSpace(req.SecretKey) == "" {
		return fmt.Errorf("访问密钥不能为空")
	}

	// 验证区域列表
	if len(req.Regions) > 0 {
		for _, region := range req.Regions {
			if strings.TrimSpace(region) == "" {
				return fmt.Errorf("区域名称不能为空")
			}
		}
	}

	// 验证描述长度
	if len(req.Description) > 500 {
		return fmt.Errorf("描述长度不能超过500个字符")
	}

	return nil
}

// validateUpdateCloudAccountRequest 验证更新云账号请求
func (h *TreeCloudHandler) validateUpdateCloudAccountRequest(req *model.UpdateCloudAccountReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	if req.ID <= 0 {
		return fmt.Errorf("无效的账号ID")
	}

	// 验证必填字段
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("账号名称不能为空")
	}

	if len(strings.TrimSpace(req.Name)) > 100 {
		return fmt.Errorf("账号名称长度不能超过100个字符")
	}

	if strings.TrimSpace(string(req.Provider)) == "" {
		return fmt.Errorf("云服务提供商不能为空")
	}

	if !h.isValidProvider(string(req.Provider)) {
		return fmt.Errorf("不支持的云服务提供商: %s", req.Provider)
	}

	if strings.TrimSpace(req.AccountId) == "" {
		return fmt.Errorf("账户ID不能为空")
	}

	if len(strings.TrimSpace(req.AccountId)) > 100 {
		return fmt.Errorf("账户ID长度不能超过100个字符")
	}

	if strings.TrimSpace(req.AccessKey) == "" {
		return fmt.Errorf("访问密钥ID不能为空")
	}

	if len(strings.TrimSpace(req.AccessKey)) > 100 {
		return fmt.Errorf("访问密钥ID长度不能超过100个字符")
	}

	// 验证区域列表
	if len(req.Regions) > 0 {
		for _, region := range req.Regions {
			if strings.TrimSpace(region) == "" {
				return fmt.Errorf("区域名称不能为空")
			}
		}
	}

	// 验证描述长度
	if len(req.Description) > 500 {
		return fmt.Errorf("描述长度不能超过500个字符")
	}

	return nil
}

// validateBatchDeleteRequest 验证批量删除请求
func (h *TreeCloudHandler) validateBatchDeleteRequest(req *model.BatchDeleteCloudAccountsReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	if len(req.AccountIDs) == 0 {
		return fmt.Errorf("账号ID列表不能为空")
	}

	if len(req.AccountIDs) > 50 {
		return fmt.Errorf("单次最多只能删除50个账号")
	}

	// 验证ID的有效性
	for _, id := range req.AccountIDs {
		if id <= 0 {
			return fmt.Errorf("无效的账号ID: %d", id)
		}
	}

	// 检查是否有重复ID
	idMap := make(map[int]bool)
	for _, id := range req.AccountIDs {
		if idMap[id] {
			return fmt.Errorf("存在重复的账号ID: %d", id)
		}
		idMap[id] = true
	}

	return nil
}

// validateBatchTestRequest 验证批量测试请求
func (h *TreeCloudHandler) validateBatchTestRequest(req *model.BatchTestCloudAccountsReq) error {
	if req == nil {
		return fmt.Errorf("请求参数不能为空")
	}

	if len(req.AccountIDs) == 0 {
		return fmt.Errorf("账号ID列表不能为空")
	}

	if len(req.AccountIDs) > 20 {
		return fmt.Errorf("单次最多只能测试20个账号")
	}

	// 验证ID的有效性
	for _, id := range req.AccountIDs {
		if id <= 0 {
			return fmt.Errorf("无效的账号ID: %d", id)
		}
	}

	// 检查是否有重复ID
	idMap := make(map[int]bool)
	for _, id := range req.AccountIDs {
		if idMap[id] {
			return fmt.Errorf("存在重复的账号ID: %d", id)
		}
		idMap[id] = true
	}

	return nil
}

// isValidProvider 验证云服务提供商是否有效
func (h *TreeCloudHandler) isValidProvider(provider string) bool {
	validProviders := []string{"aliyun", "huawei", "aws", "tencent", "local"}
	for _, validProvider := range validProviders {
		if provider == validProvider {
			return true
		}
	}
	return false
}
