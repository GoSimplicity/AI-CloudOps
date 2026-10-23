package terminal

import (
	"errors"
	"io"

	"github.com/gorilla/websocket"
)

const (
	// JSON可能将一个stdin字节转义为六个字符，消息预算与数据预算分开
	maxTerminalMessageBytes = 512 * 1024
	maxTerminalStdinBytes   = 64 * 1024
	maxTerminalControlBytes = 1024
)

var errTerminalInputLimit = errors.New("终端输入超过大小限制")

// readInputMessage 对解压后的内容验限，不能仅依赖线路长度的SetReadLimit
// 完整验收后才允许Read把任何该消息字节交给远端
func (t *Session) readInputMessage() ([]byte, error) {
	t.conn.SetReadLimit(maxTerminalMessageBytes * 2)
	_, stream, err := t.conn.NextReader()
	if errors.Is(err, websocket.ErrReadLimit) {
		return nil, errTerminalInputLimit
	}
	if err != nil {
		return nil, err
	}
	raw, err := io.ReadAll(io.LimitReader(stream, maxTerminalMessageBytes+1))
	if len(raw) > maxTerminalMessageBytes || errors.Is(err, websocket.ErrReadLimit) {
		return nil, errTerminalInputLimit
	}
	if err != nil {
		return nil, err
	}
	return raw, nil
}
func (t *Session) rejectInput() (int, error) {
	t.pending = nil
	_ = t.Close()
	return 0, errTerminalInputLimit
}
