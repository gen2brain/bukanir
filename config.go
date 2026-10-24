package bukanir

import (
	"encoding/base64"
	"io"
	"os"
	"strings"
	"time"
)

// Version is the library version.
const Version = "3.0.0"

// Defaults applied to a zero-valued Config by withDefaults.
const (
	DefaultCacheTTL    = 24 * time.Hour
	DefaultHTTPTimeout = 30 * time.Second
)

// DefaultUserAgent is sent on metadata requests when Config.UserAgent is empty.
var DefaultUserAgent = "bukanir/" + Version

// builtinTMDBKey is injected at build time as reversed base64, left empty in source.
// OpenSubtitles and SubDL credentials are never built in; their limits are per key.
var builtinTMDBKey string

// Config configures a Client's metadata, caching, and network behavior.
type Config struct {
	// TMDBKey is the TMDB v3 API key.
	TMDBKey string
	// OpenSubtitles REST credentials.
	OpenSubtitlesKey  string
	OpenSubtitlesUser string
	OpenSubtitlesPass string
	// SubdlKey is the subdl.com API key.
	SubdlKey string

	// CacheDir holds cached metadata responses; required.
	CacheDir string
	// CacheTTL is how long a cached response stays fresh.
	CacheTTL time.Duration

	// TPBHost is the ThePirateBay/apibay API host.
	TPBHost string
	// YTSHost is the YTS API host.
	YTSHost string
	// EZTVHost is the EZTV API host.
	EZTVHost string

	// Tor routes torrent peer connections through the embedded Tor client.
	Tor bool
	// TorInstances is the number of independent Tor instances (pool size).
	TorInstances int
	// HTTPTimeout bounds each metadata request.
	HTTPTimeout time.Duration
	// UserAgent is sent on metadata requests.
	UserAgent string
	// TorLog, if set, receives the embedded Tor client's diagnostic output.
	TorLog io.Writer
}

// DefaultConfig returns a Config with caching, timeout, and user agent set.
func DefaultConfig() Config {
	return Config{
		CacheTTL:    DefaultCacheTTL,
		HTTPTimeout: DefaultHTTPTimeout,
		UserAgent:   DefaultUserAgent,
	}
}

// withDefaults returns a copy with zero fields filled in, except CacheTTL.
func (c Config) withDefaults() Config {
	if c.HTTPTimeout == 0 {
		c.HTTPTimeout = DefaultHTTPTimeout
	}
	if c.UserAgent == "" {
		c.UserAgent = DefaultUserAgent
	}
	return c
}

func (c *Config) tpbHost() string {
	if c.TPBHost != "" {
		return c.TPBHost
	}
	return DefaultTPBHost
}

func (c *Config) ytsHost() string {
	if c.YTSHost != "" {
		return c.YTSHost
	}
	return DefaultYTSHost
}

func (c *Config) eztvHost() string {
	if c.EZTVHost != "" {
		return c.EZTVHost
	}
	return DefaultEZTVHost
}

// usesOnion reports whether a source host is a .onion address.
func (c *Config) usesOnion() bool {
	for _, host := range []string{c.tpbHost(), c.ytsHost(), c.eztvHost()} {
		if strings.HasSuffix(host, ".onion") {
			return true
		}
	}
	return false
}

func (c *Config) tmdbKey() string {
	return secret(c.TMDBKey, "BUKANIR_TMDB_KEY", builtin(builtinTMDBKey))
}

func (c *Config) openSubtitlesKey() string {
	return secret(c.OpenSubtitlesKey, "BUKANIR_OPENSUBTITLES_KEY", "")
}

func (c *Config) openSubtitlesUser() string {
	return secret(c.OpenSubtitlesUser, "BUKANIR_OPENSUBTITLES_USER", "")
}

func (c *Config) openSubtitlesPass() string {
	return secret(c.OpenSubtitlesPass, "BUKANIR_OPENSUBTITLES_PASS", "")
}

func (c *Config) subdlKey() string {
	return secret(c.SubdlKey, "BUKANIR_SUBDL_KEY", "")
}

// builtin decodes a build-time key, or returns "" when none was injected.
func builtin(obfuscated string) string {
	b := []byte(obfuscated)
	for i, j := 0, len(b)-1; i < j; i, j = i+1, j-1 {
		b[i], b[j] = b[j], b[i]
	}
	key, err := base64.StdEncoding.DecodeString(string(b))
	if err != nil {
		return ""
	}
	return string(key)
}

// SecretValues returns c's active credential values, for redaction from logs.
func SecretValues(c Config) []string {
	var out []string
	for _, v := range []string{
		c.tmdbKey(),
		c.openSubtitlesKey(),
		c.openSubtitlesUser(),
		c.openSubtitlesPass(),
		c.subdlKey(),
	} {
		if len(v) >= 6 {
			out = append(out, v)
		}
	}
	return out
}

// secret resolves a credential to the first non-empty of explicit, env, and builtin.
func secret(explicit, env, builtin string) string {
	if explicit != "" {
		return explicit
	}
	if v := os.Getenv(env); v != "" {
		return v
	}
	return builtin
}
