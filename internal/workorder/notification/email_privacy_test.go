package notification

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/textproto"
	"strings"
	"testing"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

type privacyEmailConfig struct{ localEmailConfig }

func (privacyEmailConfig) GetUsername() string { return "private-sender@example.test" }
func (privacyEmailConfig) GetPassword() string { return "private-credential-marker" }

func TestSMTPDiagnosticsDoNotExposePrivateData(t *testing.T) {
	for _, mode := range []string{"success", "rejected", "lost-receipt"} {
		t.Run(mode, func(t *testing.T) {
			listener, err := net.Listen("tcp", "127.0.0.1:0")
			if err != nil {
				t.Fatal(err)
			}
			defer listener.Close()
			served := make(chan struct{})
			go func() {
				defer close(served)
				conn, err := listener.Accept()
				if err != nil {
					return
				}
				defer conn.Close()
				_ = conn.SetDeadline(time.Now().Add(5 * time.Second))
				peer := textproto.NewConn(conn)
				_ = peer.PrintfLine("220 local ESMTP")
				for {
					line, err := peer.ReadLine()
					if err != nil {
						return
					}
					switch {
					case strings.HasPrefix(line, "EHLO"), strings.HasPrefix(line, "HELO"):
						_ = peer.PrintfLine("250 local")
					case strings.HasPrefix(line, "MAIL FROM"), strings.HasPrefix(line, "RCPT TO"):
						_ = peer.PrintfLine("250 accepted")
					case line == "DATA":
						_ = peer.PrintfLine("354 continue")
						_, _ = io.Copy(io.Discard, peer.DotReader())
						if mode == "lost-receipt" {
							return
						}
						if mode == "rejected" {
							_ = peer.PrintfLine("550 private-provider-marker private-recipient@example.test private-credential-marker")
						} else {
							_ = peer.PrintfLine("250 queued")
						}
					case line == "QUIT":
						_ = peer.PrintfLine("221 bye")
						return
					default:
						_ = peer.PrintfLine("500 unsupported")
					}
				}
			}()
			core, logs := observer.New(zap.DebugLevel)
			manager, err := NewManager(localChannels{email: privacyEmailConfig{localEmailConfig{port: listener.Addr().(*net.TCPAddr).Port}}}, nil, zap.New(core))
			if err != nil {
				t.Fatal(err)
			}
			result, sendErr := manager.SendNotification(context.Background(), &SendRequest{Channel: "email", RecipientAddr: "private-recipient@example.test", Subject: "private-subject-marker", Content: "private-body-marker"})
			if (sendErr == nil) != (mode == "success") {
				t.Fatal("unexpected delivery outcome")
			}
			var diagnostic strings.Builder
			for _, entry := range logs.All() {
				diagnostic.WriteString(entry.Message)
				encoded, _ := json.Marshal(entry.ContextMap())
				diagnostic.Write(encoded)
			}
			if sendErr != nil {
				diagnostic.WriteString(sendErr.Error())
			}
			if result != nil {
				diagnostic.WriteString(result.ErrorMessage)
			}
			for _, marker := range []string{"private-recipient@example.test", "private-sender@example.test", "private-subject-marker", "private-body-marker", "private-provider-marker", "private-credential-marker"} {
				if strings.Contains(diagnostic.String(), marker) {
					t.Errorf("private diagnostic data exposed")
				}
			}
			<-served
		})
	}
}

func TestSMTPManagerPreservesCancellationWithoutLeakingAddress(t *testing.T) {
	core, logs := observer.New(zap.DebugLevel)
	manager, err := NewManager(localChannels{email: privacyEmailConfig{localEmailConfig{port: 2525}}}, nil, zap.New(core))
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	response, err := manager.SendNotification(ctx, &SendRequest{Channel: "email", RecipientAddr: "private-recipient@example.test", Subject: "private-subject-marker"})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation semantics lost: %v", err)
	}
	var output strings.Builder
	for _, entry := range logs.All() {
		output.WriteString(entry.Message)
		encoded, _ := json.Marshal(entry.ContextMap())
		output.Write(encoded)
	}
	if response != nil {
		output.WriteString(response.ErrorMessage)
	}
	if strings.Contains(output.String(), "private-recipient@example.test") || strings.Contains(output.String(), "private-subject-marker") {
		t.Fatal("cancellation diagnostics exposed private data")
	}
}
