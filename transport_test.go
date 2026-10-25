package bukanir

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type fakeTor struct {
	client *http.Client
	closed bool
}

func (f *fakeTor) HTTPClient() *http.Client { return f.client }
func (f *fakeTor) DialContext(context.Context, string, string) (net.Conn, error) {
	return nil, errors.New("fakeTor: dial not supported")
}
func (f *fakeTor) Close() error { f.closed = true; return nil }

// swapTorProvider replaces the package factory for a test and restores it.
func swapTorProvider(t *testing.T, fn func(context.Context, Config, string, int) (torProvider, error)) {
	t.Helper()
	orig := newTorProvider
	newTorProvider = fn
	t.Cleanup(func() { newTorProvider = orig })
}

func TestTransportDirect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	// Tor disabled: no provider should ever be built.
	swapTorProvider(t, func(context.Context, Config, string, int) (torProvider, error) {
		t.Fatal("newTorProvider called with Tor disabled")
		return nil, nil
	})

	tr := newTransport(context.Background(), Config{Tor: false}, "")
	c, err := tr.client()
	if err != nil {
		t.Fatal(err)
	}
	if c != tr.direct {
		t.Fatal("expected the direct client")
	}
	resp, err := c.Get(srv.URL)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

func TestTransportTorLazyOnce(t *testing.T) {
	var calls int32
	fake := &fakeTor{client: &http.Client{}}
	swapTorProvider(t, func(_ context.Context, _ Config, _ string, n int) (torProvider, error) {
		atomic.AddInt32(&calls, 1)
		if n != DefaultTorInstances {
			t.Errorf("instances = %d, want %d", n, DefaultTorInstances)
		}
		return fake, nil
	})

	tr := newTransport(context.Background(), Config{Tor: true}, "")

	// Concurrent first-use must bootstrap exactly once.
	var wg sync.WaitGroup
	for range 16 {
		wg.Go(func() {
			c, err := tr.clientFor("http://example.onion/")
			if err != nil || c != fake.client {
				t.Errorf("clientFor(.onion) = %v, %v", c, err)
			}
		})
	}
	wg.Wait()

	if calls != 1 {
		t.Fatalf("provider built %d times, want 1", calls)
	}
	if fake.client.Timeout != DefaultHTTPTimeout {
		t.Errorf("timeout not applied to tor client: %v", fake.client.Timeout)
	}

	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
	if !fake.closed {
		t.Error("Close did not close the provider")
	}
}

func TestTransportTorErrorCached(t *testing.T) {
	var calls int32
	swapTorProvider(t, func(context.Context, Config, string, int) (torProvider, error) {
		atomic.AddInt32(&calls, 1)
		return nil, errors.New("bootstrap failed")
	})

	tr := newTransport(context.Background(), Config{Tor: true, TorInstances: 1}, "")
	for range 3 {
		if _, err := tr.clientFor("http://example.onion/"); err == nil {
			t.Fatal("expected error")
		}
	}
	if calls != 1 {
		t.Fatalf("factory called %d times, want 1 (error cached)", calls)
	}
	// Close with no provider must be a no-op.
	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestTransportCloseDuringBootstrap(t *testing.T) {
	started := make(chan struct{})
	release := make(chan struct{})
	fake := &fakeTor{client: &http.Client{}}
	swapTorProvider(t, func(context.Context, Config, string, int) (torProvider, error) {
		close(started)
		<-release
		return fake, nil
	})

	tr := newTransport(context.Background(), Config{Tor: true, TorInstances: 1}, "")
	done := make(chan error, 1)
	go func() { done <- tr.ensureTor() }()
	<-started
	if st, _ := tr.torStatus(); st != TorStarting {
		t.Errorf("state while bootstrapping = %d, want TorStarting", st)
	}

	if err := tr.Close(); err != nil {
		t.Fatal(err)
	}
	close(release)
	if err := <-done; !errors.Is(err, errTransportClosed) {
		t.Fatalf("ensureTor after Close = %v, want errTransportClosed", err)
	}
	if !fake.closed {
		t.Error("provider finished after Close was not closed")
	}
	<-tr.done
}

func TestTransportTorStatus(t *testing.T) {
	fail := errors.New("no network")
	for _, tc := range []struct {
		name string
		err  error
		want TorState
	}{
		{"ready", nil, TorReady},
		{"failed", fail, TorFailed},
	} {
		t.Run(tc.name, func(t *testing.T) {
			swapTorProvider(t, func(context.Context, Config, string, int) (torProvider, error) {
				if tc.err != nil {
					return nil, tc.err
				}
				return &fakeTor{client: &http.Client{}}, nil
			})
			tr := newTransport(context.Background(), Config{Tor: true, TorInstances: 1}, "")
			if st, _ := tr.torStatus(); st != TorOff {
				t.Errorf("state before start = %d, want TorOff", st)
			}
			tr.ensureTor()
			<-tr.done
			st, err := tr.torStatus()
			if st != tc.want || !errors.Is(err, tc.err) {
				t.Errorf("state = %d, %v; want %d, %v", st, err, tc.want, tc.err)
			}
			tr.Close()
			if st, _ := tr.torStatus(); st != TorOff {
				t.Errorf("state after Close = %d, want TorOff", st)
			}
		})
	}
}

func TestTransportDoneWithoutStart(t *testing.T) {
	tr := newTransport(context.Background(), Config{Tor: true}, "")
	tr.Close()
	select {
	case <-tr.done:
	default:
		t.Fatal("done not closed by Close of an unstarted transport")
	}
	if err := tr.ensureTor(); !errors.Is(err, errTransportClosed) {
		t.Errorf("ensureTor after Close = %v, want errTransportClosed", err)
	}
}

func TestTransportInstancesFromConfig(t *testing.T) {
	swapTorProvider(t, func(_ context.Context, _ Config, _ string, n int) (torProvider, error) {
		if n != 5 {
			t.Errorf("instances = %d, want 5", n)
		}
		return &fakeTor{client: &http.Client{}}, nil
	})
	tr := newTransport(context.Background(), Config{Tor: true, TorInstances: 5, HTTPTimeout: time.Second}, "")
	if _, err := tr.clientFor("http://example.onion/"); err != nil {
		t.Fatal(err)
	}
}
