// +build windows

package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"

	"github.com/gen2brain/bukanir/lib/bukanir"
)

//go:generate qtmoc
type Object3 struct {
	core.QObject

	_ func() `signal:paused`
	_ func() `signal:unpaused`
	_ func() `signal:startFile`
	_ func() `signal:fileLoaded`
	_ func() `signal:endFile`
	_ func() `signal:shutdown`
}

type Player struct {
	*Object3

	Window *Window
	Cmd    *exec.Cmd

	args    []string
	started bool
}

func NewPlayer(w *Window) *Player {
	return &Player{NewObject3(w), w, nil, []string{}, false}
}

func (p *Player) Init() {
	p.args = make([]string, 0)

	x := (widgets.QApplication_Desktop().Width() / 2) - (960 / 2)
	p.SetOption("geometry", fmt.Sprintf("960+%d+%d", x, p.Window.Y()+100))

	p.SetOption("osc", "yes")
	p.SetOption("ytdl", "no")

	p.SetOption("vo", "direct3d,opengl,sdl,null")
	p.SetOption("ao", "wasapi,sdl,null")

	p.SetOption("cache-default", "128")
	p.SetOption("cache-seek-min", "32")
	p.SetOption("cache-secs", "1.0")

	p.SetOption("input-default-bindings", "yes")
	p.SetOption("input-vo-keyboard", "yes")

	if p.Window.Settings.Fullscreen {
		p.SetOption("fullscreen", "yes")
	}

	if p.Window.Settings.StopScreensaver {
		p.SetOption("stop-screensaver", "yes")
	}

	p.SetOption("volume-max", strconv.Itoa(p.Window.Settings.VolumeMax))

	p.SetOption("sub-scale", strconv.FormatFloat(p.Window.Settings.Scale, 'f', -1, 64))
	//p.SetOption("sub-color", p.Window.Settings.Color)
	p.SetOption("sub-text-color", p.Window.Settings.Color)
	if strings.ToLower(p.Window.Settings.Codepage) != "auto" {
		p.SetOption("sub-codepage", strings.ToLower(p.Window.Settings.Codepage))
	}
}

func (p *Player) SetOption(name, value string) {
	p.args = append(p.args, fmt.Sprintf("--%s=%s", name, value))
}

func (p *Player) Wait(movie bukanir.TMovie, imdbId string) (bool, string) {
	if !bukanir.TorrentWaitStartup() {
		return false, ""
	}

	var file bukanir.TFileInfo

	retry := 0
	ready := false
	subs := false

	for !ready {
		started := bukanir.TorrentStarted()
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

					if !subs && p.Window.Settings.Subtitles {
						subDir := strings.Replace(file.Url, "http://127.0.0.1:5001/files", "", -1)
						subDir = filepath.Dir(subDir)
						subDir, _ = url.QueryUnescape(subDir)

						if p.Window.Settings.KeepFiles && p.Window.Settings.DlPath != "" {
							subDir = filepath.Join(p.Window.Settings.DlPath, subDir)
						} else {
							subDir = filepath.Join(tempDir, subDir)
						}

						_, err := os.Stat(subDir)
						if err == nil {
							subs = true
							go func() {
								p.AddSubtitles(movie, imdbId, subDir)
							}()
						}
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

func (p *Player) Status() {
}

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

	for _, sub := range subs[:cnt] {
		subPath, err := bukanir.UnzipSubtitle(sub.DownloadLink, subDir)

		if err != nil {
			log.Printf("ERROR: UnzipSubtitle: %s\n", err.Error())
			continue
		}

		if subPath != "" && subPath != "empty" {
			p.SetOption("sub-file", subPath)
		}
	}
}

func (p *Player) Play(url string, title string) {
	if title != "" {
		p.SetOption("force-media-title", title)
	}

	p.args = append(p.args, url)
	p.Cmd = exec.Command("mpv", p.args...)

	p.started = true
	p.StartFile()

	err := p.Cmd.Start()
	if err != nil {
		log.Printf("ERROR: Run: %s\n", err.Error())
		p.started = false
		p.Shutdown()
		return
	}

	p.FileLoaded()
	p.Cmd.Wait()

	p.started = false
	p.Shutdown()
}

func (p *Player) Stop() {
	if p.IsStarted() {
		p.Cmd.Process.Kill()
	}
}

func (p *Player) IsPaused() bool {
	return false
}

func (p *Player) IsStarted() bool {
	return p.started
}
