package main

import (
	"bytes"
	"crypto/md5"
	"encoding/hex"
	"image"
	_ "image/jpeg"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/gen2brain/iup-go/iup"
	xdraw "golang.org/x/image/draw"
)

const (
	posterW = 200
	posterH = 300
)

var posterHTTP = &http.Client{Timeout: 20 * time.Second}

// loadPoster fetches, decodes, and resizes a poster, delivering it via PostMessage.
func loadPoster(url string, target iup.Ihandle) {
	loadPosterSized(url, target, posterW, posterH)
}

func loadPosterSized(url string, target iup.Ihandle, w, h int) {
	if url == "" {
		return
	}
	data, err := fetchPoster(url)
	if err != nil {
		return
	}
	src, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return
	}
	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	if targetAlive(target) {
		iup.PostMessage(target, "", 0, dst)
	}
}

// fetchPoster returns the poster bytes, using an on-disk cache.
func fetchPoster(url string) ([]byte, error) {
	dir := posterCacheDir()
	sum := md5.Sum([]byte(url))
	path := filepath.Join(dir, hex.EncodeToString(sum[:]))
	if b, err := os.ReadFile(path); err == nil {
		return b, nil
	}
	resp, err := posterHTTP.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(path, b, 0o644)
	return b, nil
}

func posterCacheDir() string {
	return filepath.Join(cacheDir, "images")
}
