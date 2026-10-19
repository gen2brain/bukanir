package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gen2brain/bukanir"
	"github.com/gen2brain/go-mpv"
	"github.com/gen2brain/iup-go/iup"
)

func postStatus(s string) {
	iup.PostMessage(iup.GetHandle("status"), s, 0, nil)
}

// postProgress shows the status-bar progress bar at frac (0..1).
func postProgress(frac float64) {
	iup.PostMessage(iup.GetHandle("progress"), "", int(frac*1000), nil)
}

func postProgressHide() {
	iup.PostMessage(iup.GetHandle("progress"), "hide", 0, nil)
}

// enableControl re-enables a control from any goroutine if it is still in the tree.
func enableControl(h iup.Ihandle) {
	iup.PostMessage(iup.GetHandle("tabs"), "enable", 0, h)
}

func releaseMemory() {
	debug.FreeOSMemory()
}

func humanRate(n int64) string {
	switch {
	case n >= 1<<20:
		return fmt.Sprintf("%.1f MiB", float64(n)/(1<<20))
	case n >= 1<<10:
		return fmt.Sprintf("%.0f KiB", float64(n)/(1<<10))
	}
	return fmt.Sprintf("%d B", n)
}

// onTrailer downloads the trailer, serves it over a local HTTP server, plays it in mpv.
func onTrailer(s bukanir.TSummary, tab, btn iup.Ihandle) {
	disablePlayButton(btn)
	phase := beginPlayback(tab)
	postStatus("Loading trailer...")
	go func() {
		sess := newSession(tab)
		defer releaseMemory()
		defer enableControl(btn)
		defer phase.end()
		defer sess.end()
		keys := s.TrailerKeys
		if len(keys) == 0 {
			keys = []string{s.TrailerKey}
		}
		video, audio, err := client.TrailerStreams(sess.ctx, keys...)
		if err != nil {
			switch {
			case sess.ctx.Err() != nil:
			case errors.Is(err, bukanir.ErrTrailerUnavailable):
				log.Printf("PLAYER: trailer %q: %v", s.Title, err)
				postStatus("Trailer unavailable: YouTube does not allow it to be played here")
			default:
				postStatus("error: " + err.Error())
			}
			return
		}
		defer video.Body.Close()

		url, stop := serveStream(video.Body, video.Size, video.MimeType)
		defer stop()
		audioURL := ""
		if audio != nil {
			defer audio.Body.Close()
			var stopAudio func()
			audioURL, stopAudio = serveStream(audio.Body, audio.Size, audio.MimeType)
			defer stopAudio()
		}

		onStart := func() {
			phase.started()
			postStatus("Playing trailer...")
			log.Printf("PLAYER: trailer %q", s.Title)
		}
		if err := playExternal(sess, s.Title+" - Trailer", url, audioURL, nil, onStart, nil); err != nil {
			log.Printf("ERROR: PLAYER: %v", err)
			postStatus("error: " + err.Error())
		}
		postStatus("Ready")
	}()
}

// onWatch streams the movie's torrent to a local HTTP server and plays it in mpv.
func onWatch(m bukanir.TMovie, s bukanir.TSummary, tab, btn iup.Ihandle) {
	if m.Magnet == "" {
		postStatus("No magnet for this title")
		return
	}
	disablePlayButton(btn)
	phase := beginPlayback(tab)
	postStatus("Starting torrent...")
	go func() {
		sess := newSession(tab)
		defer releaseMemory()
		defer enableControl(btn)
		defer phase.end()
		defer sess.end()
		var dialer bukanir.PeerDialer
		if settings.UseTor {
			postStatus("Starting Tor...")
			d, err := client.PeerDialer()
			if err != nil {
				postStatus("Tor error: " + err.Error())
				return
			}
			dialer = d
		}

		tr, err := bukanir.NewTorrent(m.Magnet, settings.toTorrentConfig(dialer))
		if err != nil {
			postStatus("error: " + err.Error())
			return
		}
		defer tr.Close()
		if m.FileName != "" {
			tr.PreferFile(m.FileName, m.FileIdx)
		}

		metaTimeout := 90 * time.Second
		if settings.UseTor {
			metaTimeout = 180 * time.Second // Tor peer discovery is slower
		}
		ctx, cancel := context.WithTimeout(sess.ctx, metaTimeout)
		defer cancel()
		postStatus("Fetching metadata...")

		metaDone := make(chan struct{})
		go func() {
			ticker := time.NewTicker(5 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-metaDone:
					return
				case <-ticker.C:
					st := tr.Status()
					log.Printf("TORRENT: metadata wait - known=%d pending=%d halfopen=%d active=%d seeds=%d",
						st.KnownPeers, st.PendingPeers, st.HalfOpen, st.Peers, st.Seeds)
				}
			}
		}()
		err = tr.WaitForMetadata(ctx)
		close(metaDone)
		if err != nil {
			if sess.ctx.Err() == nil {
				postStatus("metadata: " + err.Error())
			}
			return
		}
		url, err := tr.Serve()
		if err != nil {
			postStatus("serve: " + err.Error())
			return
		}

		subsCh := make(chan []string, 1)
		go func() {
			if !settings.SubtitlesEnabled {
				log.Println("SUBTITLES: disabled in settings")
				subsCh <- nil
				return
			}
			subs := fetchSubtitles(sess.ctx, s, m)
			if b := torrentSubtitles(sess.ctx, tr); len(b) > 0 {
				log.Printf("SUBTITLES: %d bundled in the torrent", len(b))
				subs = append(subs, b...)
			}
			subsCh <- subs
		}()

		waitForBuffer(sess.ctx, tr)
		subs := <-subsCh
		defer removeFiles(subs)
		if sess.ctx.Err() != nil {
			postProgressHide()
			postStatus("Ready")
			return
		}

		var paused atomic.Bool
		done := make(chan struct{})
		go statsPoller(tr, &paused, done)

		onStart := func() {
			phase.started()
			postProgressHide()
			postStatus("Playing " + s.Title)
			log.Printf("PLAYER: playing %q", s.Title)
		}
		onPause := func(p bool) {
			paused.Store(p)
			phase.setPaused(p)
			if !p {
				postProgressHide()
				postStatus("Playing " + s.Title)
			}
		}
		if err := playExternal(sess, s.Title, url, "", subs, onStart, onPause); err != nil {
			log.Printf("ERROR: PLAYER: %v", err)
			postStatus("error: " + err.Error())
		}
		close(done)
		postProgressHide()
		postStatus("Ready")
	}()
}

func torrentSubtitles(parent context.Context, tr *bukanir.Torrent) []string {
	var paths []string
	for _, f := range tr.Files() {
		if !isSubtitle(f.Path) {
			continue
		}
		ctx, cancel := context.WithTimeout(parent, 30*time.Second)
		r, err := tr.FileReader(ctx, f.Index)
		if err != nil {
			cancel()
			continue
		}
		data, err := io.ReadAll(io.LimitReader(r, 5<<20))
		r.Close()
		cancel()
		if err != nil || len(data) == 0 {
			continue
		}
		if p := writeTempSub(filepath.Base(f.Path), data); p != "" {
			paths = append(paths, p)
		}
	}
	return paths
}

func isSubtitle(p string) bool {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".srt", ".sub", ".ass", ".ssa", ".vtt":
		return true
	}
	return false
}

// fetchSubtitles downloads the best-matching subtitles and returns their temp paths.
const maxSubtitles = 2

func fetchSubtitles(parent context.Context, s bukanir.TSummary, m bukanir.TMovie) []string {
	ctx, cancel := context.WithTimeout(parent, 30*time.Second)
	defer cancel()

	lang := settings.subtitleLanguage()
	subs, err := client.Subtitles(ctx, bukanir.SubtitleQuery{
		IMDbID:   s.IMDbID,
		TMDBID:   m.ID,
		Title:    s.Title,
		Year:     s.Year,
		Season:   m.Season,
		Episode:  m.Episode,
		Language: lang,
		Release:  subtitleRelease(m),
	})
	if err != nil {
		log.Printf("SUBTITLES: search failed (lang=%s imdb=%s): %v", lang, s.IMDbID, err)
		return nil
	}
	log.Printf("SUBTITLES: %d candidates for %q (lang=%s) from %s", len(subs), s.Title, lang, providerCounts(subs))

	var paths []string
	openSubUsed := false
	for _, sub := range subs {
		if len(paths) >= maxSubtitles {
			break
		}
		if sub.Provider == "opensubtitles" && openSubUsed {
			continue // OpenSubtitles has a daily download quota; take at most one
		}
		data, name, err := client.DownloadSubtitle(ctx, sub)
		if err != nil {
			log.Printf("SUBTITLES: download from %s failed: %v", sub.Provider, err)
			continue
		}
		p := writeTempSub(name, data)
		if p == "" {
			continue
		}
		if sub.Provider == "opensubtitles" {
			openSubUsed = true
		}
		log.Printf("SUBTITLES: downloaded %q from %s (score=%.2f)", name, sub.Provider, sub.Score)
		paths = append(paths, p)
	}
	return paths
}

func providerCounts(subs []bukanir.Subtitle) string {
	counts := map[string]int{}
	var order []string
	for _, s := range subs {
		if counts[s.Provider] == 0 {
			order = append(order, s.Provider)
		}
		counts[s.Provider]++
	}
	if len(order) == 0 {
		return "none"
	}
	var parts []string
	for _, p := range order {
		parts = append(parts, fmt.Sprintf("%s=%d", p, counts[p]))
	}
	return strings.Join(parts, " ")
}

func writeTempSub(name string, data []byte) string {
	ext := filepath.Ext(name)
	if ext == "" {
		ext = ".srt"
	}
	base := tmpDir
	os.MkdirAll(base, 0o700)
	f, err := os.CreateTemp(base, "sub-*"+ext)
	if err != nil {
		return ""
	}
	defer f.Close()
	if _, err := f.Write(data); err != nil {
		return ""
	}
	return f.Name()
}

func removeFiles(paths []string) {
	for _, p := range paths {
		os.Remove(p)
	}
}

// statsPoller reports torrent stats to the status bar while paused.
func statsPoller(tr *bukanir.Torrent, paused *atomic.Bool, done chan struct{}) {
	ticker := time.NewTicker(time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if !paused.Load() {
				continue
			}
			st := tr.Status()
			postStatus(fmt.Sprintf("Paused   %.1f%%   down %s/s   up %s/s   %d peers / %d seeds",
				st.Progress*100, humanRate(st.DownloadRate), humanRate(st.UploadRate), st.Peers, st.Seeds))
			postProgress(st.Progress)
		}
	}
}

// waitForBuffer blocks until roughly 1% of the file is downloaded, the deadline passes or ctx ends.
func waitForBuffer(ctx context.Context, tr *bukanir.Torrent) {
	deadline := time.Now().Add(90 * time.Second)
	for time.Now().Before(deadline) && ctx.Err() == nil {
		st := tr.Status()
		target := max(st.TotalSize/100, 1<<20)
		frac := 0.0
		if target > 0 {
			frac = float64(st.Downloaded) / float64(target)
		}
		if frac > 1 {
			frac = 1
		}
		postProgress(frac)
		postStatus(fmt.Sprintf("Buffering %d%%  %s/s  %d peers",
			int(frac*100), humanRate(st.DownloadRate), st.Peers))
		if st.Downloaded >= target {
			return
		}
		select {
		case <-ctx.Done():
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// serveStream serves r once over a local HTTP server and returns its URL and a stop func.
func serveStream(r io.Reader, size int64, contentType string) (url string, stop func()) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return "", func() {}
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", contentType)
		if size > 0 {
			w.Header().Set("Content-Length", strconv.FormatInt(size, 10))
		}
		if req.Method == http.MethodHead {
			return
		}
		io.Copy(w, r)
	})}
	go srv.Serve(ln)
	return "http://" + ln.Addr().String() + "/", func() { srv.Close() }
}

type session struct {
	tab    iup.Ihandle
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.Mutex
	player *mpv.Mpv
}

var (
	sessionsMu sync.Mutex
	sessions   = map[iup.Ihandle][]*session{}
)

func newSession(tab iup.Ihandle) *session {
	ctx, cancel := context.WithCancel(context.Background())
	s := &session{tab: tab, ctx: ctx, cancel: cancel}
	sessionsMu.Lock()
	sessions[tab] = append(sessions[tab], s)
	sessionsMu.Unlock()
	return s
}

func (s *session) end() {
	s.cancel()
	sessionsMu.Lock()
	defer sessionsMu.Unlock()
	list := sessions[s.tab]
	for i, o := range list {
		if o == s {
			sessions[s.tab] = append(list[:i], list[i+1:]...)
			break
		}
	}
	if len(sessions[s.tab]) == 0 {
		delete(sessions, s.tab)
	}
}

func (s *session) setPlayer(m *mpv.Mpv) {
	s.mu.Lock()
	s.player = m
	cancelled := s.ctx.Err() != nil
	s.mu.Unlock()
	if cancelled {
		m.Command([]string{"quit"})
	}
}

func (s *session) quit() {
	s.cancel()
	s.mu.Lock()
	m := s.player
	s.mu.Unlock()
	if m != nil {
		m.Command([]string{"quit"})
	}
}

// stopSessions cancels every playback started from tab and quits its player if one runs.
func stopSessions(tab iup.Ihandle) {
	sessionsMu.Lock()
	list := append([]*session(nil), sessions[tab]...)
	sessionsMu.Unlock()
	for _, s := range list {
		s.quit()
	}
}

// playExternal opens an mpv window and plays url with any external subtitles.
func playExternal(sess *session, title, url, audioURL string, subtitles []string, onStart func(), onPause func(bool)) error {
	m := mpv.New()
	m.SetOptionString("title", title)
	m.SetOptionString("force-media-title", title)
	m.SetOptionString("force-window", "yes")
	m.SetOptionString("geometry", "50%:50%")
	m.SetOptionString("input-default-bindings", "yes")
	m.SetOptionString("input-vo-keyboard", "yes")
	m.SetOption("osc", mpv.FormatFlag, true)
	m.SetOptionString("idle", "no")
	m.SetOptionString("keep-open", "no")
	// Bound the demuxer RAM cache; the torrent already buffers to disk.
	m.SetOptionString("demuxer-max-bytes", "48MiB")
	m.SetOptionString("demuxer-max-back-bytes", "24MiB")
	if settings.MPVConfig {
		m.SetOptionString("config", "yes")
	}

	if err := m.Initialize(); err != nil {
		m.TerminateDestroy()
		return err
	}
	defer m.TerminateDestroy()
	m.SetPropertyString("ytdl", "no")
	applyPlayerSettings(m)
	applyPlayerOptions(m, settings.MPVOptions)
	sess.setPlayer(m)
	m.ObserveProperty(0, "pause", mpv.FormatFlag)

	if err := m.Command([]string{"loadfile", url}); err != nil {
		return err
	}

	started := false
	for {
		e := m.WaitEvent(1)
		if e == nil {
			continue
		}
		switch e.EventID {
		case mpv.EventShutdown:
			return nil
		case mpv.EventFileLoaded:
			if !started {
				started = true
				// Add tracks here; sub-add and audio-add fail silently until the file is loaded.
				if audioURL != "" {
					if err := m.Command([]string{"audio-add", audioURL, "select"}); err != nil {
						log.Printf("PLAYER: audio-add failed: %v", err)
					}
				}
				for i, sub := range subtitles {
					flag := "auto"
					if i == 0 {
						flag = "select"
					}
					if err := m.Command([]string{"sub-add", sub, flag}); err != nil {
						log.Printf("SUBTITLES: sub-add %q failed: %v", sub, err)
					} else {
						log.Printf("SUBTITLES: loaded %s", sub)
					}
				}
				if onStart != nil {
					onStart()
				}
			}
		case mpv.EventPropertyChange:
			if !started || onPause == nil {
				continue
			}
			if p := e.Property(); p.Name == "pause" {
				if v, ok := p.Data.(int); ok {
					onPause(v != 0)
				}
			}
		}
	}
}

// applyPlayerSettings applies the user's player preferences to an initialized mpv instance.
func applyPlayerSettings(m *mpv.Mpv) {
	if settings.Fullscreen {
		m.SetPropertyString("fullscreen", "yes")
	}
	m.SetPropertyString("stop-screensaver", strings.ToLower(yesNo(settings.StopScreensaver)))
	if settings.VolumeMax >= 100 {
		m.SetPropertyString("volume-max", strconv.Itoa(settings.VolumeMax))
	}
	if settings.SubScale > 0 {
		m.SetPropertyString("sub-scale", strconv.FormatFloat(settings.SubScale, 'f', 2, 64))
	}
	if settings.SubColor != "" {
		m.SetPropertyString("sub-color", settings.SubColor)
	}
}

// applyPlayerOptions applies space-separated "name=value" or "--name=value" options from the settings.
func applyPlayerOptions(m *mpv.Mpv, opts string) {
	for _, f := range strings.Fields(opts) {
		name, value, found := strings.Cut(strings.TrimPrefix(f, "--"), "=")
		if !found {
			value = "yes"
			if rest, ok := strings.CutPrefix(name, "no-"); ok {
				name, value = rest, "no"
			}
		}
		if err := m.SetPropertyString(name, value); err != nil {
			log.Printf("PLAYER: option %s=%s: %v", name, value, err)
		}
	}
}

// subtitleRelease is the release name subtitles are matched against: the episode file for a pack.
func subtitleRelease(m bukanir.TMovie) string {
	if m.FileName == "" {
		return m.Release
	}
	base := filepath.Base(m.FileName)
	return strings.TrimSuffix(base, filepath.Ext(base))
}
