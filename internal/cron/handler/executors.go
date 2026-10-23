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

package handler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"go.uber.org/zap"
)

// CommandExecutor 命令执行器
type CommandExecutor struct {
	logger *zap.Logger
}

func NewCommandExecutor(logger *zap.Logger) *CommandExecutor {
	return &CommandExecutor{
		logger: logger,
	}
}

type CommandTask struct {
	Command     string             `json:"command"`
	Args        model.StringList   `json:"args"`
	WorkDir     string             `json:"work_dir"`
	Environment model.KeyValueList `json:"environment"`
	Timeout     int                `json:"timeout"`
}

func (h *CommandExecutor) Execute(ctx context.Context, task *CommandTask) (string, error) {
	timeout := time.Duration(task.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute // 默认5分钟
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var cmd *exec.Cmd
	if len(task.Args) > 0 {
		args := make([]string, len(task.Args))
		for i, arg := range task.Args {
			args[i] = string(arg)
		}
		cmd = exec.CommandContext(ctx, task.Command, args...)
	} else {
		cmd = exec.CommandContext(ctx, task.Command)
	}

	if task.WorkDir != "" {
		cmd.Dir = task.WorkDir
	}

	if len(task.Environment) > 0 {
		env := os.Environ()
		for _, kv := range task.Environment {
			env = append(env, fmt.Sprintf("%s=%s", kv.Key, kv.Value))
		}
		cmd.Env = env
	}

	// 执行命令并捕获输出
	stdout, stderr := newOutputBuffer(), newOutputBuffer()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	h.logger.Info("执行命令",
		zap.String("command", task.Command),
		zap.Any("args", task.Args))

	err := runTaskCommand(cmd)

	// 合并标准输出和错误输出
	output := strings.TrimSpace(stdout.String())
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n" + strings.TrimSpace(stderr.String())
		} else {
			output = strings.TrimSpace(stderr.String())
		}
	}

	if err != nil {
		h.logger.Error("命令执行失败",
			zap.String("command", task.Command),
			zap.Error(err))
		return output, fmt.Errorf("命令执行失败: %w", err)
	}

	h.logger.Info("命令执行成功", zap.String("command", task.Command))
	return output, nil
}

// HTTPExecutor HTTP执行器
type HTTPExecutor struct {
	logger *zap.Logger
	client *http.Client
}

// NewHTTPExecutor 创建HTTP执行器
func NewHTTPExecutor(logger *zap.Logger) *HTTPExecutor {
	return &HTTPExecutor{
		logger: logger,
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// HTTPTask HTTP任务配置
type HTTPTask struct {
	Method  string             `json:"method"`
	URL     string             `json:"url"`
	Headers model.KeyValueList `json:"headers"`
	Body    string             `json:"body"`
	Timeout int                `json:"timeout"`
}

// Execute 执行HTTP请求，返回响应字符串
func (h *HTTPExecutor) Execute(ctx context.Context, task *HTTPTask) (string, error) {
	timeout := time.Duration(task.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 30 * time.Second // 默认30秒
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var bodyReader io.Reader
	if task.Body != "" {
		bodyReader = strings.NewReader(task.Body)
	}

	req, err := http.NewRequestWithContext(ctx, task.Method, task.URL, bodyReader)
	if err != nil {
		return "", fmt.Errorf("创建HTTP请求失败: %w", err)
	}

	for _, header := range task.Headers {
		req.Header.Set(header.Key, header.Value)
	}

	h.logger.Info("执行HTTP请求",
		zap.String("method", task.Method),
		zap.String("url", task.URL))

	resp, err := h.client.Do(req)
	if err != nil {
		h.logger.Error("HTTP请求失败", zap.Error(err))
		return "", fmt.Errorf("HTTP请求失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxTaskOutputBytes+1))
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	if len(body) > maxTaskOutputBytes {
		return strings.ToValidUTF8(string(body[:maxTaskOutputBytes]), "") + outputTruncatedMarker, fmt.Errorf("HTTP响应超过%d字节限制", maxTaskOutputBytes)
	}

	response := string(body)

	h.logger.Info("HTTP请求完成",
		zap.String("url", task.URL),
		zap.Int("statusCode", resp.StatusCode))

	if resp.StatusCode >= 400 {
		return response, fmt.Errorf("HTTP请求失败，状态码: %d", resp.StatusCode)
	}

	return response, nil
}

// ScriptExecutor 脚本执行器
type ScriptExecutor struct {
	logger *zap.Logger
}

func NewScriptExecutor(logger *zap.Logger) *ScriptExecutor {
	return &ScriptExecutor{
		logger: logger,
	}
}

type ScriptTask struct {
	Type    string `json:"type"`
	Content string `json:"content"`
	Timeout int    `json:"timeout"`
}

func (h *ScriptExecutor) Execute(ctx context.Context, task *ScriptTask) (string, error) {
	timeout := time.Duration(task.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute // 默认5分钟
	}

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 根据脚本类型选择解释器
	var interpreter string
	argument := "-c"
	switch strings.ToLower(task.Type) {
	case "shell", "bash", "sh":
		interpreter = "/bin/bash"
	case "python", "python3":
		interpreter = "python3"
	case "node", "nodejs", "javascript":
		interpreter = "node"
		argument = "-e"
	default:
		return "", fmt.Errorf("不支持的脚本类型: %s", task.Type)
	}

	// 执行脚本
	cmd := exec.CommandContext(ctx, interpreter, argument, task.Content)

	// 执行并捕获输出
	stdout, stderr := newOutputBuffer(), newOutputBuffer()
	cmd.Stdout = stdout
	cmd.Stderr = stderr

	h.logger.Info("执行脚本",
		zap.String("type", task.Type),
		zap.String("interpreter", interpreter))

	err := runTaskCommand(cmd)

	// 合并标准输出和错误输出
	output := strings.TrimSpace(stdout.String())
	if stderr.Len() > 0 {
		if output != "" {
			output += "\n" + strings.TrimSpace(stderr.String())
		} else {
			output = strings.TrimSpace(stderr.String())
		}
	}

	if err != nil {
		h.logger.Error("脚本执行失败",
			zap.String("type", task.Type),
			zap.Error(err))
		return output, fmt.Errorf("脚本执行失败: %w", err)
	}

	h.logger.Info("脚本执行成功", zap.String("type", task.Type))
	return output, nil
}

// 为 MySQL TEXT 的 65535 字节预算预留双流标记空间，超限内容继续消费
const maxTaskOutputBytes = (32 << 10) - 128
const outputTruncatedMarker = "[输出已截断]"

type outputBuffer struct {
	buffer    bytes.Buffer
	truncated bool
}

func newOutputBuffer() *outputBuffer { return &outputBuffer{} }

// Write 按 io.Writer 契约返回已消费长度，丢弃超限部分
func (b *outputBuffer) Write(p []byte) (int, error) {
	size := len(p)
	remaining := maxTaskOutputBytes - b.buffer.Len()
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, _ = b.buffer.Write(p)
	return size, nil
}

func (b *outputBuffer) Len() int { return b.buffer.Len() }

func (b *outputBuffer) String() string {
	if b.truncated {
		return strings.ToValidUTF8(b.buffer.String(), "") + outputTruncatedMarker
	}
	return b.buffer.String()
}

// runTaskCommand 同时限制进程终止和继承管道的等待时间
func runTaskCommand(cmd *exec.Cmd) error {
	cmd.WaitDelay = 2 * time.Second
	configureTaskProcess(cmd)
	err := cmd.Run()
	// 父进程先退出而子进程仍持有管道时，WaitDelay返回后也清理进程组
	if err != nil && cmd.Process != nil && cmd.Cancel != nil {
		_ = cmd.Cancel()
	}
	return err
}
