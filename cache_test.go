package bukanir

import (
	"context"
	"os"
	"testing"
	"time"
)

func TestCacheRoundTrip(t *testing.T) {
	c := newCache(t.TempDir(), time.Hour)
	key := c.key("search", "matrix")
	if err := c.put(key, []byte("payload")); err != nil {
		t.Fatal(err)
	}
	got, ok := c.get(key)
	if !ok || string(got) != "payload" {
		t.Fatalf("get = %q, %v; want payload, true", got, ok)
	}
}

func TestCacheMiss(t *testing.T) {
	c := newCache(t.TempDir(), time.Hour)
	if _, ok := c.get(c.key("absent")); ok {
		t.Fatal("expected miss for absent key")
	}
}

func TestCacheExpiry(t *testing.T) {
	c := newCache(t.TempDir(), time.Hour)
	key := c.key("k")
	if err := c.put(key, []byte("v")); err != nil {
		t.Fatal(err)
	}
	// Age the file past the TTL.
	old := time.Now().Add(-2 * time.Hour)
	if err := os.Chtimes(c.path(key), old, old); err != nil {
		t.Fatal(err)
	}
	if _, ok := c.get(key); ok {
		t.Fatal("expected expired entry to miss")
	}
}

func TestCacheDisabled(t *testing.T) {
	for _, c := range []*cache{
		newCache("", time.Hour),  // no dir
		newCache(t.TempDir(), 0), // no ttl
		newCache(t.TempDir(), -time.Second),
	} {
		if err := c.put(c.key("k"), []byte("v")); err != nil {
			t.Errorf("disabled put returned error: %v", err)
		}
		if _, ok := c.get(c.key("k")); ok {
			t.Error("disabled cache returned a hit")
		}
	}
}

func TestCacheKeyStable(t *testing.T) {
	c := newCache(t.TempDir(), time.Hour)
	if k1, k2 := c.key("a", "b"), c.key("a", "b"); k1 != k2 {
		t.Error("key not stable")
	}
	if c.key("a", "b") == c.key("ab", "") {
		t.Error("key collision across part boundaries")
	}
}

func TestCacheLookupRefresh(t *testing.T) {
	c := newCache(t.TempDir(), time.Hour)
	key := c.key("category", "201")
	if err := c.put(key, []byte("old")); err != nil {
		t.Fatal(err)
	}
	if got, ok := c.lookup(context.Background(), key); !ok || string(got) != "old" {
		t.Fatalf("lookup = %q, %v; want old, true", got, ok)
	}
	if _, ok := c.lookup(Refresh(context.Background()), key); ok {
		t.Fatal("lookup with Refresh hit the cache")
	}
}
