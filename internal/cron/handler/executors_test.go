package handler

import (
	"io"
	"strings"
	"testing"
)

func TestOutputBuffer(t *testing.T) {
	for _, tt := range []struct {
		name      string
		writes    []string
		length    int
		truncated bool
	}{
		{"empty", []string{""}, 0, false},
		{"normal", []string{"hello", " world"}, 11, false},
		{"exact limit", []string{strings.Repeat("a", maxTaskOutputBytes)}, maxTaskOutputBytes, false},
		{"single oversized write", []string{strings.Repeat("a", maxTaskOutputBytes+1)}, maxTaskOutputBytes, true},
		{"unicode boundary", []string{strings.Repeat("a", maxTaskOutputBytes-1), "中"}, maxTaskOutputBytes, true},
		{"multiple writes", []string{strings.Repeat("a", maxTaskOutputBytes-1), "bc", "d"}, maxTaskOutputBytes, true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			buffer := newOutputBuffer()
			for _, p := range tt.writes {
				n, err := io.WriteString(buffer, p)
				if err != nil || n != len(p) {
					t.Fatalf("write returned (%d, %v), want (%d, nil)", n, err, len(p))
				}
			}
			if buffer.Len() != tt.length {
				t.Fatalf("length = %d, want %d", buffer.Len(), tt.length)
			}
			if strings.HasSuffix(buffer.String(), outputTruncatedMarker) != tt.truncated {
				t.Fatal("unexpected truncation marker")
			}
			want := strings.Join(tt.writes, "")[:tt.length]
			if tt.truncated {
				want = strings.ToValidUTF8(want, "") + outputTruncatedMarker
			}
			if buffer.String() != want {
				t.Fatal("buffer did not preserve output prefix")
			}
		})
	}
}
