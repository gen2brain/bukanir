// Package bukanir is a library for discovering and streaming movies and TV shows from BitTorrent.
package bukanir

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ErrTorDisabled is returned by PeerDialer when Config.Tor is not set.
var ErrTorDisabled = errors.New("bukanir: Tor is not enabled")

// Client discovers and streams movies and TV shows.
type Client struct {
	cfg       Config
	tr        *transport
	cache     *cache
	subsCache *cache

	tmdbBase           string // TMDB API base URL
	tpbBase            string // ThePirateBay API base URL
	ytsBase            string // YTS API base URL
	eztvBase           string // EZTV API base URL
	torrentioBase      string // Torrentio add-on base URL
	subsBase           string // OpenSubtitles API base URL
	gestdownBase       string // gestdown.info API base URL
	subdlBase          string // subdl.com API base URL
	yifyBase           string // YIFY subtitles base URL
	movieSubtitlesBase string // moviesubtitles.org base URL

	subProviders []subtitleProvider // subtitle sources queried by Subtitles

	tmdbMu   sync.Mutex
	tmdbConf *tmdbConfiguration // memoized image configuration

	tmdbMemoMu sync.Mutex
	tmdbMemo   map[string]*tmdbLookup // memoized TMDB responses by URL

	subsMu    sync.Mutex
	subsToken string // memoized OpenSubtitles login token
	subsErr   error  // memoized login failure
}

// New creates a Client.
func New(ctx context.Context, cfg Config) (*Client, error) {
	cfg = cfg.withDefaults()

	root := cfg.CacheDir
	if root == "" {
		if d, err := os.UserCacheDir(); err == nil {
			root = filepath.Join(d, "bukanir")
		} else {
			root = filepath.Join(os.TempDir(), "bukanir")
		}
	}

	var subsTTL time.Duration
	if cfg.CacheTTL > 0 {
		subsTTL = subtitleCacheTTL
	}
	c := &Client{
		cfg:                cfg,
		tr:                 newTransport(ctx, cfg, filepath.Join(root, "tor")),
		cache:              newCache(filepath.Join(root, "cache"), cfg.CacheTTL),
		subsCache:          newCache(filepath.Join(root, "subtitles"), subsTTL),
		tmdbBase:           tmdbBaseURL,
		tpbBase:            hostBaseURL(cfg.tpbHost()),
		ytsBase:            hostBaseURL(cfg.ytsHost()),
		eztvBase:           hostBaseURL(cfg.eztvHost()),
		torrentioBase:      torrentioBaseURL,
		subsBase:           openSubtitlesBaseURL,
		gestdownBase:       gestdownBaseURL,
		subdlBase:          subdlBaseURL,
		yifyBase:           yifyBaseURL,
		movieSubtitlesBase: movieSubtitlesBaseURL,
	}
	c.subProviders = []subtitleProvider{
		gestdownProvider{c},
		yifyProvider{c},
		movieSubtitlesProvider{c},
	}
	if cfg.openSubtitlesKey() != "" && cfg.openSubtitlesUser() != "" && cfg.openSubtitlesPass() != "" {
		c.subProviders = append(c.subProviders, openSubtitlesProvider{c})
	}
	if cfg.subdlKey() != "" {
		c.subProviders = append(c.subProviders, subdlProvider{c})
	}
	if cfg.Tor {
		warmGateways()
	}
	return c, nil
}

// PeerDialer returns a dialer that routes BitTorrent peer connections through the embedded Tor client.
func (c *Client) PeerDialer() (PeerDialer, error) {
	if !c.cfg.Tor {
		return nil, ErrTorDisabled
	}
	return c.tr.dialer()
}

// TorState is the state of the embedded Tor client.
type TorState int

const (
	TorOff TorState = iota
	TorStarting
	TorReady
	TorFailed
)

// TorStatus returns the state of the embedded Tor client and, for TorFailed, its error.
func (c *Client) TorStatus() (TorState, error) { return c.tr.torStatus() }

// TorDone is closed when the embedded Tor client has finished starting or the Client is closed.
func (c *Client) TorDone() <-chan struct{} { return c.tr.done }

// StartTor bootstraps the embedded Tor client in the background when peers or a .onion source host use it.
func (c *Client) StartTor() {
	if c.cfg.Tor || c.cfg.usesOnion() {
		c.tr.start()
	}
}

// Close releases resources, including the embedded Tor client.
func (c *Client) Close() error {
	return c.tr.Close()
}
