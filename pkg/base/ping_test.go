package base

import (
	"net"
	"strconv"
	"testing"
	"time"
)

func TestProbeTCP80RejectsEmptyAddress(t *testing.T) {
	if ProbeTCP80("") || ProbeTCP80("   ") || Ping("") || Ping("   ") {
		t.Fatal("空地址不应视为可达")
	}
}

func TestProbeTCP(t *testing.T) {
	if probeTCP("", "80", time.Second) || probeTCP("   ", "80", time.Second) {
		t.Fatal("空地址不应视为可达")
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := strconv.Itoa(ln.Addr().(*net.TCPAddr).Port)
	if !probeTCP("127.0.0.1", port, time.Second) {
		t.Fatal("已监听端口应探测成功")
	}
	_ = ln.Close()
	if probeTCP("127.0.0.1", port, 200*time.Millisecond) {
		t.Fatal("关闭后的端口不应探测成功")
	}
}
