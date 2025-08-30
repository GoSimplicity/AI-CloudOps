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
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"path/filepath"
	"regexp"
	"strings"

	"golang.org/x/tools/go/packages"
)

// Parser AST解析器
type Parser struct {
	projectRoot string
	fileSet     *token.FileSet
	packages    map[string]*PackageInfo
	handlers    map[string]*HandlerInfo
	routes      []RouteInfo
	structs     map[string]*StructInfo
	verbose     bool
}

// NewParser 创建新的解析器
func NewParser(projectRoot string, verbose bool) *Parser {
	return &Parser{
		projectRoot: projectRoot,
		fileSet:     token.NewFileSet(),
		packages:    make(map[string]*PackageInfo),
		handlers:    make(map[string]*HandlerInfo),
		routes:      make([]RouteInfo, 0),
		structs:     make(map[string]*StructInfo),
		verbose:     verbose,
	}
}

// ParseProject 解析整个项目
func (p *Parser) ParseProject() error {
	if p.verbose {
		fmt.Println("🔍 开始解析项目...")
	}

	// 加载包信息
	cfg := &packages.Config{
		Mode: packages.NeedName | packages.NeedFiles | packages.NeedCompiledGoFiles |
			packages.NeedImports | packages.NeedTypes | packages.NeedSyntax |
			packages.NeedTypesInfo,
		Dir: p.projectRoot,
	}

	pkgs, err := packages.Load(cfg, "./...")
	if err != nil {
		return fmt.Errorf("加载包失败: %v", err)
	}

	// 处理每个包
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			continue // 跳过有错误的包
		}

		packageInfo := &PackageInfo{
			Name:  pkg.Name,
			Path:  pkg.PkgPath,
			Files: pkg.Syntax,
		}

		p.packages[pkg.PkgPath] = packageInfo

		// 解析包中的文件
		for _, file := range pkg.Syntax {
			p.parseFile(file, pkg.PkgPath)
		}
	}

	// 解析路由
	if err := p.parseRoutes(); err != nil {
		return fmt.Errorf("解析路由失败: %v", err)
	}

	if p.verbose {
		fmt.Printf("✅ 解析完成: 找到 %d 个处理器, %d 个路由, %d 个结构体\n",
			len(p.handlers), len(p.routes), len(p.structs))
	}

	return nil
}

// parseFile 解析单个文件
func (p *Parser) parseFile(file *ast.File, packagePath string) {
	ast.Inspect(file, func(n ast.Node) bool {
		switch node := n.(type) {
		case *ast.FuncDecl:
			p.parseHandler(node, packagePath)
		case *ast.TypeSpec:
			if structType, ok := node.Type.(*ast.StructType); ok {
				p.parseStruct(node.Name.Name, structType, packagePath)
			}
		}
		return true
	})
}

// parseStruct 解析结构体
func (p *Parser) parseStruct(name string, structType *ast.StructType, packagePath string) {
	if p.verbose {
		fmt.Printf("📦 解析结构体: %s (包: %s)\n", name, packagePath)
	}

	structInfo := &StructInfo{
		Name:          name,
		Fields:        make([]FieldInfo, 0),
		Package:       packagePath,
		EmbeddedTypes: make([]string, 0),
	}

	if structType.Fields != nil {
		for _, field := range structType.Fields.List {
			p.parseStructField(field, structInfo)
		}
	}

	// 构建完整的结构体名称（包含包路径）
	fullName := fmt.Sprintf("%s.%s", packagePath, name)
	p.structs[fullName] = structInfo

	if p.verbose {
		fmt.Printf("✅ 结构体解析完成: %s (字段数: %d, 嵌入类型数: %d)\n",
			name, len(structInfo.Fields), len(structInfo.EmbeddedTypes))
	}
}

// parseStructField 解析结构体字段
func (p *Parser) parseStructField(field *ast.Field, structInfo *StructInfo) {
	// 处理嵌入类型（匿名字段）
	if len(field.Names) == 0 {
		embeddedType := p.exprToString(field.Type)
		structInfo.EmbeddedTypes = append(structInfo.EmbeddedTypes, embeddedType)
		if p.verbose {
			fmt.Printf("  🔗 嵌入类型: %s\n", embeddedType)
		}
		return
	}

	// 处理命名字段
	for _, name := range field.Names {
		if !name.IsExported() {
			continue // 跳过未导出的字段
		}

		fieldType := p.exprToString(field.Type)
		tag := ""
		if field.Tag != nil {
			tag = field.Tag.Value
		}

		fieldInfo := FieldInfo{
			Name:        name.Name,
			Type:        fieldType,
			Tag:         tag,
			JSONName:    p.extractJSONName(tag),
			FormName:    p.extractFormName(tag),
			URIName:     p.extractURIName(tag),
			Required:    p.isRequired(tag),
			Description: p.extractDescription(tag),
		}

		structInfo.Fields = append(structInfo.Fields, fieldInfo)

		if p.verbose {
			fmt.Printf("  📋 字段: %s %s (json: %s, form: %s, uri: %s, required: %t)\n",
				fieldInfo.Name, fieldInfo.Type, fieldInfo.JSONName,
				fieldInfo.FormName, fieldInfo.URIName, fieldInfo.Required)
		}
	}
}

// extractDescription 从tag或注释中提取描述
func (p *Parser) extractDescription(tag string) string {
	// 从comment tag中提取描述
	re := regexp.MustCompile(`comment:"([^"]*)"`)
	matches := re.FindStringSubmatch(tag)
	if len(matches) > 1 {
		return matches[1]
	}

	// 从gorm tag中提取描述
	re = regexp.MustCompile(`gorm:"[^"]*comment:([^;"]*)[;"]*"`)
	matches = re.FindStringSubmatch(tag)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}

	return ""
}

// parseHandler 解析处理函数
func (p *Parser) parseHandler(funcDecl *ast.FuncDecl, packagePath string) {
	handlerName := funcDecl.Name.Name
	receiverType := ""

	// 获取接收者类型
	if funcDecl.Recv != nil && len(funcDecl.Recv.List) > 0 {
		if starExpr, ok := funcDecl.Recv.List[0].Type.(*ast.StarExpr); ok {
			if ident, ok := starExpr.X.(*ast.Ident); ok {
				receiverType = ident.Name
			}
		}
	}

	// 检查是否是路由注册器、gin.Context处理器或者是Handler结构体的方法
	isRouteRegister := handlerName == "RegisterRouters" || handlerName == "RegisterRoutes"
	isGinHandler := false
	isHandlerMethod := false

	if funcDecl.Type.Params != nil && len(funcDecl.Type.Params.List) > 0 {
		for _, param := range funcDecl.Type.Params.List {
			if p.isGinContext(param.Type) {
				isGinHandler = true
				break
			}
			if p.isGinEngine(param.Type) {
				isGinHandler = true
				break
			}
		}
	}

	// 检查是否是Handler结构体的方法（包含"Handler"的接收者类型）
	if receiverType != "" && strings.Contains(receiverType, "Handler") {
		isHandlerMethod = true
	}

	// 只有当函数是导出的且满足条件时才解析
	if funcDecl.Name.IsExported() && (isRouteRegister || isGinHandler || isHandlerMethod) {
		handlerInfo := &HandlerInfo{
			Name:         handlerName,
			FuncDecl:     funcDecl,
			ReceiverType: receiverType,
			PackageName:  packagePath,
		}

		key := fmt.Sprintf("%s.%s", receiverType, handlerName)
		if receiverType == "" {
			key = handlerName
		}

		p.handlers[key] = handlerInfo

		if p.verbose {
			if isRouteRegister {
				fmt.Printf("📝 找到路由注册器: %s (方法: %s)\n", key, handlerName)
			} else if isGinHandler {
				fmt.Printf("📝 找到Gin处理器: %s (方法: %s)\n", key, handlerName)
			} else if isHandlerMethod {
				fmt.Printf("📝 找到Handler方法: %s (方法: %s)\n", key, handlerName)
			}
		}
	}
}

// isGinContext 检查是否是gin.Context类型
func (p *Parser) isGinContext(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.StarExpr:
		if selectorExpr, ok := t.X.(*ast.SelectorExpr); ok {
			if ident, ok := selectorExpr.X.(*ast.Ident); ok {
				return ident.Name == "gin" && selectorExpr.Sel.Name == "Context"
			}
		}
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == "gin" && t.Sel.Name == "Context"
		}
	}
	return false
}

// isGinEngine 检查是否是gin.Engine类型
func (p *Parser) isGinEngine(expr ast.Expr) bool {
	switch t := expr.(type) {
	case *ast.StarExpr:
		if selectorExpr, ok := t.X.(*ast.SelectorExpr); ok {
			if ident, ok := selectorExpr.X.(*ast.Ident); ok {
				return ident.Name == "gin" && selectorExpr.Sel.Name == "Engine"
			}
		}
	case *ast.SelectorExpr:
		if ident, ok := t.X.(*ast.Ident); ok {
			return ident.Name == "gin" && t.Sel.Name == "Engine"
		}
	}
	return false
}

// parseRoutes 解析路由注册
func (p *Parser) parseRoutes() error {
	// 查找web.go中的路由注册
	webFiles, err := filepath.Glob(filepath.Join(p.projectRoot, "pkg/di/web.go"))
	if err != nil {
		return err
	}

	for _, webFile := range webFiles {
		if err := p.parseWebFile(webFile); err != nil {
			return err
		}
	}

	// 查找各个handler中的RegisterRouters方法
	registeredCount := 0
	if p.verbose {
		fmt.Printf("🔍 开始查找 RegisterRouters/RegisterRoutes 方法...\n")
	}

	for handlerKey, handler := range p.handlers {
		if p.verbose && strings.Contains(strings.ToLower(handler.Name), "register") {
			fmt.Printf("🔍 发现注册相关方法: %s -> %s (接收者: %s)\n", handlerKey, handler.Name, handler.ReceiverType)
		}

		if handler.Name == "RegisterRouters" || handler.Name == "RegisterRoutes" {
			if p.verbose {
				fmt.Printf("🔧 正在解析路由注册方法: %s (接收者: %s)\n", handlerKey, handler.ReceiverType)
			}
			if err := p.parseRegisterRoutes(handler); err != nil {
				if p.verbose {
					fmt.Printf("⚠️  解析路由注册失败 %s: %v\n", handlerKey, err)
				}
			} else {
				registeredCount++
			}
		}
	}

	if p.verbose {
		fmt.Printf("📊 共处理了 %d 个 RegisterRouters 方法\n", registeredCount)
	}

	return nil
}

// parseWebFile 解析web.go文件
func (p *Parser) parseWebFile(filePath string) error {
	file, err := parser.ParseFile(p.fileSet, filePath, nil, parser.ParseComments)
	if err != nil {
		return err
	}

	// 查找InitGinServer函数
	ast.Inspect(file, func(n ast.Node) bool {
		if funcDecl, ok := n.(*ast.FuncDecl); ok {
			if funcDecl.Name.Name == "InitGinServer" {
				p.parseInitGinServer(funcDecl)
			}
		}
		return true
	})

	return nil
}

// parseInitGinServer 解析InitGinServer函数
func (p *Parser) parseInitGinServer(funcDecl *ast.FuncDecl) {
	if funcDecl.Body == nil {
		return
	}

	for _, stmt := range funcDecl.Body.List {
		if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
			if callExpr, ok := exprStmt.X.(*ast.CallExpr); ok {
				if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
					if selectorExpr.Sel.Name == "RegisterRouters" || selectorExpr.Sel.Name == "RegisterRoutes" {
						// 找到路由注册调用
						if ident, ok := selectorExpr.X.(*ast.Ident); ok {
							handlerName := ident.Name
							if p.verbose {
								fmt.Printf("🔗 找到路由注册: %s\n", handlerName)
							}
						}
					}
				}
			}
		}
	}
}

// parseRegisterRoutes 解析RegisterRoutes方法
func (p *Parser) parseRegisterRoutes(handler *HandlerInfo) error {
	if handler.FuncDecl.Body == nil {
		if p.verbose {
			fmt.Printf("⚠️  RegisterRoutes 方法没有函数体: %s\n", handler.Name)
		}
		return nil
	}

	if p.verbose {
		fmt.Printf("🔧 开始解析 RegisterRoutes 方法: %s.%s\n", handler.ReceiverType, handler.Name)
	}

	// 存储路由组映射：变量名 -> 路径
	routeGroups := make(map[string]string)

	// 第一遍：查找路由组定义
	for _, stmt := range handler.FuncDecl.Body.List {
		p.findRouteGroups(stmt, routeGroups)
	}

	if p.verbose && len(routeGroups) > 0 {
		fmt.Printf("📂 找到路由组: %v\n", routeGroups)
	}

	// 第二遍：解析路由定义
	routesFound := 0
	for _, stmt := range handler.FuncDecl.Body.List {
		routesInStmt := p.parseRoutesInStatement(stmt, routeGroups, handler)
		routesFound += routesInStmt
	}

	if p.verbose {
		fmt.Printf("✅ %s.%s 解析完成，找到 %d 个路由\n", handler.ReceiverType, handler.Name, routesFound)
	}

	return nil
}

// findRouteGroups 查找路由组定义
func (p *Parser) findRouteGroups(stmt ast.Stmt, routeGroups map[string]string) {
	switch s := stmt.(type) {
	case *ast.AssignStmt:
		// 查找形如 group := server.Group("/api/xxx") 的语句
		if len(s.Lhs) == 1 && len(s.Rhs) == 1 {
			if ident, ok := s.Lhs[0].(*ast.Ident); ok {
				if callExpr, ok := s.Rhs[0].(*ast.CallExpr); ok {
					if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
						if selectorExpr.Sel.Name == "Group" && len(callExpr.Args) > 0 {
							if basicLit, ok := callExpr.Args[0].(*ast.BasicLit); ok {
								groupPath := strings.Trim(basicLit.Value, "\"")
								routeGroups[ident.Name] = groupPath
								if p.verbose {
									fmt.Printf("📂 找到路由组: %s -> %s\n", ident.Name, groupPath)
								}
							}
						}
					}
				}
			}
		}
	case *ast.BlockStmt:
		// 递归处理嵌套的块
		for _, subStmt := range s.List {
			p.findRouteGroups(subStmt, routeGroups)
		}
	}
}

// parseRoutesInStatement 在语句中解析路由
func (p *Parser) parseRoutesInStatement(stmt ast.Stmt, routeGroups map[string]string, handler *HandlerInfo) int {
	routesCount := 0

	switch s := stmt.(type) {
	case *ast.BlockStmt:
		// 解析块中的路由（通常在 {} 中）
		if p.verbose {
			fmt.Printf("🔍 解析代码块中的路由 (语句数: %d)\n", len(s.List))
		}
		for _, subStmt := range s.List {
			routesCount += p.parseRoutesInStatement(subStmt, routeGroups, handler)
		}
	case *ast.ExprStmt:
		// 解析表达式语句中的路由定义
		if p.parseRouteExprStatement(s, routeGroups, handler) {
			routesCount++
		}
	}

	return routesCount
}

// parseRouteExprStatement 解析路由表达式语句
func (p *Parser) parseRouteExprStatement(stmt *ast.ExprStmt, routeGroups map[string]string, handler *HandlerInfo) bool {
	if callExpr, ok := stmt.X.(*ast.CallExpr); ok {
		if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			method := strings.ToUpper(selectorExpr.Sel.Name)

			if p.verbose {
				fmt.Printf("🔍 检查方法调用: %s\n", method)
			}

			// 检查是否是HTTP方法
			if p.isHTTPMethod(method) && len(callExpr.Args) >= 2 {
				// 获取路径
				var path string
				if basicLit, ok := callExpr.Args[0].(*ast.BasicLit); ok {
					path = strings.Trim(basicLit.Value, "\"")
				}

				// 获取处理函数名
				var handlerFunc string
				var targetHandler *HandlerInfo

				if selectorExpr2, ok := callExpr.Args[1].(*ast.SelectorExpr); ok {
					if ident, ok := selectorExpr2.X.(*ast.Ident); ok {
						methodName := selectorExpr2.Sel.Name
						// 处理 h.methodName 或 k.methodName 的情况，通常指向当前handler
						receiverShort := strings.ToLower(handler.ReceiverType)
						if ident.Name == "h" || ident.Name == "a" || ident.Name == "k" ||
							ident.Name == "u" || strings.HasPrefix(receiverShort, ident.Name) {
							handlerFunc = methodName
							targetHandler = p.findHandlerMethod(handler.ReceiverType, methodName)
							if targetHandler == nil {
								targetHandler = handler // 如果找不到，使用当前handler
							}
						} else {
							handlerFunc = fmt.Sprintf("%s.%s", ident.Name, methodName)
						}
					}
				} else if ident, ok := callExpr.Args[1].(*ast.Ident); ok {
					handlerFunc = ident.Name
					targetHandler = handler
				}

				// 获取路由组前缀
				var groupPath string
				if ident, ok := selectorExpr.X.(*ast.Ident); ok {
					if groupPrefix, exists := routeGroups[ident.Name]; exists {
						groupPath = groupPrefix
					}
				}

				// 组合完整路径
				fullPath := path
				if groupPath != "" {
					fullPath = groupPath + path
				}

				// 如果还没有找到targetHandler，再次尝试查找
				if targetHandler == nil && handlerFunc != "" {
					targetHandler = p.findHandlerMethod(handler.ReceiverType, handlerFunc)
				}

				// 创建路由信息
				route := RouteInfo{
					Method:      method,
					Path:        fullPath,
					Handler:     handlerFunc,
					HandlerInfo: targetHandler,
					Group:       groupPath,
				}

				p.routes = append(p.routes, route)

				if p.verbose {
					fmt.Printf("🚏 找到路由: %s %s -> %s\n", method, fullPath, handlerFunc)
				}

				return true
			} else if p.verbose && p.isHTTPMethod(method) {
				fmt.Printf("⚠️  HTTP方法 %s 参数不足 (参数数量: %d)\n", method, len(callExpr.Args))
			}
		}
	}

	return false
}

// findHandlerMethod 查找handler方法
func (p *Parser) findHandlerMethod(receiverType, methodName string) *HandlerInfo {
	// 首先尝试完整匹配
	key := fmt.Sprintf("%s.%s", receiverType, methodName)
	if handler, exists := p.handlers[key]; exists {
		return handler
	}

	// 然后遍历所有handlers查找匹配的方法
	for _, handler := range p.handlers {
		if handler.ReceiverType == receiverType && handler.Name == methodName {
			return handler
		}
	}

	return nil
}

// isHTTPMethod 检查是否是HTTP方法
func (p *Parser) isHTTPMethod(method string) bool {
	httpMethods := []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	for _, m := range httpMethods {
		if method == m {
			return true
		}
	}
	return false
}

// exprToString 将表达式转换为字符串
func (p *Parser) exprToString(expr ast.Expr) string {
	switch t := expr.(type) {
	case *ast.Ident:
		return t.Name
	case *ast.StarExpr:
		return "*" + p.exprToString(t.X)
	case *ast.ArrayType:
		return "[]" + p.exprToString(t.Elt)
	case *ast.SelectorExpr:
		return p.exprToString(t.X) + "." + t.Sel.Name
	case *ast.MapType:
		return "map[" + p.exprToString(t.Key) + "]" + p.exprToString(t.Value)
	case *ast.InterfaceType:
		return "interface{}"
	default:
		return "unknown"
	}
}

// extractJSONName 从标签中提取JSON名称
func (p *Parser) extractJSONName(tag string) string {
	return p.extractTagValue(tag, "json")
}

// extractFormName 从标签中提取Form名称
func (p *Parser) extractFormName(tag string) string {
	return p.extractTagValue(tag, "form")
}

// extractURIName 从标签中提取URI名称
func (p *Parser) extractURIName(tag string) string {
	return p.extractTagValue(tag, "uri")
}

// extractTagValue 从标签中提取指定tag的值
func (p *Parser) extractTagValue(tagString, tagName string) string {
	pattern := fmt.Sprintf(`%s:"([^"]*)"`, tagName)
	re := regexp.MustCompile(pattern)
	matches := re.FindStringSubmatch(tagString)
	if len(matches) > 1 {
		tagValue := matches[1]
		parts := strings.Split(tagValue, ",")
		if len(parts) > 0 && parts[0] != "" && parts[0] != "-" {
			return parts[0]
		}
	}
	return ""
}

// isRequired 检查字段是否必需
func (p *Parser) isRequired(tag string) bool {
	// 检查binding tag中是否包含required
	re := regexp.MustCompile(`binding:"([^"]*)"`)
	matches := re.FindStringSubmatch(tag)
	if len(matches) > 1 {
		bindingTag := matches[1]
		return strings.Contains(bindingTag, "required")
	}
	return false
}

// GetHandlers 获取所有处理器
func (p *Parser) GetHandlers() map[string]*HandlerInfo {
	return p.handlers
}

// GetRoutes 获取所有路由
func (p *Parser) GetRoutes() []RouteInfo {
	return p.routes
}

// GetStructs 获取所有结构体
func (p *Parser) GetStructs() map[string]*StructInfo {
	return p.structs
}

// GetPackages 获取所有包
func (p *Parser) GetPackages() map[string]*PackageInfo {
	return p.packages
}
