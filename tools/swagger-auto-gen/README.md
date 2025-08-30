# Swagger Auto-Gen Tool

AI-CloudOps 项目的 Swagger 文档自动生成工具。

## 功能特性

- 自动解析 Go 代码中的路由和结构体定义
- 生成符合 OpenAPI 2.0 规范的 Swagger 文档
- 支持 JSON、YAML 和 Go docs 格式输出
- 智能识别参数类型（路径、查询、请求体）
- 支持 Gin 框架的路由解析
- 生产环境优化，性能高效

## 使用方法

### 编译

```bash
cd tools/swagger-auto-gen
go build -o ../../bin/swagger-auto-gen .
```

### 运行

```bash
# 在项目根目录运行
./bin/swagger-auto-gen -root . -output ./docs

# 详细输出模式
./bin/swagger-auto-gen -root . -output ./docs -v

# 禁用Swagger生成
./bin/swagger-auto-gen -root . -output ./docs -enabled=false
```

### 参数说明

- `-root`: 项目根目录路径（默认: `.`）
- `-output`: 输出目录路径（默认: `./docs`）
- `-v`: 详细输出模式（默认: `false`）
- `-enabled`: 是否启用Swagger生成（默认: `true`）

## 输出文件

- `swagger.json` - JSON 格式的 Swagger 文档
- `swagger.yaml` - YAML 格式的 Swagger 文档  
- `docs.go` - Go 代码形式的文档，用于 gin-swagger

## 支持的注解

工具会自动识别以下 Go 结构体标签：

- `json:"fieldname"` - JSON 字段名
- `form:"fieldname"` - 查询参数名
- `uri:"fieldname"` - 路径参数名
- `binding:"required"` - 必填字段

## 环境变量控制

工具支持通过环境变量控制行为，这在不同环境下特别有用：

### 文档生成控制

- `SWAGGER_ENABLED`: 是否启用Swagger生成（默认: `true`）
  - 设置为 `false`、`0`、`no`、`n`、`off` 则禁用生成
  - 在生产环境中可以设置为 `false` 以禁用Swagger

- `SWAGGER_ROOT`: 项目根目录路径（等同于 `-root` 参数）

- `SWAGGER_OUTPUT`: 输出目录路径（等同于 `-output` 参数）

- `SWAGGER_VERBOSE`: 是否启用详细输出（等同于 `-v` 参数）

### 应用服务器控制

在启动 AI-CloudOps 应用服务器时，同样可以通过环境变量控制 Swagger：

- `SWAGGER_ENABLED`: 控制是否启用Swagger路由和文档加载
  - `false`: 完全禁用Swagger，不注册 `/swagger/*` 路由，提高安全性和性能
  - `true`: 启用Swagger文档（默认开发环境启用，生产环境禁用）

- `GIN_MODE`: Gin框架模式
  - `release` 或 `production`: 生产模式，默认禁用Swagger
  - `debug`: 开发模式，默认启用Swagger

### 使用示例

```bash
# 禁用Swagger文档生成
SWAGGER_ENABLED=false ./bin/swagger-auto-gen

# 设置输出目录和详细模式
SWAGGER_OUTPUT=./api-docs SWAGGER_VERBOSE=true ./bin/swagger-auto-gen

# 启动应用服务器时禁用Swagger
SWAGGER_ENABLED=false go run main.go

# 生产环境启动（自动禁用Swagger）
GIN_MODE=release go run main.go

# 开发环境强制启用Swagger
SWAGGER_ENABLED=true GIN_MODE=debug go run main.go
```

## 注意事项

1. 确保 Go 代码能够正常编译
2. 路由注册方法需要命名为 `RegisterRouters` 或 `RegisterRoutes`
3. 工具会自动排除测试文件和内部结构体
4. 生产环境建议关闭详细输出模式（不使用 `-v` 参数）
5. 在生产环境中可以设置 `SWAGGER_ENABLED=false` 完全禁用Swagger

## 版本要求

- Go 1.21+
- 依赖包会自动管理，无需手动安装
