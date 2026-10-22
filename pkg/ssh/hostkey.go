package ssh

import (
	"fmt"
	"os"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

// HostKeyCallback uses a provisioned OpenSSH trust store, never trust-on-first-use.
func HostKeyCallback() (ssh.HostKeyCallback, error) {
	if file := os.Getenv("SSH_KNOWN_HOSTS"); file != "" {
		return knownhosts.New(file)
	}
	if os.Getenv("ENV") == "production" {
		return nil, fmt.Errorf("生产 SSH 连接必须配置 SSH_KNOWN_HOSTS")
	}
	return ssh.InsecureIgnoreHostKey(), nil
}
