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
	"github.com/GoSimplicity/AI-CloudOps/internal/mcp/service"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/pkg/utils"
	"github.com/gin-gonic/gin"
)

type McpHandler struct {
	svc service.McpService
}

func NewMcpHandler(svc service.McpService) *McpHandler {
	return &McpHandler{
		svc: svc,
	}
}

func (m *McpHandler) RegisterRouters(server *gin.Engine) {
	mcpGroup := server.Group("/api/mcps")

	// 工具相关接口
	mcpGroup.GET("/tools", m.GetTools)
	mcpGroup.GET("/tools/:name", m.GetTool)
	mcpGroup.POST("/tools", m.CreateTool)
	mcpGroup.PUT("/tools", m.UpdateTool)
	mcpGroup.DELETE("/tools/:name", m.DeleteTool)
	mcpGroup.POST("/tools/call", m.CallTool)

	// MCP配置相关接口
	mcpGroup.GET("/configs", m.GetMCPConfigs)
	mcpGroup.GET("/configs/:id", m.GetMCPConfigByID)
	mcpGroup.POST("/configs", m.CreateMCPConfig)
	mcpGroup.PUT("/configs", m.UpdateMCPConfig)
	mcpGroup.DELETE("/configs/:id", m.DeleteMCPConfig)
	mcpGroup.POST("/configs/:id/connect", m.ConnectMCP)
	mcpGroup.POST("/configs/:id/disconnect", m.DisconnectMCP)
	mcpGroup.POST("/configs/test", m.TestMCPConnection)

	// 工具白名单相关接口
	mcpGroup.GET("/whitelists", m.GetToolWhitelists)
	mcpGroup.GET("/whitelists/:id", m.GetToolWhitelistByID)
	mcpGroup.POST("/whitelists", m.CreateToolWhitelist)
	mcpGroup.PUT("/whitelists", m.UpdateToolWhitelist)
	mcpGroup.DELETE("/whitelists/:id", m.DeleteToolWhitelist)
	mcpGroup.POST("/whitelists/add", m.AddToolToWhitelist)
	mcpGroup.POST("/whitelists/remove", m.RemoveToolFromWhitelist)

	// 工具黑名单相关接口
	mcpGroup.GET("/blacklists", m.GetToolBlacklists)
	mcpGroup.GET("/blacklists/:id", m.GetToolBlacklistByID)
	mcpGroup.POST("/blacklists", m.CreateToolBlacklist)
	mcpGroup.PUT("/blacklists", m.UpdateToolBlacklist)
	mcpGroup.DELETE("/blacklists/:id", m.DeleteToolBlacklist)
	mcpGroup.POST("/blacklists/add", m.AddToolToBlacklist)
	mcpGroup.POST("/blacklists/remove", m.RemoveToolFromBlacklist)
}

// GetTools 获取工具列表
// @Summary 获取工具列表
// @Description 获取所有可用的MCP工具列表
// @Tags MCP-Tool
// @Accept json
// @Produce json
// @Param request body model.GetToolsReq true "获取工具列表请求"
// @Success 200 {object} utils.ApiResponse{data=[]model.Tool} "成功获取工具列表"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/mcps/tools [get]
func (m *McpHandler) GetTools(ctx *gin.Context) {
	var req model.GetToolsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.GetTools(ctx, &req)
	})
}

// GetTool 获取指定工具
// @Summary 获取指定工具
// @Description 根据工具名称获取工具详情
// @Tags MCP-Tool
// @Accept json
// @Produce json
// @Param name path string true "工具名称"
// @Success 200 {object} utils.ApiResponse "成功获取工具详情"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/mcps/tools/{name} [get]
func (m *McpHandler) GetTool(ctx *gin.Context) {
	var req model.GetToolReq
	req.Name = ctx.Param("name")

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.GetTool(ctx, &req)
	})
}

// CreateTool 创建工具
// @Summary 创建工具
// @Description 创建新的MCP工具
// @Tags MCP-Tool
// @Accept json
// @Produce json
// @Param request body model.CreateToolReq true "创建工具请求"
// @Success 200 {object} utils.ApiResponse "成功创建工具"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/mcps/tools [post]
func (m *McpHandler) CreateTool(ctx *gin.Context) {
	var req model.CreateToolReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.CreateTool(ctx, &req)
	})
}

func (m *McpHandler) UpdateTool(ctx *gin.Context) {
	var req model.UpdateToolReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.UpdateTool(ctx, &req)
	})
}

func (m *McpHandler) DeleteTool(ctx *gin.Context) {
	var req model.DeleteToolReq
	req.Name = ctx.Param("name")

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, m.svc.DeleteTool(ctx, &req)
	})
}

func (m *McpHandler) CallTool(ctx *gin.Context) {
	var req model.CallToolReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.CallTool(ctx, &req)
	})
}

// GetMCPConfigs 获取MCP配置列表
// @Summary 获取MCP配置列表
// @Description 获取所有MCP服务配置列表
// @Tags MCP-Config
// @Accept json
// @Produce json
// @Param request body model.GetMCPConfigsReq true "获取MCP配置列表请求"
// @Success 200 {object} utils.ApiResponse{data=[]model.MCPConfig} "成功获取MCP配置列表"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/mcps/configs [get]
func (m *McpHandler) GetMCPConfigs(ctx *gin.Context) {
	var req model.GetMCPConfigsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.GetMCPConfigs(ctx, &req)
	})
}

func (m *McpHandler) GetMCPConfigByID(ctx *gin.Context) {
	var req model.GetMCPConfigReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.GetMCPConfigByID(ctx, &req)
	})
}

// CreateMCPConfig 创建MCP配置
// @Summary 创建MCP配置
// @Description 创建新的MCP服务配置
// @Tags MCP-Config
// @Accept json
// @Produce json
// @Param request body model.CreateMCPConfigReq true "创建MCP配置请求"
// @Success 200 {object} utils.ApiResponse "成功创建MCP配置"
// @Failure 400 {object} utils.ApiResponse "请求参数错误"
// @Failure 500 {object} utils.ApiResponse "服务器内部错误"
// @Router /api/mcps/configs [post]
func (m *McpHandler) CreateMCPConfig(ctx *gin.Context) {
	var req model.CreateMCPConfigReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.CreateMCPConfig(ctx, &req)
	})
}

func (m *McpHandler) UpdateMCPConfig(ctx *gin.Context) {
	var req model.UpdateMCPConfigReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.UpdateMCPConfig(ctx, &req)
	})
}

func (m *McpHandler) DeleteMCPConfig(ctx *gin.Context) {
	var req model.DeleteMCPConfigReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, m.svc.DeleteMCPConfig(ctx, &req)
	})
}

func (m *McpHandler) ConnectMCP(ctx *gin.Context) {
	var req model.ConnectMCPReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, m.svc.ConnectMCP(ctx, &req)
	})
}

func (m *McpHandler) DisconnectMCP(ctx *gin.Context) {
	var req model.DisconnectMCPReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, m.svc.DisconnectMCP(ctx, &req)
	})
}

func (m *McpHandler) TestMCPConnection(ctx *gin.Context) {
	var req model.TestMCPConnectionReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, m.svc.TestMCPConnection(ctx, &req)
	})
}

// 工具白名单相关接口实现
func (m *McpHandler) GetToolWhitelists(ctx *gin.Context) {
	var req model.GetToolWhitelistsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.GetToolWhitelists(ctx, &req)
	})
}

func (m *McpHandler) GetToolWhitelistByID(ctx *gin.Context) {
	var req model.GetToolWhitelistReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.GetToolWhitelistByID(ctx, &req)
	})
}

func (m *McpHandler) CreateToolWhitelist(ctx *gin.Context) {
	var req model.CreateToolWhitelistReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.CreateToolWhitelist(ctx, &req)
	})
}

func (m *McpHandler) UpdateToolWhitelist(ctx *gin.Context) {
	var req model.UpdateToolWhitelistReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.UpdateToolWhitelist(ctx, &req)
	})
}

func (m *McpHandler) DeleteToolWhitelist(ctx *gin.Context) {
	var req model.DeleteToolWhitelistReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, m.svc.DeleteToolWhitelist(ctx, &req)
	})
}

func (m *McpHandler) AddToolToWhitelist(ctx *gin.Context) {
	var req model.AddToolToWhitelistReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, m.svc.AddToolToWhitelist(ctx, &req)
	})
}

func (m *McpHandler) RemoveToolFromWhitelist(ctx *gin.Context) {
	var req model.RemoveToolFromWhitelistReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, m.svc.RemoveToolFromWhitelist(ctx, &req)
	})
}

// 工具黑名单相关接口实现
func (m *McpHandler) GetToolBlacklists(ctx *gin.Context) {
	var req model.GetToolBlacklistsReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.GetToolBlacklists(ctx, &req)
	})
}

func (m *McpHandler) GetToolBlacklistByID(ctx *gin.Context) {
	var req model.GetToolBlacklistReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.GetToolBlacklistByID(ctx, &req)
	})
}

func (m *McpHandler) CreateToolBlacklist(ctx *gin.Context) {
	var req model.CreateToolBlacklistReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.CreateToolBlacklist(ctx, &req)
	})
}

func (m *McpHandler) UpdateToolBlacklist(ctx *gin.Context) {
	var req model.UpdateToolBlacklistReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return m.svc.UpdateToolBlacklist(ctx, &req)
	})
}

func (m *McpHandler) DeleteToolBlacklist(ctx *gin.Context) {
	var req model.DeleteToolBlacklistReq

	id, err := utils.GetParamID(ctx)
	if err != nil {
		return
	}

	req.ID = id

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, m.svc.DeleteToolBlacklist(ctx, &req)
	})
}

func (m *McpHandler) AddToolToBlacklist(ctx *gin.Context) {
	var req model.AddToolToBlacklistReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, m.svc.AddToolToBlacklist(ctx, &req)
	})
}

func (m *McpHandler) RemoveToolFromBlacklist(ctx *gin.Context) {
	var req model.RemoveToolFromBlacklistReq

	utils.HandleRequest(ctx, &req, func() (interface{}, error) {
		return nil, m.svc.RemoveToolFromBlacklist(ctx, &req)
	})
}
