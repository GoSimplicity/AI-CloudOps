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

package generator

import (
	"go/ast"
)

// SwaggerInfo Swagger文档基本信息
type SwaggerInfo struct {
	Title       string   `json:"title"`
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Host        string   `json:"host"`
	BasePath    string   `json:"basePath"`
	Schemes     []string `json:"schemes"`
}

// APIEndpoint API端点信息
type APIEndpoint struct {
	Path        string                `json:"path"`
	Method      string                `json:"method"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	Responses   map[string]Response   `json:"responses"`
	Security    []map[string][]string `json:"security,omitempty"`
	OperationID string                `json:"operationId,omitempty"`
}

// Parameter 参数信息
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // query, path, header, body, formData
	Type        string  `json:"type,omitempty"`
	Format      string  `json:"format,omitempty"`
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
	Items       *Items  `json:"items,omitempty"`
}

// Response 响应信息
type Response struct {
	Description string            `json:"description"`
	Schema      *Schema           `json:"schema,omitempty"`
	Headers     map[string]Header `json:"headers,omitempty"`
}

// Header 响应头信息
type Header struct {
	Type        string `json:"type"`
	Format      string `json:"format,omitempty"`
	Description string `json:"description,omitempty"`
}

// Schema 数据模型
type Schema struct {
	Type                 string             `json:"type,omitempty"`
	Format               string             `json:"format,omitempty"`
	Ref                  string             `json:"$ref,omitempty"`
	Items                *Schema            `json:"items,omitempty"`
	Properties           map[string]*Schema `json:"properties,omitempty"`
	Required             []string           `json:"required,omitempty"`
	Description          string             `json:"description,omitempty"`
	Example              interface{}        `json:"example,omitempty"`
	AdditionalProperties interface{}        `json:"additionalProperties,omitempty"`
}

// Items 数组项信息
type Items struct {
	Type   string `json:"type,omitempty"`
	Format string `json:"format,omitempty"`
	Ref    string `json:"$ref,omitempty"`
}

// Definition 数据模型定义
type Definition struct {
	Type        string             `json:"type"`
	Properties  map[string]*Schema `json:"properties,omitempty"`
	Required    []string           `json:"required,omitempty"`
	Description string             `json:"description,omitempty"`
}

// SwaggerDoc 完整的Swagger文档
type SwaggerDoc struct {
	Swagger             string                            `json:"swagger"`
	Info                SwaggerInfo                       `json:"info"`
	Host                string                            `json:"host,omitempty"`
	BasePath            string                            `json:"basePath,omitempty"`
	Schemes             []string                          `json:"schemes,omitempty"`
	Consumes            []string                          `json:"consumes,omitempty"`
	Produces            []string                          `json:"produces,omitempty"`
	Paths               map[string]map[string]APIEndpoint `json:"paths"`
	Definitions         map[string]Definition             `json:"definitions,omitempty"`
	SecurityDefinitions map[string]SecurityDefinition     `json:"securityDefinitions,omitempty"`
	Tags                []Tag                             `json:"tags,omitempty"`
}

// SecurityDefinition 安全定义
type SecurityDefinition struct {
	Type        string `json:"type"`
	Name        string `json:"name,omitempty"`
	In          string `json:"in,omitempty"`
	Description string `json:"description,omitempty"`
}

// Tag 标签
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// HandlerInfo 处理函数信息
type HandlerInfo struct {
	Name         string
	FuncDecl     *ast.FuncDecl
	ReceiverType string
	PackageName  string
	FilePath     string
}

// RouteInfo 路由信息
type RouteInfo struct {
	Method      string
	Path        string
	Handler     string
	HandlerInfo *HandlerInfo
	Middleware  []string
	Group       string
}

// StructInfo 结构体信息
type StructInfo struct {
	Name          string
	Fields        []FieldInfo
	Package       string
	File          string
	EmbeddedTypes []string // 嵌入的类型列表（如 ListReq）
}

// FieldInfo 字段信息
type FieldInfo struct {
	Name         string
	Type         string
	Tag          string
	JSONName     string
	FormName     string // form tag 用于查询参数
	URIName      string // uri tag 用于路径参数
	Required     bool
	Description  string
	EmbeddedType string // 嵌套类型名称（如果是嵌套结构体）
}

// PackageInfo 包信息
type PackageInfo struct {
	Name  string
	Path  string
	Files []*ast.File
}
