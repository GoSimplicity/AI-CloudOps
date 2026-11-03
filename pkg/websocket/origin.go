package websocket

import (
	"net/http"
	"net/url"
	"os"
	"strings"
)

func CheckOrigin(r *http.Request) bool {
	return NewOriginChecker(os.Getenv("SERVER_ALLOWED_ORIGINS"))(r)
}

// NewOriginChecker captures the configured allowlist for an upgrader's lifetime.
// CheckOrigin remains the environment-backed adapter for standalone consumers.
func NewOriginChecker(rawAllowedOrigins string) func(*http.Request) bool {
	allowedOrigins := make(map[string]struct{})
	for _, origin := range strings.Split(rawAllowedOrigins, ",") {
		if origin = strings.TrimSpace(origin); origin != "" {
			allowedOrigins[origin] = struct{}{}
		}
	}
	return func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		u, err := url.Parse(origin)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.User != nil {
			return false
		}
		if strings.EqualFold(u.Host, r.Host) {
			return true
		}
		_, allowed := allowedOrigins[origin]
		return allowed
	}
}
