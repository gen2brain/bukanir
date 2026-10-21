package bukanir

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"sync/atomic"
	"testing"
	"time"

	atorrent "github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/bencode"
	"github.com/anacrolix/torrent/metainfo"
	"github.com/anacrolix/torrent/mse"
)

// nopDialer disables real networking (TCP/uTP/DHT) so a session stays offline.
type nopDialer struct{}

func (nopDialer) DialContext(context.Context, string, string) (net.Conn, error) {
	return nil, errors.New("offline")
}

func TestTorrentNoMetadata(t *testing.T) {
	magnet := "magnet:?xt=urn:btih:aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa&dn=test"
	tr, err := NewTorrent(magnet, TorrentConfig{DownloadDir: t.TempDir(), Dialer: nopDialer{}})
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()

	url, err := tr.Serve()
	if err != nil {
		t.Fatal(err)
	}
	if url == "" || tr.URL() == "" {
		t.Fatal("empty stream URL")
	}

	st := tr.Status()
	if st.State != "metadata" {
		t.Errorf("state = %q, want metadata", st.State)
	}
	if len(tr.Files()) != 0 {
		t.Errorf("files before metadata = %d, want 0", len(tr.Files()))
	}

	// /status endpoint responds.
	resp, err := http.Get(url + "status")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != 200 {
		t.Errorf("/status = %d", resp.StatusCode)
	}
}

// seedFile seeds content from a fresh client and returns it with the magnet link.
func seedFile(t *testing.T, name string, content []byte, opts ...func(*atorrent.ClientConfig)) (*atorrent.Client, string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, name), content, 0o644); err != nil {
		t.Fatal(err)
	}
	return seedPath(t, dir, name, opts...)
}

// seedPath seeds the file or directory name inside dir and returns the client with the magnet link.
func seedPath(t *testing.T, dir, name string, opts ...func(*atorrent.ClientConfig)) (*atorrent.Client, string) {
	t.Helper()
	info := metainfo.Info{PieceLength: 64 * 1024}
	if err := info.BuildFromFilePath(filepath.Join(dir, name)); err != nil {
		t.Fatal(err)
	}
	mi := &metainfo.MetaInfo{}
	var err error
	mi.InfoBytes, err = bencode.Marshal(info)
	if err != nil {
		t.Fatal(err)
	}

	port, _ := FreePort("")
	cfg := atorrent.NewDefaultClientConfig()
	cfg.DataDir = dir
	cfg.Seed = true
	cfg.NoDHT = true
	cfg.DisableUTP = true
	cfg.ListenPort = port
	for _, o := range opts {
		o(cfg)
	}
	cl, err := atorrent.NewClient(cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { cl.Close() })

	tor, err := cl.AddTorrent(mi)
	if err != nil {
		t.Fatal(err)
	}
	<-tor.GotInfo()
	tor.VerifyDataContext(context.Background())

	magnet := "magnet:?xt=urn:btih:" + mi.HashInfoBytes().HexString() + "&dn=" + name
	return cl, magnet
}

func TestTorrentStreamLocal(t *testing.T) {
	if testing.Short() {
		t.Skip("spins up two torrent clients; skipped under -short")
	}
	content := make([]byte, 300*1024)
	for i := range content {
		content[i] = byte(i * 7)
	}
	seeder, magnet := seedFile(t, "movie.bin", content)

	tr, err := NewTorrent(magnet, TorrentConfig{DownloadDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()

	// Connect directly to the seeder instead of relying on trackers/DHT.
	if n := tr.t.AddClientPeer(seeder); n == 0 {
		t.Fatal("no seeder peers added")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tr.WaitForMetadata(ctx); err != nil {
		t.Fatalf("metadata: %v", err)
	}

	url, err := tr.Serve()
	if err != nil {
		t.Fatal(err)
	}

	// Stream the file back over HTTP and verify the bytes.
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(content) {
		t.Fatalf("streamed %d bytes, want %d", len(got), len(content))
	}
	for i := range got {
		if got[i] != content[i] {
			t.Fatalf("byte %d differs", i)
		}
	}

	files := tr.Files()
	if len(files) != 1 || files[0].Path != "movie.bin" || files[0].Size != int64(len(content)) {
		t.Errorf("files = %+v", files)
	}
}

func TestCryptoSelector(t *testing.T) {
	for _, tc := range []struct {
		require  bool
		provided mse.CryptoMethod
		want     mse.CryptoMethod
	}{
		{false, mse.AllSupportedCrypto, mse.CryptoMethodRC4},
		{false, mse.CryptoMethodRC4, mse.CryptoMethodRC4},
		{false, mse.CryptoMethodPlaintext, mse.CryptoMethodPlaintext},
		{true, mse.AllSupportedCrypto, mse.CryptoMethodRC4},
		{true, mse.CryptoMethodPlaintext, 0},
	} {
		if got := cryptoSelector(tc.require)(tc.provided); got != tc.want {
			t.Errorf("require=%v provided=%d: got %d, want %d", tc.require, tc.provided, got, tc.want)
		}
	}
}

func TestTorrentEncryption(t *testing.T) {
	if testing.Short() {
		t.Skip("spins up torrent clients; skipped under -short")
	}
	for _, tc := range []struct {
		name    string
		require bool
		plain   bool
		connect bool
	}{
		{"prefer", false, false, true},
		{"prefer plain peer", false, true, true},
		{"require", true, false, true},
		{"require plain peer", true, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var rc4Only atomic.Bool
			seeder, magnet := seedFile(t, "movie.bin", bytes.Repeat([]byte(t.Name()), 4096), func(cc *atorrent.ClientConfig) {
				if tc.plain {
					cc.HeaderObfuscationPolicy = atorrent.HeaderObfuscationPolicy{Preferred: false, RequirePreferred: true}
				}
				sel := cc.CryptoSelector
				cc.CryptoSelector = func(p mse.CryptoMethod) mse.CryptoMethod {
					if p == mse.CryptoMethodRC4 {
						rc4Only.Store(true)
					}
					return sel(p)
				}
			})

			tr, err := NewTorrent(magnet, TorrentConfig{DownloadDir: t.TempDir(), RequireEncryption: tc.require})
			if err != nil {
				t.Fatal(err)
			}
			defer tr.Close()
			if n := tr.t.AddClientPeer(seeder); n == 0 {
				t.Fatal("no seeder peers added")
			}

			timeout := 30 * time.Second
			if !tc.connect {
				timeout = 5 * time.Second
			}
			ctx, cancel := context.WithTimeout(context.Background(), timeout)
			defer cancel()
			err = tr.WaitForMetadata(ctx)
			if tc.connect && err != nil {
				t.Fatalf("metadata: %v", err)
			}
			if !tc.connect && err == nil {
				t.Fatal("got metadata from a plain peer with encryption required")
			}
			if tc.connect && !tc.plain && !rc4Only.Load() {
				t.Error("client did not offer RC4 only")
			}
		})
	}
}

func TestTorrentPreferFile(t *testing.T) {
	if testing.Short() {
		t.Skip("spins up two torrent clients; skipped under -short")
	}
	dir := t.TempDir()
	pack := filepath.Join(dir, "Show.S01")
	if err := os.Mkdir(pack, 0o755); err != nil {
		t.Fatal(err)
	}
	ep1 := make([]byte, 400*1024)
	ep2 := make([]byte, 200*1024)
	for i := range ep1 {
		ep1[i] = byte(i * 3)
	}
	for i := range ep2 {
		ep2[i] = byte(i * 11)
	}
	if err := os.WriteFile(filepath.Join(pack, "Show.S01E01.mkv"), ep1, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(pack, "Show.S01E02.mkv"), ep2, 0o644); err != nil {
		t.Fatal(err)
	}
	seeder, magnet := seedPath(t, dir, "Show.S01")

	tr, err := NewTorrent(magnet, TorrentConfig{DownloadDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()
	tr.PreferFile("Show.S01E02.mkv", 0)
	if n := tr.t.AddClientPeer(seeder); n == 0 {
		t.Fatal("no seeder peers added")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tr.WaitForMetadata(ctx); err != nil {
		t.Fatalf("metadata: %v", err)
	}
	url, err := tr.Serve()
	if err != nil {
		t.Fatal(err)
	}

	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != len(ep2) {
		t.Fatalf("streamed %d bytes, want the smaller preferred episode (%d)", len(got), len(ep2))
	}
	for i := range got {
		if got[i] != ep2[i] {
			t.Fatalf("byte %d differs", i)
		}
	}
}

// countingDialer dials directly but records that the Dialer path was taken.
type countingDialer struct {
	calls atomic.Int32
	d     net.Dialer
}

func (c *countingDialer) DialContext(ctx context.Context, network, addr string) (net.Conn, error) {
	c.calls.Add(1)
	return c.d.DialContext(ctx, network, addr)
}

// TestTorrentStreamViaDialer checks peer connections go through the Dialer end to end.
func TestTorrentStreamViaDialer(t *testing.T) {
	if testing.Short() {
		t.Skip("spins up two torrent clients; skipped under -short")
	}
	content := make([]byte, 300*1024)
	for i := range content {
		content[i] = byte(i * 5)
	}
	seeder, magnet := seedFile(t, "movie.bin", content)

	dialer := &countingDialer{}
	tr, err := NewTorrent(magnet, TorrentConfig{DownloadDir: t.TempDir(), Dialer: dialer})
	if err != nil {
		t.Fatal(err)
	}
	defer tr.Close()

	if n := tr.t.AddClientPeer(seeder); n == 0 {
		t.Fatal("no seeder peers added")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := tr.WaitForMetadata(ctx); err != nil {
		t.Fatalf("metadata: %v", err)
	}

	url, err := tr.Serve()
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	got, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, content) {
		t.Fatalf("streamed %d bytes, want %d", len(got), len(content))
	}
	if dialer.calls.Load() == 0 {
		t.Fatal("peers were not dialed through the provided Dialer")
	}
}

func TestPreferredFileIndex(t *testing.T) {
	paths := []string{"Show.S01/Show.S01E01.mkv", "Show.S01/Show.S01E02.mkv", "Show.S01/sample.mkv"}
	cases := []struct {
		name  string
		index int
		want  int
	}{
		{"Show.S01E02.mkv", 0, 1},
		{"show.s01e02.MKV", 0, 1},
		{"Missing.mkv", 2, 2},
		{"Missing.mkv", 9, -1},
		{"", 0, 0},
	}
	for _, tc := range cases {
		if got := preferredFileIndex(paths, tc.name, tc.index); got != tc.want {
			t.Errorf("preferredFileIndex(%q, %d) = %d, want %d", tc.name, tc.index, got, tc.want)
		}
	}
}

func TestFreePort(t *testing.T) {
	port, err := FreePort("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if port < 1 || port > 65535 {
		t.Fatalf("port %d out of range", port)
	}
	// The reported port must actually be bindable.
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
	if err != nil {
		t.Fatalf("reported free port %d is not bindable: %v", port, err)
	}
	l.Close()
}

func TestFreePortDistinct(t *testing.T) {
	// Hold the first port open so the OS must hand out a different one.
	p1, err := FreePort("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(p1)))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	p2, err := FreePort("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	if p2 == p1 {
		t.Fatalf("FreePort returned the in-use port %d", p1)
	}
}

func TestFreePortInRange(t *testing.T) {
	// Occupy a known port, then require the range to route around it.
	p, err := FreePort("127.0.0.1")
	if err != nil {
		t.Fatal(err)
	}
	l, err := net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(p)))
	if err != nil {
		t.Fatal(err)
	}
	defer l.Close()

	// A one-port range over the occupied port must fail.
	if _, err := FreePortInRange("127.0.0.1", p, p); err == nil {
		t.Errorf("FreePortInRange(%d,%d) succeeded, want error (port in use)", p, p)
	}

	// A wider range must find a free port, and it must not be the occupied one.
	got, err := FreePortInRange("127.0.0.1", p, p+16)
	if err != nil {
		t.Fatalf("FreePortInRange(%d,%d): %v", p, p+16, err)
	}
	if got == p {
		t.Errorf("FreePortInRange returned the occupied port %d", p)
	}
	if got < p || got > p+16 {
		t.Errorf("FreePortInRange returned %d, outside [%d,%d]", got, p, p+16)
	}
}

func TestFreePortInRangeInvalid(t *testing.T) {
	for _, tc := range []struct{ low, high int }{
		{0, 10},
		{100, 50},
		{1, 70000},
	} {
		if _, err := FreePortInRange("127.0.0.1", tc.low, tc.high); err == nil {
			t.Errorf("FreePortInRange(%d,%d) = nil error, want error", tc.low, tc.high)
		}
	}
}
