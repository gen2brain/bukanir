// Command bukanir is the desktop GUI for the bukanir movie/TV streaming library.
package main

import (
	"context"
	"log"
	"runtime/debug"

	"github.com/gen2brain/bukanir"
	"github.com/gen2brain/go-mpv"
	"github.com/gen2brain/iup-go/iup"
)

const appName = "Bukanir"

// appVersion is shown in the title bar.
var appVersion string

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

// client is the shared library client.
var client *bukanir.Client

// settings holds the persisted user preferences, loaded at startup.
var settings *Settings

func main() {
	iup.Open()
	defer iup.Close()

	// libmpv requires LC_NUMERIC=C; iup/GTK changes the locale on Open.
	mpv.SetLocale(mpv.LCNumeric, "C")

	iup.SetGlobal("APPNAME", "bukanir")
	iup.SetGlobal("APPID", "io.github.gen2brain.bukanir")
	initDirs()
	settings = loadSettings()
	iup.SetGlobal("APPEARANCE", settings.appearance())

	loadIcons()
	loadLoadingAnimation()
	registerTabIndicators()

	cfg := settings.toConfig()
	cfg.TorLog = logs
	logs.setRedactor(bukanir.SecretValues(cfg))

	c, err := bukanir.New(context.Background(), cfg)
	if err != nil {
		log.Fatal(err) // still on stderr: the viewer is not up yet
	}

	log.SetOutput(logs)
	bukanir.SetVerbose(true)

	client = c
	defer client.Close()

	dlg := iup.Dialog(layout()).SetAttributes(map[string]string{
		"TITLE":      appName + " " + appVersion,
		"ICON":       "appicon",
		"RASTERSIZE": "1000x720",
		"SHRINK":     "YES",
	}).SetHandle("dlg")
	dlg.SetCallback("K_ANY", iup.KAnyFunc(onKey))
	dlg.SetCallback("THEMECHANGED_CB", iup.ThemeChangedFunc(onThemeChanged))
	dlg.SetCallback("CLOSE_CB", iup.CloseFunc(func(iup.Ihandle) int { return iup.CLOSE })) // end the loop; iup.Close then destroys child dialogs
	// A MultiBox reports its wrapped height one layout pass late; refresh again after a resize.
	dlg.SetCallback("RESIZE_CB", iup.ResizeFunc(func(ih iup.Ihandle, _, _ int) int {
		iup.Refresh(ih)
		return iup.DEFAULT
	}))

	iup.Map(dlg)
	iup.ShowXY(dlg, iup.CENTER, iup.CENTER)
	updateTorStatus()
	startup()
	loadMenus()
	iup.MainLoop()
}

// startup kicks off the initial content load in the background.
func startup() {
	go func() {
		movies, err := loadTop()
		if err != nil {
			iup.PostMessage(iup.GetHandle("status"), "error: "+err.Error(), 0, nil)
			return
		}
		iup.PostMessage(iup.GetHandle("content0"), "", 0, movies)
	}()
}
