//go:build media

// Command bukanir is the desktop GUI for the bukanir movie/TV streaming library.
package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"runtime/debug"
	"syscall"
	"time"

	"github.com/gen2brain/bukanir"
	"github.com/gen2brain/iup-go/iup"
)

const appName = "Bukanir"

// quitWait bounds how long quitting waits for playbacks to clean up.
const quitWait = 10 * time.Second

// appVersion is shown in the About dialog.
var appVersion string

// client is the shared library client.
var client *bukanir.Client

// settings holds the persisted user preferences, loaded at startup.
var settings *Settings

var quitting bool

// teardown closes the client and removes this run's temp dir; quitApp runs it once playbacks stop.
var teardown func()

func init() {
	if appVersion != "" {
		return
	}
	if info, ok := debug.ReadBuildInfo(); ok {
		if v := info.Main.Version; v != "" && v != "(devel)" {
			appVersion = v
		}
		for _, kv := range info.Settings {
			if kv.Key == "vcs.revision" && kv.Value != "" {
				appVersion = kv.Value
				if len(appVersion) > 7 {
					appVersion = appVersion[:7]
				}
			}
		}
	}
	if appVersion == "" {
		appVersion = bukanir.Version
	}
}

func init() { iup.EntryPoint(main) }

func main() {
	iup.Open()
	defer iup.Close()
	iup.MediaOpen()

	iup.SetGlobal("APPNAME", "bukanir")
	iup.SetGlobal("APPID", "io.github.gen2brain.bukanir")
	initDirs()
	removeTemp, err := initTemp()
	if err != nil {
		log.Fatal(err)
	}
	settings = loadSettings()
	loadFavorites()
	iup.SetGlobal("APPEARANCE", settings.appearance())

	loadIcons()
	loadLoadingAnimation()
	registerTabIndicators()

	cfg := settings.toConfig()
	logs.setRedactor(bukanir.SecretValues(cfg))

	c, err := bukanir.New(context.Background(), cfg)
	if err != nil {
		log.Fatal(err) // still on stderr: the viewer is not up yet
	}

	log.SetOutput(logs)
	bukanir.SetVerbose(true)

	client = c
	client.StartTor()
	teardown = func() {
		client.Close()
		removeTemp()
	}

	dlg := mainDialog()
	dlg.SetCallback("THEMECHANGED_CB", iup.ThemeChangedFunc(onThemeChanged))
	dlg.SetCallback("CLOSE_CB", iup.CloseFunc(func(iup.Ihandle) int {
		quitApp()
		return iup.IGNORE
	}))

	showMainDialog(dlg)
	watchTor(client)
	quitOnSignal()
	startup(context.Background())
	loadMenus()
	iup.MainLoop()
}

// quitApp stops every playback and ends the main loop once their cleanup is done; iup.Close then destroys the dialogs.
func quitApp() {
	if quitting {
		return
	}
	quitting = true
	iup.SetGlobal("LOCKLOOP", "YES")
	iup.Hide(iup.GetHandle("dlg"))
	stopAllSessions()
	go func() {
		done := make(chan struct{})
		go func() {
			playbacks.Wait()
			close(done)
		}()
		select {
		case <-done:
		case <-time.After(quitWait):
			log.Println("QUIT: playbacks did not stop in time")
		}
		teardown()
		runOnMain(exitApp)
	}()
}

// quitOnSignal quits like closing the window on an interrupt or terminate signal; a second signal kills the process.
func quitOnSignal() {
	sig := make(chan os.Signal, 1)
	signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-sig
		signal.Stop(sig)
		runOnMain(quitApp)
	}()
}

// startup kicks off the initial content load in the background.
func startup(ctx context.Context) {
	tab := iup.GetHandle("content0")
	loadInTab(tab, func() {
		movies, err := loadTop(ctx)
		if err != nil {
			iup.PostMessage(iup.GetHandle("status"), "error: "+err.Error(), 0, nil)
			return
		}
		iup.PostMessage(tab, "", 0, movies)
	})
}
