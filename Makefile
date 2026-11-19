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
	@goimports -w -local github.com/GoSimplicity/AI-CloudOps ./
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

# 清理本地构建/运行产物
clean:
	@./scripts/clean-local.sh

# 深度清理（包含本地数据卷）
clean-data:
	@./scripts/clean-local.sh --data

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

# 当前入口说明。默认 make 仍是 generate，避免无参数时改跑发布流程。
help:
	@printf '%s\n' \
		'本机开发：' \
		'  make generate          go generate ./...' \
		'  make fmt / fmt-check / fmt-imports / lint' \
		'  make dev-setup         安装 air，不启动服务' \
		'  make dev-air           本机 Air 热重载' \
		'  make clean             清理 bin、tmp、logs' \
		'  make clean-data        额外清理本地 data 目录，不用于发布栈' \
		'隔离栈计划（只打印，不启动）：' \
		'  bash scripts/release-start.sh' \
		'编排文件：deploy/release/compose.yaml' \
		'已停用且不会调用 Docker：' \
		'  make docker-build docker-start-env docker-start-server' \
		'  make docker-stop-env docker-stop-server docker-net-remove dev stop'

# 历史根目录 Compose 与缺失的 docker-compose-env.yaml 不是当前发布入口。
# 这些目标只拒绝，不构建镜像、不启停容器、不删网络或卷。
retired_docker = printf '%s\n' '此目标已停用，不会构建镜像、启动容器、停止容器或删除网络。' '历史根目录 Compose 与不存在的 docker-compose-env.yaml 不是当前发布入口。' '查看离线计划：bash scripts/release-start.sh' '编排文件：deploy/release/compose.yaml' >&2; exit 1

docker-build docker-start-env docker-start-server docker-stop-server docker-stop-env docker-net-remove dev stop:
	@$(retired_docker)

.PHONY: help docker-build docker-start-env docker-start-server docker-stop-server docker-stop-env docker-net-remove dev stop
