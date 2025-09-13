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

package ssh

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

// AuthMode SSH认证模式
type AuthMode int8

const (
	AuthModePassword AuthMode = 1 // 密码认证
	AuthModeKey      AuthMode = 2 // 密钥认证
)

// Config SSH连接配置
type Config struct {
	Host     string   `json:"host"`     // 服务器地址
	Port     int      `json:"port"`     // SSH端口，默认22
	Username string   `json:"username"` // 用户名
	Password string   `json:"password"` // 密码（密码认证时使用）
	Key      string   `json:"key"`      // 私钥内容（密钥认证时使用）
	Mode     AuthMode `json:"mode"`     // 认证方式
	Timeout  int      `json:"timeout"`  // 连接超时时间（秒），默认10
}

// Client SSH客户端接口
type Client interface {
	// Connect 建立SSH连接
	Connect(config *Config) error
	// Run 执行单个命令
	Run(command string) (string, error)
	// CreateSession 创建新的SSH会话
	CreateSession(userID int) error
	// GetSession 获取用户会话
	GetSession(userID int) *ssh.Session
	// CloseSession 关闭指定用户会话
	CloseSession(userID int) error
	// AddPublicKey 添加公钥到远程主机
	AddPublicKey(publicKey string) error
	// Close 关闭SSH客户端
	Close() error
	// WebTerminal 提供Web终端功能
	WebTerminal(userID int, conn *websocket.Conn) error
}

// client SSH客户端实现
type client struct {
	config     *Config              // 连接配置
	sshClient  *ssh.Client          // SSH客户端
	sessions   map[int]*ssh.Session // 用户会话映射
	sessionMux sync.RWMutex         // 会话锁
	logger     *zap.Logger          // 日志记录器
}

// NewClient 创建SSH客户端
func NewClient(logger *zap.Logger) Client {
	if logger == nil {
		logger = zap.NewNop()
	}

	return &client{
		sessions: make(map[int]*ssh.Session),
		logger:   logger,
	}
}

// Connect 建立SSH连接
func (c *client) Connect(config *Config) error {
	if err := c.validateConfig(config); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	c.config = config

	// 设置默认参数
	if config.Port <= 0 || config.Port > 65535 {
		config.Port = 22
		c.logger.Warn("端口号无效，使用默认端口22", zap.Int("原端口", config.Port))
	}

	if config.Timeout <= 0 {
		config.Timeout = 10
	}

	// 构建SSH配置
	sshConfig := &ssh.ClientConfig{
		User:            config.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境建议使用安全的主机密钥验证
		Timeout:         time.Duration(config.Timeout) * time.Second,
	}

	// 配置认证方法
	auth, err := c.getAuthMethod(config)
	if err != nil {
		return fmt.Errorf("配置认证方法失败: %w", err)
	}
	sshConfig.Auth = []ssh.AuthMethod{auth}

	// 建立连接
	addr := fmt.Sprintf("%s:%d", config.Host, config.Port)
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		c.logger.Error("SSH连接失败",
			zap.String("地址", addr),
			zap.String("用户名", config.Username),
			zap.Error(err))
		return fmt.Errorf("SSH连接失败: %w", err)
	}

	c.sshClient = sshClient
	c.logger.Info("SSH连接成功", zap.String("地址", addr))

	return nil
}

// Run 执行单个命令
func (c *client) Run(command string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	if c.sshClient == nil {
		return "", fmt.Errorf("SSH客户端未连接")
	}

	// 创建新会话执行命令
	session, err := c.sshClient.NewSession()
	if err != nil {
		c.logger.Error("创建命令执行会话失败", zap.Error(err))
		return "", fmt.Errorf("创建会话失败: %w", err)
	}
	defer func() {
		if closeErr := session.Close(); closeErr != nil {
			c.logger.Error("关闭会话失败", zap.Error(closeErr))
		}
	}()

	// 执行命令并获取输出
	c.logger.Debug("执行命令", zap.String("命令", command))
	output, err := session.CombinedOutput(command)
	result := string(output)

	if err != nil {
		c.logger.Error("命令执行失败",
			zap.String("命令", command),
			zap.String("输出", result),
			zap.Error(err))
		return result, fmt.Errorf("执行命令失败: %w", err)
	}

	c.logger.Debug("命令执行成功",
		zap.String("命令", command),
		zap.Int("输出长度", len(result)))

	return result, nil
}

// CreateSession 创建SSH会话
func (c *client) CreateSession(userID int) error {
	if c.sshClient == nil {
		return fmt.Errorf("SSH客户端未连接")
	}

	session, err := c.sshClient.NewSession()
	if err != nil {
		c.logger.Error("创建SSH会话失败", zap.Error(err))
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}

	c.sessionMux.Lock()
	// 如果已存在会话则先关闭
	if existingSession, exists := c.sessions[userID]; exists {
		if closeErr := existingSession.Close(); closeErr != nil {
			c.logger.Error("关闭已存在的会话失败", zap.Int("用户ID", userID), zap.Error(closeErr))
		}
	}
	c.sessions[userID] = session
	c.sessionMux.Unlock()

	c.logger.Info("SSH会话创建成功", zap.Int("用户ID", userID))
	return nil
}

// GetSession 获取用户会话
func (c *client) GetSession(userID int) *ssh.Session {
	c.sessionMux.RLock()
	defer c.sessionMux.RUnlock()
	return c.sessions[userID]
}

// CloseSession 关闭指定用户会话
func (c *client) CloseSession(userID int) error {
	c.sessionMux.Lock()
	defer c.sessionMux.Unlock()

	session, exists := c.sessions[userID]
	if !exists {
		return fmt.Errorf("用户%d的会话不存在", userID)
	}

	if err := session.Close(); err != nil {
		c.logger.Error("关闭SSH会话失败", zap.Int("用户ID", userID), zap.Error(err))
		return fmt.Errorf("关闭SSH会话失败: %w", err)
	}

	delete(c.sessions, userID)
	c.logger.Info("SSH会话已关闭", zap.Int("用户ID", userID))

	return nil
}

// AddPublicKey 添加公钥到远程主机
func (c *client) AddPublicKey(publicKey string) error {
	if publicKey == "" {
		return fmt.Errorf("公钥内容不能为空")
	}

	cleanedKey := strings.TrimSpace(publicKey)
	command := fmt.Sprintf(`mkdir -p -m 700 ~/.ssh && echo '%s' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`, cleanedKey)

	result, err := c.Run(command)
	if err != nil {
		c.logger.Error("添加公钥到远程主机失败",
			zap.Error(err),
			zap.String("命令结果", result))
		return fmt.Errorf("添加公钥失败: %w", err)
	}

	c.logger.Info("成功添加公钥到远程主机")
	return nil
}

// Close 关闭SSH客户端
func (c *client) Close() error {
	var errors []string

	// 关闭所有会话
	c.sessionMux.Lock()
	for userID, session := range c.sessions {
		if session != nil {
			if err := session.Close(); err != nil {
				c.logger.Error("关闭会话失败", zap.Int("用户ID", userID), zap.Error(err))
				errors = append(errors, fmt.Sprintf("关闭用户%d会话失败: %v", userID, err))
			}
		}
	}
	c.sessions = make(map[int]*ssh.Session)
	c.sessionMux.Unlock()

	// 关闭客户端连接
	if c.sshClient != nil {
		if err := c.sshClient.Close(); err != nil {
			c.logger.Error("关闭SSH客户端失败", zap.Error(err))
			errors = append(errors, fmt.Sprintf("关闭SSH客户端失败: %v", err))
		}
		c.sshClient = nil
	}

	if len(errors) > 0 {
		return fmt.Errorf("关闭过程中发生错误: %s", strings.Join(errors, "; "))
	}

	c.logger.Info("SSH客户端已成功关闭")
	return nil
}

// validateConfig 验证配置参数
func (c *client) validateConfig(config *Config) error {
	if config == nil {
		return fmt.Errorf("配置不能为空")
	}
	if config.Host == "" {
		return fmt.Errorf("主机地址不能为空")
	}
	if config.Username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if config.Mode != AuthModePassword && config.Mode != AuthModeKey {
		return fmt.Errorf("认证方式必须是密码或密钥")
	}
	if config.Mode == AuthModePassword && config.Password == "" {
		return fmt.Errorf("密码认证模式下密码不能为空")
	}
	if config.Mode == AuthModeKey && config.Key == "" {
		return fmt.Errorf("密钥认证模式下私钥不能为空")
	}
	return nil
}

// getAuthMethod 获取认证方法
func (c *client) getAuthMethod(config *Config) (ssh.AuthMethod, error) {
	switch config.Mode {
	case AuthModePassword:
		c.logger.Info("使用密码认证模式")
		return ssh.Password(config.Password), nil
	case AuthModeKey:
		signer, err := ssh.ParsePrivateKey([]byte(config.Key))
		if err != nil {
			c.logger.Error("解析SSH私钥失败", zap.Error(err))
			return nil, fmt.Errorf("解析私钥失败: %w", err)
		}
		c.logger.Info("使用私钥认证模式")
		return ssh.PublicKeys(signer), nil
	default:
		return nil, fmt.Errorf("不支持的认证方式: %d", config.Mode)
	}
}
