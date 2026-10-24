package ssh

import (
	"fmt"
	"os"

	"github.com/GoSimplicity/AI-CloudOps/pkg/runenv"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// HostKeyCallback uses a provisioned OpenSSH trust store, never trust-on-first-use.
func HostKeyCallback() (ssh.HostKeyCallback, error) {
	if file := os.Getenv("SSH_KNOWN_HOSTS"); file != "" {
		return knownhosts.New(file)
	}
	if runenv.IsProduction() {
		return nil, fmt.Errorf("生产 SSH 连接必须配置 SSH_KNOWN_HOSTS")
	}
	return ssh.InsecureIgnoreHostKey(), nil
}
