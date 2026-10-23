package ssh

import (
	"fmt"
	"net/url"
	"strconv"
)

const (
	TerminalModePTY         = "pty-v1"
	MaxTerminalRows         = 500
	MaxTerminalCols         = 1000
	MaxTerminalControlBytes = 1024
)

type TerminalOptions struct {
	Mode string
	Rows int
	Cols int
}

func (o TerminalOptions) Normalize() (TerminalOptions, error) {
	if o.Mode == "" {
		return TerminalOptions{Rows: 25, Cols: 80}, nil
	}
	if o.Mode != TerminalModePTY {
		return o, fmt.Errorf("不支持的终端模式")
	}
	if o.Rows == 0 {
		o.Rows = 25
	}
	if o.Cols == 0 {
		o.Cols = 80
	}
	if err := ValidateTerminalSize(o.Rows, o.Cols); err != nil {
		return o, err
	}
	return o, nil
}
func ValidateTerminalSize(rows, cols int) error {
	if rows < 1 || rows > MaxTerminalRows || cols < 1 || cols > MaxTerminalCols {
		return fmt.Errorf("终端尺寸超出允许范围")
	}
	return nil
}

// ParseTerminalOptions 只由显式版本选择PTY，旧客户端输入不做JSON猜测
func ParseTerminalOptions(query url.Values) (TerminalOptions, error) {
	if len(query["terminal_mode"]) > 1 {
		return TerminalOptions{}, fmt.Errorf("终端模式参数无效")
	}
	options := TerminalOptions{Mode: query.Get("terminal_mode"), Rows: 25, Cols: 80}
	if options.Mode == "" {
		return options.Normalize()
	}
	if options.Mode != TerminalModePTY {
		return options, fmt.Errorf("不支持的终端模式")
	}
	for name, target := range map[string]*int{"rows": &options.Rows, "cols": &options.Cols} {
		if values, exists := query[name]; exists {
			if len(values) != 1 {
				return options, fmt.Errorf("终端尺寸参数无效")
			}
			value, err := strconv.Atoi(values[0])
			if err != nil || value < 1 {
				return options, fmt.Errorf("终端尺寸参数无效")
			}
			*target = value
		}
	}
	return options.Normalize()
}
