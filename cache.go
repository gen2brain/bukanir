package bukanir

import (
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// cache is an on-disk cache of gzip-compressed responses.
type cache struct {
	dir string
	ttl time.Duration
}

func newCache(dir string, ttl time.Duration) *cache {
	return &cache{dir: dir, ttl: ttl}
}

func (c *cache) enabled() bool { return c != nil && c.dir != "" && c.ttl > 0 }

// key hashes its parts into a stable filename-safe cache key.
func (c *cache) key(parts ...string) string {
	sum := sha256.Sum256([]byte(strings.Join(parts, "\x00")))
	return hex.EncodeToString(sum[:])
}

func (c *cache) path(key string) string {
	return filepath.Join(c.dir, key+".gz")
}

// get returns the cached bytes for key if present and still fresh.
func (c *cache) get(key string) ([]byte, bool) {
	if !c.enabled() {
		return nil, false
	}
	path := c.path(key)
	info, err := os.Stat(path)
	if err != nil || time.Since(info.ModTime()) > c.ttl {
		return nil, false
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	zr, err := gzip.NewReader(f)
	if err != nil {
		return nil, false
	}
	defer zr.Close()
	data, err := io.ReadAll(zr)
	if err != nil {
		return nil, false
	}
	return data, true
}

// put stores data for key, replacing any existing entry.
func (c *cache) put(key string, data []byte) error {
	if !c.enabled() {
		return nil
	}
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return err
	}
	var buf bytes.Buffer
	zw, _ := gzip.NewWriterLevel(&buf, gzip.BestCompression)
	if _, err := zw.Write(data); err != nil {
		return err
	}
	if err := zw.Close(); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(c.dir, "tmp-*")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(buf.Bytes()); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	return os.Rename(tmpName, c.path(key))
}
