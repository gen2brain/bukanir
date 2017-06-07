package main

// #include <mpv/client.h>
// #cgo darwin CFLAGS: -I/usr/local/include
import "C"

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"
	"gitlab.com/hannahxy/go-mpv"

	"github.com/gen2brain/bukanir/lib"
)

// Object3 type
type Object3 struct {
	core.QObject

	_ func() `signal:"paused"`
	_ func() `signal:"unpaused"`
	_ func() `signal:"startFile"`
	_ func() `signal:"fileLoaded"`
	_ func() `signal:"endFile"`
	_ func() `signal:"shutdown"`
}

// Player type
type Player struct {
	*Object3

	Mpv    *mpv.Mpv
	Window *Window

	paused  bool
	started bool
}

// NewPlayer returns new player
func NewPlayer(w *Window) *Player {
	return &Player{NewObject3(w), nil, w, false, false}
}

// Init initialize player
func (p *Player) Init() {
	p.Mpv = mpv.Create()
	p.Mpv.RequestLogMessages("info")
	//p.Mpv.RequestLogMessages("v")

	x := (widgets.QApplication_Desktop().Width() / 2) - (960 / 2)
	p.SetOptionString("geometry", fmt.Sprintf("960+%d+%d", x, p.Window.Y()+100))

	p.SetOptionString("osc", "yes")
	p.SetOptionString("ytdl", "no")

	switch runtime.GOOS {
	case "windows":
		p.SetOptionString("vo", "direct3d,opengl,sdl,null")
		p.SetOptionString("ao", "wasapi,sdl,null")
	case "linux":
		p.SetOptionString("vo", "opengl,sdl,xv,null")
		p.SetOptionString("ao", "alsa,sdl,null")
	case "darwin":
		p.SetOptionString("vo", "opengl,sdl,null")
		p.SetOptionString("ao", "coreaudio,sdl,null")
	}

	p.SetOption("cache-default", mpv.FORMAT_INT64, 128)
	p.SetOption("cache-seek-min", mpv.FORMAT_INT64, 32)
	p.SetOption("cache-secs", mpv.FORMAT_DOUBLE, 1.0)

	p.SetOptionString("input-default-bindings", "yes")
	p.SetOptionString("input-vo-keyboard", "yes")

	if p.Window.Settings.Fullscreen {
		p.SetOptionString("fullscreen", "yes")
	}

	if p.Window.Settings.StopScreensaver {
		p.SetOptionString("stop-screensaver", "yes")
	}

	p.SetOption("volume-max", mpv.FORMAT_INT64, p.Window.Settings.VolumeMax)

	p.SetOption("sub-scale", mpv.FORMAT_DOUBLE, p.Window.Settings.Scale)
	p.SetOptionString("sub-color", p.Window.Settings.Color)
	if strings.ToLower(p.Window.Settings.Codepage) != "auto" {
		p.SetOptionString("sub-codepage", strings.ToLower(p.Window.Settings.Codepage))
	}

	err := p.Mpv.Initialize()
	if err != nil {
		log.Printf("ERROR: Initialize: %s\n", err.Error())
	}
}

// SetOption sets option
func (p *Player) SetOption(name string, format mpv.Format, data interface{}) {
	err := p.Mpv.SetOption(name, format, data)
	if err != nil {
		log.Printf("ERROR: SetOption %s: %s\n", name, err.Error())
	}
}

// SetOptionString sets string option
func (p *Player) SetOptionString(name, value string) {
	err := p.Mpv.SetOptionString(name, value)
	if err != nil {
		log.Printf("ERROR: SetOptionString %s: %s\n", name, err.Error())
	}
}

// Wait waits for torrent to download enough data
func (p *Player) Wait(movie bukanir.TMovie, imdbId string) (bool, string) {
	if !bukanir.TorrentWaitStartup() {
		return false, ""
	}

	var file bukanir.TFileInfo

	retry := 0
	ready := false

	if p.Window.Settings.Subtitles {
		var subDir string
		if p.Window.Settings.KeepFiles && p.Window.Settings.DlPath != "" {
			subDir = p.Window.Settings.DlPath
		} else {
			subDir = tempDir
		}

		_, err := os.Stat(subDir)
		if err == nil {
			go func() {
				p.AddSubtitles(movie, imdbId, subDir)
			}()
		}
	}

	for !ready {
		started := bukanir.TorrentRunning()
		if !started {
			if retry > 3 {
				return false, ""
			}
			retry += 1
		}

		s, err := bukanir.TorrentStatus()
		if err == nil {
			var status bukanir.TStatus
			err = json.Unmarshal([]byte(s), &status)

			if err == nil && status.State != -1 {
				if status.State == 3 {
					d := fmt.Sprintf("D:%.2fkB/s U:%.2fkB/s S:%d (%d) P:%d (%d)",
						status.DownloadRate, status.UploadRate, status.NumSeeds, status.TotalSeeds, status.NumPeers, status.TotalPeers)
					p.Window.LabelStatus.ValueChanged(status.StateStr + "... " + d)
				} else {
					p.Window.LabelStatus.ValueChanged(status.StateStr + "...")
				}

				if status.State >= 3 && !ready {
					f := bukanir.TorrentLargestFile()
					err = json.Unmarshal([]byte(f), &file)
					if err != nil {
						continue
					}

					required := file.Size / 100
					value := float64(status.TotalDownload) / float64(required) * 100
					p.Window.ProgressBar.ValueChanged(int(value))

					if status.TotalDownload >= required {
						p.Window.ProgressBar.SetVisible(false)
						p.Window.LabelStatus.ValueChanged("")
						ready = true
						break
					}
				}
			}
		}

		time.Sleep(1 * time.Second)
	}

	return true, file.Url
}

// Status shows torrent status on pause
func (p *Player) Status() {
	for p.IsPaused() {
		s, err := bukanir.TorrentStatus()
		if err == nil {
			var status bukanir.TStatus
			err = json.Unmarshal([]byte(s), &status)
			if err == nil {
				if status.State == 3 {
					progress := fmt.Sprintf("%s... (%.2f%%)", status.StateStr, status.Progress*100)
					d := fmt.Sprintf(" D:%.2fkB/s U:%.2fkB/s S:%d (%d) P:%d (%d)",
						status.DownloadRate, status.UploadRate, status.NumSeeds, status.TotalSeeds, status.NumPeers, status.TotalPeers)
					p.Window.LabelStatus.ValueChanged(progress + d)
				} else {
					state := fmt.Sprintf("%s...", status.StateStr)
					p.Window.LabelStatus.ValueChanged(state)
				}
			}
		}

		time.Sleep(1 * time.Second)
	}

	p.Window.LabelStatus.ValueChanged("")
}

// AddSubtitles add subtitles to player
func (p *Player) AddSubtitles(m bukanir.TMovie, imdbId string, subDir string) {
	str, err := bukanir.Subtitle(m.Title, m.Year, m.Release, p.Window.Settings.Language, m.Category, m.Season, m.Episode, imdbId)
	if err != nil {
		log.Printf("ERROR: Subtitle: %s\n", err.Error())
		return
	}

	var subs []bukanir.TSubtitle
	err = json.Unmarshal([]byte(str), &subs)
	if err != nil {
		log.Printf("ERROR: Unmarshal: %s\n", err.Error())
		return
	}

	cnt := len(subs)

	if cnt == 0 {
		return
	}

	if cnt >= 5 {
		cnt = 5
	}

	list := make([]*mpv.Node, 0)
	for _, sub := range subs[:cnt] {
		subPath, err := bukanir.UnzipSubtitle(sub.DownloadLink, subDir)

		if err != nil {
			log.Printf("ERROR: UnzipSubtitle: %s\n", err.Error())
			continue
		}

		if subPath != "" && subPath != "empty" {
			list = append(list, &mpv.Node{subPath, mpv.FORMAT_STRING})
		}
	}

	node := &mpv.Node{list, mpv.FORMAT_NODE_ARRAY}
	p.SetOption("sub-file", mpv.FORMAT_NODE, node)
}

// Play plays movie
func (p *Player) Play(url string, title string) {
	if title != "" {
		p.SetOptionString("force-media-title", title)
	}

	err := p.Mpv.Command([]string{"loadfile", url})
	if err != nil {
		log.Printf("ERROR: loadfile: %s\n", err.Error())
	}

	for {
		e := p.Mpv.WaitEvent(10000)
		p.handleEvent(e)

		if e.Event_Id == mpv.EVENT_SHUTDOWN || e.Event_Id == mpv.EVENT_END_FILE {
			p.paused = false
			p.started = false
			break
		}
	}

	p.Mpv.TerminateDestroy()
	p.Shutdown()
}

// Stop stops movie
func (p *Player) Stop() {
	if p.IsStarted() {
		err := p.Mpv.Command([]string{"stop"})
		if err != nil {
			log.Printf("ERROR: stop: %s\n", err.Error())
		}
	}
}

// IsPaused checks if player is paused
func (p *Player) IsPaused() bool {
	return p.paused
}

// IsStarted checks if player is started
func (p *Player) IsStarted() bool {
	return p.started
}

// handleEvent handles player events
func (p *Player) handleEvent(e *mpv.Event) {
	switch e.Event_Id {
	case mpv.EVENT_PAUSE:
		p.paused = true
		p.Paused()
	case mpv.EVENT_UNPAUSE:
		p.paused = false
		p.Unpaused()
	case mpv.EVENT_START_FILE:
		p.started = true
		p.StartFile()
	case mpv.EVENT_FILE_LOADED:
		p.FileLoaded()
	case mpv.EVENT_END_FILE:
		p.EndFile()
	case mpv.EVENT_LOG_MESSAGE:
		s := (*C.struct_mpv_event_log_message)(e.Data)
		msg := C.GoString((*C.char)(s.text))
		log.Printf("MPV: %s\n", strings.TrimSpace(msg))
	}
}
