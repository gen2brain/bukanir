package main

//go:generate goversioninfo -icon=dist/windows/bukanir.ico -o resource_windows.syso

import (
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/widgets"

	"github.com/gen2brain/bukanir/lib"
)

var (
	tabs    []Tab
	tempDir string

	tr func(string) string
)

// Tab type
type Tab struct {
	Query    string
	Category int
	Genre    int
	Movie    bukanir.TMovie
	Widget   *List
	Widget2  *Summary
}

func main() {
	tempDir, _ = ioutil.TempDir(os.TempDir(), "bukanir")
	defer os.RemoveAll(tempDir)

	logFile, _ := os.Create(filepath.Join(tempDir, "log.txt"))
	defer logFile.Close()

	app := widgets.NewQApplication(len(os.Args), os.Args)

	v := flag.Bool("verbose", false, "Show verbose output")
	flag.Parse()
	if *v {
		log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	} else {
		log.SetOutput(logFile)
	}

	tabs = make([]Tab, 0)

	bukanir.SetVerbose(true)
	defer bukanir.TorStop()

	locale := core.NewQLocale().System().Name()
	translator := core.NewQTranslator(app)
	if translator.Load(":qml/i18n/bukanir."+locale, ":/qml/i18n", "", "") {
		app.InstallTranslator(translator)
	}

	tr = func(source string) string {
		return translator.Translate("global", source, "", -1)
	}

	setLocale(LC_NUMERIC, "C")

	window := NewWindow()
	window.Center()
	window.AddWidgets()
	window.ConnectSignals()
	window.Show()
	window.Init()

	app.Exec()
}
