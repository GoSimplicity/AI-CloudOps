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

// parseHandler 解析处理函数
func (p *Parser) parseHandler(funcDecl *ast.FuncDecl, packagePath string) {
	if funcDecl.Type.Params == nil || len(funcDecl.Type.Params.List) == 0 {
		return
	}

	// 检查是否是gin.Context参数
	for _, param := range funcDecl.Type.Params.List {
		if p.isGinContext(param.Type) {
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
				fmt.Printf("📝 找到处理器: %s\n", key)
			}
			break
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

// parseStruct 解析结构体
func (p *Parser) parseStruct(name string, structType *ast.StructType, packagePath string) {
	if structType.Fields == nil {
		return
	}

	structInfo := &StructInfo{
		Name:    name,
		Package: packagePath,
		Fields:  make([]FieldInfo, 0),
	}

	for _, field := range structType.Fields.List {
		for _, fieldName := range field.Names {
			fieldInfo := FieldInfo{
				Name: fieldName.Name,
				Type: p.exprToString(field.Type),
			}

			// 解析标签
			if field.Tag != nil {
				tag := strings.Trim(field.Tag.Value, "`")
				fieldInfo.Tag = tag
				fieldInfo.JSONName = p.extractJSONName(tag)
				fieldInfo.Required = !strings.Contains(tag, "omitempty")
			}

			structInfo.Fields = append(structInfo.Fields, fieldInfo)
		}
	}

	key := fmt.Sprintf("%s.%s", packagePath, name)
	p.structs[key] = structInfo

	if p.verbose {
		fmt.Printf("🏗️  找到结构体: %s (字段数: %d)\n", key, len(structInfo.Fields))
	}
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
	for _, handler := range p.handlers {
		if handler.Name == "RegisterRouters" || handler.Name == "RegisterRoutes" {
			if err := p.parseRegisterRoutes(handler); err != nil {
				if p.verbose {
					fmt.Printf("⚠️  解析路由注册失败: %v\n", err)
				}
			}
		}
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
		return nil
	}

	currentGroup := ""

	for _, stmt := range handler.FuncDecl.Body.List {
		switch s := stmt.(type) {
		case *ast.AssignStmt:
			// 查找路由组定义
			if len(s.Lhs) == 1 && len(s.Rhs) == 1 {
				if ident, ok := s.Lhs[0].(*ast.Ident); ok {
					if callExpr, ok := s.Rhs[0].(*ast.CallExpr); ok {
						if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
							if selectorExpr.Sel.Name == "Group" {
								currentGroup = ident.Name
								if len(callExpr.Args) > 0 {
									if basicLit, ok := callExpr.Args[0].(*ast.BasicLit); ok {
										groupPath := strings.Trim(basicLit.Value, "\"")
										if p.verbose {
											fmt.Printf("📂 找到路由组: %s -> %s\n", currentGroup, groupPath)
										}
									}
								}
							}
						}
					}
				}
			}
		case *ast.BlockStmt:
			// 解析路由组内的路由定义
			p.parseRouteBlock(s, currentGroup, handler)
		case *ast.ExprStmt:
			// 直接的路由定义
			p.parseRouteStatement(s, "", handler)
		}
	}

	return nil
}

// parseRouteBlock 解析路由块
func (p *Parser) parseRouteBlock(block *ast.BlockStmt, group string, handler *HandlerInfo) {
	for _, stmt := range block.List {
		if exprStmt, ok := stmt.(*ast.ExprStmt); ok {
			p.parseRouteStatement(exprStmt, group, handler)
		}
	}
}

// parseRouteStatement 解析路由语句
func (p *Parser) parseRouteStatement(stmt *ast.ExprStmt, group string, handler *HandlerInfo) {
	if callExpr, ok := stmt.X.(*ast.CallExpr); ok {
		if selectorExpr, ok := callExpr.Fun.(*ast.SelectorExpr); ok {
			method := strings.ToUpper(selectorExpr.Sel.Name)

			// 检查是否是HTTP方法
			if p.isHTTPMethod(method) && len(callExpr.Args) >= 2 {
				// 获取路径
				var path string
				if basicLit, ok := callExpr.Args[0].(*ast.BasicLit); ok {
					path = strings.Trim(basicLit.Value, "\"")
				}

				// 获取处理函数名
				var handlerFunc string
				if selectorExpr2, ok := callExpr.Args[1].(*ast.SelectorExpr); ok {
					if ident, ok := selectorExpr2.X.(*ast.Ident); ok {
						handlerFunc = fmt.Sprintf("%s.%s", ident.Name, selectorExpr2.Sel.Name)
					}
				} else if ident, ok := callExpr.Args[1].(*ast.Ident); ok {
					handlerFunc = ident.Name
				}

				route := RouteInfo{
					Method:      method,
					Path:        path,
					Handler:     handlerFunc,
					HandlerInfo: handler,
					Group:       group,
				}

				p.routes = append(p.routes, route)

				if p.verbose {
					fmt.Printf("🚏 找到路由: %s %s -> %s\n", method, path, handlerFunc)
				}
			}
		}
	}
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
	re := regexp.MustCompile(`json:"([^"]*)"`)
	matches := re.FindStringSubmatch(tag)
	if len(matches) > 1 {
		jsonTag := matches[1]
		parts := strings.Split(jsonTag, ",")
		if len(parts) > 0 && parts[0] != "" {
			return parts[0]
		}
	}
	return ""
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
