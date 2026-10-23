package ssh

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestCommandOutputWriterContract(t *testing.T) {
	for _, tt := range []struct {
		name      string
		writes    []string
		truncated bool
	}{
		{"empty", []string{""}, false},
		{"exact", []string{strings.Repeat("x", MaxCommandOutputBytes)}, false},
		{"multiple", []string{strings.Repeat("x", MaxCommandOutputBytes-1), "more"}, true},
		{"unicode", []string{strings.Repeat("x", MaxCommandOutputBytes-1), "中"}, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			output := &commandOutput{}
			for _, p := range tt.writes {
				n, err := output.Write([]byte(p))
				if n != len(p) || err != nil {
					t.Fatal("writer did not consume full input")
				}
			}
			result := output.String()
			if len(result) > MaxCommandOutputBytes+len(outputTruncated) || strings.HasSuffix(result, outputTruncated) != tt.truncated {
				t.Fatal("unexpected output boundary")
			}
		})
	}
}
func TestRunContextRejectsCancelledRequest(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := RunContext(ctx, nil, "command"); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation not retained: %v", err)
	}
}
