package ssh

import (
	"fmt"
	"os"

	"github.com/GoSimplicity/AI-CloudOps/pkg/envmode"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// HostKeyCallback uses a provisioned OpenSSH trust store, never trust-on-first-use.
// Without SSH_KNOWN_HOSTS, host key checking is skipped only when ENV explicitly
// resolves to development (see pkg/envmode).
func HostKeyCallback() (ssh.HostKeyCallback, error) {
	if file := os.Getenv("SSH_KNOWN_HOSTS"); file != "" {
		return knownhosts.New(file)
	}
	// 失败即关闭：只有 ENV 明确为开发环境时才允许跳过主机密钥校验
	if envmode.IsProduction() {
		return nil, fmt.Errorf("生产（或 ENV 未设置/无法识别）时 SSH 连接必须配置 SSH_KNOWN_HOSTS")
	}
	return ssh.InsecureIgnoreHostKey(), nil
}
