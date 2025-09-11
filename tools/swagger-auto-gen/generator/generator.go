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
	"go/ast"
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
			Description: "AI-CloudOps云原生运维平台API文档",
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

	if route.HandlerInfo == nil || route.HandlerInfo.FuncDecl == nil || route.HandlerInfo.FuncDecl.Body == nil {
		return parameters
	}

	// 分析函数体中的参数绑定调用
	bindingInfo := g.analyzeParameterBindings(route.HandlerInfo.FuncDecl.Body)

	// 1. 处理路径参数（URI绑定）
	for structName, _ := range bindingInfo.URIBindings {
		uriParams := g.extractURIParametersFromStruct(structName)
		parameters = append(parameters, uriParams...)
	}

	// 2. 处理查询参数（Query绑定）
	for structName, _ := range bindingInfo.QueryBindings {
		queryParams := g.extractQueryParametersFromStruct(structName)
		parameters = append(parameters, queryParams...)
	}

	// 3. 处理请求体参数（Body绑定）
	for structName, _ := range bindingInfo.BodyBindings {
		parameters = append(parameters, Parameter{
			Name:        "body",
			In:          "body",
			Description: "请求体",
			Schema: &Schema{
				Ref: fmt.Sprintf("#/definitions/%s", g.getShortName(structName)),
			},
		})
		break // 只处理第一个body绑定
	}

	return parameters
}

// ParameterBindingInfo 参数绑定信息
type ParameterBindingInfo struct {
	URIBindings   map[string]bool // 结构体名 -> 是否存在
	QueryBindings map[string]bool // 结构体名 -> 是否存在
	BodyBindings  map[string]bool // 结构体名 -> 是否存在
}

// analyzeParameterBindings 分析函数体中的参数绑定调用
func (g *SwaggerGenerator) analyzeParameterBindings(body *ast.BlockStmt) *ParameterBindingInfo {
	bindingInfo := &ParameterBindingInfo{
		URIBindings:   make(map[string]bool),
		QueryBindings: make(map[string]bool),
		BodyBindings:  make(map[string]bool),
	}

	// 遍历函数体，查找绑定调用
	ast.Inspect(body, func(n ast.Node) bool {
		if callExpr, ok := n.(*ast.CallExpr); ok {
			if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
				methodName := selectorExpr.Sel.Name

				// 检查不同的绑定方法
				switch methodName {
				case "ShouldBindUri":
					if structType := g.extractStructFromBindCall(callExpr, body); structType != "" {
						bindingInfo.URIBindings[structType] = true
					}
				case "ShouldBindQuery":
					if structType := g.extractStructFromBindCall(callExpr, body); structType != "" {
						bindingInfo.QueryBindings[structType] = true
					}
				case "ShouldBind", "ShouldBindJSON":
					if structType := g.extractStructFromBindCall(callExpr, body); structType != "" {
						bindingInfo.BodyBindings[structType] = true
					}
				case "HandleRequest":
					// 检查 utils.HandleRequest 的第二个参数
					if len(callExpr.Args) >= 2 {
						if structType := g.extractStructFromHandleRequest(callExpr.Args[1], body); structType != "" {
							bindingInfo.BodyBindings[structType] = true
						}
					}
				}
			}
		}
		return true
	})

	return bindingInfo
}

// extractStructFromBindCall 从绑定调用中提取结构体类型
func (g *SwaggerGenerator) extractStructFromBindCall(callExpr *ast.CallExpr, body *ast.BlockStmt) string {
	if len(callExpr.Args) == 0 {
		return ""
	}

	// 处理 &req 形式的参数
	if unaryExpr, ok := callExpr.Args[0].(*ast.UnaryExpr); ok {
		if unaryExpr.Op.String() == "&" {
			if ident, ok := unaryExpr.X.(*ast.Ident); ok {
				// 查找变量声明来确定类型
				return g.findVariableTypeInFunction(body, ident.Name)
			}
		}
	}

	return ""
}

// extractStructFromHandleRequest 从HandleRequest调用中提取结构体类型
func (g *SwaggerGenerator) extractStructFromHandleRequest(arg ast.Expr, body *ast.BlockStmt) string {
	// 处理 &req 形式的参数
	if unaryExpr, ok := arg.(*ast.UnaryExpr); ok {
		if unaryExpr.Op.String() == "&" {
			if ident, ok := unaryExpr.X.(*ast.Ident); ok {
				return g.findVariableTypeInFunction(body, ident.Name)
			}
		}
	}

	// 处理 nil 参数
	if ident, ok := arg.(*ast.Ident); ok {
		if ident.Name == "nil" {
			return ""
		}
	}

	return ""
}

// findVariableTypeInFunction 在函数中查找变量类型
func (g *SwaggerGenerator) findVariableTypeInFunction(body *ast.BlockStmt, varName string) string {
	var varType string

	ast.Inspect(body, func(n ast.Node) bool {
		// 查找变量声明语句 var req model.UserLoginReq
		if declStmt, ok := n.(*ast.DeclStmt); ok {
			if genDecl, ok := declStmt.Decl.(*ast.GenDecl); ok {
				for _, spec := range genDecl.Specs {
					if valueSpec, ok := spec.(*ast.ValueSpec); ok {
						for i, name := range valueSpec.Names {
							if name.Name == varName && valueSpec.Type != nil {
								varType = g.exprToString(valueSpec.Type)
								return false
							}
							// 处理短声明 req := model.UserLoginReq{}
							if name.Name == varName && i < len(valueSpec.Values) {
								if compositeLit, ok := valueSpec.Values[i].(*ast.CompositeLit); ok {
									varType = g.exprToString(compositeLit.Type)
									return false
								}
							}
						}
					}
				}
			}
		}

		// 查找短声明语句 req := model.UserLoginReq{}
		if assignStmt, ok := n.(*ast.AssignStmt); ok {
			if assignStmt.Tok.String() == ":=" {
				for i, lhs := range assignStmt.Lhs {
					if ident, ok := lhs.(*ast.Ident); ok {
						if ident.Name == varName && i < len(assignStmt.Rhs) {
							if compositeLit, ok := assignStmt.Rhs[i].(*ast.CompositeLit); ok {
								varType = g.exprToString(compositeLit.Type)
								return false
							}
						}
					}
				}
			}
		}

		return true
	})

	return varType
}

// extractURIParametersFromStruct 从结构体中提取URI参数
func (g *SwaggerGenerator) extractURIParametersFromStruct(structName string) []Parameter {
	parameters := make([]Parameter, 0)

	structs := g.parser.GetStructs()
	structInfo, exists := structs[structName]
	if !exists {
		return parameters
	}

	// 遍历结构体字段，查找有 uri tag 的字段
	for _, field := range structInfo.Fields {
		if field.URIName != "" {
			param := Parameter{
				Name:        field.URIName,
				In:          "path",
				Type:        g.mapGoTypeToSwagger(field.Type),
				Description: g.getFieldDescription(field),
				Required:    true, // URI参数总是必需的
			}
			parameters = append(parameters, param)
		}
	}

	return parameters
}

// extractQueryParametersFromStruct 从结构体中提取查询参数
func (g *SwaggerGenerator) extractQueryParametersFromStruct(structName string) []Parameter {
	parameters := make([]Parameter, 0)

	structs := g.parser.GetStructs()
	structInfo, exists := structs[structName]
	if !exists {
		return parameters
	}

	// 处理继承的基础结构体（如 ListReq）
	for _, embedded := range structInfo.EmbeddedTypes {
		if embeddedParams := g.extractQueryParametersFromStruct(embedded); len(embeddedParams) > 0 {
			parameters = append(parameters, embeddedParams...)
		}
	}

	// 遍历结构体字段，查找有 form tag 的字段
	for _, field := range structInfo.Fields {
		if field.FormName != "" {
			param := Parameter{
				Name:        field.FormName,
				In:          "query",
				Type:        g.mapGoTypeToSwagger(field.Type),
				Description: g.getFieldDescription(field),
				Required:    field.Required,
			}
			parameters = append(parameters, param)
		}
	}

	return parameters
}

// getFieldDescription 获取字段描述
func (g *SwaggerGenerator) getFieldDescription(field FieldInfo) string {
	if field.Description != "" {
		return field.Description
	}
	return fmt.Sprintf("%s参数", field.Name)
}

// mapGoTypeToSwagger 将Go类型映射为Swagger类型
func (g *SwaggerGenerator) mapGoTypeToSwagger(goType string) string {
	switch {
	case strings.HasPrefix(goType, "string"):
		return "string"
	case strings.HasPrefix(goType, "int"), strings.HasPrefix(goType, "uint"):
		return "integer"
	case strings.HasPrefix(goType, "float"):
		return "number"
	case strings.HasPrefix(goType, "bool"):
		return "boolean"
	case strings.HasPrefix(goType, "time.Time"):
		return "string"
	default:
		return "string"
	}
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
	// 支持 :param 和 *param 格式
	re := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
	path = re.ReplaceAllString(path, "{$1}")

	// 处理通配符参数 *param
	re2 := regexp.MustCompile(`\*([a-zA-Z_][a-zA-Z0-9_]*)`)
	path = re2.ReplaceAllString(path, "{$1}")

	return path
}

// extractPathParams 提取路径参数
func (g *SwaggerGenerator) extractPathParams(path string) []string {
	params := make([]string, 0)

	// 提取 :param 格式的参数
	re := regexp.MustCompile(`:([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches := re.FindAllStringSubmatch(path, -1)
	for _, match := range matches {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	// 提取 *param 格式的参数
	re2 := regexp.MustCompile(`\*([a-zA-Z_][a-zA-Z0-9_]*)`)
	matches2 := re2.FindAllStringSubmatch(path, -1)
	for _, match := range matches2 {
		if len(match) > 1 {
			params = append(params, match[1])
		}
	}

	return params
}

// hasRequestBody 检查是否有请求体
func (g *SwaggerGenerator) hasRequestBody(method string) bool {
	return method == "POST" || method == "PUT" || method == "PATCH"
}

// exprToString 将表达式转换为字符串
func (g *SwaggerGenerator) exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + g.exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + g.exprToString(t.Elt)
	case *ast.SelectorExpr:
		return g.exprToString(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + g.exprToString(t.Key) + "]" + g.exprToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return "unknown"
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
	excludePatterns := []string{"wire", "test", "Test", "Mock", "mock"}

	name := structInfo.Name
	pkg := structInfo.Package

	for _, pattern := range excludePatterns {
		if strings.Contains(name, pattern) || strings.Contains(pkg, pattern) {
			return false
		}
	}

	// 只包含有JSON标签的结构体
	for _, field := range structInfo.Fields {
		if field.JSONName != "" && field.JSONName != "-" {
			return true
		}
	}

	return false
}

// getShortName 获取短名称
func (g *SwaggerGenerator) getShortName(fullName string) string {
	parts := strings.Split(fullName, ".")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return fullName
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
	if len(slice) <= 1 {
		return slice
	}

	keys := make(map[string]bool, len(slice))
	result := make([]string, 0, len(slice))

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
		return fmt.Errorf("JSON序列化失败: %w", err)
	}

	if err := os.WriteFile(jsonFile, data, 0644); err != nil {
		return fmt.Errorf("写入JSON文件失败: %w", err)
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
		return fmt.Errorf("YAML序列化失败: %w", err)
	}

	if err := os.WriteFile(yamlFile, data, 0644); err != nil {
		return fmt.Errorf("写入YAML文件失败: %w", err)
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
		return fmt.Errorf("JSON序列化失败: %w", err)
	}

	// 使用模板生成内容
	content := strings.ReplaceAll(DocsGoTemplate, "{{.DocJSON}}", string(jsonData))
	content = strings.ReplaceAll(content, "{{.Version}}", doc.Info.Version)
	content = strings.ReplaceAll(content, "{{.Host}}", doc.Host)
	content = strings.ReplaceAll(content, "{{.BasePath}}", doc.BasePath)
	content = strings.ReplaceAll(content, "{{.Title}}", doc.Info.Title)
	content = strings.ReplaceAll(content, "{{.Description}}", doc.Info.Description)
	content = strings.ReplaceAll(content, "{{.InstanceName}}", "swagger")

	// 处理 Schemes 数组
	var schemesStr string
	for i, scheme := range doc.Schemes {
		if i > 0 {
			schemesStr += ", "
		}
		schemesStr += `"` + scheme + `"`
	}
	content = strings.ReplaceAll(content, "{{range .Schemes}}\"{{.}}\", {{end}}", schemesStr)

	if err := os.WriteFile(goFile, []byte(content), 0644); err != nil {
		return fmt.Errorf("写入Go文件失败: %w", err)
	}

	if g.verbose {
		fmt.Printf("📄 Go文档已生成: %s\n", goFile)
	}

	return nil
}
