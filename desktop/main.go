package main

//go:generate goversioninfo -icon=dist/windows/bukanir.ico -o resource_windows_386.syso
//go:generate goversioninfo -64 -icon=dist/windows/bukanir.ico -o resource_windows_amd64.syso

import (
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/postfinance/single"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"

	"github.com/gen2brain/bukanir/lib"
)

var (
	tempDir string
	tr      func(string) string
)

func init() {
	setLocale(LcNumeric, "C")
	os.Setenv("SDL_RENDER_DRIVER", "software")

	switch runtime.GOOS {
	case "linux":
		os.Setenv("QT_QPA_PLATFORM", "xcb")
		//if os.Getenv("WAYLAND_DISPLAY") != "" {
		//	os.Setenv("QT_QPA_PLATFORM", "wayland-egl")
		//} else {
		//	os.Setenv("QT_QPA_PLATFORM", "xcb")
		//}
	case "windows":
		os.Setenv("QT_QPA_PLATFORM", "windows")
	case "darwin":
		os.Setenv("QT_QPA_PLATFORM", "cocoa")
	}
}

func main() {
	one, err := single.New("bukanir", single.WithLockPath(os.TempDir()))
	if err != nil {
		log.Fatal(err)
	}
	err = one.Lock()
	if err != nil {
		log.Fatal(err)
	}
	defer one.Unlock()

	tempDir, _ = ioutil.TempDir(os.TempDir(), "bukanir")
	defer os.RemoveAll(tempDir)

	logFile, _ := os.Create(filepath.Join(tempDir, "log.txt"))
	defer logFile.Close()

	app := widgets.NewQApplication(len(os.Args), os.Args)

	var v bool
	if inSlice("-verbose", os.Args[1:]) || inSlice("--verbose", os.Args[1:]) {
		v = true
	}
	if v {
		log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	} else {
		log.SetOutput(logFile)
	}

	bukanir.SetVerbose(true)
	defer bukanir.TorStop()

	locale := core.NewQLocale().System().Name()
	translator := core.NewQTranslator(app)
	translator.Load(":qml/i18n/bukanir."+locale+".qm", ":/qml/i18n", "", "")
	app.InstallTranslator(translator)

	tr = func(source string) string {
		return translator.Translate("global", source, "", -1)
	}

	window := NewWindow()
	window.Center()
	window.AddWidgets()
	window.ConnectSignals()
	window.Init()
	window.Show()

	app.Exec()
}
