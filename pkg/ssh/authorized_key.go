package ssh

import (
	"fmt"
	"strings"

	"golang.org/x/crypto/ssh"
)

// AuthorizedKeyCommand 验证单个公钥并将整行作为 shell 字面量安全写入
func AuthorizedKeyCommand(publicKey string) (string, error) {
	key := strings.TrimSpace(publicKey)
	if key == "" || len(key) > 16*1024 || strings.ContainsAny(key, "\r\n") {
		return "", fmt.Errorf("必须提供单行有效 SSH 公钥")
	}
	if _, _, _, rest, err := ssh.ParseAuthorizedKey([]byte(key)); err != nil || len(strings.TrimSpace(string(rest))) != 0 {
		return "", fmt.Errorf("SSH 公钥格式无效")
	}
	// 保留合法 authorized_keys 选项与注释，单引号也只作为数据写入
	quoted := "'" + strings.ReplaceAll(key, "'", "'\"'\"'") + "'"
	return "umask 077; mkdir -p ~/.ssh && chmod 700 ~/.ssh && printf '%s\\n' " + quoted + " >> ~/.ssh/authorized_keys && chmod 600 ~/.ssh/authorized_keys", nil
}
