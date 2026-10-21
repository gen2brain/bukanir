//go:build media

package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"math"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gen2brain/iup-go/iup"
)

// engineChoices are the IupVideo engines offered in the settings, in AUTO order.
var engineChoices = []string{"NATIVE", "MPV", "VLC", "GSTREAMER"}

// availableEngines returns the engines IupVideo can load, optionally only those that play a separate audio URL.
func availableEngines(separateAudio bool) []string {
	v := iup.Video()
	defer v.Destroy()
	if separateAudio {
		v.SetAttribute("AUDIOURL", "http://127.0.0.1/")
	}
	var engines []string
	for _, e := range engineChoices {
		v.SetAttribute("BACKEND", e)
		if v.GetAttribute("AVAILABLE") == "YES" {
			engines = append(engines, e)
		}
	}
	return engines
}

// playbackEngines returns the engines a playback tries in order: the chosen one, or every available one for AUTO.
func playbackEngines(separateAudio bool) []string {
	engines := availableEngines(separateAudio)
	for _, e := range engines {
		if e == settings.Engine {
			return []string{e}
		}
	}
	return engines
}

// audioDevices returns the names of the audio outputs of engine, without the default output.
func audioDevices(engine string) []string {
	v := iup.Video()
	defer v.Destroy()
	v.SetAttribute("BACKEND", engine)
	var names []string
	for i, count := 1, v.GetInt("AUDIODEVICECOUNT"); i < count; i++ {
		names = append(names, v.GetAttribute("AUDIODEVICENAME"+strconv.Itoa(i)))
	}
	return names
}

// audioDeviceIndex returns the AUDIODEVICE index of the output named name, or 0 for the default output.
func audioDeviceIndex(v iup.Ihandle, name string) int {
	if name == "" {
		return 0
	}
	for i, count := 1, v.GetInt("AUDIODEVICECOUNT"); i < count; i++ {
		if v.GetAttribute("AUDIODEVICENAME"+strconv.Itoa(i)) == name {
			return i
		}
	}
	return 0
}

// engineName is the label of an engine in the settings and the player.
func engineName(engine string) string {
	switch engine {
	case "NATIVE":
		return "System"
	case "MPV":
		return "mpv"
	case "VLC":
		return "VLC"
	case "GSTREAMER":
		return "GStreamer"
	}
	return "Automatic"
}

// subtitleFile is an external subtitle file offered in the player.
type subtitleFile struct {
	path, name string
}

// playback describes one video to play in a player window.
type playback struct {
	title     string
	url       string
	audioURL  string
	subtitles []subtitleFile
	engines   []string
	onStart   func()
	onPause   func(bool)
}

// runOnMain runs fn on the main loop.
func runOnMain(fn func()) {
	iup.PostMessage(iup.GetHandle("tabs"), "run", 0, fn)
}

// playVideo opens a player window for pb and blocks until it closes; called from a playback goroutine.
func playVideo(sess *session, pb playback) error {
	if len(pb.engines) == 0 {
		return errors.New("no video engine is available")
	}
	done := make(chan error, 1)
	runOnMain(func() { openVideoPlayer(sess, pb, done) })
	return <-done
}

// videoPlayer is one player window.
type videoPlayer struct {
	sess   *session
	pb     playback
	done   chan error
	err    error
	engine int

	dlg, video, controls, playBtn, elapsed, position, subBtn, muteBtn, volume, fullBtn, timer iup.Ihandle

	started, closed, chosen bool
	seek                    float64
	delay                   float64
	hold, notice            int
	tracks, subtitle, last  int
}

func openVideoPlayer(sess *session, pb playback, done chan error) {
	if sess.ctx.Err() != nil {
		done <- nil
		return
	}
	p := &videoPlayer{sess: sess, pb: pb, done: done, seek: -1}
	p.build()
	sess.setPlayer(p)
	iup.ShowXY(p.dlg, iup.CENTER, iup.CENTER)
	if settings.Fullscreen {
		p.setFullscreen(true)
	}
	iup.SetFocus(p.video)
	p.load()
	p.timer.SetAttribute("RUN", "YES")
}

func (p *videoPlayer) build() {
	r, g, b := parseHexColor(settings.SubColor)
	p.video = iup.Video().SetAttributes(map[string]string{
		"EXPAND": "YES", "CANFOCUS": "YES", "RASTERSIZE": "960x540",
		"SUBTITLEFONTSIZE": strconv.Itoa(int(math.Round(18 * settings.SubScale))),
		"SUBTITLECOLOR":    fmt.Sprintf("%d %d %d", r, g, b),
		"MPVCONFIG":        yesNo(settings.MPVConfig),
		"VLCCONFIG":        yesNo(settings.VLCConfig),
	})
	if settings.MPVOptions != "" {
		p.video.SetAttribute("MPVOPTIONS", settings.MPVOptions)
	}
	if settings.VLCOptions != "" {
		p.video.SetAttribute("VLCOPTIONS", settings.VLCOptions)
	}
	p.video.SetCallback("STATECHANGED_CB", iup.StateChangedFunc(p.onState))
	p.video.SetCallback("ERROR_CB", iup.ErrorFunc(p.onError))
	p.video.SetCallback("PLAYEND_CB", iup.PlayEndFunc(func(iup.Ihandle) int {
		p.close()
		return iup.DEFAULT
	}))
	p.video.SetCallback("BUTTON_CB", iup.ButtonFunc(func(_ iup.Ihandle, button, pressed, _, _ int, status string) int {
		if button == iup.BUTTON1 && pressed != 0 && iup.IsDouble(status) {
			p.setFullscreen(p.dlg.GetAttribute("FULLSCREEN") != "YES")
		}
		return iup.DEFAULT
	}))

	p.playBtn = iconButton("player-pause", p.togglePause).SetAttribute("TIP", "Pause (Space)")
	p.elapsed = iup.Label("0:00:00 / 0:00:00").SetAttributes("ELLIPSIS=YES, SIZE=100x")
	p.position = iup.Val("HORIZONTAL").SetAttributes("MIN=0, MAX=1, EXPAND=HORIZONTAL, ACTIVE=NO")
	p.position.SetCallback("VALUECHANGED_CB", iup.ValueChangedFunc(func(ih iup.Ihandle) int {
		p.seek = ih.GetDouble("VALUE") * p.video.GetDouble("DURATION")
		p.hold = 2
		return iup.DEFAULT
	}))
	p.subBtn = iconButton("player-subtitles", func() { p.cycleSubtitle(1) }).SetAttributes("TITLE=-/0, IMAGEPOSITION=LEFT, SPACING=4")
	p.subBtn.SetCallback("BUTTON_CB", iup.ButtonFunc(func(_ iup.Ihandle, button, pressed, _, _ int, _ string) int {
		if button == iup.BUTTON3 && pressed == 0 {
			p.cycleSubtitle(-1)
		}
		return iup.DEFAULT
	}))
	p.muteBtn = iconButton("player-volume", p.toggleMute).SetAttribute("TIP", "Mute (m)")
	p.volume = iup.Val("HORIZONTAL").SetAttributes("MIN=0, MAX=100, VALUE=100")
	p.volume.SetCallback("VALUECHANGED_CB", iup.ValueChangedFunc(func(ih iup.Ihandle) int {
		p.video.SetAttribute("VOLUME", strconv.Itoa(ih.GetInt("VALUE")))
		return iup.DEFAULT
	}))
	p.fullBtn = iconButton("player-fullscreen", func() { p.setFullscreen(true) }).SetAttribute("TIP", "Fullscreen (f)")

	p.controls = iup.Hbox(p.playBtn, p.elapsed, p.position, p.subBtn, p.muteBtn, p.volume, p.fullBtn).SetAttributes("ALIGNMENT=ACENTER, GAP=6, NMARGIN=6x4")
	p.dlg = iup.Dialog(iup.Vbox(p.video, p.controls)).SetAttributes(map[string]string{
		"TITLE": plainText(p.pb.title), "ICON": "appicon", "SHRINK": "YES",
	})
	p.dlg.SetCallback("CLOSE_CB", iup.CloseFunc(func(iup.Ihandle) int {
		p.close()
		return iup.IGNORE
	}))
	p.dlg.SetCallback("K_ANY", iup.KAnyFunc(p.onKey))

	p.timer = iup.Timer().SetAttribute("TIME", "250")
	p.timer.SetCallback("ACTION_CB", iup.TimerActionFunc(func(iup.Ihandle) int {
		p.tick()
		return iup.DEFAULT
	}))
}

// setIcon switches an icon button to another tinted icon.
func setIcon(b iup.Ihandle, icon string) {
	monoButtons[b] = icon
	b.SetAttribute("IMAGE", tintedName(icon))
}

// load opens the source with the current engine; the first downloaded subtitle is shown until a track in the subtitle language appears.
func (p *videoPlayer) load() {
	engine := p.pb.engines[p.engine]
	p.video.SetAttribute("URL", nil)
	p.video.SetAttribute("BACKEND", engine)
	p.video.SetAttribute("AUDIODEVICE", strconv.Itoa(audioDeviceIndex(p.video, settings.AudioDevice)))
	if p.pb.audioURL != "" {
		p.video.SetAttribute("AUDIOURL", p.pb.audioURL)
	}
	p.video.SetAttribute("URL", p.pb.url)
	p.tracks, p.subtitle, p.last, p.chosen = 0, -1, 0, false
	if len(p.pb.subtitles) > 0 {
		p.selectSubtitle(1, false)
	}
	if p.delay != 0 {
		p.video.SetAttribute("SUBTITLEDELAY", strconv.FormatFloat(p.delay, 'f', 1, 64))
	}
	p.video.SetAttribute("PLAY", "YES")
	p.updateSubtitleButton()
	log.Printf("PLAYER: %q with %s", p.pb.title, engineName(engine))
}

func (p *videoPlayer) onState(_ iup.Ihandle, state string) int {
	switch state {
	case "PLAYING":
		setIcon(p.playBtn, "player-pause")
		p.playBtn.SetAttribute("TIP", "Pause (Space)")
		if !p.started {
			p.started = true
			if p.pb.onStart != nil {
				p.pb.onStart()
			}
		} else if p.pb.onPause != nil {
			p.pb.onPause(false)
		}
	case "PAUSED", "STOPPED":
		setIcon(p.playBtn, "player-play")
		p.playBtn.SetAttribute("TIP", "Play (Space)")
		if p.started && p.pb.onPause != nil {
			p.pb.onPause(true)
		}
	}
	return iup.DEFAULT
}

func (p *videoPlayer) onError(ih iup.Ihandle, message string) int {
	if ih.GetAttribute("STATE") != "ERROR" {
		log.Printf("SUBTITLES: %s", message)
		p.notify(message)
		return iup.DEFAULT
	}
	log.Printf("PLAYER: %s: %s", engineName(p.pb.engines[p.engine]), message)
	if !p.started && p.engine+1 < len(p.pb.engines) {
		p.engine++
		p.load()
		return iup.DEFAULT
	}
	p.err = errors.New(message)
	p.close()
	return iup.DEFAULT
}

// selectSubtitle shows subtitle n: 0 is off, embedded tracks come first, then the downloaded files.
func (p *videoPlayer) selectSubtitle(n int, announce bool) {
	switch {
	case n <= 0:
		n = 0
		p.video.SetAttribute("SUBTITLE", "OFF")
	case n <= p.tracks:
		p.video.SetAttribute("SUBTITLE", strconv.Itoa(n))
	default:
		p.video.SetAttribute("SUBTITLEFILE", p.pb.subtitles[n-p.tracks-1].path)
	}
	p.subtitle = n
	if n > 0 {
		p.last = n
	}
	p.updateSubtitleButton()
	if announce {
		p.notify("Subtitles: " + p.subtitleName(n))
	}
}

// cycleSubtitle moves through the subtitles and off, like the mpv subtitle button.
func (p *videoPlayer) cycleSubtitle(delta int) {
	states := p.tracks + len(p.pb.subtitles) + 1
	if states == 1 {
		return
	}
	p.chosen = true
	p.selectSubtitle(((max(p.subtitle, 0)+delta)%states+states)%states, true)
}

// toggleSubtitles hides the subtitles or shows the last shown one.
func (p *videoPlayer) toggleSubtitles() {
	p.chosen = true
	if p.subtitle > 0 {
		p.selectSubtitle(0, true)
	} else if p.tracks+len(p.pb.subtitles) > 0 {
		p.selectSubtitle(max(p.last, 1), true)
	}
}

func (p *videoPlayer) subtitleName(n int) string {
	switch {
	case n <= 0:
		return "Off"
	case n <= p.tracks:
		name := p.video.GetAttribute("SUBTITLENAME" + strconv.Itoa(n))
		lang := p.video.GetAttribute("SUBTITLELANGUAGE" + strconv.Itoa(n))
		switch {
		case name == "" && lang == "":
			return fmt.Sprintf("Track %d", n)
		case name == "":
			return lang
		case lang != "":
			return name + " (" + lang + ")"
		}
		return name
	}
	return p.pb.subtitles[n-p.tracks-1].name
}

func (p *videoPlayer) updateSubtitleButton() {
	total := p.tracks + len(p.pb.subtitles)
	title := fmt.Sprintf("%d/%d", max(p.subtitle, 0), total)
	tip := "Subtitles: " + p.subtitleName(p.subtitle)
	if p.subtitle < 0 {
		title, tip = fmt.Sprintf("-/%d", total), "Subtitles: as flagged in the file"
	}
	p.subBtn.SetAttribute("TITLE", title)
	p.subBtn.SetAttribute("TIP", plainText(tip)+" (j)")
	p.subBtn.SetAttribute("ACTIVE", yesNo(total > 0))
}

// updateTracks follows the embedded subtitle tracks, which become known after loading, and prefers one in the subtitle language.
func (p *videoPlayer) updateTracks() {
	count := p.video.GetInt("SUBTITLECOUNT")
	if count == p.tracks {
		return
	}
	if p.subtitle > p.tracks {
		p.subtitle += count - p.tracks
	}
	if p.last > p.tracks {
		p.last += count - p.tracks
	}
	p.tracks = count
	if !p.chosen {
		for i := 1; i <= count; i++ {
			if languageMatches(p.video.GetAttribute("SUBTITLELANGUAGE"+strconv.Itoa(i)), settings.subtitleLanguage()) {
				p.selectSubtitle(i, false)
				break
			}
		}
	}
	p.updateSubtitleButton()
}

// languageCodes are the ISO 639-2 codes of the subtitle languages.
var languageCodes = map[string][]string{
	"ar": {"ara"}, "bg": {"bul"}, "cs": {"cze", "ces"}, "da": {"dan"}, "de": {"ger", "deu"}, "el": {"gre", "ell"},
	"en": {"eng"}, "es": {"spa"}, "fa": {"per", "fas"}, "fi": {"fin"}, "fr": {"fre", "fra"}, "he": {"heb"},
	"hi": {"hin"}, "hr": {"hrv"}, "hu": {"hun"}, "id": {"ind"}, "it": {"ita"}, "ja": {"jpn"}, "ko": {"kor"},
	"nl": {"dut", "nld"}, "no": {"nor", "nob", "nno"}, "pl": {"pol"}, "pt": {"por"}, "ro": {"rum", "ron"},
	"ru": {"rus"}, "sk": {"slo", "slk"}, "sl": {"slv"}, "sr": {"srp"}, "sv": {"swe"}, "th": {"tha"}, "tr": {"tur"},
	"uk": {"ukr"}, "vi": {"vie"}, "zh": {"chi", "zho"},
}

// languageMatches reports whether a track language, as a code or a name, is the language with the given ISO 639-1 code.
func languageMatches(lang, code string) bool {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		return false
	}
	if lang == code || strings.HasPrefix(lang, code+"-") || strings.HasPrefix(lang, code+"_") || slices.Contains(languageCodes[code], lang) {
		return true
	}
	for _, l := range subtitleLanguages {
		if l.Code == code && strings.EqualFold(l.Name, lang) {
			return true
		}
	}
	return false
}

// notify shows a short message in place of the time.
func (p *videoPlayer) notify(message string) {
	p.elapsed.SetAttribute("TITLE", plainText(message))
	p.notice = 8
}

func (p *videoPlayer) tick() {
	if p.closed {
		return
	}
	p.updateTracks()
	if p.seek >= 0 {
		p.video.SetAttribute("POSITION", strconv.FormatFloat(p.seek, 'f', 3, 64))
		p.seek = -1
	}
	current, duration := p.video.GetDouble("POSITION"), p.video.GetDouble("DURATION")
	seekable := p.video.GetAttribute("SEEKABLE") == "YES" && duration > 0
	if p.position.GetAttribute("ACTIVE") != yesNo(seekable) {
		p.position.SetAttribute("ACTIVE", yesNo(seekable))
	}
	switch state := p.video.GetAttribute("STATE"); {
	case p.notice > 0:
		p.notice--
	case state == "LOADING":
		p.elapsed.SetAttribute("TITLE", "Loading...")
	case state == "BUFFERING":
		p.elapsed.SetAttribute("TITLE", "Buffering...")
	default:
		p.elapsed.SetAttribute("TITLE", clock(current)+" / "+clock(duration))
	}
	if p.hold > 0 {
		p.hold--
	} else if seekable {
		p.position.SetAttribute("VALUE", strconv.FormatFloat(current/duration, 'f', 6, 64))
	}
}

// clock formats seconds as h:mm:ss, or m:ss under an hour.
func clock(seconds float64) string {
	s := int(seconds + 0.5)
	if s >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", s/3600, s/60%60, s%60)
	}
	return fmt.Sprintf("%d:%02d", s/60, s%60)
}

func (p *videoPlayer) togglePause() {
	if p.video.GetAttribute("STATE") == "PLAYING" || p.video.GetAttribute("STATE") == "BUFFERING" {
		p.video.SetAttribute("PAUSE", "YES")
	} else {
		p.video.SetAttribute("PLAY", "YES")
	}
}

func (p *videoPlayer) toggleMute() {
	mute := p.video.GetAttribute("MUTE") != "YES"
	p.video.SetAttribute("MUTE", yesNo(mute))
	if mute {
		setIcon(p.muteBtn, "player-volume-off")
		p.notify("Mute")
	} else {
		setIcon(p.muteBtn, "player-volume")
		p.notify(fmt.Sprintf("Volume %d%%", p.volume.GetInt("VALUE")))
	}
}

func (p *videoPlayer) seekBy(delta float64) {
	if p.video.GetAttribute("SEEKABLE") != "YES" {
		return
	}
	base := p.seek
	if base < 0 {
		base = p.video.GetDouble("POSITION")
	}
	target := max(base+delta, 0)
	if d := p.video.GetDouble("DURATION"); d > 0 {
		target = min(target, d)
	}
	p.seek = target
	p.hold = 2
}

func (p *videoPlayer) setVolume(volume int) {
	volume = min(max(volume, 0), 100)
	p.volume.SetAttribute("VALUE", strconv.Itoa(volume))
	p.video.SetAttribute("VOLUME", strconv.Itoa(volume))
	p.notify(fmt.Sprintf("Volume %d%%", volume))
}

// adjust changes a picture adjustment for this playback, like the mpv keys 1 to 8.
func (p *videoPlayer) adjust(attribute, label string, delta int) {
	value := min(max(p.video.GetInt(attribute)+delta, -100), 100)
	p.video.SetAttribute(attribute, strconv.Itoa(value))
	p.notify(fmt.Sprintf("%s %d", label, value))
}

func (p *videoPlayer) addDelay(delta float64) {
	p.delay = math.Round((p.delay+delta)*10) / 10
	p.video.SetAttribute("SUBTITLEDELAY", strconv.FormatFloat(p.delay, 'f', 1, 64))
	p.notify(fmt.Sprintf("Subtitle delay %+.1f s", p.delay))
}

// onKey handles the mpv keys the player supports.
func (p *videoPlayer) onKey(_ iup.Ihandle, key int) int {
	switch key {
	case iup.K_ESC:
		if p.dlg.GetAttribute("FULLSCREEN") != "YES" {
			return iup.CONTINUE
		}
		p.setFullscreen(false)
	case iup.K_SP, iup.K_p:
		p.togglePause()
	case iup.K_f:
		p.setFullscreen(p.dlg.GetAttribute("FULLSCREEN") != "YES")
	case iup.K_m:
		p.toggleMute()
	case iup.K_q:
		p.close()
	case iup.K_v:
		p.toggleSubtitles()
	case iup.K_j:
		p.cycleSubtitle(1)
	case iup.K_J:
		p.cycleSubtitle(-1)
	case iup.K_z:
		p.addDelay(-0.1)
	case iup.K_x:
		p.addDelay(0.1)
	case iup.K_LEFT:
		p.seekBy(-5)
	case iup.K_RIGHT:
		p.seekBy(5)
	case iup.K_DOWN:
		p.seekBy(-60)
	case iup.K_UP:
		p.seekBy(60)
	case iup.K_9:
		p.setVolume(p.volume.GetInt("VALUE") - 2)
	case iup.K_0:
		p.setVolume(p.volume.GetInt("VALUE") + 2)
	case iup.K_1:
		p.adjust("CONTRAST", "Contrast", -1)
	case iup.K_2:
		p.adjust("CONTRAST", "Contrast", 1)
	case iup.K_3:
		p.adjust("BRIGHTNESS", "Brightness", -1)
	case iup.K_4:
		p.adjust("BRIGHTNESS", "Brightness", 1)
	case iup.K_5:
		p.adjust("GAMMA", "Gamma", -1)
	case iup.K_6:
		p.adjust("GAMMA", "Gamma", 1)
	case iup.K_7:
		p.adjust("SATURATION", "Saturation", -1)
	case iup.K_8:
		p.adjust("SATURATION", "Saturation", 1)
	default:
		return iup.CONTINUE
	}
	return iup.IGNORE
}

// setFullscreen shows only the video over the whole screen, or restores the window with its controls.
func (p *videoPlayer) setFullscreen(on bool) {
	p.controls.SetAttribute("FLOATING", yesNo(on))
	p.controls.SetAttribute("VISIBLE", yesNo(!on))
	p.dlg.SetAttribute("FULLSCREEN", yesNo(on))
	iup.Refresh(p.dlg)
	iup.SetFocus(p.video)
}

// close stops playback and destroys the window on the next loop pass; the playback goroutine then finishes.
func (p *videoPlayer) close() {
	if p.closed {
		return
	}
	p.closed = true
	p.timer.SetAttribute("RUN", "NO")
	p.video.SetAttribute("URL", nil)
	iup.Hide(p.dlg)
	p.sess.setPlayer(nil)
	for _, b := range []iup.Ihandle{p.playBtn, p.subBtn, p.muteBtn, p.fullBtn} {
		delete(monoButtons, b)
	}
	runOnMain(func() {
		p.timer.Destroy()
		p.dlg.Destroy()
		p.done <- p.err
	})
}

// streamFile serves a downloading stream over HTTP with range requests, reading what is already downloaded.
type streamFile struct {
	file    *os.File
	size    int64
	mu      sync.Mutex
	cond    *sync.Cond
	written int64
	done    bool
	err     error
}

// serveStream copies r of the given size to a temp file and serves it on a local URL until stop is called.
func serveStream(r io.Reader, size int64, contentType string) (url string, stop func(), err error) {
	if size <= 0 {
		return "", nil, errors.New("stream size is unknown")
	}
	f, err := os.CreateTemp(tmpDir, "stream-*")
	if err != nil {
		return "", nil, err
	}
	sf := &streamFile{file: f, size: size}
	sf.cond = sync.NewCond(&sf.mu)
	go sf.fill(r)

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		sf.close()
		return "", nil, err
	}
	srv := &http.Server{Handler: http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", contentType)
		http.ServeContent(w, req, "", time.Time{}, &streamReader{sf: sf, ctx: req.Context()})
	})}
	go srv.Serve(ln)
	stop = func() {
		srv.Close()
		sf.close()
	}
	return "http://" + ln.Addr().String() + "/" + filepath.Base(f.Name()), stop, nil
}

func (sf *streamFile) fill(r io.Reader) {
	buf := make([]byte, 256<<10)
	for {
		n, err := r.Read(buf)
		if n > 0 {
			sf.mu.Lock()
			if sf.done {
				sf.mu.Unlock()
				return
			}
			_, werr := sf.file.WriteAt(buf[:n], sf.written)
			if werr == nil {
				sf.written += int64(n)
			}
			sf.mu.Unlock()
			sf.cond.Broadcast()
			if werr != nil {
				err = werr
			}
		}
		if err != nil {
			sf.mu.Lock()
			if err != io.EOF && sf.err == nil {
				sf.err = err
			}
			if sf.written < sf.size && sf.err == nil {
				sf.err = io.ErrUnexpectedEOF
			}
			sf.mu.Unlock()
			sf.cond.Broadcast()
			return
		}
	}
}

func (sf *streamFile) close() {
	sf.mu.Lock()
	if !sf.done {
		sf.done = true
		sf.file.Close()
		os.Remove(sf.file.Name())
	}
	sf.mu.Unlock()
	sf.cond.Broadcast()
}

// streamReader is one request's view of a streamFile; reads wait for the bytes to arrive.
type streamReader struct {
	sf     *streamFile
	ctx    context.Context
	offset int64
}

func (s *streamReader) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
	case io.SeekCurrent:
		offset += s.offset
	case io.SeekEnd:
		offset += s.sf.size
	}
	if offset < 0 {
		return 0, errors.New("negative offset")
	}
	s.offset = offset
	return offset, nil
}

func (s *streamReader) Read(p []byte) (int, error) {
	sf := s.sf
	if s.offset >= sf.size {
		return 0, io.EOF
	}
	stop := context.AfterFunc(s.ctx, sf.cond.Broadcast)
	defer stop()
	sf.mu.Lock()
	for sf.written <= s.offset && !sf.done && sf.err == nil && s.ctx.Err() == nil {
		sf.cond.Wait()
	}
	if sf.done || s.ctx.Err() != nil {
		sf.mu.Unlock()
		return 0, io.ErrClosedPipe
	}
	if sf.written <= s.offset {
		err := sf.err
		sf.mu.Unlock()
		return 0, err
	}
	size := int64(len(p))
	if avail := sf.written - s.offset; size > avail {
		size = avail
	}
	n, err := sf.file.ReadAt(p[:size], s.offset)
	sf.mu.Unlock()
	s.offset += int64(n)
	if err == io.EOF {
		err = nil
	}
	return n, err
}
