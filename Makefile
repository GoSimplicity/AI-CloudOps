generate:
	go generate ./...

# 生成 Swagger API 文档 (传统方式，需要手动注释)
swagger-manual:
	@echo "正在生成API文档（传统方式）..."
	@swag init --output ./docs --parseDependency --parseInternal --exclude ./internal/*/service --dir ./ --generalInfo main.go
	@echo "API文档已生成到 docs/ 目录"
	@echo "访问地址: http://localhost:8889/swagger/index.html"

# 自动生成 Swagger API 文档（无需手动注释，自动同步到 docs.go）
swagger:
	@echo "🚀 正在生成API文档..."
	@echo "📄 使用自动生成工具生成文档..."
	@bash scripts/generate-swagger.sh
	@echo "✅ Swagger 文档生成完成！"

# 禁用 Swagger 生成（设置环境变量）
swagger-disable:
	@echo "🔒 禁用 Swagger 文档生成..."
	@export SWAGGER_ENABLED=false
	@export SWAGGER_VERBOSE=false
	@echo "✅ Swagger 生成已禁用"
	@echo "💡 提示: 使用 'export SWAGGER_ENABLED=false' 永久禁用"

# 兼容旧的命令名
openai: swagger

# 检查 Swagger 注解完整性
swagger-check:
	@echo "检查 Swagger 注解..."
	@echo "正在统计 API 函数..."
	@total_funcs=$$(grep -c "func.*Handler.*[Gg]et\|[Pp]ost\|[Pp]ut\|[Dd]elete" internal/*/api/*.go); \
	 swagger_funcs=$$(grep -c "@Summary" internal/*/api/*.go); \
	 echo "总 API 函数数量: $$total_funcs"; \
	 echo "包含 Swagger 注解的函数: $$swagger_funcs"; \
	 if [ $$swagger_funcs -lt $$total_funcs ]; then \
	     echo "⚠️  发现 $$(( $$total_funcs - $$swagger_funcs )) 个函数缺少 Swagger 注解"; \
	     echo "缺少注解的文件:"; \
	     grep -L "@Summary" internal/*/api/*.go || true; \
	 else \
	     echo "✅ 所有 API 函数都包含 Swagger 注解"; \
	 fi

# 启动本地服务器并打开 Swagger UI
swagger-serve:
	@echo "启动服务器..."
	@if pgrep -f "AI-CloudOps" > /dev/null; then \
		echo "服务器已在运行"; \
	else \
		echo "请先启动服务器: make dev 或 go run main.go"; \
	fi
	@echo "Swagger UI 访问地址: http://localhost:8889/swagger/index.html"
	@if command -v open > /dev/null; then \
		open http://localhost:8889/swagger/index.html; \
	elif command -v xdg-open > /dev/null; then \
		xdg-open http://localhost:8889/swagger/index.html; \
	fi

# 验证生成的 API 文档
swagger-validate:
	@echo "验证 Swagger 文档..."
	@if [ -f "docs/swagger.json" ]; then \
		echo "✅ swagger.json 文件存在"; \
		echo "文件大小: $$(du -h docs/swagger.json | cut -f1)"; \
	else \
		echo "❌ swagger.json 文件不存在，请先运行 make swagger"; \
		exit 1; \
	fi
	@if [ -f "docs/swagger.yaml" ]; then \
		echo "✅ swagger.yaml 文件存在"; \
		echo "文件大小: $$(du -h docs/swagger.yaml | cut -f1)"; \
	else \
		echo "❌ swagger.yaml 文件不存在，请先运行 make swagger"; \
		exit 1; \
	fi
	@api_count=$$(grep -c '"paths"' docs/swagger.json 2>/dev/null || echo "0"); \
	 echo "API 路径数量: $$api_count"

# 清理生成的文档
swagger-clean:
	@echo "清理 Swagger 文档..."
	@rm -f docs/docs.go docs/swagger.json docs/swagger.yaml
	@echo "✅ 文档已清理"

# 完整的 Swagger 工作流（包含自动同步）
swagger-all: swagger-clean swagger swagger-validate swagger-check
	@echo "🎉 Swagger 文档生成并同步完成！"

# 安装 Git hooks 和自动同步机制
swagger-setup:
	@echo "设置 Swagger 自动生成和同步..."
	@bash scripts/setup-git-hooks.sh

# 启用自动监控模式（开发时使用）
swagger-watch:
	@echo "⚠️  自动监控功能已被禁用以防止循环生成问题"
	@echo "💡 建议手动使用: make swagger"
	@echo "🔧 如需启用监控，请联系开发者进行安全配置"

# 快速构建（包含 swagger 生成）
build-with-docs: swagger
	@echo "构建项目（包含文档）..."
	@go build -o bin/ai-cloudops main.go
	@echo "✅ 构建完成，可执行文件: bin/ai-cloudops"

# 开发模式启动（自动生成文档）
dev-with-docs: swagger
	@echo "开发模式启动（包含最新文档）..."
	@go run main.go

# 检查并自动修复 Swagger 注解
swagger-fix:
	@echo "检查并修复 Swagger 注解..."
	@bash scripts/swagger-helper.sh fix

# 安装开发工具
install-dev-tools:
	@echo "安装开发工具..."
	@go install github.com/air-verse/air@latest
	@go install github.com/swaggo/swag/cmd/swag@latest
	@echo "✅ 开发工具安装完成"

# 使用 Air 启动开发服务器（支持热重载和自动生成 Swagger）
dev-air: 
	@if ! command -v air &> /dev/null; then \
		echo "❌ air 工具未安装，正在安装..."; \
		go install github.com/air-verse/air@latest; \
	fi
	@echo "🚀 启动开发服务器 (Air + 自动 Swagger 生成)..."
	@air

# 检查文档同步状态
swagger-sync-check:
	@echo "检查 Swagger 文档同步状态..."
	@bash scripts/swagger-auto-sync.sh verify

# 手动同步文档
swagger-sync:
	@echo "手动同步 Swagger 文档..."
	@bash scripts/swagger-auto-sync.sh sync

# 强制重新同步文档
swagger-sync-force:
	@echo "强制重新同步 Swagger 文档..."
	@bash scripts/swagger-auto-sync.sh force

# 开发环境完整设置
dev-setup: swagger-setup install-dev-tools
	@echo "🎉 开发环境设置完成！"
	@echo ""
	@echo "可用命令:"
	@echo "  make dev-air           # 使用 Air 热重载启动"
	@echo "  make swagger-watch     # 监控 Swagger 文档变化"
	@echo "  make swagger           # 生成文档并自动同步"
	@echo "  make swagger-sync      # 手动同步文档"
	@echo "  make swagger-sync-check # 检查同步状态"
	@echo "  go generate            # 使用 Go generate 生成文档"

docker-build:
	docker build -t Bamboo/gomodd:v1.23.1 .

docker-start-env:
	docker-compose -f docker-compose-env.yaml up -d

docker-start-server:
	docker-compose -f docker-compose.yaml up -d

docker-stop-server:
	docker-compose -f docker-compose.yaml down

docker-stop-env:
	docker-compose -f docker-compose-env.yaml down

docker-net-remove:
	docker network rm cloudOps_net

dev: docker-build docker-start-env docker-start-server

stop: docker-stop-env docker-stop-server docker-net-remove