package utils

import (
	"errors"
	"path/filepath"
	"strings"
)

// allowedPodExecShells 是 Pod 终端允许执行的 shell 程序名。
// 只接受精确名称，不接受路径、参数或额外空白。
var allowedPodExecShells = [...]string{"sh", "bash"}

// IsAllowedPodExecShell 判断 shell 是否在允许列表中。
func IsAllowedPodExecShell(shell string) bool {
	for _, allowed := range allowedPodExecShells {
		if shell == allowed {
			return true
		}
	}
	return false
}

// ValidatePodFilePath 校验上传/下载路径。
// 清洗后仍含 ".." 段、绝对路径借 ".." 逃逸、NUL、换行、回车、过长或以 ~ 开头的路径都会拒绝。
func ValidatePodFilePath(path string) error {
	if path == "" {
		return errors.New("文件路径不能为空")
	}
	if strings.Contains(path, "\x00") || strings.ContainsAny(path, "\r\n") || len(path) > 4096 {
		return errors.New("无效的文件路径格式")
	}
	if hasDotDotSegment(path) {
		return errors.New("无效的文件路径格式")
	}

	cleaned := filepath.Clean(path)
	if len(cleaned) > 4096 || strings.HasPrefix(path, "~") || strings.HasPrefix(cleaned, "~") || hasDotDotSegment(cleaned) {
		return errors.New("无效的文件路径格式")
	}
	if strings.Contains(cleaned, "\x00") || strings.ContainsAny(cleaned, "\r\n") {
		return errors.New("无效的文件路径格式")
	}
	return nil
}

func hasDotDotSegment(path string) bool {
	normalized := strings.ReplaceAll(path, "\\", "/")
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return true
		}
	}
	return false
}
