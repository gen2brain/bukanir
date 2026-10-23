package bukanir

import (
	"strings"
	"testing"
	"time"
)

func TestSecretPrecedence(t *testing.T) {
	// explicit wins over env and builtin
	t.Setenv("BUKANIR_TEST_SECRET", "fromEnv")
	if got := secret("explicit", "BUKANIR_TEST_SECRET", "builtin"); got != "explicit" {
		t.Errorf("explicit: got %q, want %q", got, "explicit")
	}
	// env wins over builtin when explicit is empty
	if got := secret("", "BUKANIR_TEST_SECRET", "builtin"); got != "fromEnv" {
		t.Errorf("env: got %q, want %q", got, "fromEnv")
	}
	// builtin when neither explicit nor env is set
	if got := secret("", "BUKANIR_TEST_UNSET", "builtin"); got != "builtin" {
		t.Errorf("builtin: got %q, want %q", got, "builtin")
	}
	// empty when nothing is available
	if got := secret("", "BUKANIR_TEST_UNSET", ""); got != "" {
		t.Errorf("empty: got %q, want empty", got)
	}
}

func TestCredentialAccessors(t *testing.T) {
	c := Config{TMDBKey: "field-key"}
	if got := c.tmdbKey(); got != "field-key" {
		t.Errorf("tmdbKey field: got %q", got)
	}

	t.Setenv("BUKANIR_OPENSUBTITLES_USER", "env-user")
	empty := Config{}
	if got := empty.openSubtitlesUser(); got != "env-user" {
		t.Errorf("openSubtitlesUser env: got %q, want env-user", got)
	}
	// A set field beats the env var.
	set := Config{OpenSubtitlesUser: "field-user"}
	if got := set.openSubtitlesUser(); got != "field-user" {
		t.Errorf("openSubtitlesUser field: got %q, want field-user", got)
	}
}

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.CacheTTL != DefaultCacheTTL {
		t.Errorf("CacheTTL = %v, want %v", c.CacheTTL, DefaultCacheTTL)
	}
	if c.HTTPTimeout != DefaultHTTPTimeout {
		t.Errorf("HTTPTimeout = %v, want %v", c.HTTPTimeout, DefaultHTTPTimeout)
	}
	if c.UserAgent != DefaultUserAgent {
		t.Errorf("UserAgent = %q, want %q", c.UserAgent, DefaultUserAgent)
	}
}

func TestWithDefaults(t *testing.T) {
	// Zero HTTPTimeout and UserAgent are filled; zero CacheTTL is preserved.
	got := Config{}.withDefaults()
	if got.HTTPTimeout != DefaultHTTPTimeout {
		t.Errorf("HTTPTimeout = %v, want %v", got.HTTPTimeout, DefaultHTTPTimeout)
	}
	if got.UserAgent != DefaultUserAgent {
		t.Errorf("UserAgent = %q, want %q", got.UserAgent, DefaultUserAgent)
	}
	if got.CacheTTL != 0 {
		t.Errorf("CacheTTL = %v, want 0 (caching disabled)", got.CacheTTL)
	}

	// Explicit values are preserved.
	custom := Config{HTTPTimeout: time.Second, UserAgent: "x", CacheTTL: time.Hour}.withDefaults()
	if custom.HTTPTimeout != time.Second || custom.UserAgent != "x" || custom.CacheTTL != time.Hour {
		t.Errorf("withDefaults overwrote explicit values: %+v", custom)
	}
}

func TestSecretValuesRedaction(t *testing.T) {
	c := Config{
		TMDBKey:           "abcdef123456",
		OpenSubtitlesPass: "short", // below the length floor, must be skipped
		SubdlKey:          "subdl_longkeyvalue",
	}
	vals := SecretValues(c)
	joined := strings.Join(vals, ",")
	if !strings.Contains(joined, "abcdef123456") || !strings.Contains(joined, "subdl_longkeyvalue") {
		t.Errorf("expected long secrets present, got %v", vals)
	}
	for _, v := range vals {
		if v == "short" {
			t.Errorf("trivially short secret not filtered")
		}
	}
}

func TestBuiltin(t *testing.T) {
	if got := builtin("=8GbsVGa"); got != "hello" {
		t.Errorf("builtin() = %q, want hello", got)
	}
	if got := builtin(""); got != "" {
		t.Errorf("builtin(empty) = %q", got)
	}
	if got := builtin("not base64!"); got != "" {
		t.Errorf("builtin(garbage) = %q", got)
	}
}

func TestSourceHosts(t *testing.T) {
	var def Config
	if def.tpbHost() != DefaultTPBHost || def.ytsHost() != DefaultYTSHost || def.eztvHost() != DefaultEZTVHost {
		t.Errorf("defaults = %q, %q, %q", def.tpbHost(), def.ytsHost(), def.eztvHost())
	}
	cfg := Config{EZTVHost: "eztv.wf"}
	if got := hostBaseURL(cfg.eztvHost()); got != "https://eztv.wf" {
		t.Errorf("eztv base = %q", got)
	}
	if got := hostBaseURL("example.onion"); got != "http://example.onion" {
		t.Errorf("onion base = %q", got)
	}
	if def.usesOnion() || cfg.usesOnion() {
		t.Error("usesOnion() = true without a .onion host")
	}
	if onion := (Config{TPBHost: "example.onion"}); !onion.usesOnion() {
		t.Error("usesOnion() = false with a .onion TPB host")
	}
}
