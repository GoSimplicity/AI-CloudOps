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

type EcsSSH interface {
	Connect(ip string, port int, username string, password string, key string, mode int8, userID int) error
	AddPublicKeyToRemoteHost(publicKey string) error
	Run(command string) (string, error)
	Close() error
	Web2SSH(ws *websocket.Conn)
}

type ecsSSH struct {
	IP         string               // 服务器IP地址
	Port       int                  // SSH端口号，默认22
	Username   string               // SSH用户名
	Mode       int8                 // 认证方式：1:密码,2:密钥
	Password   string               // 密码（当Mode为password时使用）
	Key        string               // SSH私钥内容（当Mode为key时使用）
	Client     *ssh.Client          // SSH客户端连接
	UserID     int                  // 用户ID，用于区分不同用户的会话
	Sessions   map[int]*ssh.Session // 用户会话映射表，key为UserID，value为对应的SSH会话
	sessionMu  sync.RWMutex         // 保护Sessions的读写锁
	LastResult string               // 最近一次执行命令的结果
	logger     *zap.Logger          // 日志记录器
}

// NewSSH 创建新的SSH连接管理器
func NewSSH(logger *zap.Logger) EcsSSH {
	return &ecsSSH{
		logger:   logger,
		Sessions: make(map[int]*ssh.Session), // 初始化会话映射表
	}
}

// Connect 建立SSH连接
func (s *ecsSSH) Connect(ip string, port int, username string, password string, key string, mode int8, userID int) error {
	// 参数验证
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

	// 设置连接参数
	s.IP = ip
	s.Port = port
	s.Username = username
	s.Password = password
	s.Key = key
	s.Mode = mode
	s.UserID = userID

	// 初始化Sessions映射（双重检查锁定模式）
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

	// 配置SSH客户端
	config := &ssh.ClientConfig{
		User:            s.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // 注意：生产环境应使用更安全的主机密钥验证
		Timeout:         10 * time.Second,            // 连接超时时间
	}

	// 根据认证方式配置认证方法
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
	addr := fmt.Sprintf("%s:%d", ip, port)
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		s.logger.Error("SSH连接失败",
			zap.String("地址", addr),
			zap.String("用户名", username),
			zap.Error(err))
		return fmt.Errorf("SSH连接失败: %w", err)
	}
	s.Client = sshClient
	s.logger.Info("SSH连接成功", zap.String("地址", addr))

	// 创建新的SSH会话
	session, err := s.Client.NewSession()
	if err != nil {
		s.logger.Error("创建SSH会话失败", zap.Error(err))
		// 连接失败时清理资源
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
	if publicKey == "" {
		return fmt.Errorf("公钥内容不能为空")
	}

	// 清理公钥内容，移除多余的空白字符
	cleanedKey := strings.TrimSpace(publicKey)

	// 构建安全的命令：创建.ssh目录、添加公钥、设置正确的权限
	command := fmt.Sprintf(`mkdir -p -m 700 ~/.ssh && echo '%s' >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys`, cleanedKey)

	result, err := s.Run(command)
	if err != nil {
		s.logger.Error("添加公钥到远程主机失败",
			zap.Error(err),
			zap.String("命令结果", result))
		return fmt.Errorf("添加公钥失败: %w", err)
	}

	s.logger.Info("成功添加公钥到远程主机")
	return nil
}

// Run 执行单个Shell命令
func (s *ecsSSH) Run(command string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("命令不能为空")
	}

	// 如果客户端未连接，尝试重新连接
	if s.Client == nil {
		s.logger.Warn("SSH客户端未连接，尝试重新连接")
		if err := s.Connect(s.IP, s.Port, s.Username, s.Password, s.Key, s.Mode, s.UserID); err != nil {
			return "", fmt.Errorf("重新连接失败: %w", err)
		}
	}

	// 为每个命令创建新的会话（避免会话状态污染）
	session, err := s.Client.NewSession()
	if err != nil {
		s.logger.Error("创建命令执行会话失败", zap.Error(err))
		return "", fmt.Errorf("创建会话失败: %w", err)
	}

	// 确保会话在函数结束时关闭
	defer func() {
		if closeErr := session.Close(); closeErr != nil {
			s.logger.Error("关闭会话失败", zap.Error(closeErr))
		}
	}()

	// 执行命令并获取输出（合并stdout和stderr）
	s.logger.Debug("执行命令", zap.String("命令", command))
	buf, err := session.CombinedOutput(command)
	s.LastResult = string(buf)

	if err != nil {
		s.logger.Error("命令执行失败",
			zap.String("命令", command),
			zap.String("输出", s.LastResult),
			zap.Error(err))
		return s.LastResult, fmt.Errorf("执行命令失败: %w", err)
	}

	s.logger.Debug("命令执行成功",
		zap.String("命令", command),
		zap.String("输出长度", fmt.Sprintf("%d字符", len(s.LastResult))))
	return s.LastResult, nil
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

// MyReader 实现从WebSocket读取数据的io.Reader接口
type MyReader struct {
	ws *websocket.Conn // WebSocket连接
}

// MyWriter 实现向WebSocket写入数据的io.Writer接口
type MyWriter struct {
	ws *websocket.Conn // WebSocket连接
}

// Read 从WebSocket读取用户输入的命令数据
func (r MyReader) Read(p []byte) (n int, err error) {
	// 从WebSocket客户端接收消息
	_, message, err := r.ws.ReadMessage()
	if err != nil {
		// 读取失败时发送关闭消息
		err := r.ws.WriteMessage(websocket.CloseMessage, []byte{})
		if err != nil {
			return 0, err
		}
		return 0, fmt.Errorf("从WebSocket读取消息失败: %w", err)
	}

	// 将接收到的消息转换为命令字符串
	cmdStr := string(message)

	// 确保命令以换行符结尾（终端需要）
	if !strings.HasSuffix(cmdStr, "\n") {
		cmdStr = cmdStr + "\n"
	}

	// 将命令复制到读取缓冲区
	cmdBytes := []byte(cmdStr)
	n = copy(p, cmdBytes)

	return n, nil
}

// Write 向WebSocket发送终端输出数据
func (w MyWriter) Write(p []byte) (n int, err error) {
	// 空数据直接返回
	if len(p) == 0 {
		return 0, nil
	}

	// 向WebSocket客户端发送文本消息
	err = w.ws.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, fmt.Errorf("向WebSocket写入消息失败: %w", err)
	}

	return len(p), nil
}

// Web2SSH 实现基于WebSocket的Web终端SSH功能
func (s *ecsSSH) Web2SSH(ws *websocket.Conn) {
	// 确保在函数结束时清理所有资源
	defer func() {
		// 关闭WebSocket连接
		if err := ws.Close(); err != nil {
			s.logger.Error("关闭WebSocket连接失败", zap.Error(err))
		}

		// 关闭当前用户的SSH会话
		s.sessionMu.RLock()
		session := s.Sessions[s.UserID]
		s.sessionMu.RUnlock()

		if session != nil {
			if err := session.Close(); err != nil {
				s.logger.Error("关闭SSH会话失败", zap.Int("用户ID", s.UserID), zap.Error(err))
			}
			// 从映射中移除已关闭的会话
			s.sessionMu.Lock()
			delete(s.Sessions, s.UserID)
			s.sessionMu.Unlock()
		}

		// 关闭SSH客户端连接
		if s.Client != nil {
			if err := s.Client.Close(); err != nil {
				s.logger.Error("关闭SSH客户端失败", zap.Error(err))
			}
		}

		s.logger.Info("Web终端SSH会话已清理", zap.Int("用户ID", s.UserID))
	}()

	// 检查SSH会话是否存在
	s.sessionMu.RLock()
	session := s.Sessions[s.UserID]
	s.sessionMu.RUnlock()

	if session == nil {
		s.logger.Error("未找到SSH会话", zap.Int("用户ID", s.UserID))
		// 向客户端发送错误消息
		err := ws.WriteMessage(websocket.TextMessage, []byte("错误: SSH会话未建立\r\n"))
		if err != nil {
			s.logger.Error("向WebSocket发送错误消息失败", zap.Error(err))
			return
		}
		return
	}

	// 配置伪终端模式
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // 禁用回显（避免重复显示用户输入）
		ssh.TTY_OP_ISPEED: 14400, // 输入波特率
		ssh.TTY_OP_OSPEED: 14400, // 输出波特率
	}

	// 请求伪终端（PTY）
	// xterm: 终端类型
	// 25: 终端行数
	// 80: 终端列数
	if err := session.RequestPty("xterm", 25, 80, modes); err != nil {
		s.logger.Error("请求伪终端失败", zap.Error(err))
		err := ws.WriteMessage(websocket.TextMessage, []byte("错误: 无法创建终端\r\n"))
		if err != nil {
			s.logger.Error("向WebSocket发送错误消息失败", zap.Error(err))
			return
		}
		return
	}

	// 将WebSocket连接设置为SSH会话的输入输出流
	session.Stdin = &MyReader{ws}  // 标准输入：从WebSocket读取用户命令
	session.Stdout = &MyWriter{ws} // 标准输出：向WebSocket发送命令结果
	session.Stderr = &MyWriter{ws} // 错误输出：向WebSocket发送错误信息

	// 启动交互式Shell
	if err := session.Shell(); err != nil {
		s.logger.Error("启动Shell失败", zap.Error(err))
		err := ws.WriteMessage(websocket.TextMessage, []byte("错误: 无法启动Shell\r\n"))
		if err != nil {
			s.logger.Error("向WebSocket发送错误消息失败", zap.Error(err))
			return
		}
		return
	}

	s.logger.Info("Web终端SSH会话已启动", zap.Int("用户ID", s.UserID))

	// 等待SSH会话结束（阻塞直到用户退出或连接断开）
	if err := session.Wait(); err != nil {
		s.logger.Info("SSH会话结束", zap.Int("用户ID", s.UserID), zap.Error(err))
	} else {
		s.logger.Info("SSH会话正常结束", zap.Int("用户ID", s.UserID))
	}
}
