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
 */

// Package main provides code generation directives for AI-CloudOps project.
//
// This file contains all go:generate directives for the project.
// Run `go generate` in the project root to execute all generation tasks.
package main

// Generate Swagger API documentation (Auto-generated, no manual annotations required)
//go:generate bash -c "echo '🔄 正在生成 Swagger 文档...'"
//go:generate bash -c "echo '[INFO] 使用自动生成模式（无需手动注释）...'"
//go:generate bash -c "echo '[INFO] 构建自动生成工具...'"
//go:generate bash -c "cd tools/swagger-auto-gen && go build -o ../../bin/swagger-auto-gen ."
//go:generate bash -c "echo '[INFO] 分析项目结构并生成文档...'"
//go:generate bash -c "./bin/swagger-auto-gen -root . -output ./docs -v"
//go:generate bash -c "if [ -f docs/swagger.json ]; then echo '[SUCCESS] 文档生成成功！文件大小: $(du -h docs/swagger.json | cut -f1)'; else echo '[ERROR] 文档生成失败'; fi"
//go:generate bash -c "echo '[INFO] 访问地址: http://localhost:8889/swagger/index.html'"
//go:generate bash -c "echo '✅ Swagger 文档生成完成！'"

// Generate wire dependency injection code (if wire.go exists)
//go:generate bash -c "if [ -f 'pkg/di/wire.go' ]; then cd pkg/di && wire; echo '✅ Wire 依赖注入代码生成完成'; fi"

// Generate additional project files
//go:generate bash -c "echo '📋 代码生成任务完成 - $(date)'"
