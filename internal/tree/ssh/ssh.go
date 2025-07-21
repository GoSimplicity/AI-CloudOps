package ssh

import (
	"fmt"
	"strings"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type EcsSSH struct {
	IP         string               // IP地址
	Port       int                  // 端口号
	Username   string               // 用户名
	Mode       string               // 认证方式[password:密码，key:秘钥认证]
	Password   string               // 密码
	Key        string               // 认证私钥
	Client     *ssh.Client          // ssh客户端
	UserID     int                  // 用户ID
	Sessions   map[int]*ssh.Session // ssh会话对象
	Channel    ssh.Channel          // ssh通信管道
	LastResult string               // 最近一次执行命令的结果
	l          *zap.Logger
}

func NewSSH(l *zap.Logger) *EcsSSH {
	return &EcsSSH{
		l:        l,
		Sessions: make(map[int]*ssh.Session), // 初始化Sessions map
	}
}

// Connect SSH连接
func (s *EcsSSH) Connect(ip string, port int, username string, password string, key string, mode string, userID int) error {
	s.IP = ip
	s.Port = port
	s.Username = username
	s.Password = password
	s.Key = key
	s.Mode = mode
	s.UserID = userID

	if s.Sessions == nil {
		s.Sessions = make(map[int]*ssh.Session)
	}

	config := &ssh.ClientConfig{
		User:            s.Username,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
		Timeout:         10 * time.Second,
	}

	if port <= 0 || port > 65535 {
		port = 22
	}

	var auth ssh.AuthMethod
	if mode == "key" {
		signer, err := ssh.ParsePrivateKey([]byte(key))
		if err != nil {
			s.l.Error("SSH key signer failed", zap.Error(err))
			return fmt.Errorf("parse private key failed: %w", err)
		}
		auth = ssh.PublicKeys(signer)
	} else {
		auth = ssh.Password(password)
	}
	config.Auth = []ssh.AuthMethod{auth}

	addr := fmt.Sprintf("%s:%d", ip, port)
	sshClient, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		s.l.Error("SSH dial failed", zap.Error(err))
		return fmt.Errorf("ssh dial failed: %w", err)
	}
	s.Client = sshClient

	session, err := s.Client.NewSession()
	if err != nil {
		s.l.Error("Create SSH session failed", zap.Error(err))
		return fmt.Errorf("create session failed: %w", err)
	}
	s.Sessions[userID] = session // 将会话存储在映射中

	return nil
}

// AddPublicKeyToRemoteHost 将公钥写入目标主机
func (s *EcsSSH) AddPublicKeyToRemoteHost(publicKey string) error {
	command := fmt.Sprintf("mkdir -p -m 700 ~/.ssh && echo %v >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys", strings.TrimSpace(publicKey))
	_, err := s.Run(command)
	if err != nil {
		return fmt.Errorf("添加主机失败: %w", err)
	}
	return nil
}

// Run 执行Shell命令
func (s *EcsSSH) Run(command string) (string, error) {
	if s.Client == nil {
		if err := s.Connect(s.IP, s.Port, s.Username, s.Password, s.Key, s.Mode, s.UserID); err != nil {
			return "", err
		}
	}

	session, err := s.Client.NewSession()
	if err != nil {
		return "", fmt.Errorf("create session failed: %w", err)
	}
	defer session.Close()

	buf, err := session.CombinedOutput(command)
	s.LastResult = string(buf)
	if err != nil {
		return s.LastResult, fmt.Errorf("execute command failed: %w", err)
	}
	return s.LastResult, nil
}

/*
*************************** Web2SSH ***************************
 */

type MyReader struct {
	// 监听websocket
	ws *websocket.Conn
}

type MyWriter struct {
	ws *websocket.Conn
}

// MyReader 从WebSocket读取数据
func (r MyReader) Read(p []byte) (n int, err error) {
	// 从客户端接收命令
	_, message, err := r.ws.ReadMessage()
	if err != nil {
		r.ws.WriteMessage(websocket.CloseMessage, []byte{})
		return 0, err
	}

	// 将命令转换为字节并添加换行符
	cmdStr := string(message)
	if !strings.HasSuffix(cmdStr, "\n") {
		cmdStr = cmdStr + "\n"
	}

	// 复制命令到缓冲区
	cmdBytes := []byte(cmdStr)
	n = copy(p, cmdBytes)

	return n, nil
}

// MyWriter 向WebSocket写入数据
func (w MyWriter) Write(p []byte) (n int, err error) {
	if len(p) == 0 {
		return 0, nil
	}

	// 发送数据
	err = w.ws.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, fmt.Errorf("write websocket message failed: %w", err)
	}

	return len(p), nil
}

// Web2SSH 实现Web SSH功能
func (s *EcsSSH) Web2SSH(ws *websocket.Conn) {
	defer func() {
		ws.Close()
		if s.Sessions != nil {
			for _, session := range s.Sessions {
				session.Close()
			}
		}
		if s.Client != nil {
			s.Client.Close()
		}
	}()

	if s.Sessions == nil || s.Sessions[s.UserID] == nil {
		s.l.Error("SSH session not found")
		return
	}

	// 配置和创建一个伪终端
	modes := ssh.TerminalModes{
		ssh.ECHO:          0,     // 关闭回显
		ssh.TTY_OP_ISPEED: 14400, // 设置传输速率
		ssh.TTY_OP_OSPEED: 14400, // 设置传输速率
	}

	// 激活终端
	if err := s.Sessions[s.UserID].RequestPty("xterm", 25, 80, modes); err != nil {
		s.l.Error("Request pty failed", zap.Error(err))
		return
	}

	// 设置输入输出
	s.Sessions[s.UserID].Stdin = &MyReader{ws}
	s.Sessions[s.UserID].Stdout = &MyWriter{ws}
	s.Sessions[s.UserID].Stderr = &MyWriter{ws}

	// 激活shell
	if err := s.Sessions[s.UserID].Shell(); err != nil {
		s.l.Error("Start shell failed", zap.Error(err))
		return
	}

	s.l.Info("WebSocket connected to SSH session")

	// 等待SSH会话结束
	if err := s.Sessions[s.UserID].Wait(); err != nil {
		s.l.Error("SSH session ended with error", zap.Error(err))
	}

	s.l.Info("WebSocket disconnected from SSH session")
}
