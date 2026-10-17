package bukanir

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"
)

// testClient builds a Client that talks directly (no Tor) with a temp cache.
func testClient(t *testing.T) *Client {
	t.Helper()
	c, err := New(context.Background(), Config{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { c.Close() })
	return c
}

func TestParseRetryAfter(t *testing.T) {
	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if d, ok := parseRetryAfter("3", now); !ok || d != 3*time.Second {
		t.Errorf("seconds: %v %v", d, ok)
	}
	if d, ok := parseRetryAfter(now.Add(10*time.Second).Format(http.TimeFormat), now); !ok || d <= 0 {
		t.Errorf("http-date future: %v %v", d, ok)
	}
	if _, ok := parseRetryAfter("", now); ok {
		t.Error("empty should be false")
	}
	if _, ok := parseRetryAfter("garbage", now); ok {
		t.Error("garbage should be false")
	}
	if _, ok := parseRetryAfter("-5", now); ok {
		t.Error("negative should be false")
	}
}

func TestGetBytesOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("User-Agent") != DefaultUserAgent {
			t.Errorf("user-agent = %q", r.Header.Get("User-Agent"))
		}
		w.Write([]byte("hello"))
	}))
	defer srv.Close()

	body, err := testClient(t).getBytes(context.Background(), srv.URL)
	if err != nil || string(body) != "hello" {
		t.Fatalf("getBytes = %q, %v", body, err)
	}
}

func TestGetBytesRetries429(t *testing.T) {
	old := retryBackoff
	retryBackoff = time.Millisecond
	t.Cleanup(func() { retryBackoff = old })

	var n int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&n, 1) <= 2 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.Write([]byte("finally"))
	}))
	defer srv.Close()

	body, err := testClient(t).getBytes(context.Background(), srv.URL)
	if err != nil || string(body) != "finally" {
		t.Fatalf("getBytes = %q, %v", body, err)
	}
	if n != 3 {
		t.Errorf("server hit %d times, want 3", n)
	}
}

func TestGetBytesError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	if _, err := testClient(t).getBytes(context.Background(), srv.URL); err == nil {
		t.Fatal("expected error on 404")
	}
}

func TestGetBytesContextCancel(t *testing.T) {
	old := retryBackoff
	retryBackoff = 50 * time.Millisecond
	t.Cleanup(func() { retryBackoff = old })

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTooManyRequests) // always rate-limited
	}))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	_, err := testClient(t).getBytes(ctx, srv.URL)
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("err = %v, want context.DeadlineExceeded", err)
	}
}

func TestGetJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"name":"Neo","age":30}`))
	}))
	defer srv.Close()

	var out struct {
		Name string `json:"name"`
		Age  int    `json:"age"`
	}
	if err := testClient(t).getJSON(context.Background(), srv.URL, &out); err != nil {
		t.Fatal(err)
	}
	if out.Name != "Neo" || out.Age != 30 {
		t.Fatalf("decoded %+v", out)
	}
}

func TestNewAndClose(t *testing.T) {
	c, err := New(context.Background(), Config{CacheDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if err := c.Close(); err != nil {
		t.Fatal(err)
	}
}
