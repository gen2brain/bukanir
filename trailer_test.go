package bukanir

import (
	"context"
	"io"
	"strings"
	"testing"
	"time"
)

func TestTrailer(t *testing.T) {
	if testing.Short() {
		t.Skip("resolves a real YouTube stream; skipped under -short")
	}
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// "Me at the zoo" — the first, and very stable, YouTube video.
	url, err := c.Trailer(ctx, "jNQXAC9IVRw")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(url, "http") || !strings.Contains(url, "googlevideo.com") {
		t.Fatalf("unexpected trailer URL: %q", url)
	}
}

func TestTrailerBadID(t *testing.T) {
	if testing.Short() {
		t.Skip("contacts YouTube; skipped under -short")
	}
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if _, err := c.Trailer(ctx, "!!!not-a-video!!!"); err == nil {
		t.Fatal("expected error for invalid video id")
	}
}

func TestTrailerStream(t *testing.T) {
	if testing.Short() {
		t.Skip("streams a real YouTube video; skipped under -short")
	}
	c := testClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	start := time.Now()
	s, err := c.Summary(ctx, 660120, CategoryMovies, 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("summary in %v, trailer key %q", time.Since(start), s.TrailerKey)
	if s.TrailerKey == "" {
		t.Skip("no trailer key for the movie")
	}

	start = time.Now()
	stream, size, err := c.TrailerStream(ctx, s.TrailerKey)
	if err != nil {
		t.Fatal(err)
	}
	defer stream.Close()
	t.Logf("stream resolved in %v, size %d", time.Since(start), size)

	start = time.Now()
	buf := make([]byte, 256<<10)
	n, err := io.ReadFull(stream, buf)
	t.Logf("read %d bytes in %v, err %v", n, time.Since(start), err)
	if n == 0 {
		t.Fatal("no data from the trailer stream")
	}
}
