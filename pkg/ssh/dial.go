package ssh

import (
	"context"
	"errors"
	"fmt"
	"net"
	"os"
	"strings"
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
		return nil, normalizeDialError(ctx, err)
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
		return nil, normalizeDialError(ctx, err)
	}
	if !stop() || ctx.Err() != nil {
		_ = connection.Close()
		if err := normalizeDialError(ctx, ctx.Err()); err != nil {
			return nil, err
		}
		return nil, context.DeadlineExceeded
	}
	if err := conn.SetDeadline(time.Time{}); err != nil {
		_ = connection.Close()
		return nil, err
	}
	return ssh.NewClient(connection, channels, requests), nil
}

// normalizeDialError 把连接期限触发的超时统一成 context.DeadlineExceeded。
func normalizeDialError(ctx context.Context, err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(ctx.Err(), context.Canceled) || errors.Is(err, context.Canceled) {
		if errors.Is(ctx.Err(), context.Canceled) {
			return ctx.Err()
		}
		return err
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(err, context.DeadlineExceeded) || errors.Is(err, os.ErrDeadlineExceeded) || netTimeout(err) {
		return fmt.Errorf("%w: %v", context.DeadlineExceeded, err)
	}
	return err
}

func netTimeout(err error) bool {
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return true
	}
	return strings.Contains(err.Error(), "i/o timeout")
}
