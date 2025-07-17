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

package dao

import (
	"context"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"gorm.io/gorm"
)

type McpDAO interface {
	// 工具相关操作
	GetTools(ctx context.Context, req *model.GetToolsReq) ([]model.Tool, int, error)
	GetTool(ctx context.Context, req *model.GetToolReq) (*model.Tool, error)

	// MCP配置相关操作
	GetMCPConfig(ctx context.Context) ([]model.MCPConfig, error)
	GetMCPConfigByID(ctx context.Context, req *model.GetMCPConfigReq) (*model.MCPConfig, error)
	UpdateMCPConfig(ctx context.Context, id int, req *model.UpdateMCPConfigReq) error

	// 工具白名单相关操作
	GetToolWhitelist(ctx context.Context) ([]model.ToolWhitelist, error)
	GetToolWhitelistByID(ctx context.Context, req *model.GetToolWhitelistReq) (*model.ToolWhitelist, error)
	UpdateToolWhitelist(ctx context.Context, id int, req *model.UpdateToolWhitelistReq) error

	// 工具黑名单相关操作
	GetToolBlacklist(ctx context.Context) ([]model.ToolBlacklist, error)
	GetToolBlacklistByID(ctx context.Context, req *model.GetToolBlacklistReq) (*model.ToolBlacklist, error)
	UpdateToolBlacklist(ctx context.Context, id int, req *model.UpdateToolBlacklistReq) error
}

type mcpDAO struct {
	db *gorm.DB
}

func NewMcpDAO(db *gorm.DB) McpDAO {
	return &mcpDAO{db: db}
}

// GetTools 获取工具列表
func (m *mcpDAO) GetTools(ctx context.Context, req *model.GetToolsReq) ([]model.Tool, int, error) {
	return nil, 0, nil
}

// GetTool 获取单个工具
func (m *mcpDAO) GetTool(ctx context.Context, req *model.GetToolReq) (*model.Tool, error) {
	return nil, nil
}

// GetMCPConfig 获取MCP配置列表
func (m *mcpDAO) GetMCPConfig(ctx context.Context) ([]model.MCPConfig, error) {
	return nil, nil
}

// GetMCPConfigByID 根据ID获取MCP配置
func (m *mcpDAO) GetMCPConfigByID(ctx context.Context, req *model.GetMCPConfigReq) (*model.MCPConfig, error) {
	return nil, nil
}

// UpdateMCPConfig 更新MCP配置
func (m *mcpDAO) UpdateMCPConfig(ctx context.Context, id int, req *model.UpdateMCPConfigReq) error {
	return nil
}

// GetToolWhitelist 获取工具白名单列表
func (m *mcpDAO) GetToolWhitelist(ctx context.Context) ([]model.ToolWhitelist, error) {
	return nil, nil
}

// GetToolWhitelistByID 根据ID获取工具白名单
func (m *mcpDAO) GetToolWhitelistByID(ctx context.Context, req *model.GetToolWhitelistReq) (*model.ToolWhitelist, error) {
	return nil, nil
}

// UpdateToolWhitelist 更新工具白名单
func (m *mcpDAO) UpdateToolWhitelist(ctx context.Context, id int, req *model.UpdateToolWhitelistReq) error {
	return nil
}

// GetToolBlacklist 获取工具黑名单列表
func (m *mcpDAO) GetToolBlacklist(ctx context.Context) ([]model.ToolBlacklist, error) {
	return nil, nil
}

// GetToolBlacklistByID 根据ID获取工具黑名单
func (m *mcpDAO) GetToolBlacklistByID(ctx context.Context, req *model.GetToolBlacklistReq) (*model.ToolBlacklist, error) {
	return nil, nil
}

// UpdateToolBlacklist 更新工具黑名单
func (m *mcpDAO) UpdateToolBlacklist(ctx context.Context, id int, req *model.UpdateToolBlacklistReq) error {
	return nil
}
