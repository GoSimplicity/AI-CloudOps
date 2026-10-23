//go:build smoke

package ssh

import (
	"net"
	"testing"
	"time"

	cryptossh "golang.org/x/crypto/ssh"
)

func TestSSHHandshakeDeadline(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	accepted := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err == nil {
			accepted <- conn
		}
	}()
	started := time.Now()
	_, err = DialWithTimeout("tcp", listener.Addr().String(), &cryptossh.ClientConfig{User: "test", HostKeyCallback: cryptossh.InsecureIgnoreHostKey(), Timeout: 100 * time.Millisecond})
	if err == nil {
		t.Fatal("silent SSH server completed handshake")
	}
	if time.Since(started) > time.Second {
		t.Fatal("handshake ignored connection deadline")
	}
	select {
	case conn := <-accepted:
		defer conn.Close()
		if err := conn.SetReadDeadline(time.Now().Add(time.Second)); err != nil {
			t.Fatal(err)
		}
		buffer := make([]byte, 1024)
		for {
			if _, err := conn.Read(buffer); err != nil {
				if timeout, ok := err.(net.Error); ok && timeout.Timeout() {
					t.Fatal("failed handshake leaked TCP connection")
				}
				break
			}
		}
	case <-time.After(time.Second):
		t.Fatal("test listener did not accept connection")
	}
}
