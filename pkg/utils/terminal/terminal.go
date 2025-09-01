package terminal

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/websocket"
	"io"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/remotecommand"
	"time"
)

const (
	writeWait         = 10 * time.Second
	endOfTransmission = "\u0004"

	pongWait = 30 * time.Second
	// Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10
)

type Interface interface {
	HandleSession(ctx context.Context, shell, namespace, podName, containerName string, conn *websocket.Conn)
}

type TerminalSessionHandler interface {
	io.Reader
	io.Writer
	remotecommand.TerminalSizeQueue
}

type Session struct {
	conn     *websocket.Conn
	sizeChan chan remotecommand.TerminalSize
}

/*
		 OP      DIRECTION  USED  				DESCRIPTION
		 ---------------------------------------------------------------------
	     stdin   fe->be     Data           		Keystrokes/paste buffer
	     resize  fe->be     RowSize, ColSize    New terminal size
		 stdout  be->fe     Data           		Output from the process
*/
type Message struct {
	Op      string `json:"op"`
	Data    string `json:"data"`
	RowSize uint16 `json:"row_size"`
	ColSize uint16 `json:"col_size"`
}

func (t Session) Write(p []byte) (int, error) {
	msg, err := json.Marshal(Message{
		Op:   "stdout",
		Data: string(p),
	})
	if err != nil {
		return 0, err
	}
	if err := t.conn.SetWriteDeadline(time.Now().Add(5 * time.Second)); err != nil {
		return 0, err
	}
	if err = t.conn.WriteMessage(websocket.TextMessage, msg); err != nil {
		return 0, err
	}
	return len(p), nil
}

func (t Session) Close() error {
	close(t.sizeChan)
	return t.conn.Close()
}

func (t Session) Read(p []byte) (int, error) {
	var msg Message
	if err := t.conn.ReadJSON(&msg); err != nil {
		return copy(p, endOfTransmission), err
	}

	switch msg.Op {
	case "stdin":
		return copy(p, msg.Data), nil
	case "resize":
		t.sizeChan <- remotecommand.TerminalSize{Width: msg.ColSize, Height: msg.RowSize}
		return 0, nil
	default:
		return copy(p, endOfTransmission), fmt.Errorf("unknown message type '%s'", msg.Op)
	}
}

func (t Session) Next() *remotecommand.TerminalSize {
	size := <-t.sizeChan
	if size.Height == 0 && size.Width == 0 {
		return nil
	}
	return &size
}

type terminaler struct {
	client kubernetes.Interface
	config *rest.Config
	//options *Options
}

func NewTerminalerHandler(client kubernetes.Interface, config *rest.Config) Interface {
	return &terminaler{client: client, config: config}
}

func (t *terminaler) HandleSession(ctx context.Context, shell, namespace, podName, containerName string, conn *websocket.Conn) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	go wait.UntilWithContext(ctx, func(ctx context.Context) {

		if err := conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(writeWait)); err != nil {
			//klog.V(4).Infof("failed to send ping packet: %s, closing websocket connection", err)
			cancel()
			_ = conn.Close()
		}
	}, pingPeriod)

	conn.SetReadDeadline(time.Now().Add(pongWait)) // nolint
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(pongWait)) // nolint
		return nil
	})

	t.handler(ctx, shell, namespace, podName, containerName, conn)
}

func (t *terminaler) handler(ctx context.Context, shell, namespace, podName, containerName string, conn *websocket.Conn) {
	var err error

	session := &Session{conn: conn, sizeChan: make(chan remotecommand.TerminalSize)}

	cmd := []string{"sh"}
	if checkShell(shell) {
		cmd = []string{shell}
		err = t.processor(ctx, namespace, podName, containerName, cmd, session)
	}
	if err != nil && !errors.Is(err, context.Canceled) {
		session.Close()
		return
	}
	session.Close()
}

func (t *terminaler) processor(ctx context.Context, namespace, podName, containerName string, cmd []string, handler TerminalSessionHandler) error {
	req := t.client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")
	req.VersionedParams(&v1.PodExecOptions{
		Container: containerName,
		Command:   cmd,
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
		TTY:       true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(t.config, "POST", req.URL())
	if err != nil {
		return err
	}

	return exec.StreamWithContext(ctx, remotecommand.StreamOptions{
		Stdin:             handler,
		Stdout:            handler,
		Stderr:            handler,
		TerminalSizeQueue: handler,
		Tty:               true,
	})
}

func checkShell(shell string) bool {

	for _, validShell := range []string{"bash", "sh"} {
		if validShell == shell {
			return true
		}
	}
	return false
}
