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

package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/GoSimplicity/AI-CloudOps/tools/swagger-auto-gen/generator"
)

func main() {
	var (
		projectRoot = flag.String("root", ".", "项目根目录")
		outputDir   = flag.String("output", "./docs", "输出目录")
		verbose     = flag.Bool("v", false, "详细输出")
	)
	flag.Parse()

	// 获取绝对路径
	absRoot, err := filepath.Abs(*projectRoot)
	if err != nil {
		log.Fatalf("获取项目根目录绝对路径失败: %v", err)
	}

	absOutput, err := filepath.Abs(*outputDir)
	if err != nil {
		log.Fatalf("获取输出目录绝对路径失败: %v", err)
	}

	// 创建输出目录
	if err := os.MkdirAll(absOutput, 0755); err != nil {
		log.Fatalf("创建输出目录失败: %v", err)
	}

	fmt.Println("🚀 AI-CloudOps 自动 Swagger 文档生成器")
	fmt.Printf("📁 项目根目录: %s\n", absRoot)
	fmt.Printf("📄 输出目录: %s\n", absOutput)

	// 创建生成器
	gen := generator.NewSwaggerGenerator(absRoot, absOutput, *verbose)

	// 生成文档
	if err := gen.Generate(); err != nil {
		log.Fatalf("生成 Swagger 文档失败: %v", err)
	}

	fmt.Println("✅ Swagger 文档生成完成！")
}
