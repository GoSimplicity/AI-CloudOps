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
	"regexp"
	"strings"
	"time"

	"github.com/GoSimplicity/AI-CloudOps/internal/config"
	"github.com/GoSimplicity/AI-CloudOps/internal/model"
	"github.com/GoSimplicity/AI-CloudOps/internal/tree/dao"
	treeSSH "github.com/GoSimplicity/AI-CloudOps/internal/tree/ssh"
	treeUtils "github.com/GoSimplicity/AI-CloudOps/internal/tree/utils"
	"go.uber.org/zap"
)

type sshConnection interface {
	ConnectContext(context.Context, string, int, string, string, string, int8, int) error
	RunContext(context.Context, string) (string, error)
	Close() error
}

type SSHExecutor struct {
	passwordEncryptionKey string
	logger                *zap.Logger
	treeLocalDAO          dao.TreeLocalDAO
	newSSH                func() sshConnection
}

func NewSSHExecutor(cfg *config.Config, logger *zap.Logger, treeLocalDAO dao.TreeLocalDAO) (*SSHExecutor, error) {
	if cfg == nil {
		return nil, fmt.Errorf("配置不能为空")
	}
	if logger == nil {
		logger = zap.NewNop()
	}
	return &SSHExecutor{
		passwordEncryptionKey: cfg.Tree.PasswordEncryptionKey,
		logger:                logger,
		treeLocalDAO:          treeLocalDAO,
		newSSH:                func() sshConnection { return treeSSH.NewSSH(logger) },
	}, nil
}

func (s *SSHExecutor) ExecuteSSHJob(ctx context.Context, job *model.CronJob) (string, error) {
	if ctx == nil || job == nil {
		return "", fmt.Errorf("SSH上下文和任务不能为空")
	}
	if job.JobType != model.CronJobTypeSSH {
		return "", fmt.Errorf("任务类型不是SSH执行任务")
	}
	if job.SSHResourceID == nil || *job.SSHResourceID <= 0 {
		return "", fmt.Errorf("SSH资源ID不能为空")
	}
	if strings.TrimSpace(job.SSHCommand) == "" {
		return "", fmt.Errorf("SSH执行命令不能为空")
	}
	command, err := s.buildSSHCommand(job)
	if err != nil {
		return "", err
	}
	timeout := time.Duration(job.Timeout) * time.Second
	if timeout <= 0 {
		timeout = 5 * time.Minute
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	resource, err := s.treeLocalDAO.GetByID(ctx, *job.SSHResourceID)
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("获取SSH资源失败")
	}
	if resource == nil {
		return "", fmt.Errorf("SSH资源不存在")
	}
	if resource.Status != model.RUNNING {
		return "", fmt.Errorf("SSH资源状态不可用")
	}
	// DAO对象可能被缓存或共享，解密仅用于当前连接的局部副本
	credentials := *resource
	switch credentials.AuthMode {
	case model.AuthModePassword:
		credentials.Password, err = treeUtils.DecryptPassword(credentials.Password, s.passwordEncryptionKey)
		if err != nil {
			return "", fmt.Errorf("SSH密码解密失败")
		}
	case model.AuthModeKey:
	default:
		return "", fmt.Errorf("SSH认证方式无效")
	}
	connection := s.newSSH()
	defer func() {
		if connection.Close() != nil {
			s.logger.Warn("SSH连接清理失败")
		}
	}()
	if err := connection.ConnectContext(ctx, credentials.IpAddr, credentials.Port, credentials.Username, credentials.Password, credentials.Key, int8(credentials.AuthMode), 0); err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("SSH连接失败: %w", err)
	}
	output, err := connection.RunContext(ctx, command)
	if err != nil {
		s.logger.Error("SSH任务执行失败", zap.Int("资源ID", *job.SSHResourceID), zap.Int("输出长度", len(output)))
		return output, fmt.Errorf("SSH命令执行失败: %w", err)
	}
	s.logger.Info("SSH任务执行成功", zap.Int("资源ID", *job.SSHResourceID), zap.Int("输出长度", len(output)))
	return output, nil
}

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func shellLiteral(value string) string { return "'" + strings.ReplaceAll(value, "'", `'"'"'`) + "'" }

// buildSSHCommand 目录与环境仅作为数据，用户显式提供的命令保留shell语义
func (s *SSHExecutor) buildSSHCommand(job *model.CronJob) (string, error) {
	if strings.ContainsRune(job.SSHWorkDir, 0) || strings.ContainsRune(job.SSHCommand, 0) {
		return "", fmt.Errorf("SSH目录或命令包含空字节")
	}
	parts := []string{}
	if job.SSHWorkDir != "" {
		parts = append(parts, "cd -- "+shellLiteral(job.SSHWorkDir))
	}
	for _, kv := range job.SSHEnvironment {
		if !environmentName.MatchString(kv.Key) || strings.ContainsRune(kv.Value, 0) {
			return "", fmt.Errorf("SSH环境变量名或值无效")
		}
		parts = append(parts, "export "+kv.Key+"="+shellLiteral(kv.Value))
	}
	// 分组使cd/export失败阻止整个命令块，换行保留末尾注释的合法语义
	parts = append(parts, "(\n"+job.SSHCommand+"\n)")
	return strings.Join(parts, " && "), nil
}
