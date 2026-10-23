//go:build linux

package config

import (
	"syscall"
	"testing"
	"unsafe"

	"github.com/spf13/viper"
)

// 第三轮 J-3：配置文件只打开、读取一次，viper 与结构扫描用同一份字节（两次读取之间文件被替换时两边会不一致）。
// 用 inotify 统计加载期间对配置文件的 open 次数（同时监听 close，避免相邻的相同事件被内核合并）。
func TestLoadWebhookReadsConfigOnce(t *testing.T) {
	p := writeWebhookFileNamed(t, "webhook.yaml", strictHead+"  fixed_workers: 7\n")
	fd, err := syscall.InotifyInit1(syscall.IN_NONBLOCK | syscall.IN_CLOEXEC)
	if err != nil {
		t.Skipf("inotify 不可用: %v", err)
	}
	defer syscall.Close(fd)
	if _, err := syscall.InotifyAddWatch(fd, p, syscall.IN_OPEN|syscall.IN_CLOSE); err != nil {
		t.Skipf("inotify 不可用: %v", err)
	}
	viper.Reset()
	t.Cleanup(viper.Reset)
	clearAllWebhookEnv(t)
	_ = captureConfigWarn(t)
	cfg, _, lerr := loadWebhookFromFile(p)
	expectLoaded(t, cfg, lerr)
	opens := 0
	buf := make([]byte, 4096)
	for {
		n, err := syscall.Read(fd, buf)
		if n <= 0 || err != nil {
			break
		}
		for off := 0; off+syscall.SizeofInotifyEvent <= n; {
			ev := (*syscall.InotifyEvent)(unsafe.Pointer(&buf[off]))
			if ev.Mask&syscall.IN_OPEN != 0 {
				opens++
			}
			off += syscall.SizeofInotifyEvent + int(ev.Len)
		}
	}
	if opens != 1 {
		t.Fatalf("配置文件应只打开一次, 实际 %d 次", opens)
	}
}
