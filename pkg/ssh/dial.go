package ssh

import (
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// DialWithTimeout 将连接期限覆盖 TCP 拨号与 SSH 握手，成功后清除会话期限
func DialWithTimeout(network, address string, config *ssh.ClientConfig) (*ssh.Client, error) {
	if config == nil {
		return nil, fmt.Errorf("SSH 配置不能为空")
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	deadline := time.Now().Add(timeout)
	conn, err := net.DialTimeout(network, address, timeout)
	if err != nil {
		return nil, err
	}
	if err := conn.SetDeadline(deadline); err != nil {
		conn.Close()
		return nil, err
	}
	connection, channels, requests, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		conn.Close()
		return nil, err
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		connection.Close()
		return nil, err
	}
	return ssh.NewClient(connection, channels, requests), nil
}
