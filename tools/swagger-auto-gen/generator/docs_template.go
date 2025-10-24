package generator

// DocsGoTemplate 是生成docs.go文件的模板
const DocsGoTemplate = `// Package docs 自动生成的API文档包. DO NOT EDIT.

package docs

import (
	"os"
	"strings"

	"github.com/swaggo/swag"
)

// SwaggerInfo holds exported Swagger Info so clients can modify it
var SwaggerInfo = &swag.Spec{
	Version:          "{{.Version}}",
	Host:             "{{.Host}}",
	BasePath:         "{{.BasePath}}",
	Schemes:          []string{ {{range .Schemes}}"{{.}}", {{end}} },
	Title:            "{{.Title}}",
	Description:      "{{.Description}}",
	InfoInstanceName: "{{.InstanceName}}",
	SwaggerTemplate:  docTemplate,
	LeftDelim:        "{{",
	RightDelim:       "}}",
}

// isEnvTrue 检查环境变量是否为true
func isEnvTrue(key string) bool {
	value := strings.ToLower(os.Getenv(key))
	return value == "true" || value == "1" || value == "yes" || value == "y" || value == "on"
}

// isSwaggerEnabled 检查是否应该启用Swagger
func isSwaggerEnabled() bool {
	// 优先检查环境变量
	if swaggerEnabled := os.Getenv("SWAGGER_ENABLED"); swaggerEnabled != "" {
		return isEnvTrue("SWAGGER_ENABLED")
	}

	// 默认情况下，开发环境启用，生产环境禁用
	env := strings.ToLower(os.Getenv("GIN_MODE"))
	return env != "release" && env != "production"
}

func init() {
	// 只有当环境变量允许时才注册Swagger
	if isSwaggerEnabled() {
		swag.Register(SwaggerInfo.InstanceName(), SwaggerInfo)
	}
}

const docTemplate = ` + "`{{.DocJSON}}`" + `
`
