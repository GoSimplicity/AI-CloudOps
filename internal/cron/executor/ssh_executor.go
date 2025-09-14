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

package executor

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/ssh"
	"go.uber.org/zap"
)

// SSHExecutor SSH远程执行器
type SSHExecutor struct {
	logger       *zap.Logger
	treeLocalDAO dao.TreeLocalDAO
}

// NewSSHExecutor 创建SSH执行器
func NewSSHExecutor(logger *zap.Logger, treeLocalDAO dao.TreeLocalDAO) *SSHExecutor {
	return &SSHExecutor{
		logger:       logger,
		treeLocalDAO: treeLocalDAO,
	}
}

// ExecuteSSHJob 执行SSH任务
func (e *SSHExecutor) ExecuteSSHJob(ctx context.Context, job *model.CronJob) (string, error) {
	e.logger.Info("开始执行SSH任务",
		zap.String("任务名称", job.Name),
		zap.Int("资源ID", job.SSHResourceID))

	// 验证SSH任务配置
	if job.JobType != model.CronJobTypeSSH {
		return "", fmt.Errorf("任务类型不是SSH执行任务")
	}

	if job.SSHResourceID == 0 {
		return "", fmt.Errorf("SSH资源ID不能为空")
	}

	if job.SSHCommand == "" {
		return "", fmt.Errorf("SSH执行命令不能为空")
	}

	// 获取SSH资源信息
	resource, err := e.treeLocalDAO.GetByID(ctx, job.SSHResourceID)
	if err != nil {
		e.logger.Error("获取SSH资源失败",
			zap.Int("资源ID", job.SSHResourceID),
			zap.Error(err))
		return "", fmt.Errorf("获取SSH资源失败: %w", err)
	}

	if resource == nil {
		return "", fmt.Errorf("SSH资源不存在: ID=%d", job.SSHResourceID)
	}

	// 验证资源状态
	if resource.Status != model.RUNNING {
		return "", fmt.Errorf("SSH资源状态不可用: %v", resource.Status)
	}

	// 创建SSH连接
	sshClient := ssh.NewSSH(e.logger)
	defer func() {
		if closeErr := sshClient.Close(); closeErr != nil {
			e.logger.Error("关闭SSH连接失败", zap.Error(closeErr))
		}
	}()

	// 根据认证方式连接SSH
	var authMode int8
	if resource.AuthMode == model.AuthModePassword {
		authMode = 1
	} else {
		authMode = 2
	}

	err = sshClient.Connect(
		resource.IpAddr,
		resource.Port,
		resource.Username,
		resource.Password,
		resource.Key,
		authMode,
		0, // 系统任务使用用户ID 0
	)
	if err != nil {
		e.logger.Error("SSH连接失败",
			zap.String("地址", resource.IpAddr),
			zap.Int("端口", resource.Port),
			zap.String("用户名", resource.Username),
			zap.Error(err))
		return "", fmt.Errorf("SSH连接失败: %w", err)
	}

	e.logger.Info("SSH连接成功", zap.String("地址", resource.IpAddr))

	// 构建执行命令
	command := e.buildSSHCommand(job)

	// 设置超时控制
	cmdCtx, cancel := context.WithTimeout(ctx, time.Duration(job.Timeout)*time.Second)
	defer cancel()

	// 在goroutine中执行命令，支持超时控制
	resultCh := make(chan string, 1)
	errorCh := make(chan error, 1)

	go func() {
		output, err := sshClient.Run(command)
		if err != nil {
			errorCh <- err
		} else {
			resultCh <- output
		}
	}()

	// 等待命令执行完成或超时
	select {
	case <-cmdCtx.Done():
		return "", fmt.Errorf("SSH命令执行超时: %d秒", job.Timeout)
	case err := <-errorCh:
		e.logger.Error("SSH命令执行失败",
			zap.String("命令", command),
			zap.Error(err))
		return "", fmt.Errorf("SSH命令执行失败: %w", err)
	case output := <-resultCh:
		e.logger.Info("SSH命令执行成功",
			zap.String("命令", command),
			zap.String("输出长度", fmt.Sprintf("%d字符", len(output))))
		return output, nil
	}
}

// buildSSHCommand 构建SSH执行命令
func (e *SSHExecutor) buildSSHCommand(job *model.CronJob) string {
	var commandParts []string

	// 设置工作目录
	if job.SSHWorkDir != "" {
		commandParts = append(commandParts, fmt.Sprintf("cd %s", job.SSHWorkDir))
	}

	// 设置环境变量
	if len(job.SSHEnvironment) > 0 {
		for key, value := range job.SSHEnvironment {
			commandParts = append(commandParts, fmt.Sprintf("export %d=%s", key, value))
		}
	}

	// 添加要执行的命令
	commandParts = append(commandParts, job.SSHCommand)

	// 使用 && 连接所有命令，确保按顺序执行
	return strings.Join(commandParts, " && ")
}

// ValidateSSHJob 验证SSH任务配置
func (e *SSHExecutor) ValidateSSHJob(ctx context.Context, job *model.CronJob) error {
	if job.JobType != model.CronJobTypeSSH {
		return fmt.Errorf("任务类型不是SSH执行任务")
	}

	if job.SSHResourceID == 0 {
		return fmt.Errorf("SSH资源ID不能为空")
	}

	if job.SSHCommand == "" {
		return fmt.Errorf("SSH执行命令不能为空")
	}

	// 验证SSH资源是否存在且可用
	resource, err := e.treeLocalDAO.GetByID(ctx, job.SSHResourceID)
	if err != nil {
		return fmt.Errorf("获取SSH资源失败: %w", err)
	}

	if resource == nil {
		return fmt.Errorf("SSH资源不存在: ID=%d", job.SSHResourceID)
	}

	return nil
}

// TestSSHConnection 测试SSH连接
func (e *SSHExecutor) TestSSHConnection(ctx context.Context, resourceID int) error {
	resource, err := e.treeLocalDAO.GetByID(ctx, resourceID)
	if err != nil {
		return fmt.Errorf("获取SSH资源失败: %w", err)
	}

	if resource == nil {
		return fmt.Errorf("SSH资源不存在: ID=%d", resourceID)
	}

	sshClient := ssh.NewSSH(e.logger)
	defer sshClient.Close()

	var authMode int8
	if resource.AuthMode == model.AuthModePassword {
		authMode = 1
	} else {
		authMode = 2
	}

	err = sshClient.Connect(
		resource.IpAddr,
		resource.Port,
		resource.Username,
		resource.Password,
		resource.Key,
		authMode,
		0,
	)
	if err != nil {
		return fmt.Errorf("SSH连接测试失败: %w", err)
	}

	// 执行简单的测试命令
	_, err = sshClient.Run("echo 'SSH连接测试成功'")
	if err != nil {
		return fmt.Errorf("SSH命令测试失败: %w", err)
	}

	return nil
}
