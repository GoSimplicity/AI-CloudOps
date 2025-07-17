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

package model

// Tool 工具定义
type Tool struct {
	Name        string                 `json:"name"`        // 工具名称
	Description string                 `json:"description"` // 工具描述
	Parameters  ToolParameters         `json:"parameters"`  // 工具参数定义
	Metadata    map[string]interface{} `json:"metadata"`    // 工具元数据
}

// ToolParameters 工具参数定义
type ToolParameters struct {
	Type       string                 `json:"type"`                 // 参数类型
	Properties map[string]PropertyDef `json:"properties,omitempty"` // 参数属性定义
	Required   []string               `json:"required,omitempty"`   // 必需的参数列表
}

// PropertyDef 参数属性定义
type PropertyDef struct {
	Type        string        `json:"type,omitempty"`        // 属性类型
	Description string        `json:"description,omitempty"` // 属性描述
	Default     interface{}   `json:"default,omitempty"`     // 默认值
	Enum        []interface{} `json:"enum,omitempty"`        // 枚举值
	Minimum     *float64      `json:"minimum,omitempty"`     // 最小值
	Maximum     *float64      `json:"maximum,omitempty"`     // 最大值
	Items       *ItemDef      `json:"items,omitempty"`       // 数组项定义
}

// ItemDef 数组项定义
type ItemDef struct {
	Type string        `json:"type,omitempty"` // 项类型
	Enum []interface{} `json:"enum,omitempty"` // 枚举值
}

// MCPConfig MCP配置
type MCPConfig struct {
	Model
	URL       string         `json:"url" gorm:"type:varchar(255)"` // MCP服务URL
	Whitelist *ToolWhitelist `json:"whitelist,omitempty" gorm:"-"` // 工具白名单
	Blacklist *ToolBlacklist `json:"blacklist,omitempty" gorm:"-"` // 工具黑名单
}

// TableName 设置表名
func (MCPConfig) TableName() string {
	return "ac_mcp_config"
}

// ToolWhitelist 工具白名单
type ToolWhitelist struct {
	Model
	Tools []string `json:"tools" gorm:"type:json"` // 白名单工具列表
}

// TableName 设置表名
func (ToolWhitelist) TableName() string {
	return "ac_mcp_tool_whitelist"
}

// ToolBlacklist 工具黑名单
type ToolBlacklist struct {
	Model
	Tools []string `json:"tools" gorm:"type:json"` // 黑名单工具列表
}

// TableName 设置表名
func (ToolBlacklist) TableName() string {
	return "ac_mcp_tool_blacklist"
}

// GetToolsReq 获取工具列表请求
type GetToolsReq struct {
	ListReq
	Name        string `json:"name" form:"name" binding:"omitempty"`               // 工具名称筛选
	Description string `json:"description" form:"description" binding:"omitempty"` // 工具描述筛选
}

// GetToolReq 获取单个工具请求
type GetToolReq struct {
	Name string `json:"name" form:"name" binding:"required"` // 工具名称
}

// CreateToolReq 创建工具请求
type CreateToolReq struct {
	Name        string                 `json:"name" binding:"required"`        // 工具名称
	Description string                 `json:"description" binding:"required"` // 工具描述
	Parameters  ToolParameters         `json:"parameters"`                     // 工具参数定义
	Metadata    map[string]interface{} `json:"metadata"`                       // 工具元数据
}

// UpdateToolReq 更新工具请求
type UpdateToolReq struct {
	Name        string                 `json:"name" binding:"required"` // 工具名称
	Description string                 `json:"description"`             // 工具描述
	Parameters  ToolParameters         `json:"parameters"`              // 工具参数定义
	Metadata    map[string]interface{} `json:"metadata"`                // 工具元数据
}

// DeleteToolReq 删除工具请求
type DeleteToolReq struct {
	Name string `json:"name" form:"name" binding:"required"` // 工具名称
}

// CallToolReq 调用工具请求
type CallToolReq struct {
	Name      string                 `json:"name" binding:"required"` // 工具名称
	Arguments map[string]interface{} `json:"arguments"`               // 工具调用参数
}

// GetMCPConfigsReq 获取MCP配置列表请求
type GetMCPConfigsReq struct {
	ListReq
	URL string `json:"url" form:"url" binding:"omitempty"` // MCP服务URL筛选
}

// GetMCPConfigReq 获取MCP配置请求
type GetMCPConfigReq struct {
	ID int `json:"id" form:"id" binding:"required"` // MCP配置ID
}

// CreateMCPConfigReq 创建MCP配置请求
type CreateMCPConfigReq struct {
	URL string `json:"url" binding:"required"` // MCP服务URL
}

// UpdateMCPConfigReq 更新MCP配置请求
type UpdateMCPConfigReq struct {
	ID  int    `json:"id" binding:"required"`  // MCP配置ID
	URL string `json:"url" binding:"required"` // MCP服务URL
}

// DeleteMCPConfigReq 删除MCP配置请求
type DeleteMCPConfigReq struct {
	ID int `json:"id" form:"id" binding:"required"` // MCP配置ID
}

// GetToolWhitelistsReq 获取工具白名单列表请求
type GetToolWhitelistsReq struct {
	ListReq
}

// GetToolWhitelistReq 获取工具白名单请求
type GetToolWhitelistReq struct {
	ID int `json:"id" form:"id" binding:"required"` // 白名单ID
}

// CreateToolWhitelistReq 创建工具白名单请求
type CreateToolWhitelistReq struct {
	Tools []string `json:"tools" binding:"required"` // 白名单工具列表
}

// UpdateToolWhitelistReq 更新工具白名单请求
type UpdateToolWhitelistReq struct {
	ID    int      `json:"id" binding:"required"`    // 白名单ID
	Tools []string `json:"tools" binding:"required"` // 白名单工具列表
}

// DeleteToolWhitelistReq 删除工具白名单请求
type DeleteToolWhitelistReq struct {
	ID int `json:"id" form:"id" binding:"required"` // 白名单ID
}

// AddToolToWhitelistReq 添加工具到白名单请求
type AddToolToWhitelistReq struct {
	ID       int    `json:"id" binding:"required"`        // 白名单ID
	ToolName string `json:"tool_name" binding:"required"` // 工具名称
}

// RemoveToolFromWhitelistReq 从白名单移除工具请求
type RemoveToolFromWhitelistReq struct {
	ID       int    `json:"id" binding:"required"`        // 白名单ID
	ToolName string `json:"tool_name" binding:"required"` // 工具名称
}

// GetToolBlacklistsReq 获取工具黑名单列表请求
type GetToolBlacklistsReq struct {
	ListReq
}

// GetToolBlacklistReq 获取工具黑名单请求
type GetToolBlacklistReq struct {
	ID int `json:"id" form:"id" binding:"required"` // 黑名单ID
}

// CreateToolBlacklistReq 创建工具黑名单请求
type CreateToolBlacklistReq struct {
	Tools []string `json:"tools" binding:"required"` // 黑名单工具列表
}

// UpdateToolBlacklistReq 更新工具黑名单请求
type UpdateToolBlacklistReq struct {
	ID    int      `json:"id" binding:"required"`    // 黑名单ID
	Tools []string `json:"tools" binding:"required"` // 黑名单工具列表
}

// DeleteToolBlacklistReq 删除工具黑名单请求
type DeleteToolBlacklistReq struct {
	ID int `json:"id" form:"id" binding:"required"` // 黑名单ID
}

// AddToolToBlacklistReq 添加工具到黑名单请求
type AddToolToBlacklistReq struct {
	ID     int `json:"id" binding:"required"`      // 黑名单ID
	ToolID int `json:"tool_id" binding:"required"` // 工具ID
}

// RemoveToolFromBlacklistReq 从黑名单移除工具请求
type RemoveToolFromBlacklistReq struct {
	ID     int `json:"id" binding:"required"`      // 黑名单ID
	ToolID int `json:"tool_id" binding:"required"` // 工具ID
}

// ConnectMCPReq 连接MCP服务请求
type ConnectMCPReq struct {
	ID int `json:"id" binding:"required"` // MCP配置ID
}

// DisconnectMCPReq 断开MCP服务请求
type DisconnectMCPReq struct {
	ID int `json:"id" binding:"required"` // MCP配置ID
}

// GetMCPStatusReq 获取MCP服务状态请求
type GetMCPStatusReq struct {
	ID int `json:"id" binding:"required"` // MCP配置ID
}

// TestMCPConnectionReq 测试MCP连接请求
type TestMCPConnectionReq struct {
	URL string `json:"url" binding:"required"` // MCP服务URL
}
