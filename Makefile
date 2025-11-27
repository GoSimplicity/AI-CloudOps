generate:
	go generate ./...

# 格式化代码
fmt:
	@echo "🎨 正在格式化代码..."
	@go fmt ./...
	@echo "✅ 代码格式化完成"

# 格式化代码并整理导入 (需要安装 goimports)
fmt-imports:
	@echo "🎨 正在格式化代码并整理导入..."
	@if ! command -v goimports &> /dev/null; then \
		echo "⚠️  goimports 未安装，正在安装..."; \
		go install golang.org/x/tools/cmd/goimports@latest; \
	fi
	@goimports -w -local github.com/yourusername ./
	@echo "✅ 代码格式化和导入整理完成"

# 检查代码格式 (用于 CI)
fmt-check:
	@echo "🔍 检查代码格式..."
	@unformatted=$$(gofmt -l .); \
	if [ -n "$$unformatted" ]; then \
		echo "❌ 以下文件需要格式化:"; \
		echo "$$unformatted"; \
		exit 1; \
	else \
		echo "✅ 所有文件格式正确"; \
	fi

# 代码检查和格式化 (包含 go vet)
lint:
	@echo "🔍 正在进行代码检查..."
	@go vet ./...
	@echo "✅ 代码检查完成"

# 安装开发工具
install-dev-tools:
	@echo "安装开发工具..."
	@go install github.com/air-verse/air@latest
	@echo "✅ 开发工具安装完成"

# 使用 Air 启动开发服务器
dev-air: 
	@if ! command -v air &> /dev/null; then \
		echo "❌ air 工具未安装，正在安装..."; \
		go install github.com/air-verse/air@latest; \
	fi
	@echo "🚀 启动开发服务器 (Air 热重载)..."
	@air

# 开发环境设置（不包含自动生成功能）
dev-setup: install-dev-tools
	@echo "🎉 开发环境设置完成！"
	@echo ""
	@echo "可用命令:"
	@echo "  make dev-air           # 使用 Air 热重载启动"
	@echo "  make fmt               # 格式化代码"
	@echo "  make fmt-imports       # 格式化代码并整理导入"
	@echo "  make fmt-check         # 检查代码格式"
	@echo "  make lint              # 代码检查 (go vet)"

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
