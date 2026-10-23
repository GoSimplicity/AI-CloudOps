package ssh

import (
	"context"
	"fmt"
	"net"
	"time"

	"golang.org/x/crypto/ssh"
)

// DialWithTimeout 保留既有入口，期限覆盖TCP拨号与完整SSH握手
func DialWithTimeout(network, address string, config *ssh.ClientConfig) (*ssh.Client, error) {
	return DialContext(context.Background(), network, address, config)
}

func DialContext(ctx context.Context, network, address string, config *ssh.ClientConfig) (*ssh.Client, error) {
	if ctx == nil || config == nil {
		return nil, fmt.Errorf("SSH上下文和配置不能为空")
	}
	timeout := config.Timeout
	if timeout <= 0 {
		timeout = 10 * time.Second
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	conn, err := (&net.Dialer{}).DialContext(ctx, network, address)
	if err != nil {
		return nil, err
	}
	deadline, _ := ctx.Deadline()
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return nil, err
	}
	stop := context.AfterFunc(ctx, func() { _ = conn.Close() })
	defer stop()
	connection, channels, requests, err := ssh.NewClientConn(conn, address, config)
	if err != nil {
		_ = conn.Close()
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, err
	}
	if !stop() || ctx.Err() != nil {
		_ = connection.Close()
		return nil, ctx.Err()
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		_ = connection.Close()
		return nil, err
	}
	return ssh.NewClient(connection, channels, requests), nil
}
