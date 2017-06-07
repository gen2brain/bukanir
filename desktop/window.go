package main

import (
	"encoding/json"
	"fmt"
	"log"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/hpcloud/tail"
	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/network"
	"github.com/therecipe/qt/widgets"

	"github.com/gen2brain/bukanir/lib"
)

var (
	textMeta      string = "Downloading torrents metadata..."
	textMovieMeta string = "Downloading movie metadata..."
)

// Window type
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

	Side widgets.QTabBar__ButtonPosition

	Manager *network.QNetworkAccessManager
}

// NewWindow returns new window
func NewWindow() *Window {
	w := widgets.NewQWidget(nil, 0)
	w.SetGeometry2(0, 0, 870, 675)
	w.SetWindowTitle("Bukanir")
	w.SetWindowIcon(gui.NewQIcon5(":/qml/images/bukanir.png"))

	side := widgets.QTabBar__LeftSide
	if runtime.GOOS == "darwin" {
		side = widgets.QTabBar__RightSide
	}

	window := &Window{w, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, nil, side, nil}
	window.Log = NewLog(window.QWidget_PTR())
	window.Client = NewClient()
	window.Settings = NewSettings(window.QWidget_PTR())

	window.Manager = network.NewQNetworkAccessManager(w)
	cache := network.NewQNetworkDiskCache(w)
	cache.SetCacheDirectory(filepath.Join(cacheDir(), "images"))
	window.Manager.SetCache(cache)

	return window
}

// Center centers window
func (w *Window) Center() {
	size := w.Size()
	desktop := widgets.QApplication_Desktop()
	width, height := size.Width(), size.Height()
	dwidth, dheight := desktop.Width(), desktop.Height()
	cw, ch := (dwidth/2)-(width/2), (dheight/2)-(height/2)
	w.Move2(cw, ch)
}

// AddWidgets adds widgets to window
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
	w.TabWidget.TabBar().SetTabsClosable(true)
	w.TabWidget.TabBar().SetExpanding(false)
	w.TabWidget.TabBar().SetElideMode(core.Qt__ElideRight)
	w.TabWidget.TabBar().SetDocumentMode(true)
	w.TabWidget.SetStyleSheet(`
		QTabWidget { background-color: black; }
		QTabWidget::pane { background-color: black; }
	`)

	w.StatusBar = widgets.NewQStatusBar(w)
	w.StatusBar.SetSizeGripEnabled(false)

	w.LabelStatus = &LabelStatus{NewObject2(w), widgets.NewQLabel(w.StatusBar, 0)}
	w.LabelStatus.SetIndent(3)

	w.Movie = gui.NewQMovie3(":/qml/images/loading.gif", core.NewQByteArray2("GIF", 3), w)
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

// ConnectSignals connects signals
func (w *Window) ConnectSignals() {
	w.Settings.ConnectSignals()

	w.Log.ConnectValueChanged(func(value string) {
		if strings.Contains(strings.ToLower(value), "error:") {
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
			w.setLoading(t.Widget.QWidget_PTR(), true)
			w.LabelStatus.SetText(textMeta)

			t.Widget.Started = true
			go w.Client.Search(t.Widget, t.Query, w.Settings.Limit, 1, w.Settings.Days, 3, w.Settings.TPBHost, w.Settings.EZTVHost)
		} else if t.Category != 0 {
			w.setLoading(t.Widget.QWidget_PTR(), true)
			w.LabelStatus.SetText(textMeta)

			t.Widget.Started = true
			go w.Client.Top(t.Widget, t.Category, w.Settings.Limit, 1, w.Settings.Days, w.Settings.TPBHost)
		} else if t.Genre != 0 {
			w.setLoading(t.Widget.QWidget_PTR(), true)
			w.LabelStatus.SetText(textMeta)

			t.Widget.Started = true
			go w.Client.Genre(t.Widget, t.Genre, w.Settings.Limit, 1, w.Settings.Days, w.Settings.TPBHost)
		} else if t.Movie.Id != 0 {
			w.setLoading(t.Widget2.QWidget_PTR(), true)
			w.LabelStatus.SetText(textMovieMeta)

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
		w.TabWidget.TabBar().SetTabButton(index, w.Side, label)
	})

	w.TabWidget.ConnectTabCloseRequested(func(index int) {
		t := tabs[index]
		if t.Widget2 != nil {
			t.Widget2.DisconnectFinished()
			t.Widget2.DisconnectFinished2()
			t.Widget2.Player.DisconnectShutdown()
			t.Widget2.Player.DisconnectFileLoaded()
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
		} else if t.Widget != nil {
			t.Widget.DisconnectFinished()

			w.LabelStatus.SetText("")
			w.Toolbar.Input.SetText("")

			if t.Widget.Started {
				go func() {
					bukanir.Cancel()
				}()
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

// Top movies
func (w *Window) Top(title string, category int) {
	tab := NewList(w.TabWidget)

	tab.ConnectItemActivated(func(item *widgets.QListWidgetItem) {
		data := item.Data(int(core.Qt__UserRole)).ToString()

		var movie bukanir.TMovie
		err := json.Unmarshal([]byte(data), &movie)
		if err != nil {
			log.Printf("ERROR: Unmarshal: %s\n", err.Error())
			return
		}

		w.Summary(movie)
	})

	tab.ConnectFinished(func(data string) {
		w.setLoading(tab.QWidget_PTR(), false)
		w.LabelStatus.SetText("")

		if data != "" && data != "empty" {
			tab.Init(w.Manager, data)
			tab.Started = false
		}
	})

	w.TabWidget.AddTab(tab, title)
	w.setLoading(tab.QWidget_PTR(), true)
	w.LabelStatus.SetText(textMeta)

	tab.Started = true
	tabs = append(tabs, Tab{"", category, 0, bukanir.TMovie{}, tab, nil})

	go w.Client.Top(tab, category, w.Settings.Limit, 0, w.Settings.Days, w.Settings.TPBHost)
}

// Search movies
func (w *Window) Search(title, query string, pages int) {
	query = strings.TrimSpace(query)
	if query == "" {
		return
	}

	tab := NewList(w.TabWidget)

	tab.ConnectItemActivated(func(item *widgets.QListWidgetItem) {
		data := item.Data(int(core.Qt__UserRole)).ToString()

		var movie bukanir.TMovie
		err := json.Unmarshal([]byte(data), &movie)
		if err != nil {
			log.Printf("ERROR: Unmarshal: %s\n", err.Error())
			return
		}

		w.Summary(movie)
	})

	tab.ConnectFinished(func(data string) {
		w.setLoading(tab.QWidget_PTR(), false)
		w.LabelStatus.SetText("")
		w.Toolbar.Input.SetText("")

		if data != "" && data != "empty" {
			tab.Init(w.Manager, data)
			tab.Started = false
		}
	})

	w.TabWidget.AddTab(tab, title)
	w.setLoading(tab.QWidget_PTR(), true)
	w.LabelStatus.SetText(textMeta)

	tab.Started = true
	tabs = append(tabs, Tab{query, 0, 0, bukanir.TMovie{}, tab, nil})

	go w.Client.Search(tab, query, w.Settings.Limit, 0, w.Settings.Days, pages, w.Settings.TPBHost, w.Settings.EZTVHost)
}

// Genre search movies by genre
func (w *Window) Genre(title string, id int) {
	tab := NewList(w.TabWidget)

	tab.ConnectItemActivated(func(item *widgets.QListWidgetItem) {
		data := item.Data(int(core.Qt__UserRole)).ToString()

		var movie bukanir.TMovie
		err := json.Unmarshal([]byte(data), &movie)
		if err != nil {
			log.Printf("ERROR: Unmarshal: %s\n", err.Error())
			return
		}

		w.Summary(movie)
	})

	tab.ConnectFinished(func(data string) {
		w.setLoading(tab.QWidget_PTR(), false)
		w.LabelStatus.SetText("")

		if data != "" && data != "empty" {
			tab.Init(w.Manager, data)
			tab.Started = false
		}
	})

	w.TabWidget.AddTab(tab, title)
	w.setLoading(tab.QWidget_PTR(), true)
	w.LabelStatus.SetText(tr("Downloading torrents metadata..."))

	tab.Started = true
	tabs = append(tabs, Tab{"", 0, id, bukanir.TMovie{}, tab, nil})

	go w.Client.Genre(tab, id, w.Settings.Limit, 0, w.Settings.Days, w.Settings.TPBHost)
}

// Summary shows movie summary
func (w *Window) Summary(movie bukanir.TMovie) {
	summary := NewSummary(w.TabWidget)
	summary.Player = NewPlayer(w)

	summary.ConnectFinished(func(data string) {
		w.setLoading(summary.QWidget_PTR(), false)
		w.LabelStatus.SetText("")

		if data != "" && data != "empty" {
			summary.Init(movie, data)
			summary.Started = false

			reply := w.Manager.Get(network.NewQNetworkRequest(core.NewQUrl3(movie.PosterXLarge, core.QUrl__TolerantMode)))
			reply.ConnectFinished(func() {
				if reply.IsReadable() && reply.Error() == network.QNetworkReply__NoError {
					data := reply.ReadAll()
					if data.ConstData() != "" {
						pixmap := gui.NewQPixmap()
						ok := pixmap.LoadFromData2(data, "JPG", core.Qt__AutoColor)
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
					}
				}
				if reply.IsFinished() {
					reply.DeleteLater()
				}
			})
		}
	})

	summary.Watch.ConnectClicked(func(bool) {
		idx := w.TabWidget.CurrentIndex()
		label := widgets.NewQLabelFromPointer(w.TabWidget.TabBar().TabButton(idx, w.Side).Pointer())

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
			w.LabelStatus.SetText(tr("Torrent started..."))
		}

		summary.Player.ConnectStartFile(func() {
			w.LabelStatus.SetText(tr("Opening player..."))
		})

		summary.Player.ConnectFileLoaded(func() {
			w.LabelStatus.SetText("")
			w.StatusBar.ShowMessage(tr("Playing..."), 3000)
			label.SetPixmap(playPixmap)
		})

		summary.Player.ConnectPaused(func() {
			label.SetPixmap(pausePixmap)
			go summary.Player.Status()
		})

		summary.Player.ConnectUnpaused(func() {
			w.LabelStatus.SetText("")
			w.StatusBar.ShowMessage(tr("Playing..."), 1500)
			label.SetPixmap(playPixmap)
		})

		summary.Player.ConnectShutdown(func() {
			go func() {
				if bukanir.TorrentRunning() {
					bukanir.TorrentStop()
				}
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
				summary.TorrentStarted = false
			}
		}()

		go func() {
			if !summary.Player.IsStarted() {
				summary.Player.Init()
				ok, uri := summary.Player.Wait(movie, summary.ImdbId)
				if ok {
					summary.Player.Play(uri, fmt.Sprintf("%s (%s)", movie.Title, movie.Year))
				}
			}
		}()

	})

	summary.Trailer.ConnectClicked(func(bool) {
		idx := w.TabWidget.CurrentIndex()
		label := widgets.NewQLabelFromPointer(w.TabWidget.TabBar().TabButton(idx, w.Side).Pointer())

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
	w.setLoading(summary.QWidget_PTR(), true)
	w.LabelStatus.SetText(tr("Downloading movie metadata..."))

	summary.Started = true
	tabs = append(tabs, Tab{"", 0, 0, movie, nil, summary})

	go w.Client.Summary(summary, movie)
}

// TailLog watch log file changes
func (w *Window) TailLog() {
	poll := false
	if runtime.GOOS == "windows" {
		poll = true
	}

	t, err := tail.TailFile(filepath.Join(tempDir, "log.txt"), tail.Config{Follow: true, Poll: poll})
	if err != nil {
		log.Printf("ERROR: Trailer: %s\n", err.Error())
		return
	}

	for line := range t.Lines {
		w.Log.ValueChanged(line.Text)
	}
}

// Init initialize window
func (w *Window) Init() {
	go w.TailLog()

	w.Top("Top Movies", bukanir.CategoryMovies)

	go w.Client.Popular(w.Toolbar)
	go w.Client.TopRated(w.Toolbar)
	go w.Client.Genres(w.Toolbar)
}

// setLoading sets loading movie
func (w *Window) setLoading(widget *widgets.QWidget, visible bool) {
	for x := 0; x < w.TabWidget.Count(); x++ {
		tab := w.TabWidget.Widget(x)
		if tab.Pointer() == widget.Pointer() {
			label := w.TabWidget.TabBar().TabButton(x, w.Side)
			label.SetVisible(visible)
		}
	}
}
