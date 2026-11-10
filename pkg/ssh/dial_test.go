package ssh

import (
	"context"
	"errors"
	"io"
	"net"
	"os"
	"testing"
	"time"

	cryptossh "golang.org/x/crypto/ssh"
)

type timeoutNetError struct{}

func (timeoutNetError) Error() string   { return "timeout" }
func (timeoutNetError) Timeout() bool   { return true }
func (timeoutNetError) Temporary() bool { return true }

func TestNormalizeDialErrorMapsTimeouts(t *testing.T) {
	background := context.Background()
	for _, err := range []error{
		errors.New("read tcp 127.0.0.1:1->127.0.0.1:2: i/o timeout"),
		timeoutNetError{},
		os.ErrDeadlineExceeded,
		context.DeadlineExceeded,
	} {
		got := normalizeDialError(background, err)
		if !errors.Is(got, context.DeadlineExceeded) {
			t.Fatalf("%v => %v", err, got)
		}
	}
	if got := normalizeDialError(background, errors.New("connection refused")); errors.Is(got, context.DeadlineExceeded) || got.Error() != "connection refused" {
		t.Fatalf("non-timeout = %v", got)
	}
	canceled, cancel := context.WithCancel(background)
	cancel()
	if got := normalizeDialError(canceled, errors.New("i/o timeout")); !errors.Is(got, context.Canceled) {
		t.Fatalf("canceled dial = %v", got)
	}
}

func TestDialContextTimeoutIsDeadlineExceeded(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer listener.Close()
	go func() {
		conn, acceptErr := listener.Accept()
		if acceptErr == nil {
			_, _ = io.Copy(io.Discard, conn)
			_ = conn.Close()
		}
	}()
	started := time.Now()
	_, err = DialContext(context.Background(), "tcp", listener.Addr().String(), &cryptossh.ClientConfig{
		User:            "fixture",
		HostKeyCallback: cryptossh.InsecureIgnoreHostKey(),
		Timeout:         200 * time.Millisecond,
	})
	if err == nil || time.Since(started) > 2*time.Second || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("handshake timeout = %v after %s", err, time.Since(started))
	}

	listener.Close()
	_, err = DialContext(context.Background(), "tcp", listener.Addr().String(), &cryptossh.ClientConfig{
		User:            "fixture",
		HostKeyCallback: cryptossh.InsecureIgnoreHostKey(),
		Timeout:         time.Second,
	})
	if err == nil || errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("refused dial = %v", err)
	}
}
