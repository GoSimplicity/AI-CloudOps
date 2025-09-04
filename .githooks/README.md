# Git Hooks 使用说明

## Pre-commit Hook

这个pre-commit钩子会在每次提交时自动执行以下检查：

### 执行步骤

1. **`go generate ./...`** - 自动生成代码（如Wire依赖注入、Swagger文档等）
2. **代码格式化** - 自动运行`go fmt`并将格式化后的文件添加到暂存区
3. **静态分析** - 运行`go vet`检查潜在问题
4. **编译检查** - 运行`go build`确保代码可以正常编译

### 跳过选项

如果遇到临时问题需要跳过某些检查，可以使用以下环境变量：

```bash
# 跳过 go generate（用于Wire依赖注入问题等）
SKIP_GENERATE=1 git commit -m "修复登录功能"

# 跳过静态分析检查
SKIP_VET=1 git commit -m "临时提交，稍后修复警告"

# 跳过编译检查
SKIP_BUILD=1 git commit -m "WIP: 添加新功能"

# 组合使用多个跳过选项
SKIP_GENERATE=1 SKIP_VET=1 git commit -m "快速修复"
```

### 常见问题解决

#### Wire 依赖注入错误
```
wire: no provider found for invalid type
```

**解决方案：**
1. 检查`pkg/di/wire.go`中的provider配置
2. 确保所有依赖都已正确定义
3. 运行`go mod tidy`
4. 临时跳过：`SKIP_GENERATE=1 git commit -m "你的消息"`

#### 格式化问题
钩子会自动格式化代码并添加到暂存区，无需手动处理。

#### 编译错误
解决所有编译错误后重新提交，或使用`SKIP_BUILD=1`临时跳过。

### 设置和配置

确保Git配置了正确的钩子路径：
```bash
git config core.hooksPath .githooks
```

检查钩子权限：
```bash
chmod +x .githooks/pre-commit
```

### 钩子输出示例

```
[PRE-COMMIT] 当前分支: feature/user-auth
[PRE-COMMIT] 检测到 Go 文件变更，开始执行检查...
[PRE-COMMIT] 步骤 1/4: 执行 go generate ./...
[PRE-COMMIT] go generate 执行成功
[PRE-COMMIT] 步骤 2/4: 检查代码格式...
[PRE-COMMIT] 代码格式检查完成
[PRE-COMMIT] 步骤 3/4: 执行静态分析 (go vet)...
[PRE-COMMIT] 静态分析通过
[PRE-COMMIT] 步骤 4/4: 执行编译检查...
[PRE-COMMIT] 编译检查通过
[PRE-COMMIT] 所有检查通过，准备提交
```

### 团队协作建议

1. **首次设置**：每个团队成员需要运行`git config core.hooksPath .githooks`
2. **定期更新**：当钩子更新时，团队成员需要重新设置权限
3. **问题反馈**：如果钩子出现问题，请及时反馈给团队
4. **谨慎跳过**：只在确实需要时使用跳过选项，并在后续提交中修复问题

### 禁用钩子

如果需要完全禁用钩子：
```bash
git config core.hooksPath ""
# 或者
git commit --no-verify -m "绕过钩子提交"
```
