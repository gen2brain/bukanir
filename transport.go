package bukanir

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"

	arti "github.com/gen2brain/go-arti"
)

// DefaultTorInstances is the number of Tor instances used when Config.TorInstances is zero.
const DefaultTorInstances = 4

type torProvider interface {
	HTTPClient() *http.Client
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
	Close() error
}

// newTorProvider builds a bootstrapped Tor provider.
var newTorProvider = func(ctx context.Context, cacheDir string, instances int, stderr io.Writer) (torProvider, error) {
	var opts []arti.Option
	if stderr != nil {
		opts = append(opts, arti.WithStderr(stderr))
	}
	return arti.NewPool(ctx, cacheDir, instances, opts...)
}

// transport hands out an *http.Client for metadata requests, routing through Tor when configured.
type transport struct {
	cfg       Config
	ctx       context.Context
	cacheDir  string
	instances int

	direct *http.Client

	once sync.Once
	tor  torProvider
	torc *http.Client
	err  error
}

// newTransport builds a transport; torCacheDir persists the Tor dir cache, empty uses a temp dir.
func newTransport(ctx context.Context, cfg Config, torCacheDir string) *transport {
	cfg = cfg.withDefaults()
	instances := cfg.TorInstances
	if instances < 1 {
		instances = DefaultTorInstances
	}
	return &transport{
		cfg:       cfg,
		ctx:       ctx,
		cacheDir:  torCacheDir,
		instances: instances,
		direct: &http.Client{
			Timeout: cfg.HTTPTimeout,
			Transport: &http.Transport{
				Proxy:       http.ProxyFromEnvironment,
				DialContext: (&net.Dialer{Timeout: cfg.HTTPTimeout}).DialContext,
				// force HTTP/1.1
				TLSNextProto:          map[string]func(string, *tls.Conn) http.RoundTripper{},
				MaxIdleConnsPerHost:   8,
				TLSClientConfig:       &tls.Config{MinVersion: tls.VersionTLS12},
				TLSHandshakeTimeout:   cfg.HTTPTimeout,
				ExpectContinueTimeout: cfg.HTTPTimeout,
			},
		},
	}
}

// ensureTor bootstraps the Tor provider once and caches it (or the error).
func (t *transport) ensureTor() error {
	t.once.Do(func() {
		p, err := newTorProvider(t.ctx, t.cacheDir, t.instances, t.cfg.TorLog)
		if err != nil {
			t.err = err
			return
		}
		c := p.HTTPClient()
		c.Timeout = t.cfg.HTTPTimeout
		t.tor, t.torc = p, c
	})
	return t.err
}

// client is the HTTP client for non-.onion metadata: always direct.
func (t *transport) client() (*http.Client, error) {
	return t.direct, nil
}

// clientFor returns the HTTP client to use for a URL; .onion is routed through Tor.
func (t *transport) clientFor(rawURL string) (*http.Client, error) {
	if isOnionURL(rawURL) {
		if err := t.ensureTor(); err != nil {
			return nil, err
		}
		return t.torc, nil
	}
	return t.client()
}

func isOnionURL(rawURL string) bool {
	u, err := url.Parse(rawURL)
	if err != nil {
		return false
	}
	return strings.HasSuffix(u.Hostname(), ".onion")
}

// dialer returns a PeerDialer routing through Tor, bootstrapping it on first use.
func (t *transport) dialer() (PeerDialer, error) {
	if err := t.ensureTor(); err != nil {
		return nil, err
	}
	return t.tor, nil
}

func (t *transport) Close() error {
	if t.tor != nil {
		return t.tor.Close()
	}
	return nil
}
