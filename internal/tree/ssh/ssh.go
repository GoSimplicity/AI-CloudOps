package ssh

import (
	"context"
	"fmt"
	sshclient "github.com/GoSimplicity/AI-CloudOps/pkg/ssh"
	"net"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type EcsSSH interface {
	Connect(ip string, port int, username string, password string, key string, mode int8, userID int) error
	ConnectContext(context.Context, string, int, string, string, string, int8, int) error
	AddPublicKeyToRemoteHost(publicKey string) error
	Run(command string) (string, error)
	RunContext(context.Context, string) (string, error)
	Close() error
	Web2SSH(ws *websocket.Conn)
	Web2SSHContext(context.Context, *websocket.Conn) error
	Web2SSHWithOptionsContext(context.Context, *websocket.Conn, sshclient.TerminalOptions) error
}

type ecsSSH struct {
	IP         string               // 服务器IP地址
	Port       int                  // SSH端口号，默认22
	Username   string               // SSH用户名
	Mode       int8                 // 认证方式：1:密码,2:密钥
	Password   string               // 密码
	Key        string               // SSH私钥内容
	Client     *ssh.Client          // SSH客户端连接
	UserID     int                  // 用户ID，用于区分不同用户的会话
	Sessions   map[int]*ssh.Session // 用户会话映射表，key为UserID，value为对应的SSH会话
	sessionMu  sync.RWMutex         // 保护Sessions的读写锁
	LastResult string               // 最近一次执行命令的结果
	logger     *zap.Logger          // 日志记录器
}

// NewSSH 创建新的SSH连接管理器
func NewSSH(logger *zap.Logger) EcsSSH {
	if logger == nil {
		logger = zap.NewNop()
	}
	return &ecsSSH{
		logger:   logger,
		Sessions: make(map[int]*ssh.Session), // 初始化会话映射表
	}
}

// Connect 建立SSH连接
func (s *ecsSSH) Connect(ip string, port int, username string, password string, key string, mode int8, userID int) error {
	return s.ConnectContext(context.Background(), ip, port, username, password, key, mode, userID)
}
func (s *ecsSSH) ConnectContext(ctx context.Context, ip string, port int, username string, password string, key string, mode int8, userID int) error {
	if ip == "" {
		return fmt.Errorf("IP地址不能为空")
	}
	if username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	if mode != 1 && mode != 2 {
		return fmt.Errorf("认证方式必须是 'password' 或 'key'")
	}
	if mode == 1 && password == "" {
		return fmt.Errorf("密码认证模式下密码不能为空")
	}
	if mode == 2 && key == "" {
		return fmt.Errorf("私钥认证模式下私钥不能为空")
	}

	s.IP = ip
	s.Port = port
	s.Username = username
	s.Password = password
	s.Key = key
	s.Mode = mode
	s.UserID = userID

	// 初始化Sessions映射
	if s.Sessions == nil {
		s.sessionMu.Lock()
		if s.Sessions == nil {
			s.Sessions = make(map[int]*ssh.Session)
		}
		s.sessionMu.Unlock()
	}

	// 验证端口范围，默认使用22端口
	if port <= 0 || port > 65535 {
		s.logger.Warn("端口号无效，使用默认端口22", zap.Int("原端口", port))
		s.Port = 22
		port = 22
	}

	hostKeyCallback, err := sshclient.HostKeyCallback()
	if err != nil {
		return err
	}

	// 配置SSH客户端
	config := &ssh.ClientConfig{
		User:            s.Username,
		HostKeyCallback: hostKeyCallback,
		Timeout:         10 * time.Second,
	}

	var auth ssh.AuthMethod
	if mode == 2 {
		// 解析私钥
		signer, err := ssh.ParsePrivateKey([]byte(key))
		if err != nil {
			s.logger.Error("解析SSH私钥失败", zap.Error(err))
			return fmt.Errorf("解析私钥失败: %w", err)
		}
		auth = ssh.PublicKeys(signer)
		s.logger.Info("使用私钥认证模式")
	} else {
		// 使用密码认证
		auth = ssh.Password(password)
		s.logger.Info("使用密码认证模式")
	}
	config.Auth = []ssh.AuthMethod{auth}

	// 建立SSH连接
	addr := net.JoinHostPort(ip, strconv.Itoa(port))
	sshClient, err := sshclient.DialContext(ctx, "tcp", addr, config)
	if err != nil {
		s.logger.Error("SSH连接失败",
			zap.String("地址", addr),
			zap.String("用户名", username),
			zap.Error(err))
		return fmt.Errorf("SSH连接失败: %w", err)
	}
	s.Client = sshClient
	s.logger.Info("SSH连接成功", zap.String("地址", addr))

	// 创建新的SSH会话也受请求取消约束
	stop := context.AfterFunc(ctx, func() { _ = sshClient.Close() })
	defer stop()
	session, err := s.Client.NewSession()
	if ctx.Err() != nil {
		_ = sshClient.Close()
		s.Client = nil
		return ctx.Err()
	}
	if err != nil {
		s.logger.Error("创建SSH会话失败", zap.Error(err))
		err := s.Client.Close()
		if err != nil {
			s.logger.Error("关闭SSH客户端失败", zap.Error(err))
			return err
		}
		s.Client = nil
		return fmt.Errorf("创建SSH会话失败: %w", err)
	}

	// 线程安全地存储会话
	s.sessionMu.Lock()
	s.Sessions[userID] = session
	s.sessionMu.Unlock()

	s.logger.Info("SSH会话创建成功", zap.Int("用户ID", userID))
	return nil
}

// AddPublicKeyToRemoteHost 将公钥添加到远程主机的authorized_keys文件中
func (s *ecsSSH) AddPublicKeyToRemoteHost(publicKey string) error {
	command, err := sshclient.AuthorizedKeyCommand(publicKey)
	if err != nil {
		return err
	}

	_, err = s.Run(command)
	if err != nil {
		s.logger.Error("添加公钥到远程主机失败",
			zap.Error(err))
		return fmt.Errorf("添加公钥失败: %w", err)
	}

	s.logger.Info("成功添加公钥到远程主机")
	return nil
}

// Run 执行单个Shell命令
func (s *ecsSSH) Run(command string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()
	return s.RunContext(ctx, command)
}
func (s *ecsSSH) RunContext(ctx context.Context, command string) (string, error) {
	if s.Client == nil {
		if err := s.ConnectContext(ctx, s.IP, s.Port, s.Username, s.Password, s.Key, s.Mode, s.UserID); err != nil {
			return "", err
		}
	}
	output, err := sshclient.RunContext(ctx, s.Client, command)
	s.sessionMu.Lock()
	s.LastResult = output
	s.sessionMu.Unlock()
	return output, err
}

// Close 关闭SSH连接和所有会话
func (s *ecsSSH) Close() error {
	var errors []string

	// 关闭所有会话
	s.sessionMu.Lock()
	for userID, session := range s.Sessions {
		if session != nil {
			if err := session.Close(); err != nil {
				s.logger.Error("关闭会话失败", zap.Int("用户ID", userID), zap.Error(err))
				errors = append(errors, fmt.Sprintf("关闭用户%d会话失败: %v", userID, err))
			}
		}
	}
	// 清空会话映射
	s.Sessions = make(map[int]*ssh.Session)
	s.sessionMu.Unlock()

	// 关闭SSH客户端连接
	if s.Client != nil {
		if err := s.Client.Close(); err != nil {
			s.logger.Error("关闭SSH客户端失败", zap.Error(err))
			errors = append(errors, fmt.Sprintf("关闭SSH客户端失败: %v", err))
		}
		s.Client = nil
	}

	if len(errors) > 0 {
		return fmt.Errorf("关闭过程中发生错误: %s", strings.Join(errors, "; "))
	}

	s.logger.Info("SSH连接已成功关闭")
	return nil
}

/*
*************************** Web终端SSH实现 ***************************
 */

// Web2SSH 保留已发布的逐帧命令格式，新增上下文入口供会话所有者取消
func (s *ecsSSH) Web2SSH(ws *websocket.Conn) {
	if s.Web2SSHContext(context.Background(), ws) != nil {
		s.logger.Info("SSH终端会话结束")
	}
}
func (s *ecsSSH) Web2SSHContext(parent context.Context, ws *websocket.Conn) error {
	return s.Web2SSHWithOptionsContext(parent, ws, sshclient.TerminalOptions{})
}
func (s *ecsSSH) Web2SSHWithOptionsContext(parent context.Context, ws *websocket.Conn, options sshclient.TerminalOptions) error {
	normalized, err := options.Normalize()
	if err != nil {
		return err
	}
	options = normalized
	if parent == nil || ws == nil {
		return fmt.Errorf("终端上下文与WebSocket不能为空")
	}
	if err := parent.Err(); err != nil {
		return err
	}
	s.sessionMu.RLock()
	session := s.Sessions[s.UserID]
	s.sessionMu.RUnlock()
	if session == nil {
		return fmt.Errorf("SSH会话未建立")
	}
	transport := s.Client
	userID := s.UserID
	ctx, cancel := context.WithCancel(parent)
	defer cancel()
	interrupted := make(chan struct{})
	stop := context.AfterFunc(ctx, func() {
		defer close(interrupted)
		_ = ws.Close()
		if transport != nil {
			_ = transport.Close()
		}
		_ = session.Close()
	})
	defer func() {
		if !stop() {
			<-interrupted
		}
	}()
	defer func() {
		_ = ws.Close()
		_ = session.Close()
		if transport != nil {
			_ = transport.Close()
		}
		s.sessionMu.Lock()
		if s.Sessions[userID] == session {
			delete(s.Sessions, userID)
		}
		s.sessionMu.Unlock()
	}()
	var setupExpired atomic.Bool
	timer := time.AfterFunc(10*time.Second, func() { setupExpired.Store(true); cancel() })
	defer timer.Stop()
	modes := ssh.TerminalModes{ssh.ECHO: 0, ssh.TTY_OP_ISPEED: 14400, ssh.TTY_OP_OSPEED: 14400}
	if options.Mode == sshclient.TerminalModePTY {
		modes[ssh.ECHO] = 1
	}
	if err := session.RequestPty("xterm", options.Rows, options.Cols, modes); err != nil {
		if parent.Err() != nil {
			return parent.Err()
		}
		if setupExpired.Load() {
			return context.DeadlineExceeded
		}
		return fmt.Errorf("请求SSH伪终端失败")
	}
	writer := sshclient.NewTerminalWriterWithOptions(ws, s.logger, options)
	session.Stdin = sshclient.NewTerminalReaderWithOptions(ws, s.logger, cancel, options, session.WindowChange)
	session.Stdout = writer
	session.Stderr = writer
	if err := session.Shell(); err != nil {
		if parent.Err() != nil {
			return parent.Err()
		}
		if setupExpired.Load() {
			return context.DeadlineExceeded
		}
		return fmt.Errorf("启动SSH终端失败")
	}
	timer.Stop()
	_ = session.Wait()
	if parent.Err() != nil {
		return parent.Err()
	}
	return nil
}
