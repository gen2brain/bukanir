//go:build systemtor

package bukanir

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"golang.org/x/net/proxy"
)

// SystemTor reports whether Tor goes through the system daemon instead of the embedded client.
const SystemTor = true

// newTorProvider connects to the system Tor daemon's SOCKS5 port.
var newTorProvider = func(ctx context.Context, cfg Config, _ string, _ int) (torProvider, error) {
	addr := cfg.torProxy()
	conn, err := (&net.Dialer{Timeout: cfg.HTTPTimeout}).DialContext(ctx, "tcp", addr)
	if err != nil {
		return nil, fmt.Errorf("bukanir: system Tor at %s: %w", addr, err)
	}
	conn.Close()
	d, err := proxy.SOCKS5("tcp", addr, nil, proxy.Direct)
	if err != nil {
		return nil, err
	}
	return &systemTor{dialer: d.(proxy.ContextDialer)}, nil
}

// systemTor routes connections through a SOCKS5 Tor proxy.
type systemTor struct {
	dialer proxy.ContextDialer
}

func (s *systemTor) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	return s.dialer.DialContext(ctx, network, addr)
}

func (s *systemTor) HTTPClient() *http.Client {
	return &http.Client{Transport: &http.Transport{DialContext: s.dialer.DialContext}}
}

func (s *systemTor) Close() error { return nil }
