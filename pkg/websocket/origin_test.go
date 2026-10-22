package websocket

import (
	"net/http/httptest"
	"testing"
)

func TestCheckOrigin(t *testing.T) {
	t.Setenv("SERVER_ALLOWED_ORIGINS", "https://console.example.com")
	for _, tt := range []struct {
		origin string
		want   bool
	}{
		{"", true}, {"https://api.example.com", true}, {"https://console.example.com", true},
		{"https://console.example.com.evil.test", false}, {"null", false}, {"ftp://api.example.com", false},
	} {
		t.Run(tt.origin, func(t *testing.T) {
			req := httptest.NewRequest("GET", "https://api.example.com/ws", nil)
			req.Header.Set("Origin", tt.origin)
			if got := CheckOrigin(req); got != tt.want {
				t.Fatalf("got %v want %v", got, tt.want)
			}
		})
	}
}
