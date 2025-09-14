package net

import (
	"context"
	"crypto/tls"
	"net"
	"net/url"
	"strings"
	"time"

	"github.com/pkg/errors"
)

type ConnOptions struct {
	Timeout   time.Duration
	TLSConfig *tls.Config
}

type ProtocolDialer interface {
	DialContext(ctx context.Context, address string, opts ConnOptions) (net.Conn, error)
}

type tcpDialer struct{}

func (d tcpDialer) DialContext(ctx context.Context, address string, opts ConnOptions) (net.Conn, error) {
	dialer := CheckDialer()
	dialer.Timeout = opts.Timeout

	if opts.TLSConfig != nil {
		return tls.DialWithDialer(dialer, "tcp", address, opts.TLSConfig)
	}
	return dialer.DialContext(ctx, "tcp", address)
}

func SockConn(ctx context.Context, daemon string, opts ConnOptions) (net.Conn, error) {
	daemonUrl, err := url.Parse(daemon)
	if err != nil {
		return nil, errors.Wrapf(err, "could not parse url %q", daemon)
	}

	var (
		dialer  ProtocolDialer
		address string
	)

	switch strings.ToLower(daemonUrl.Scheme) {
	case "tcp":
		dialer = tcpDialer{}
		address = daemonUrl.Host
	default:
		return nil, errors.Errorf("unsupported protocol scheme %q", daemonUrl.Scheme)
	}

	conn, err := dialer.DialContext(ctx, address, opts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to connect to %s", daemon)
	}
	return conn, nil
}
