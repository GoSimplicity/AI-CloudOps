generate:
	go generate ./...

# 生成 Swagger API 文档 (传统方式，需要手动注释)
swagger-manual:
	@echo "正在生成API文档（传统方式）..."
	@swag init --output ./docs --parseDependency --parseInternal --exclude ./internal/*/service --dir ./ --generalInfo main.go
	@echo "API文档已生成到 docs/ 目录"
	@echo "访问地址: http://localhost:8889/swagger/index.html"

# 自动生成 Swagger API 文档（无需手动注释）
swagger:
	@echo "🚀 正在自动生成API文档..."
	@echo "📁 构建自动生成工具..."
	@cd tools/swagger-auto-gen && go build -o ../../bin/swagger-auto-gen .
	@echo "🔍 分析项目结构并生成文档..."
	@./bin/swagger-auto-gen -root . -output ./docs -v
	@echo "✅ API文档已自动生成到 docs/ 目录"
	@echo "🌐 访问地址: http://localhost:8889/swagger/index.html"

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

# 完整的 Swagger 工作流
swagger-all: swagger-clean swagger swagger-validate swagger-check
	@echo "🎉 Swagger 文档生成完成！"

# 安装 Git hooks 自动生成 Swagger 文档
swagger-setup:
	@echo "设置 Swagger 自动生成..."
	@bash scripts/setup-git-hooks.sh

# 启用自动监控模式（开发时使用）
swagger-watch:
	@echo "启动 Swagger 文档自动监控..."
	@if [ ! -f "scripts/swagger-watcher.sh" ]; then \
		echo "❌ swagger-watcher.sh 脚本不存在，请先运行 make swagger-setup"; \
		exit 1; \
	fi
	@bash scripts/swagger-watcher.sh

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

# 开发环境完整设置
dev-setup: swagger-setup install-dev-tools
	@echo "🎉 开发环境设置完成！"
	@echo ""
	@echo "可用命令:"
	@echo "  make dev-air          # 使用 Air 热重载启动"
	@echo "  make swagger-watch     # 仅监控 Swagger 文档"
	@echo "  make swagger           # 手动生成 Swagger 文档"
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