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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// SwaggerGenerator Swagger文档生成器
type SwaggerGenerator struct {
	projectRoot string
	outputDir   string
	verbose     bool
	parser      *Parser
}

// NewSwaggerGenerator 创建新的生成器
func NewSwaggerGenerator(projectRoot, outputDir string, verbose bool) *SwaggerGenerator {
	return &SwaggerGenerator{
		projectRoot: projectRoot,
		outputDir:   outputDir,
		verbose:     verbose,
		parser:      NewParser(projectRoot, verbose),
	}
}

// Generate 生成Swagger文档
func (g *SwaggerGenerator) Generate() error {
	if g.verbose {
		fmt.Println("🔧 开始生成 Swagger 文档...")
	}

	// 解析项目
	if err := g.parser.ParseProject(); err != nil {
		return fmt.Errorf("解析项目失败: %v", err)
	}

	// 构建Swagger文档
	swaggerDoc := g.buildSwaggerDoc()

	// 生成JSON文档
	if err := g.writeJSON(swaggerDoc); err != nil {
		return fmt.Errorf("生成JSON文档失败: %v", err)
	}

	// 生成YAML文档
	if err := g.writeYAML(swaggerDoc); err != nil {
		return fmt.Errorf("生成YAML文档失败: %v", err)
	}

	// 生成Go文档
	if err := g.writeGoDoc(swaggerDoc); err != nil {
		return fmt.Errorf("生成Go文档失败: %v", err)
	}

	return nil
}

// buildSwaggerDoc 构建Swagger文档
func (g *SwaggerGenerator) buildSwaggerDoc() *SwaggerDoc {
	doc := &SwaggerDoc{
		Swagger: "2.0",
		Info: SwaggerInfo{
			Title:       "AI-CloudOps API",
			Version:     "1.0.0",
			Description: "AI-CloudOps云原生运维平台API文档 (自动生成)",
		},
		Host:        "localhost:8889",
		BasePath:    "/",
		Schemes:     []string{"http", "https"},
		Consumes:    []string{"application/json"},
		Produces:    []string{"application/json"},
		Paths:       make(map[string]map[string]APIEndpoint),
		Definitions: make(map[string]Definition),
		SecurityDefinitions: map[string]SecurityDefinition{
			"BearerAuth": {
				Type:        "apiKey",
				Name:        "Authorization",
				In:          "header",
				Description: "Bearer Token认证",
			},
		},
		Tags: make([]Tag, 0),
	}

	// 构建路径和端点
	g.buildPaths(doc)

	// 构建定义
	g.buildDefinitions(doc)

	// 构建标签
	g.buildTags(doc)

	return doc
}

// buildPaths 构建API路径
func (g *SwaggerGenerator) buildPaths(doc *SwaggerDoc) {
	routes := g.parser.GetRoutes()

	for _, route := range routes {
		path := g.normalizePath(route.Path)
		method := strings.ToLower(route.Method)

		if doc.Paths[path] == nil {
			doc.Paths[path] = make(map[string]APIEndpoint)
		}

		endpoint := g.buildEndpoint(route)
		doc.Paths[path][method] = endpoint

		if g.verbose {
			fmt.Printf("📝 添加端点: %s %s\n", route.Method, path)
		}
	}
}

// buildEndpoint 构建API端点
func (g *SwaggerGenerator) buildEndpoint(route RouteInfo) APIEndpoint {
	endpoint := APIEndpoint{
		Path:        route.Path,
		Method:      route.Method,
		Summary:     g.generateSummary(route),
		Description: g.generateDescription(route),
		Tags:        g.generateTags(route),
		Parameters:  g.generateParameters(route),
		Responses:   g.generateResponses(route),
		OperationID: g.generateOperationID(route),
	}

	// 添加安全认证
	if g.needsAuth(route) {
		endpoint.Security = []map[string][]string{
			{"BearerAuth": []string{}},
		}
	}

	return endpoint
}

// generateSummary 生成摘要
func (g *SwaggerGenerator) generateSummary(route RouteInfo) string {
	// 从路径生成摘要
	pathParts := strings.Split(strings.Trim(route.Path, "/"), "/")
	if len(pathParts) > 0 {
		resource := pathParts[len(pathParts)-1]

		// 处理路径参数
		if strings.HasPrefix(resource, ":") {
			resource = pathParts[len(pathParts)-2]
		}

		switch route.Method {
		case "GET":
			if strings.Contains(route.Path, "/:") {
				return fmt.Sprintf("获取%s详情", resource)
			}
			return fmt.Sprintf("获取%s列表", resource)
		case "POST":
			return fmt.Sprintf("创建%s", resource)
		case "PUT":
			return fmt.Sprintf("更新%s", resource)
		case "DELETE":
			return fmt.Sprintf("删除%s", resource)
		case "PATCH":
			return fmt.Sprintf("部分更新%s", resource)
		}
	}

	return fmt.Sprintf("%s %s", route.Method, route.Path)
}

// generateDescription 生成描述
func (g *SwaggerGenerator) generateDescription(route RouteInfo) string {
	if route.HandlerInfo != nil && route.HandlerInfo.FuncDecl.Doc != nil {
		return strings.TrimSpace(route.HandlerInfo.FuncDecl.Doc.Text())
	}
	return g.generateSummary(route)
}

// generateTags 生成标签
func (g *SwaggerGenerator) generateTags(route RouteInfo) []string {
	tags := make([]string, 0)

	// 从路径提取标签
	pathParts := strings.Split(strings.Trim(route.Path, "/"), "/")
	if len(pathParts) > 1 {
		// 使用第二级路径作为标签 (跳过api)
		if pathParts[0] == "api" && len(pathParts) > 2 {
			tags = append(tags, strings.Title(pathParts[1]))
		} else if len(pathParts) > 1 {
			tags = append(tags, strings.Title(pathParts[0]))
		}
	}

	// 从处理器名称提取标签
	if route.HandlerInfo != nil && route.HandlerInfo.ReceiverType != "" {
		receiverType := route.HandlerInfo.ReceiverType
		// 移除Handler后缀
		receiverType = strings.TrimSuffix(receiverType, "Handler")
		if receiverType != "" {
			tags = append(tags, receiverType)
		}
	}

	if len(tags) == 0 {
		tags = append(tags, "Default")
	}

	return g.removeDuplicates(tags)
}

// generateParameters 生成参数
func (g *SwaggerGenerator) generateParameters(route RouteInfo) []Parameter {
	parameters := make([]Parameter, 0)

	// 路径参数
	pathParams := g.extractPathParams(route.Path)
	for _, param := range pathParams {
		parameters = append(parameters, Parameter{
			Name:        param,
			In:          "path",
			Type:        "string",
			Required:    true,
			Description: fmt.Sprintf("%s ID", param),
		})
	}

	// 查询参数 (从函数参数推断)
	if route.HandlerInfo != nil {
		queryParams := g.extractQueryParams(route.HandlerInfo)
		parameters = append(parameters, queryParams...)
	}

	// 请求体参数
	if g.hasRequestBody(route.Method) {
		bodyParam := g.generateBodyParameter(route)
		if bodyParam != nil {
			parameters = append(parameters, *bodyParam)
		}
	}

	return parameters
}

// generateResponses 生成响应
func (g *SwaggerGenerator) generateResponses(route RouteInfo) map[string]Response {
	responses := make(map[string]Response)

	// 默认成功响应
	successCode := "200"
	if route.Method == "POST" {
		successCode = "201"
	}

	responses[successCode] = Response{
		Description: "成功",
		Schema: &Schema{
			Ref: "#/definitions/ApiResponse",
		},
	}

	// 错误响应
	responses["400"] = Response{
		Description: "请求参数错误",
		Schema: &Schema{
			Ref: "#/definitions/ApiResponse",
		},
	}

	responses["500"] = Response{
		Description: "服务器内部错误",
		Schema: &Schema{
			Ref: "#/definitions/ApiResponse",
		},
	}

	// 需要认证的接口添加401响应
	if g.needsAuth(route) {
		responses["401"] = Response{
			Description: "未授权",
			Schema: &Schema{
				Ref: "#/definitions/ApiResponse",
			},
		}
	}

	return responses
}

// generateOperationID 生成操作ID
func (g *SwaggerGenerator) generateOperationID(route RouteInfo) string {
	if route.HandlerInfo != nil {
		return fmt.Sprintf("%s_%s", route.HandlerInfo.ReceiverType, route.HandlerInfo.Name)
	}

	pathParts := strings.Split(strings.Trim(route.Path, "/"), "/")
	resource := "unknown"
	if len(pathParts) > 0 {
		resource = pathParts[len(pathParts)-1]
		if strings.HasPrefix(resource, ":") && len(pathParts) > 1 {
			resource = pathParts[len(pathParts)-2]
		}
	}

	return fmt.Sprintf("%s_%s", strings.ToLower(route.Method), resource)
}

// buildDefinitions 构建数据模型定义
func (g *SwaggerGenerator) buildDefinitions(doc *SwaggerDoc) {
	structs := g.parser.GetStructs()

	// 添加通用响应模型
	doc.Definitions["ApiResponse"] = Definition{
		Type: "object",
		Properties: map[string]*Schema{
			"code": {
				Type:        "integer",
				Description: "响应码",
				Example:     200,
			},
			"message": {
				Type:        "string",
				Description: "响应消息",
				Example:     "success",
			},
			"data": {
				Type:                 "object",
				Description:          "响应数据",
				AdditionalProperties: true,
			},
		},
		Description: "通用API响应",
	}

	// 添加结构体定义
	for name, structInfo := range structs {
		if g.shouldIncludeStruct(structInfo) {
			definition := g.buildDefinition(structInfo)
			shortName := g.getShortName(name)
			doc.Definitions[shortName] = definition

			if g.verbose {
				fmt.Printf("📋 添加定义: %s\n", shortName)
			}
		}
	}
}

// buildDefinition 构建单个定义
func (g *SwaggerGenerator) buildDefinition(structInfo *StructInfo) Definition {
	definition := Definition{
		Type:        "object",
		Properties:  make(map[string]*Schema),
		Required:    make([]string, 0),
		Description: fmt.Sprintf("%s数据模型", structInfo.Name),
	}

	for _, field := range structInfo.Fields {
		if field.JSONName == "" || field.JSONName == "-" {
			continue
		}

		schema := g.buildFieldSchema(field)
		definition.Properties[field.JSONName] = schema

		if field.Required {
			definition.Required = append(definition.Required, field.JSONName)
		}
	}

	return definition
}

// buildFieldSchema 构建字段Schema
func (g *SwaggerGenerator) buildFieldSchema(field FieldInfo) *Schema {
	schema := &Schema{
		Description: field.Description,
	}

	// 根据Go类型映射到Swagger类型
	switch {
	case strings.HasPrefix(field.Type, "string"):
		schema.Type = "string"
	case strings.HasPrefix(field.Type, "int"), strings.HasPrefix(field.Type, "uint"):
		schema.Type = "integer"
	case strings.HasPrefix(field.Type, "float"):
		schema.Type = "number"
	case strings.HasPrefix(field.Type, "bool"):
		schema.Type = "boolean"
	case strings.HasPrefix(field.Type, "[]"):
		schema.Type = "array"
		itemType := strings.TrimPrefix(field.Type, "[]")
		schema.Items = &Schema{
			Type: g.mapGoTypeToSwagger(itemType),
		}
	case strings.HasPrefix(field.Type, "map["):
		schema.Type = "object"
		schema.AdditionalProperties = true
	case strings.Contains(field.Type, "time.Time"):
		schema.Type = "string"
		schema.Format = "date-time"
	default:
		// 可能是自定义类型
		if g.isCustomType(field.Type) {
			schema.Ref = fmt.Sprintf("#/definitions/%s", field.Type)
		} else {
			schema.Type = "object"
		}
	}

	return schema
}

// buildTags 构建标签
func (g *SwaggerGenerator) buildTags(doc *SwaggerDoc) {
	tagMap := make(map[string]bool)

	// 从路径中提取标签
	for _, pathMethods := range doc.Paths {
		for _, endpoint := range pathMethods {
			for _, tag := range endpoint.Tags {
				if !tagMap[tag] {
					doc.Tags = append(doc.Tags, Tag{
						Name:        tag,
						Description: fmt.Sprintf("%s相关接口", tag),
					})
					tagMap[tag] = true
				}
			}
		}
	}
}

// 辅助方法

// normalizePath 标准化路径
func (g *SwaggerGenerator) normalizePath(path string) string {
	// 将Gin路径参数格式转换为Swagger格式
	re := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
	return re.ReplaceAllString(path, "{$1}")
}

// extractPathParams 提取路径参数
func (g *SwaggerGenerator) extractPathParams(path string) []string {
	re := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := re.FindAllStringSubmatch(path, -1)
	params := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}
	return params
}

// extractQueryParams 从函数参数提取查询参数
func (g *SwaggerGenerator) extractQueryParams(handler *HandlerInfo) []Parameter {
	// 这里可以进一步分析函数体中的c.Query()调用
	// 目前返回一些常见的查询参数
	return []Parameter{
		{
			Name:        "page",
			In:          "query",
			Type:        "integer",
			Description: "页码",
		},
		{
			Name:        "size",
			In:          "query",
			Type:        "integer",
			Description: "每页数量",
		},
	}
}

// hasRequestBody 检查是否有请求体
func (g *SwaggerGenerator) hasRequestBody(method string) bool {
	return method == "POST" || method == "PUT" || method == "PATCH"
}

// generateBodyParameter 生成请求体参数
func (g *SwaggerGenerator) generateBodyParameter(route RouteInfo) *Parameter {
	if !g.hasRequestBody(route.Method) {
		return nil
	}

	return &Parameter{
		Name:        "body",
		In:          "body",
		Description: "请求体",
		Schema: &Schema{
			Type: "object",
		},
	}
}

// needsAuth 检查是否需要认证
func (g *SwaggerGenerator) needsAuth(route RouteInfo) bool {
	// 不需要认证的路径
	noAuthPaths := []string{
		"/api/not_auth",
		"/health",
		"/swagger",
	}

	for _, path := range noAuthPaths {
		if strings.HasPrefix(route.Path, path) {
			return false
		}
	}

	return true
}

// shouldIncludeStruct 检查是否应该包含结构体
func (g *SwaggerGenerator) shouldIncludeStruct(structInfo *StructInfo) bool {
	// 排除一些内部结构体
	excludePatterns := []string{
		"wire",
		"test",
		"Test",
		"Mock",
		"mock",
	}

	for _, pattern := range excludePatterns {
		if strings.Contains(structInfo.Name, pattern) ||
			strings.Contains(structInfo.Package, pattern) {
			return false
		}
	}

	// 只包含有JSON标签的结构体
	hasJSONTags := false
	for _, field := range structInfo.Fields {
		if field.JSONName != "" && field.JSONName != "-" {
			hasJSONTags = true
			break
		}
	}

	return hasJSONTags
}

// getShortName 获取短名称
func (g *SwaggerGenerator) getShortName(fullName string) string {
	parts := strings.Split(fullName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
}

// mapGoTypeToSwagger 映射Go类型到Swagger类型
func (g *SwaggerGenerator) mapGoTypeToSwagger(goType string) string {
	switch goType {
	case "string":
		return "string"
	case "int", "int8", "int16", "int32", "int64", "uint", "uint8", "uint16", "uint32", "uint64":
		return "integer"
	case "float32", "float64":
		return "number"
	case "bool":
		return "boolean"
	default:
		return "string"
	}
}

// isCustomType 检查是否是自定义类型
func (g *SwaggerGenerator) isCustomType(typeName string) bool {
	structs := g.parser.GetStructs()
	for name := range structs {
		if strings.HasSuffix(name, "."+typeName) {
			return true
		}
	}
	return false
}

// removeDuplicates 移除重复项
func (g *SwaggerGenerator) removeDuplicates(slice []string) []string {
	keys := make(map[string]bool)
	result := make([]string, 0)

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

// 文件输出方法

// writeJSON 写入JSON文档
func (g *SwaggerGenerator) writeJSON(doc *SwaggerDoc) error {
	jsonFile := filepath.Join(g.outputDir, "swagger.json")

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(jsonFile, data, 0644); err != nil {
		return err
	}

	if g.verbose {
		fmt.Printf("📄 JSON文档已生成: %s\n", jsonFile)
	}

	return nil
}

// writeYAML 写入YAML文档
func (g *SwaggerGenerator) writeYAML(doc *SwaggerDoc) error {
	yamlFile := filepath.Join(g.outputDir, "swagger.yaml")

	data, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}

	if err := os.WriteFile(yamlFile, data, 0644); err != nil {
		return err
	}

	if g.verbose {
		fmt.Printf("📄 YAML文档已生成: %s\n", yamlFile)
	}

	return nil
}

// writeGoDoc 写入Go文档
func (g *SwaggerGenerator) writeGoDoc(doc *SwaggerDoc) error {
	goFile := filepath.Join(g.outputDir, "docs.go")

	jsonData, err := json.Marshal(doc)
	if err != nil {
		return err
	}

	content := fmt.Sprintf(`// Code generated by swagger-auto-gen. DO NOT EDIT.

package docs

import "github.com/swaggo/swag"

const docTemplate = `+"`%s`"+`

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "%s",
	Host:             "%s",
	BasePath:         "%s",
	Schemes:          []string{%s},
	Title:            "%s",
	Description:      "%s",
	InfoInstanceName: "swagger",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

func init() {
	swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
}
`, string(jsonData), doc.Info.Version, doc.Host, doc.BasePath,
		`"`+strings.Join(doc.Schemes, `", "`)+`"`,
		doc.Info.Title, doc.Info.Description)

	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		return err
	}

	if g.verbose {
		fmt.Printf("📄 Go文档已生成: %s\n", goFile)
	}

	return nil
}
