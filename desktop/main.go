package main

//go:generate goversioninfo -icon=dist/windows/bukanir.ico -o resource_windows.syso

import (
	"flag"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/hpcloud/tail"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"

	"github.com/gen2brain/bukanir/lib/bukanir"
)

var (
	tabs    []Tab
	posters map[string][]byte
	content []map[int]bukanir.TMovie

	tempDir string
)

type Tab struct {
	Query    string
	Category int
	Genre    int
	Movie    bukanir.TMovie
	Widget   *List
	Widget2  *Summary
}

type Window struct {
	*widgets.QWidget

	Log      *Log
	Client   *Client
	Settings *Settings

	Toolbar   *Toolbar
	TabWidget *widgets.QTabWidget

	Model     *core.QStringListModel
	Completer *widgets.QCompleter

	Movie       *gui.QMovie
	LabelStatus *LabelStatus
	StatusBar   *widgets.QStatusBar
	ProgressBar *widgets.QProgressBar
}

func NewWindow() *Window {
	w := widgets.NewQWidget(nil, 0)
	w.SetGeometry2(0, 0, 870, 675)
	w.SetWindowTitle("Bukanir")
	w.SetWindowIcon(gui.NewQIcon5(":/qml/images/bukanir.png"))

	window := &Window{w, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil}
	window.Log = NewLog(window.QWidget_PTR())
	window.Client = NewClient()
	window.Settings = NewSettings(window.QWidget_PTR())

	return window
}

func (w *Window) Center() {
	size := w.Size()
	desktop := widgets.QApplication_Desktop()
	width, height := size.Width(), size.Height()
	dwidth, dheight := desktop.Width(), desktop.Height()
	cw, ch := (dwidth/2)-(width/2), (dheight/2)-(height/2)
	w.Move2(cw, ch)
}

func (w *Window) AddWidgets() {
	w.Model = core.NewQStringListModel(w)
	w.Completer = widgets.NewQCompleter(w)
	w.Completer.SetCaseSensitivity(core.Qt__CaseInsensitive)
	w.Completer.SetMaxVisibleItems(15)
	w.Completer.SetModel(w.Model)

	w.Toolbar = NewToolbar(w.QWidget_PTR())
	w.Toolbar.Input.SetCompleter(w.Completer)

	w.TabWidget = widgets.NewQTabWidget(w)
	w.TabWidget.SetTabsClosable(true)
	w.TabWidget.TabBar().SetExpanding(false)
	w.TabWidget.TabBar().SetElideMode(core.Qt__ElideRight)
	w.TabWidget.SetStyleSheet(`
		QTabWidget { background-color: black; }
		QTabWidget::pane { background-color: black; }
	`)

	w.StatusBar = widgets.NewQStatusBar(w)
	w.StatusBar.SetSizeGripEnabled(false)

	w.LabelStatus = &LabelStatus{NewObject2(w), widgets.NewQLabel(w.StatusBar, 0)}
	w.LabelStatus.SetIndent(3)

	w.Movie = gui.NewQMovie3(":/qml/images/loading.gif", "GIF", w)
	w.Movie.Start()

	w.ProgressBar = widgets.NewQProgressBar(w.StatusBar)
	palette := w.ProgressBar.Palette()
	palette.SetColor2(gui.QPalette__Highlight, gui.NewQColor3(255, 167, 37, 255))
	w.ProgressBar.SetPalette(palette)
	w.ProgressBar.SetVisible(false)

	w.StatusBar.AddWidget(w.LabelStatus, 1)
	w.StatusBar.AddPermanentWidget(w.ProgressBar, 1)
	w.StatusBar.Layout().SetSpacing(5)

	layout := widgets.NewQVBoxLayout()
	layout.SetSpacing(0)
	layout.SetContentsMargins(0, 0, 0, 0)

	layout.AddWidget(w.Toolbar.QWidget_PTR(), 0, 0)
	layout.AddWidget(w.TabWidget, 0, 0)
	layout.AddWidget(w.StatusBar, 0, 0)

	w.SetLayout(layout)
}

func (w *Window) ConnectSignals() {
	w.Settings.ConnectSignals()

	w.Log.ConnectValueChanged(func(value string) {
		if strings.Contains(strings.ToLower(value), "error") {
			value = fmt.Sprintf("<font color=\"red\">%s</font>", value)
			w.Log.TextEdit.AppendHtml(value)
		} else {
			w.Log.TextEdit.AppendPlainText(value)
		}
	})

	w.LabelStatus.ConnectValueChanged(func(value string) {
		w.LabelStatus.SetText(value)
	})

	w.ProgressBar.ConnectValueChanged(func(value int) {
		if !w.ProgressBar.IsVisible() {
			w.ProgressBar.SetVisible(true)
		}
		w.ProgressBar.SetValue(value)
	})

	w.Toolbar.Input.ConnectReturnPressed(func() {
		if w.Toolbar.Input.HasFocus() {
			query := w.Toolbar.Input.Text()
			if query == "" {
				return
			}

			w.Search(query, query, 3)
		}
	})

	w.Toolbar.Input.ConnectTextEdited(func(text string) {
		if len(text) < 2 {
			return
		}
		go w.Client.Complete(w.Toolbar, text)
	})

	w.Toolbar.ConnectFinished(func(data string) {
		if data != "" && data != "empty" {
			w.Toolbar.Complete(w, data)
		}
	})

	w.Toolbar.Search.ConnectClicked(func(bool) {
		query := w.Toolbar.Input.Text()
		if query == "" {
			return
		}

		w.Search(query, query, 3)
	})

	w.Toolbar.Refresh.ConnectClicked(func(bool) {
		index := w.TabWidget.CurrentIndex()
		t := tabs[index]

		if t.Query != "" {
			w.setLoading(index, true)
			w.Toolbar.SetEnabled(false)
			w.LabelStatus.SetText("Downloading torrents metadata...")

			t.Widget.Started = true
			go w.Client.Search(t.Widget, t.Query, w.Settings.Limit, 1, w.Settings.Days, 3, w.Settings.Host)
		} else if t.Category != 0 {
			w.setLoading(index, true)
			w.Toolbar.SetEnabled(false)
			w.LabelStatus.SetText("Downloading torrents metadata...")

			t.Widget.Started = true
			go w.Client.Top(t.Widget, t.Category, w.Settings.Limit, 1, w.Settings.Days, w.Settings.Host)
		} else if t.Genre != 0 {
			w.setLoading(index, true)
			w.Toolbar.SetEnabled(false)
			w.LabelStatus.SetText("Downloading torrents metadata...")

			t.Widget.Started = true
			go w.Client.Genre(t.Widget, t.Genre, w.Settings.Limit, 1, w.Settings.Days, w.Settings.Host)
		} else if t.Movie.Id != 0 {
			w.setLoading(index, true)
			w.Toolbar.SetEnabled(false)
			w.LabelStatus.SetText("Downloading movie metadata...")

			t.Widget2.Started = true
			go w.Client.Summary(t.Widget2, t.Movie)
		}
	})

	w.Toolbar.Top.Menu().ConnectTriggered(func(action *widgets.QAction) {
		w.Top(fmt.Sprintf("Top %s", action.Text()), action.Data().ToInt(false))
	})

	w.Toolbar.Year.Menu().ConnectTriggered(func(action *widgets.QAction) {
		w.Search(fmt.Sprintf("Year %s", action.Text()), action.Text(), 3)
	})

	w.Toolbar.Popular.Menu().ConnectTriggered(func(action *widgets.QAction) {
		w.Search(action.Data().ToString(), action.Data().ToString(), 1)
	})

	w.Toolbar.TopRated.Menu().ConnectTriggered(func(action *widgets.QAction) {
		w.Search(action.Data().ToString(), action.Data().ToString(), 1)
	})

	w.Toolbar.Genre.Menu().ConnectTriggered(func(action *widgets.QAction) {
		w.Genre(action.Text(), action.Data().ToInt(false))
	})

	w.Toolbar.Settings.ConnectClicked(func(bool) {
		w.Settings.Sync()
		w.Settings.Set()
		w.Settings.Show()
	})

	w.Toolbar.Log.ConnectClicked(func(bool) {
		w.Log.Show()
	})

	w.Toolbar.About.ConnectClicked(func(bool) {
		NewAbout(w.QWidget).Show()
	})

	w.TabWidget.ConnectTabInserted(func(index int) {
		w.TabWidget.SetCurrentIndex(index)
		label := widgets.NewQLabel(w.TabWidget.TabBar(), 0)
		label.SetMovie(w.Movie)
		w.TabWidget.TabBar().SetTabButton(index, widgets.QTabBar__LeftSide, label)
	})

	w.TabWidget.ConnectTabCloseRequested(func(index int) {
		t := tabs[index]
		if t.Widget2 != nil {
			t.Widget2.DisconnectFinished()
			t.Widget2.DisconnectFinished2()
			t.Widget2.Player.DisconnectShutdown()
			t.Widget2.Player.Stop()

			w.LabelStatus.SetText("")
			if w.ProgressBar.IsVisible() {
				w.ProgressBar.SetVisible(false)
			}

			for _, tb := range tabs {
				if tb.Widget2 != nil && tb.Widget2 != t.Widget2 {
					tb.Widget2.Watch.SetEnabled(true)
				}
			}

			if t.Widget2.TorrentStarted {
				go func() {
					bukanir.TorrentStop()
				}()
			}

			if t.Widget2.Started {
				go func() {
					bukanir.Cancel()
				}()
			}

			if t.Widget2.Initialized {
				content = append(content[:index], content[index+1:]...)
			}
		} else if t.Widget != nil {
			t.Widget.DisconnectFinished()

			w.Toolbar.SetEnabled(true)
			w.LabelStatus.SetText("")
			w.Toolbar.Input.SetText("")

			if t.Widget.Started {
				go func() {
					bukanir.Cancel()
				}()
			}

			if t.Widget.Initialized {
				content = append(content[:index], content[index+1:]...)
			}
		}

		tabs = append(tabs[:index], tabs[index+1:]...)

		w.TabWidget.Widget(index).DeleteLater()
		w.TabWidget.RemoveTab(index)
	})

	shortcut := widgets.NewQShortcut(w.TabWidget)
	shortcut.SetKey(gui.NewQKeySequence2("Ctrl+W", gui.QKeySequence__NativeText))
	shortcut.ConnectActivated(func() {
		index := w.TabWidget.CurrentIndex()
		if index == -1 {
			return
		}

		w.TabWidget.TabCloseRequested(index)
		w.TabWidget.SetFocus2()
	})
}

func (w *Window) Top(title string, category int) {
	tab := NewList(w.TabWidget)
	index := w.TabWidget.Count()

	tab.ConnectItemActivated(func(item *widgets.QListWidgetItem) {
		tidx := w.TabWidget.CurrentIndex()
		midx := tab.Row(item)

		movie := content[tidx][midx]
		w.Summary(movie)
	})

	tab.ConnectFinished(func(data string) {
		w.setLoading(index, false)
		w.Toolbar.SetEnabled(true)
		w.LabelStatus.SetText("")

		if data != "" && data != "empty" {
			tab.Init(index, data)
			tab.Started = false
		}
	})

	w.TabWidget.AddTab(tab, title)
	w.setLoading(index, true)
	w.Toolbar.SetEnabled(false)
	w.LabelStatus.SetText("Downloading torrents metadata...")

	tab.Started = true
	tabs = append(tabs, Tab{"", category, 0, bukanir.TMovie{}, tab, nil})

	go w.Client.Top(tab, category, w.Settings.Limit, 0, w.Settings.Days, w.Settings.Host)
}

func (w *Window) Search(title, query string, pages int) {
	query = strings.TrimSpace(query)
	if query == "" {
		return
	}

	tab := NewList(w.TabWidget)
	index := w.TabWidget.Count()

	tab.ConnectItemActivated(func(item *widgets.QListWidgetItem) {
		tidx := w.TabWidget.CurrentIndex()
		midx := tab.Row(item)

		movie := content[tidx][midx]
		w.Summary(movie)
	})

	tab.ConnectFinished(func(data string) {
		w.setLoading(index, false)
		w.Toolbar.SetEnabled(true)
		w.LabelStatus.SetText("")
		w.Toolbar.Input.SetText("")

		if data != "" && data != "empty" {
			tab.Init(index, data)
			tab.Started = false
		}
	})

	w.TabWidget.AddTab(tab, title)
	w.setLoading(index, true)
	w.Toolbar.SetEnabled(false)
	w.LabelStatus.SetText("Downloading torrents metadata...")

	tab.Started = true
	tabs = append(tabs, Tab{query, 0, 0, bukanir.TMovie{}, tab, nil})

	go w.Client.Search(tab, query, w.Settings.Limit, 0, w.Settings.Days, pages, w.Settings.Host)
}

func (w *Window) Genre(title string, id int) {
	tab := NewList(w.TabWidget)
	index := w.TabWidget.Count()

	tab.ConnectItemActivated(func(item *widgets.QListWidgetItem) {
		tidx := w.TabWidget.CurrentIndex()
		midx := tab.Row(item)

		movie := content[tidx][midx]
		w.Summary(movie)
	})

	tab.ConnectFinished(func(data string) {
		w.setLoading(index, false)
		w.Toolbar.SetEnabled(true)
		w.LabelStatus.SetText("")

		if data != "" && data != "empty" {
			tab.Init(index, data)
			tab.Started = false
		}
	})

	w.TabWidget.AddTab(tab, title)
	w.setLoading(index, true)
	w.Toolbar.SetEnabled(false)
	w.LabelStatus.SetText("Downloading torrents metadata...")

	tab.Started = true
	tabs = append(tabs, Tab{"", 0, id, bukanir.TMovie{}, tab, nil})

	go w.Client.Genre(tab, id, w.Settings.Limit, 0, w.Settings.Days, w.Settings.Host)
}

func (w *Window) Summary(movie bukanir.TMovie) {
	summary := NewSummary(w.TabWidget)
	summary.Player = NewPlayer(w)

	index := w.TabWidget.Count()

	summary.ConnectFinished(func(data string) {
		w.setLoading(index, false)
		w.Toolbar.SetEnabled(true)
		w.LabelStatus.SetText("")

		if data != "" && data != "empty" {
			summary.Init(index, movie, data)
			summary.Started = false
		}
	})

	summary.ConnectFinished2(func(data string) {
		w.Client.Mutex.RLock()
		d := string(posters[data])
		w.Client.Mutex.RUnlock()

		pixmap := gui.NewQPixmap()
		ok := pixmap.LoadFromData2(d, "JPG", core.Qt__AutoColor)
		if ok {
			summary.Poster.SetPixmap(pixmap)

			var filterObject = core.NewQObject(summary)
			filterObject.ConnectEventFilter(func(watched *core.QObject, event *core.QEvent) bool {
				if event.Type() == core.QEvent__Resize {
					summary.Poster.SetPixmap(pixmap.Scaled2(summary.Poster.Width(), summary.Poster.Height(), core.Qt__KeepAspectRatio, core.Qt__SmoothTransformation))
					return true
				}
				return false
			})
			summary.Poster.InstallEventFilter(filterObject)
		}

		w.Client.Mutex.Lock()
		delete(posters, data)
		w.Client.Mutex.Unlock()
	})

	summary.Watch.ConnectClicked(func(bool) {
		idx := w.TabWidget.CurrentIndex()
		label := widgets.NewQLabelFromPointer(w.TabWidget.TabBar().TabButton(idx, widgets.QTabBar__LeftSide).Pointer())

		playPixmap := gui.NewQPixmap5(":/qml/images/play.png", "PNG", core.Qt__AutoColor)
		pausePixmap := gui.NewQPixmap5(":/qml/images/pause.png", "PNG", core.Qt__AutoColor)

		if !summary.Player.IsStarted() {
			summary.Watch.SetEnabled(false)
			summary.Trailer.SetEnabled(false)

			for _, t := range tabs {
				if t.Widget2 != nil && t.Widget2 != summary {
					t.Widget2.Watch.SetEnabled(false)
				}
			}

			label.SetVisible(true)
			w.LabelStatus.SetText("Torrent started...")
		}

		summary.Player.ConnectStartFile(func() {
			w.LabelStatus.SetText("Opening player...")
		})

		summary.Player.ConnectFileLoaded(func() {
			w.LabelStatus.SetText("")
			w.StatusBar.ShowMessage("Playing...", 3000)
			label.SetPixmap(playPixmap)
		})

		summary.Player.ConnectPaused(func() {
			label.SetPixmap(pausePixmap)
			go summary.Player.Status()
		})

		summary.Player.ConnectUnpaused(func() {
			w.LabelStatus.SetText("")
			w.StatusBar.ShowMessage("Playing...", 1500)
			label.SetPixmap(playPixmap)
		})

		summary.Player.ConnectShutdown(func() {
			go func() {
				bukanir.TorrentStop()
			}()

			summary.Watch.SetEnabled(true)
			summary.Trailer.SetEnabled(true)

			for _, t := range tabs {
				if t.Widget2 != nil && t.Widget2 != summary {
					t.Widget2.Watch.SetEnabled(true)
				}
			}

			label.SetVisible(false)
			label.SetMovie(w.Movie)

			w.TabWidget.SetFocus2()
		})

		go func() {
			if !summary.Player.IsStarted() {
				summary.TorrentStarted = true
				bukanir.TorrentStartup(w.Settings.TorrentConfig(movie.MagnetLink))
				bukanir.TorrentShutdown()
				summary.TorrentStarted = false
			}
		}()

		go func() {
			if !summary.Player.IsStarted() {
				summary.Player.Init()
				summary.Player.Wait(movie, summary.ImdbId)
			}
		}()

	})

	summary.Trailer.ConnectClicked(func(bool) {
		idx := w.TabWidget.CurrentIndex()
		label := widgets.NewQLabelFromPointer(w.TabWidget.TabBar().TabButton(idx, widgets.QTabBar__LeftSide).Pointer())

		playPixmap := gui.NewQPixmap5(":/qml/images/play.png", "PNG", core.Qt__AutoColor)
		pausePixmap := gui.NewQPixmap5(":/qml/images/pause.png", "PNG", core.Qt__AutoColor)

		summary.Trailer.SetEnabled(false)
		label.SetVisible(true)

		summary.Player.ConnectStartFile(func() {
		})

		summary.Player.ConnectFileLoaded(func() {
			label.SetPixmap(playPixmap)
		})

		summary.Player.ConnectPaused(func() {
			label.SetPixmap(pausePixmap)
		})

		summary.Player.ConnectUnpaused(func() {
			label.SetPixmap(playPixmap)
		})

		summary.Player.ConnectShutdown(func() {
			summary.Trailer.SetEnabled(true)

			label.SetVisible(false)
			label.SetMovie(w.Movie)

			w.TabWidget.SetFocus2()
		})

		go func() {
			url, err := bukanir.Trailer(summary.Video)
			if err != nil {
				log.Printf("ERROR: Trailer: %s\n", err.Error())
				return
			}

			if url != "" && url != "empty" {
				summary.Player.Init()
				summary.Player.Play(url, fmt.Sprintf("%s (%s) - Trailer", movie.Title, movie.Year))
			}
		}()
	})

	title := movie.Title
	if movie.Season != 0 && movie.Episode != 0 {
		title = fmt.Sprintf("%s S%02dE%02d", movie.Title, movie.Season, movie.Episode)
	}

	w.TabWidget.AddTab(summary, title)
	w.setLoading(index, true)
	w.Toolbar.SetEnabled(false)
	w.LabelStatus.SetText("Downloading movie metadata...")

	summary.Started = true
	tabs = append(tabs, Tab{"", 0, 0, movie, nil, summary})

	go w.Client.Summary(summary, movie)
}

func (w *Window) TailLog() {
	t, err := tail.TailFile(filepath.Join(tempDir, "log.txt"), tail.Config{Follow: true, Poll: true})
	if err != nil {
		log.Printf("ERROR: Trailer: %s\n", err.Error())
		return
	}
	for line := range t.Lines {
		w.Log.ValueChanged(line.Text)
	}
}

func (w *Window) Init() {
	go w.TailLog()

	w.Top("Top Movies", bukanir.CategoryMovies)

	go w.Client.Popular(w.Toolbar)
	go w.Client.TopRated(w.Toolbar)
	go w.Client.Genres(w.Toolbar)

	if w.LabelStatus.Text() == "" {
		w.StatusBar.ShowMessage(fmt.Sprintf("Welcome to Bukanir %s", bukanir.Version), 5000)
	}
}

func (w *Window) setLoading(index int, visible bool) {
	label := w.TabWidget.TabBar().TabButton(index, widgets.QTabBar__LeftSide)
	label.SetVisible(visible)
}

func main() {
	tempDir, _ = ioutil.TempDir(os.TempDir(), "bukanir")
	defer os.RemoveAll(tempDir)

	logFile, _ := os.Create(filepath.Join(tempDir, "log.txt"))
	defer logFile.Close()

	widgets.NewQApplication(len(os.Args), os.Args)

	v := flag.Bool("verbose", false, "Show verbose output")
	flag.Parse()
	if *v {
		log.SetOutput(io.MultiWriter(os.Stderr, logFile))
	} else {
		log.SetOutput(logFile)
	}

	tabs = make([]Tab, 0)
	posters = make(map[string][]byte)
	content = make([]map[int]bukanir.TMovie, 0)

	bukanir.SetVerbose(true)
	setLocale(LC_NUMERIC, "C")

	window := NewWindow()
	window.Center()
	window.AddWidgets()
	window.ConnectSignals()
	window.Show()
	window.Init()

	widgets.QApplication_Exec()
}
