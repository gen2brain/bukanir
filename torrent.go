package bukanir

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	"strconv"
	"strings"
	"sync"
	"time"

	atorrent "github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/mse"
	"github.com/anacrolix/torrent/storage"
	"github.com/huin/goupnp/dcps/internetgateway2"
	"golang.org/x/time/rate"
)

var (
	gwMu    sync.Mutex
	gwCache []gateway
)

// PeerDialer dials BitTorrent peer connections.
type PeerDialer interface {
	DialContext(ctx context.Context, network, addr string) (net.Conn, error)
}

// TorrentConfig configures a streaming Torrent session.
type TorrentConfig struct {
	// DownloadDir stores torrent data that KeepFiles keeps.
	DownloadDir string
	// TempDir holds the temp data dir; empty uses the system temp dir.
	TempDir string
	// ListenPort is the BitTorrent peer port. Zero picks a free one.
	ListenPort int
	// HTTPHost/HTTPPort bind the streaming server.
	HTTPHost string
	HTTPPort int
	// MaxConns caps peer connections per torrent. Zero uses the library default.
	MaxConns int
	// DownloadRate/UploadRate cap throughput in KiB/s. Zero is unlimited.
	DownloadRate int
	UploadRate   int
	// KeepFiles keeps downloaded data in DownloadDir after Close; otherwise data goes to a temp dir removed on Close.
	KeepFiles bool
	// Seed keeps uploading after the download completes.
	Seed bool
	// RequireEncryption drops peers that do not encrypt the whole connection; otherwise they connect in plain.
	RequireEncryption bool
	// Dialer, when set, carries outgoing TCP peers (e.g. Tor); uTP/DHT/incoming direct.
	Dialer PeerDialer
}

// TorrentStatus is a snapshot of a session's progress.
type TorrentStatus struct {
	Name         string  `json:"name"`
	State        string  `json:"state"` // metadata, downloading, seeding
	Progress     float64 `json:"progress"`
	DownloadRate int64   `json:"downloadRate"` // bytes/s
	UploadRate   int64   `json:"uploadRate"`   // bytes/s
	Downloaded   int64   `json:"downloaded"`
	TotalSize    int64   `json:"totalSize"`
	Peers        int     `json:"peers"`
	Seeds        int     `json:"seeds"`
	KnownPeers   int     `json:"knownPeers"`   // discovered (trackers + DHT)
	PendingPeers int     `json:"pendingPeers"` // discovered, not yet connected
	HalfOpen     int     `json:"halfOpen"`     // connection in progress
}

// TorrentFile describes one file in the torrent.
type TorrentFile struct {
	Index    int     `json:"index"`
	Path     string  `json:"path"`
	Size     int64   `json:"size"`
	Progress float64 `json:"progress"`
}

// Torrent streams a single torrent over HTTP while it downloads.
type Torrent struct {
	cfg     TorrentConfig
	client  *atorrent.Client
	store   storage.ClientImplCloser
	t       *atorrent.Torrent
	tmpDir  string
	fileIdx int

	prefName string
	prefIdx  int
	prefSet  bool

	server   *http.Server
	addr     string
	portStop func()

	stop chan struct{}
	wg   sync.WaitGroup

	mu       sync.Mutex
	prevDown int64
	prevUp   int64
	dnRate   int64
	upRate   int64
}

// NewTorrent creates a streaming session for a magnet link.
func NewTorrent(magnet string, cfg TorrentConfig) (*Torrent, error) {
	dir, tmp := cfg.DownloadDir, ""
	if dir == "" || !cfg.KeepFiles {
		d, err := os.MkdirTemp(cfg.TempDir, "bukanir-torrent-*")
		if err != nil {
			return nil, err
		}
		dir, tmp = d, d
	}

	cc := atorrent.NewDefaultClientConfig()
	cc.DataDir = dir
	pc, err := storage.NewBoltPieceCompletion(dir)
	if err != nil {
		removeTemp(tmp)
		return nil, err
	}
	// plain file I/O: the mmap default keeps the whole file resident while streaming
	store := storage.NewFileOpts(storage.NewFileClientOpts{ClientBaseDir: dir, DisableMmap: true, PieceCompletion: pc})
	cc.DefaultStorage = store
	cc.Seed = cfg.Seed
	cc.NoDefaultPortForwarding = true
	cc.Slogger = slog.New(slog.DiscardHandler)
	cc.HeaderObfuscationPolicy = atorrent.HeaderObfuscationPolicy{Preferred: true, RequirePreferred: cfg.RequireEncryption}
	cc.CryptoProvides = mse.CryptoMethodRC4
	cc.CryptoSelector = cryptoSelector(cfg.RequireEncryption)
	if cfg.MaxConns > 0 {
		cc.EstablishedConnsPerTorrent = cfg.MaxConns
	}
	if cfg.DownloadRate > 0 {
		cc.DownloadRateLimiter = rate.NewLimiter(rate.Limit(cfg.DownloadRate*1024), cfg.DownloadRate*1024)
	}
	if cfg.UploadRate > 0 {
		cc.UploadRateLimiter = rate.NewLimiter(rate.Limit(cfg.UploadRate*1024), cfg.UploadRate*1024)
	}
	if cfg.Dialer != nil {
		cc.DisableTCP = true
		cc.DialForPeerConns = true
		cc.AcceptPeerConnections = true
		cc.HandshakesTimeout = 20 * time.Second
		if cfg.MaxConns == 0 {
			cc.EstablishedConnsPerTorrent = 200
		}
	}
	if cfg.ListenPort > 0 {
		cc.ListenPort = cfg.ListenPort
	} else if p, err := FreePort(""); err == nil {
		cc.ListenPort = p
	}

	// map the port before the client binds it
	var portStop func()
	if cfg.Dialer != nil && cc.ListenPort > 0 {
		portStop = mapPort(cc.ListenPort)
	}

	cl, err := atorrent.NewClient(cc)
	if err != nil {
		if portStop != nil {
			portStop()
		}
		store.Close()
		removeTemp(tmp)
		return nil, err
	}
	if cfg.Dialer != nil {
		cl.AddDialer(atorrent.NetworkDialer{Network: "tcp", Dialer: cfg.Dialer})
	}

	tt, err := cl.AddMagnet(magnet)
	if err != nil {
		if portStop != nil {
			portStop()
		}
		cl.Close()
		store.Close()
		removeTemp(tmp)
		return nil, err
	}

	t := &Torrent{cfg: cfg, client: cl, store: store, t: tt, tmpDir: tmp, fileIdx: -1, stop: make(chan struct{}), portStop: portStop}
	t.wg.Add(1)
	go t.sampleRates()
	logf("TORRENT: added %s, fetching metadata", tt.InfoHash().HexString())
	return t, nil
}

// WaitForMetadata blocks until the torrent's metadata is available or ctx is done.
func (t *Torrent) WaitForMetadata(ctx context.Context) error {
	select {
	case <-t.t.GotInfo():
		if err := t.selectFile(); err != nil {
			logf("ERROR: TORRENT: %s", err)
			return err
		}
		logf("TORRENT: metadata %q", t.t.Name())
		return nil
	case <-ctx.Done():
		logf("ERROR: TORRENT: metadata: %s", ctx.Err())
		return ctx.Err()
	}
}

// PreferFile streams the file with this name, or the one at index, instead of the largest; call it before WaitForMetadata.
func (t *Torrent) PreferFile(name string, index int) {
	t.mu.Lock()
	t.prefName, t.prefIdx, t.prefSet = name, index, true
	t.mu.Unlock()
}

// selectFile picks the file to stream and deprioritizes the rest.
func (t *Torrent) selectFile() error {
	files := t.t.Files()
	t.mu.Lock()
	name, prefIdx, pref := t.prefName, t.prefIdx, t.prefSet
	t.mu.Unlock()

	idx := -1
	if pref {
		paths := make([]string, len(files))
		for i, f := range files {
			paths[i] = f.DisplayPath()
		}
		idx = preferredFileIndex(paths, name, prefIdx)
	}
	if idx < 0 {
		largest := int64(-1)
		for i, f := range files {
			if f.Length() > largest {
				largest, idx = f.Length(), i
			}
		}
	}
	if pref && idx >= 0 && files[idx].Length() > maxTorrentSize {
		return fmt.Errorf("bukanir: episode file is %s, over the %s limit", humanizeBytes(files[idx].Length()), humanizeBytes(maxTorrentSize))
	}

	t.mu.Lock()
	t.fileIdx = idx
	t.mu.Unlock()
	for i, f := range files {
		if i == idx {
			f.SetPriority(atorrent.PiecePriorityNormal)
		} else {
			f.SetPriority(atorrent.PiecePriorityNone)
		}
	}
	return nil
}

// preferredFileIndex finds name among the file paths, falling back to index; -1 when neither fits.
func preferredFileIndex(paths []string, name string, index int) int {
	if name != "" {
		base := strings.ToLower(path.Base(name))
		for i, p := range paths {
			if strings.ToLower(path.Base(p)) == base {
				return i
			}
		}
	}
	if index >= 0 && index < len(paths) {
		return index
	}
	return -1
}

// Serve starts the streaming HTTP server and returns its base URL.
func (t *Torrent) Serve() (string, error) {
	host := t.cfg.HTTPHost
	if host == "" {
		host = "127.0.0.1"
	}
	port := t.cfg.HTTPPort
	if port == 0 {
		p, err := FreePort(host)
		if err != nil {
			return "", err
		}
		port = p
	}
	ln, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(port)))
	if err != nil {
		return "", err
	}
	t.addr = ln.Addr().String()

	mux := http.NewServeMux()
	mux.HandleFunc("/", t.handleStream)
	mux.HandleFunc("/status", t.handleStatus)
	mux.HandleFunc("/files", t.handleFiles)
	t.server = &http.Server{Handler: mux}
	go t.server.Serve(ln)
	logf("TORRENT: serving %s", t.URL())
	return t.URL(), nil
}

// URL returns the streaming base URL, or "" if Serve has not been called.
func (t *Torrent) URL() string {
	if t.addr == "" {
		return ""
	}
	return "http://" + t.addr + "/"
}

func (t *Torrent) handleStream(w http.ResponseWriter, r *http.Request) {
	idx := t.streamIndex()
	if q := r.URL.Query().Get("file"); q != "" {
		if n, err := strconv.Atoi(q); err == nil {
			idx = n
		}
	}
	files := t.t.Files()
	if idx < 0 || idx >= len(files) {
		http.Error(w, "no such file", http.StatusNotFound)
		return
	}
	f := files[idx]

	reader := f.NewReader()
	reader.SetReadahead(5 << 20)
	reader.SetContext(r.Context())
	defer reader.Close()

	http.ServeContent(w, r, path.Base(f.Path()), time.Time{}, reader)
}

func (t *Torrent) handleStatus(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, t.Status())
}

func (t *Torrent) handleFiles(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, t.Files())
}

// Status returns a snapshot of the session's progress.
func (t *Torrent) Status() TorrentStatus {
	st := t.t.Stats()
	completed, length := t.t.BytesCompleted(), t.t.Length()

	var progress float64
	if length > 0 {
		progress = float64(completed) / float64(length)
	}
	state := "metadata"
	if t.t.Info() != nil {
		if t.t.Seeding() && completed >= length {
			state = "seeding"
		} else {
			state = "downloading"
		}
	}

	t.mu.Lock()
	dn, up := t.dnRate, t.upRate
	t.mu.Unlock()

	return TorrentStatus{
		Name:         t.t.Name(),
		State:        state,
		Progress:     progress,
		DownloadRate: dn,
		UploadRate:   up,
		Downloaded:   completed,
		TotalSize:    length,
		Peers:        st.ActivePeers,
		Seeds:        st.ConnectedSeeders,
		KnownPeers:   st.TotalPeers,
		PendingPeers: st.PendingPeers,
		HalfOpen:     st.HalfOpenPeers,
	}
}

// Files lists the torrent's files. It returns nil before metadata is available.
func (t *Torrent) Files() []TorrentFile {
	if t.t.Info() == nil {
		return nil
	}
	files := t.t.Files()
	out := make([]TorrentFile, len(files))
	for i, f := range files {
		var p float64
		if f.Length() > 0 {
			p = float64(f.BytesCompleted()) / float64(f.Length())
		}
		out[i] = TorrentFile{Index: i, Path: f.DisplayPath(), Size: f.Length(), Progress: p}
	}
	return out
}

// FileReader returns a reader for the file at index; reads block on needed pieces until ctx is done.
func (t *Torrent) FileReader(ctx context.Context, index int) (io.ReadCloser, error) {
	if t.t.Info() == nil {
		return nil, fmt.Errorf("bukanir: torrent has no metadata")
	}
	files := t.t.Files()
	if index < 0 || index >= len(files) {
		return nil, fmt.Errorf("bukanir: file index %d out of range", index)
	}
	r := files[index].NewReader()
	r.SetContext(ctx)
	return r, nil
}

func (t *Torrent) streamIndex() int {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.fileIdx
}

// sampleRates recomputes download/upload rates once per second.
func (t *Torrent) sampleRates() {
	defer t.wg.Done()
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-t.stop:
			return
		case <-ticker.C:
			st := t.t.Stats()
			down := st.ConnStats.BytesReadData.Int64()
			up := st.ConnStats.BytesWrittenData.Int64()
			t.mu.Lock()
			t.dnRate = down - t.prevDown
			t.upRate = up - t.prevUp
			t.prevDown, t.prevUp = down, up
			t.mu.Unlock()
		}
	}
}

// Close stops the server, the torrent client, and removes temp data.
func (t *Torrent) Close() error {
	if t.portStop != nil {
		t.portStop()
	}
	close(t.stop)
	t.wg.Wait()
	if t.server != nil {
		t.server.Close()
	}
	t.client.Close()
	t.store.Close()
	removeTemp(t.tmpDir)
	return nil
}

func removeTemp(dir string) {
	if dir != "" {
		os.RemoveAll(dir)
	}
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// cryptoSelector picks RC4 for incoming encrypted peers, falling back to plaintext unless encryption is required.
func cryptoSelector(require bool) mse.CryptoSelector {
	return func(provided mse.CryptoMethod) mse.CryptoMethod {
		if provided&mse.CryptoMethodRC4 != 0 {
			return mse.CryptoMethodRC4
		}
		if require {
			return 0
		}
		return mse.CryptoMethodPlaintext
	}
}

// FreePort returns a TCP port that is free on host at the moment of the call.
func FreePort(host string) (int, error) {
	l, err := net.Listen("tcp", net.JoinHostPort(host, "0"))
	if err != nil {
		return 0, err
	}
	defer l.Close()
	return l.Addr().(*net.TCPAddr).Port, nil
}

// FreePortInRange returns the first bindable TCP port in [low, high] on host.
func FreePortInRange(host string, low, high int) (int, error) {
	if low < 1 || high > 65535 || low > high {
		return 0, fmt.Errorf("bukanir: invalid port range %d-%d", low, high)
	}
	for p := low; p <= high; p++ {
		l, err := net.Listen("tcp", net.JoinHostPort(host, strconv.Itoa(p)))
		if err != nil {
			continue
		}
		l.Close()
		return p, nil
	}
	return 0, fmt.Errorf("bukanir: no free port in range %d-%d", low, high)
}

// gateway is a discovered IGD WAN service we can send SOAP to directly.
type gateway struct {
	ctrlURL string // SOAP control endpoint
	urn     string // service type
	client  string // our LAN IP toward the gateway
}

// cachedGateways discovers IGD gateways once and caches a non-empty result.
func cachedGateways() []gateway {
	gwMu.Lock()
	defer gwMu.Unlock()
	if len(gwCache) == 0 {
		gwCache = discoverGateways()
	}
	return gwCache
}

// warmGateways discovers gateways in the background to unblock the first map.
func warmGateways() { go cachedGateways() }

// discoverGateways finds IGD WAN connection services via goupnp.
func discoverGateways() []gateway {
	var out []gateway
	add := func(loc *url.URL, endpoint url.URL, urn string) {
		out = append(out, gateway{ctrlURL: endpoint.String(), urn: urn, client: localIP(loc)})
	}
	if cs, _, err := internetgateway2.NewWANIPConnection1Clients(); err == nil {
		for _, c := range cs {
			add(c.Location, c.SOAPClient.EndpointURL, c.Service.ServiceType)
		}
	}
	if cs, _, err := internetgateway2.NewWANPPPConnection1Clients(); err == nil {
		for _, c := range cs {
			add(c.Location, c.SOAPClient.EndpointURL, c.Service.ServiceType)
		}
	}
	return out
}

// localIP is this host's address on the route to the gateway.
func localIP(loc *url.URL) string {
	conn, err := net.Dial("udp", loc.Host)
	if err != nil {
		return ""
	}
	defer conn.Close()
	return conn.LocalAddr().(*net.UDPAddr).IP.String()
}

// soapCall sends a SOAP action and returns the sanitized UTF-8 body and HTTP status.
func soapCall(ctrlURL, urn, action, inner string) (string, int, error) {
	env := `<?xml version="1.0"?><s:Envelope xmlns:s="http://schemas.xmlsoap.org/soap/envelope/" s:encodingStyle="http://schemas.xmlsoap.org/soap/encoding/"><s:Body>` + inner + `</s:Body></s:Envelope>`
	req, err := http.NewRequest("POST", ctrlURL, strings.NewReader(env))
	if err != nil {
		return "", 0, err
	}
	req.Header.Set("Content-Type", `text/xml; charset="utf-8"`)
	req.Header.Set("SOAPAction", `"`+urn+"#"+action+`"`)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", 0, err
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return strings.ToValidUTF8(string(raw), ""), resp.StatusCode, nil
}

func addMapping(g gateway, proto string, port, leaseSecs int) error {
	inner := fmt.Sprintf(`<u:AddPortMapping xmlns:u="%s"><NewRemoteHost></NewRemoteHost><NewExternalPort>%d</NewExternalPort><NewProtocol>%s</NewProtocol><NewInternalPort>%d</NewInternalPort><NewInternalClient>%s</NewInternalClient><NewEnabled>1</NewEnabled><NewPortMappingDescription>bukanir</NewPortMappingDescription><NewLeaseDuration>%d</NewLeaseDuration></u:AddPortMapping>`,
		g.urn, port, proto, port, g.client, leaseSecs)
	body, status, err := soapCall(g.ctrlURL, g.urn, "AddPortMapping", inner)
	if err != nil {
		return err
	}
	if status == http.StatusOK {
		return nil
	}
	return fmt.Errorf("status %d, %s", status, soapErrorCode(body))
}

func deleteMapping(g gateway, proto string, port int) {
	inner := fmt.Sprintf(`<u:DeletePortMapping xmlns:u="%s"><NewRemoteHost></NewRemoteHost><NewExternalPort>%d</NewExternalPort><NewProtocol>%s</NewProtocol></u:DeletePortMapping>`, g.urn, port, proto)
	soapCall(g.ctrlURL, g.urn, "DeletePortMapping", inner)
}

func soapErrorCode(body string) string {
	_, after, ok := strings.Cut(body, "<errorCode>")
	if !ok {
		return "no error code"
	}
	if code, _, ok := strings.Cut(after, "</errorCode>"); ok {
		return "error " + code
	}
	return "no error code"
}

// mapPort maps the listen port (TCP+UDP) via UPnP, renewing until the returned func is called.
func mapPort(port int) func() {
	gws := cachedGateways()
	if len(gws) == 0 {
		logf("UPNP: no gateway found; incoming peers unavailable")
		return func() {}
	}
	const lease = 30 * time.Minute
	protos := []string{"TCP", "UDP"}
	remap := func() {
		for _, g := range gws {
			for _, proto := range protos {
				deleteMapping(g, proto, port)
				if err := addMapping(g, proto, port, int(lease/time.Second)); err != nil {
					logf("UPNP: map %s port %d failed: %v", proto, port, err)
				} else {
					logf("UPNP: mapped %s port %d", proto, port)
				}
			}
		}
	}
	remap()
	done := make(chan struct{})
	go func() {
		t := time.NewTicker(lease - 5*time.Minute)
		defer t.Stop()
		for {
			select {
			case <-done:
				for _, g := range gws {
					for _, proto := range protos {
						deleteMapping(g, proto, port)
					}
				}
				return
			case <-t.C:
				remap()
			}
		}
	}()
	return func() { close(done) }
}
