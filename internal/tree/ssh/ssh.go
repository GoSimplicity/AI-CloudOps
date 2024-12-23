package ssh

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
	"golang.org/x/crypto/ssh"
)

type EcsSSH struct {
	IP       string      // IP地址
	Port     int         // 端口号
	Username string      // 用户名
	Mode     string      // 认证方式[password:密码，key:秘钥认证]
	Password string      // 密码
	Key      string      // 认证私钥
	Client   *ssh.Client // ssh客户端
	// console
	Session    *ssh.Session // ssh会话对象
	Channel    ssh.Channel  // ssh通信管道
	LastResult string       // 最近一次执行命令的结果
	l          *zap.Logger
}

func NewSSH(l *zap.Logger) *EcsSSH {
	return &EcsSSH{
		l: l,
	}
}

// Connect SSH连接
func (s *EcsSSH) Connect(ip string, port int, username string, password string, key string, mode string) error {
	s.IP = ip
	s.Port = port
	s.Username = username
	s.Password = password
	s.Key = key
	s.Mode = mode

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
	s.Session = session

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
		if err := s.Connect(s.IP, s.Port, s.Username, s.Password, s.Key, s.Mode); err != nil {
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
	ws *websocket.Conn
}

type MyWriter struct {
	ws *websocket.Conn
}

// MyReader 从WebSocket读取数据
func (r MyReader) Read(p []byte) (n int, err error) {
	messageType, data, err := r.ws.ReadMessage()
	if err != nil {
		return 0, err
	}

	if messageType != websocket.TextMessage {
		return 0, nil
	}

	copy(p, data)
	return len(data), nil
}

// MyWriter 向WebSocket写入数据
func (w MyWriter) Write(p []byte) (n int, err error) {
	err = w.ws.WriteMessage(websocket.TextMessage, p)
	if err != nil {
		return 0, err
	}
	return len(p), nil
}

var (
	ansiRegex   = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	promptRegex = regexp.MustCompile(`\[?[0-9;:]*[0-9]+[@][a-zA-Z0-9]+:.*?#\s*`)
	specialSeqs = []string{
		"[?2004l",
		"[?2004h",
		"]0;",
		"[0m",
		"[01;34m",
	}
)

// cleanOutput 清理终端输出中的特殊字符
func cleanOutput(input string) string {
	result := ansiRegex.ReplaceAllString(input, "")
	result = promptRegex.ReplaceAllString(result, "")

	for _, seq := range specialSeqs {
		result = strings.ReplaceAll(result, seq, "")
	}

	lines := strings.Split(result, "\n")
	var cleanLines []string
	for _, line := range lines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		}
	}

	return strings.Join(cleanLines, "\n")
}

// handleIO 处理输入输出流
func handleIO(reader *bufio.Reader, ws *websocket.Conn, l *zap.Logger) {
	var buffer strings.Builder
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if err != io.EOF {
				l.Error("Failed to read", zap.Error(err))
			}
			return
		}
		buffer.WriteString(line)

		if strings.Contains(line, "\n") {
			if cleanedOutput := cleanOutput(buffer.String()); cleanedOutput != "" {
				if err := ws.WriteMessage(websocket.TextMessage, []byte(cleanedOutput+"\n")); err != nil {
					l.Error("Failed to write message", zap.Error(err))
					return
				}
			}
			buffer.Reset()
		}
	}
}

// Web2SSH 实现Web SSH功能
func (s *EcsSSH) Web2SSH(ws *websocket.Conn) {
	modes := ssh.TerminalModes{
		ssh.ECHO:          1,
		ssh.TTY_OP_ISPEED: 14400,
		ssh.TTY_OP_OSPEED: 14400,
		ssh.ICANON:        1,
		ssh.ICRNL:         1,
		ssh.ISIG:          1,
	}

	if err := s.Session.RequestPty("xterm-256color", 50, 200, modes); err != nil {
		s.l.Error("Request pty failed", zap.Error(err))
		return
	}

	stdin, err := s.Session.StdinPipe()
	if err != nil {
		s.l.Error("Failed to create stdin pipe", zap.Error(err))
		return
	}
	stdout, err := s.Session.StdoutPipe()
	if err != nil {
		s.l.Error("Failed to create stdout pipe", zap.Error(err))
		return
	}
	stderr, err := s.Session.StderrPipe()
	if err != nil {
		s.l.Error("Failed to create stderr pipe", zap.Error(err))
		return
	}

	if err := s.Session.Shell(); err != nil {
		s.l.Error("Failed to start shell", zap.Error(err))
		return
	}

	var wg sync.WaitGroup
	wg.Add(3)

	// 处理输入
	go func() {
		defer wg.Done()
		defer stdin.Close()
		for {
			messageType, p, err := ws.ReadMessage()
			if err != nil {
				s.l.Error("Failed to read message", zap.Error(err))
				return
			}
			if messageType == websocket.TextMessage {
				if !bytes.HasSuffix(p, []byte("\n")) {
					p = append(p, '\n')
				}
				if _, err = stdin.Write(p); err != nil {
					s.l.Error("Failed to write to stdin", zap.Error(err))
					return
				}
			}
		}
	}()

	// 处理标准输出
	go func() {
		defer wg.Done()
		handleIO(bufio.NewReader(stdout), ws, s.l)
	}()

	// 处理错误输出
	go func() {
		defer wg.Done()
		handleIO(bufio.NewReader(stderr), ws, s.l)
	}()

	wg.Wait()

	if err := s.Session.Wait(); err != nil && err != io.EOF {
		s.l.Error("SSH session ended with error", zap.Error(err))
	}
}
