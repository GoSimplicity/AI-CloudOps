package ssh

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"sync"

	"golang.org/x/crypto/ssh"
)

const MaxCommandOutputBytes = (64 << 10) - 128
const outputTruncated = "[输出已截断]"

type commandOutput struct {
	mu        sync.Mutex
	buffer    bytes.Buffer
	truncated bool
}

func (b *commandOutput) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	count := len(p)
	remaining := MaxCommandOutputBytes - b.buffer.Len()
	if len(p) > remaining {
		p = p[:remaining]
		b.truncated = true
	}
	_, _ = b.buffer.Write(p)
	return count, nil
}
func (b *commandOutput) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	output := b.buffer.String()
	if b.truncated {
		return strings.ToValidUTF8(output, "") + outputTruncated
	}
	return output
}

// RunContext 在取消时关闭该SSH传输，覆盖会话建立、exec请求与输出等待
// SSH协议关闭会话不保证远端脱离会话的后台进程终止
func RunContext(ctx context.Context, client *ssh.Client, command string) (string, error) {
	if ctx == nil {
		return "", fmt.Errorf("SSH执行上下文不能为空")
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	if client == nil {
		return "", fmt.Errorf("SSH客户端未连接")
	}
	if command == "" {
		return "", fmt.Errorf("命令不能为空")
	}
	stop := context.AfterFunc(ctx, func() { _ = client.Close() })
	defer stop()
	session, err := client.NewSession()
	if err != nil {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		return "", fmt.Errorf("创建SSH会话失败: %w", err)
	}
	defer session.Close()
	output := &commandOutput{}
	session.Stdout = output
	session.Stderr = output
	err = session.Run(command)
	if ctx.Err() != nil {
		return output.String(), ctx.Err()
	}
	if err != nil {
		return output.String(), fmt.Errorf("执行SSH命令失败: %w", err)
	}
	return output.String(), nil
}
