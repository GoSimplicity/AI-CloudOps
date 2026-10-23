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
// resolves to development/test (see pkg/envmode).
func HostKeyCallback() (ssh.HostKeyCallback, error) {
	if file := os.Getenv("SSH_KNOWN_HOSTS"); file != "" {
		return knownhosts.New(file)
	}
	// 与 config.IsProduction 复用 pkg/envmode 的同一判定：生产别名、大小写/空白变体，
	// 以及 ENV 未设置或无法识别时都按生产处理（fail closed），只有明确的开发/测试环境才允许跳过校验。
	if envmode.IsProduction() {
		return nil, fmt.Errorf("生产 SSH 连接必须配置 SSH_KNOWN_HOSTS（ENV 未设置或无法识别时同样按生产处理）")
	}
	return ssh.InsecureIgnoreHostKey(), nil
}
