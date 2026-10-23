package ssh

import (
	"net/url"
	"testing"
)

func TestTerminalVersionAndInitialSize(t *testing.T) {
	for _, tt := range []struct {
		query      string
		mode       string
		rows, cols int
		invalid    bool
	}{
		{query: "", rows: 25, cols: 80}, {query: "rows=bad&cols=0", rows: 25, cols: 80},
		{query: "terminal_mode=pty-v1", mode: TerminalModePTY, rows: 25, cols: 80},
		{query: "terminal_mode=pty-v1&rows=37&cols=96", mode: TerminalModePTY, rows: 37, cols: 96},
		{query: "terminal_mode=guess", invalid: true}, {query: "terminal_mode=pty-v1&terminal_mode=pty-v1", invalid: true},
		{query: "terminal_mode=pty-v1&rows=0", invalid: true}, {query: "terminal_mode=pty-v1&rows=501", invalid: true},
		{query: "terminal_mode=pty-v1&cols=1001", invalid: true}, {query: "terminal_mode=pty-v1&cols=", invalid: true},
		{query: "terminal_mode=pty-v1&rows=1&rows=2", invalid: true},
	} {
		query, _ := url.ParseQuery(tt.query)
		options, err := ParseTerminalOptions(query)
		if (err != nil) != tt.invalid {
			t.Fatal("terminal parameter boundary changed")
		}
		if !tt.invalid && (options.Mode != tt.mode || options.Rows != tt.rows || options.Cols != tt.cols) {
			t.Fatal("terminal version or dimensions lost")
		}
	}
}
