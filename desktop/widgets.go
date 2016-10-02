package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"

	"github.com/gen2brain/bukanir/lib/bukanir"
)

//go:generate qtmoc
type Object struct {
	core.QObject

	_ func(value string) `signal:finished`
	_ func(value string) `signal:finished2`
	_ func(value string) `signal:finished3`
	_ func(value string) `signal:finished4`
}

//go:generate qtmoc
type Object2 struct {
	core.QObject

	_ func(value string) `signal:valueChanged`
}

type LabelStatus struct {
	*Object2
	*widgets.QLabel
}

type List struct {
	*Object
	*widgets.QListWidget

	Started     bool
	Initialized bool
}

func NewList(w *widgets.QTabWidget) *List {
	listWidget := widgets.NewQListWidget(w)
	listWidget.SetUniformItemSizes(true)
	listWidget.SetViewMode(widgets.QListView__IconMode)
	listWidget.SetIconSize(core.NewQSize2(250, 375))
	listWidget.SetResizeMode(widgets.QListView__Adjust)
	listWidget.SetSizePolicy2(widgets.QSizePolicy__Expanding, widgets.QSizePolicy__Expanding)
	listWidget.SetVerticalScrollMode(widgets.QAbstractItemView__ScrollPerPixel)
	listWidget.SetDragEnabled(false)

	listWidget.SetStyleSheet(`
		QListWidget {
			border: 0;
			background-color: black;
		}

		QListWidget::item {
			border: 0;
			padding-top: 10px;
			color: white;
			background-color: black;
		}

		QListWidget::item:selected {
			color: #ffa725;
		}
	`)

	var filterObject = core.NewQObject(w)
	filterObject.ConnectEventFilter(func(watched *core.QObject, event *core.QEvent) bool {
		if event.Type() == core.QEvent__Wheel {
			wheel := gui.NewQWheelEventFromPointer(event.Pointer())
			delta := wheel.AngleDelta()
			if delta.IsNull() {
				return false
			}

			if delta.Y() > 0 {
				listWidget.VerticalScrollBar().SetValue(listWidget.VerticalScrollBar().Value() - 100)
			} else if delta.Y() < 0 {
				listWidget.VerticalScrollBar().SetValue(listWidget.VerticalScrollBar().Value() + 100)
			}
			return true
		}
		return false
	})

	listWidget.VerticalScrollBar().InstallEventFilter(filterObject)

	return &List{NewObject(w), listWidget, false, false}
}

func (l *List) Init(index int, data string) {
	var movies []bukanir.TMovie
	err := json.Unmarshal([]byte(data), &movies)
	if err != nil {
		log.Printf("ERROR: Unmarshal: %s\n", err.Error())
		return
	}

	var refresh bool = false
	if l.Count() > 0 {
		l.Clear()
		refresh = true
	}

	var mcontent map[int]bukanir.TMovie
	mcontent = make(map[int]bukanir.TMovie)

	var mutex sync.RWMutex
	var images map[string][]byte
	images = make(map[string][]byte)

	for idx, m := range movies {
		mcontent[idx] = m

		item := NewListItem(l, m)
		item.ConnectFinished(func(magnet string) {
			mutex.RLock()
			data := string(images[magnet])
			mutex.RUnlock()

			pixmap := gui.NewQPixmap()
			ok := pixmap.LoadFromData2(data, "JPG", core.Qt__AutoColor)
			if ok {
				item.SetIcon(gui.NewQIcon2(pixmap))
			}

			mutex.Lock()
			delete(images, magnet)
			mutex.Unlock()
		})

		l.AddItem2(item)

		go func(item *ListItem, movie bukanir.TMovie) {
			res, err := http.Get(movie.PosterLarge)
			if err != nil {
				log.Printf("ERROR: %s\n", err.Error())
				return
			}

			data, err := ioutil.ReadAll(res.Body)
			if err != nil {
				log.Printf("ERROR: ReadAll: %s\n", err.Error())
				return
			}
			res.Body.Close()

			mutex.Lock()
			images[movie.MagnetLink] = data
			mutex.Unlock()

			item.Finished(movie.MagnetLink)
		}(item, m)
	}

	if refresh {
		content[index] = mcontent
	} else {
		content = append(content, mcontent)
	}

	l.SetFocus2()
	l.SetCurrentRow(0)

	l.Initialized = true
}

type ListItem struct {
	*Object
	*widgets.QListWidgetItem
}

func NewListItem(l *List, m bukanir.TMovie) *ListItem {
	var desc string
	if m.Category == bukanir.CategoryTV || m.Category == bukanir.CategoryHDTV {
		desc = fmt.Sprintf("S%02dE%02d", m.Season, m.Episode)
	} else {
		desc = m.Year
		if m.Quality != "" {
			desc = fmt.Sprintf("%s (%sp)", desc, m.Quality)
		}
	}

	font := gui.NewQFont()
	font.SetPixelSize(16)

	item := widgets.NewQListWidgetItem(l, 0)
	item.SetFont(font)
	item.SetText(fmt.Sprintf("%s\n%s", m.Title, desc))
	item.SetSizeHint(core.NewQSize2(270, 425))

	return &ListItem{NewObject(l), item}
}

type Summary struct {
	*Object
	*widgets.QFrame

	Poster  *widgets.QLabel
	Watch   *widgets.QPushButton
	Trailer *widgets.QPushButton

	Video  string
	ImdbId string

	Player *Player

	Started        bool
	TorrentStarted bool
	Initialized    bool
}

func NewSummary(parent *widgets.QTabWidget) *Summary {
	frame := widgets.NewQFrame(parent, core.Qt__Widget)
	widget := widgets.NewQWidget(frame, 0)

	// Labels
	labelTitle := widgets.NewQLabel(widget, 0)
	labelTitle.SetWordWrap(true)
	labelTitle.SetObjectName("labelTitle")
	labelTitle.Font().SetPointSize(20)
	labelTitle.Font().SetBold(true)
	labelTitle.SetMinimumWidth(475)
	labelTitle.SetSizePolicy2(widgets.QSizePolicy__MinimumExpanding, widgets.QSizePolicy__Preferred)

	labelRatingYear := widgets.NewQLabel(widget, 0)
	labelRatingYear.SetWordWrap(true)
	labelRatingYear.SetObjectName("labelRatingYear")
	labelRatingYear.Font().SetPointSize(12)

	labelGenre := widgets.NewQLabel(widget, 0)
	labelGenre.SetWordWrap(true)
	labelGenre.SetObjectName("labelGenre")
	labelGenre.Font().SetPointSize(12)

	labelRuntimeSize := widgets.NewQLabel(widget, 0)
	labelRuntimeSize.SetWordWrap(true)
	labelRuntimeSize.SetObjectName("labelRuntimeSize")
	labelRuntimeSize.Font().SetPointSize(12)

	labelRelease := widgets.NewQLabel(widget, 0)
	labelRelease.SetWordWrap(true)
	labelRelease.SetObjectName("labelRelease")
	labelRelease.Font().SetPointSize(10)

	labelTagline := widgets.NewQLabel(widget, 0)
	labelTagline.SetWordWrap(true)
	labelTagline.SetObjectName("labelTagline")
	labelTagline.Font().SetPointSize(14)
	labelTagline.Font().SetBold(true)

	labelDirector := widgets.NewQLabel(widget, 0)
	labelDirector.SetWordWrap(true)
	labelDirector.SetObjectName("labelDirector")
	labelDirector.Font().SetPointSize(12)

	labelCast := widgets.NewQLabel(widget, 0)
	labelCast.SetWordWrap(true)
	labelCast.SetObjectName("labelCast")
	labelCast.Font().SetPointSize(12)

	// Overview scroll
	scrollArea := widgets.NewQScrollArea(widget)
	scrollArea.SetSizePolicy2(widgets.QSizePolicy__Preferred, widgets.QSizePolicy__Expanding)
	scrollArea.SetAlignment(core.Qt__AlignLeft | core.Qt__AlignTop)
	scrollArea.SetVerticalScrollBarPolicy(core.Qt__ScrollBarAsNeeded)
	scrollArea.SetHorizontalScrollBarPolicy(core.Qt__ScrollBarAlwaysOff)
	scrollArea.SetWidgetResizable(true)
	scrollArea.SetFrameShape(widgets.QFrame__NoFrame)

	labelOverview := widgets.NewQLabel(widget, 0)
	labelOverview.SetWordWrap(true)
	labelOverview.SetObjectName("labelOverview")
	labelOverview.Font().SetPointSize(12)
	labelOverview.SetAlignment(core.Qt__AlignLeft | core.Qt__AlignTop)

	slayout := widgets.NewQVBoxLayout()
	slayout.SetSpacing(15)
	slayout.SetContentsMargins(0, 0, 0, 0)
	labelOverview.SetLayout(slayout)

	scrollArea.SetWidget(labelOverview)

	// Labels layout
	vlayout := widgets.NewQVBoxLayout()
	vlayout.SetSpacing(15)
	vlayout.SetContentsMargins(10, 5, 10, 0)

	vlayout.AddWidget(labelTitle, 0, 0)
	vlayout.AddWidget(labelRatingYear, 0, 0)
	vlayout.AddWidget(labelGenre, 0, 0)
	vlayout.AddWidget(labelRuntimeSize, 0, 0)
	vlayout.AddWidget(labelRelease, 0, 0)
	vlayout.AddWidget(labelTagline, 0, 0)
	vlayout.AddWidget(labelDirector, 0, 0)
	vlayout.AddWidget(labelCast, 0, 0)
	vlayout.AddWidget(scrollArea, 0, 0)

	// Poster
	labelPoster := widgets.NewQLabel(widget, 0)
	labelPoster.SetSizePolicy2(widgets.QSizePolicy__MinimumExpanding, widgets.QSizePolicy__Preferred)
	labelPoster.SetScaledContents(true)
	labelPoster.SetMinimumSize2(300, 450)
	labelPoster.SetMaximumSize2(700, 1050)
	labelPoster.SetObjectName("labelPoster")

	// Horizontal layout
	hlayout := widgets.NewQHBoxLayout()
	hlayout.SetSpacing(0)
	hlayout.SetContentsMargins(0, 0, 0, 0)

	hlayout.AddWidget(labelPoster, 0, 0)
	hlayout.AddLayout(vlayout, 0)

	// Tmdb
	labelTmdb := widgets.NewQLabel(widget, 0)
	labelTmdb.SetObjectName("labelTmdb")
	labelTmdb.Font().SetPointSize(8)

	labelTmdbLogo := widgets.NewQLabel(widget, 0)
	labelTmdbLogo.SetObjectName("labelTmdbLogo")
	labelTmdbLogo.Font().SetPointSize(8)

	// Tmdb layout
	vlayout2 := widgets.NewQVBoxLayout()
	vlayout2.SetSpacing(0)
	vlayout2.SetContentsMargins(0, 0, 0, 0)

	vlayout2.AddWidget(labelTmdbLogo, 0, 0)
	vlayout2.AddWidget(labelTmdb, 0, 0)

	// Buttons
	stylesheet := `
		QPushButton {
			background-color: #ffa725;
			border-style: outset;
			border-width: 2px;
			border-radius: 5px;
			border-color: beige;
			font: bold 20px;
			min-width: 5em;
			padding: 6px;
		}

		QPushButton:pressed {
			background-color: #ac6616;
			border-style: inset;
		}

		QPushButton:disabled {
			background-color: gray;
		}
	`

	buttonWatch := widgets.NewQPushButton(widget)
	buttonWatch.SetSizePolicy2(widgets.QSizePolicy__Minimum, widgets.QSizePolicy__Fixed)
	buttonWatch.SetStyleSheet(stylesheet)
	buttonWatch.SetCursor(gui.NewQCursor2(core.Qt__PointingHandCursor))
	buttonWatch.SetText("WATCH")

	buttonTrailer := widgets.NewQPushButton(widget)
	buttonTrailer.SetSizePolicy2(widgets.QSizePolicy__Minimum, widgets.QSizePolicy__Fixed)
	buttonTrailer.SetStyleSheet(stylesheet)
	buttonTrailer.SetCursor(gui.NewQCursor2(core.Qt__PointingHandCursor))
	buttonTrailer.SetText("TRAILER")

	// Buttons layout
	hlayout2 := widgets.NewQHBoxLayout()
	hlayout2.SetSpacing(6)
	hlayout2.SetContentsMargins(10, 10, 10, 10)

	hlayout2.AddWidget(buttonWatch, 0, 0)
	hlayout2.AddSpacerItem(widgets.NewQSpacerItem(40, 20, widgets.QSizePolicy__MinimumExpanding, widgets.QSizePolicy__Preferred))
	hlayout2.AddWidget(buttonTrailer, 0, 0)

	// Widget layout
	wlayout := widgets.NewQVBoxLayout()
	wlayout.SetSpacing(6)
	wlayout.SetContentsMargins(0, 0, 0, 0)

	wlayout.AddLayout(hlayout, 0)
	wlayout.AddLayout(vlayout2, 0)
	wlayout.AddLayout(hlayout2, 0)

	widget.SetLayout(wlayout)
	widget.Resize2(800, 555)
	widget.SetStyleSheet("color: white; background-color: black;")

	// Frame layout
	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(widget, 0, 0)
	layout.SetSpacing(0)
	layout.SetContentsMargins(10, 10, 10, 10)

	frame.SetLayout(layout)
	frame.SetStyleSheet("QFrame { background-color: black; }")

	buttonWatch.SetVisible(false)
	buttonTrailer.SetVisible(false)

	return &Summary{NewObject(parent), frame, labelPoster, buttonWatch, buttonTrailer, "", "", nil, false, false, false}
}

func (l *Summary) Init(index int, m bukanir.TMovie, data string) {
	var s bukanir.TSummary
	err := json.Unmarshal([]byte(data), &s)
	if err != nil {
		log.Printf("ERROR: Unmarshal: %s\n", err.Error())
		return
	}

	l.Video = s.Video
	l.ImdbId = s.ImdbId

	labelCast := widgets.NewQLabelFromPointer(l.QFrame.FindChild("labelCast", core.Qt__FindChildrenRecursively))
	labelDirector := widgets.NewQLabelFromPointer(l.QFrame.FindChild("labelDirector", core.Qt__FindChildrenRecursively))
	labelGenre := widgets.NewQLabelFromPointer(l.QFrame.FindChild("labelGenre", core.Qt__FindChildrenRecursively))
	labelOverview := widgets.NewQLabelFromPointer(l.QFrame.FindChild("labelOverview", core.Qt__FindChildrenRecursively))
	labelRatingYear := widgets.NewQLabelFromPointer(l.QFrame.FindChild("labelRatingYear", core.Qt__FindChildrenRecursively))
	labelRuntimeSize := widgets.NewQLabelFromPointer(l.QFrame.FindChild("labelRuntimeSize", core.Qt__FindChildrenRecursively))
	labelRelease := widgets.NewQLabelFromPointer(l.QFrame.FindChild("labelRelease", core.Qt__FindChildrenRecursively))
	labelTagline := widgets.NewQLabelFromPointer(l.QFrame.FindChild("labelTagline", core.Qt__FindChildrenRecursively))
	labelTitle := widgets.NewQLabelFromPointer(l.QFrame.FindChild("labelTitle", core.Qt__FindChildrenRecursively))
	labelTmdb := widgets.NewQLabelFromPointer(l.QFrame.FindChild("labelTmdb", core.Qt__FindChildrenRecursively))
	labelTmdbLogo := widgets.NewQLabelFromPointer(l.QFrame.FindChild("labelTmdbLogo", core.Qt__FindChildrenRecursively))

	var refresh bool = false
	if labelTitle.Text() != "" {
		refresh = true
	}

	cast := ""
	if len(s.Cast) >= 4 {
		cast = "<em>Cast:</em> " + strings.Join(s.Cast[:4], ", ") + "..."
	} else if len(s.Cast) != 0 {
		cast = "<em>Cast:</em> " + strings.Join(s.Cast[:len(s.Cast)], ", ")
	}

	director := ""
	if s.Director != "" {
		director = "<em>Director:</em> " + s.Director
	}

	genre := ""
	if len(s.Genre) > 0 {
		genre = strings.Join(s.Genre, ", ")
	}

	rating := ""
	if s.Rating != 0 {
		rating = fmt.Sprintf("%.1f/10 ", s.Rating)
	}

	desc := ""
	if m.Category == bukanir.CategoryTV || m.Category == bukanir.CategoryHDTV {
		desc = fmt.Sprintf("S%02dE%02d", m.Season, m.Episode)
	} else {
		if m.Year != "" {
			desc = fmt.Sprintf("(%s)", m.Year)
		}
	}

	runtime := ""
	if s.Runtime != 0 {
		runtime = fmt.Sprintf("%dmin / ", s.Runtime)
	}

	release := ""
	if m.Release != "" {
		release = fmt.Sprintf("(%s)", m.Release)
	}

	tmdb := gui.NewQPixmap5(":/qml/images/tmdb.png", "PNG", core.Qt__AutoColor)
	tmdbText := "This product uses the TMDb API but is not endorsed or certified by TMDb."

	labelCast.SetText(cast)
	labelDirector.SetText(director)
	labelGenre.SetText(genre)
	labelOverview.SetText(s.Overview)
	labelRatingYear.SetText(rating + desc)
	labelRuntimeSize.SetText(runtime + m.SizeHuman)
	labelRelease.SetText(release)
	labelTagline.SetText(s.TagLine)
	labelTitle.SetText(m.Title)
	labelTmdbLogo.SetPixmap(tmdb)
	labelTmdb.SetText(tmdbText)

	if s.TagLine == "" {
		labelTagline.SetEnabled(false)
	}

	l.Watch.SetVisible(true)
	l.Trailer.SetVisible(true)

	if l.Video == "" {
		l.Trailer.SetVisible(false)
	}

	var mcontent map[int]bukanir.TMovie
	mcontent = make(map[int]bukanir.TMovie)

	if refresh {
		content[index] = mcontent
	} else {
		content = append(content, mcontent)
	}

	l.Initialized = true
}

type Toolbar struct {
	*Object
	*widgets.QWidget

	Search   *widgets.QPushButton
	Refresh  *widgets.QPushButton
	Log      *widgets.QPushButton
	Settings *widgets.QPushButton
	About    *widgets.QPushButton

	Input    *widgets.QLineEdit
	Top      *widgets.QToolButton
	Year     *widgets.QToolButton
	Popular  *widgets.QToolButton
	TopRated *widgets.QToolButton
	Genre    *widgets.QToolButton
}

func NewToolbar(parent *widgets.QWidget) *Toolbar {
	widget := widgets.NewQWidget(parent, 0)
	widget.SetStyleSheet("QMenu {font-size: 11px;} QMenu::item {color: #000000;}")

	lineInput := widgets.NewQLineEdit(widget)
	lineInput.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	lineInput.SetMinimumWidth(200)
	lineInput.SetPlaceholderText("Search")

	searchButton := widgets.NewQPushButton(widget)
	searchButton.SetIcon(gui.NewQIcon5(":/qml/images/search.png"))
	searchButton.SetIconSize(core.NewQSize2(20, 20))
	searchButton.SetSizePolicy2(widgets.QSizePolicy__Minimum, widgets.QSizePolicy__Fixed)
	searchButton.SetCursor(gui.NewQCursor2(core.Qt__PointingHandCursor))
	searchButton.SetToolTip("Search")

	refreshButton := widgets.NewQPushButton(widget)
	refreshButton.SetIcon(gui.NewQIcon5(":/qml/images/refresh.png"))
	refreshButton.SetIconSize(core.NewQSize2(20, 20))
	refreshButton.SetSizePolicy2(widgets.QSizePolicy__Minimum, widgets.QSizePolicy__Fixed)
	refreshButton.SetCursor(gui.NewQCursor2(core.Qt__PointingHandCursor))
	refreshButton.SetToolTip("Refresh")

	logButton := widgets.NewQPushButton(widget)
	logButton.SetIcon(gui.NewQIcon5(":/qml/images/log.png"))
	logButton.SetIconSize(core.NewQSize2(20, 20))
	logButton.SetSizePolicy2(widgets.QSizePolicy__Minimum, widgets.QSizePolicy__Fixed)
	logButton.SetCursor(gui.NewQCursor2(core.Qt__PointingHandCursor))
	logButton.SetToolTip("Log")

	settingsButton := widgets.NewQPushButton(widget)
	settingsButton.SetIcon(gui.NewQIcon5(":/qml/images/settings.png"))
	settingsButton.SetIconSize(core.NewQSize2(20, 20))
	settingsButton.SetSizePolicy2(widgets.QSizePolicy__Minimum, widgets.QSizePolicy__Fixed)
	settingsButton.SetCursor(gui.NewQCursor2(core.Qt__PointingHandCursor))
	settingsButton.SetToolTip("Settings")

	aboutButton := widgets.NewQPushButton(widget)
	aboutButton.SetStyleSheet("QPushButton {border: 0; margin: 0; padding: 0; outline: 0;}")
	aboutButton.SetIcon(gui.NewQIcon5(":/qml/images/bukanir-gray.png"))
	aboutButton.SetIconSize(core.NewQSize2(24, 24))
	aboutButton.SetCursor(gui.NewQCursor2(core.Qt__PointingHandCursor))
	aboutButton.SetToolTip("About")

	topButton := widgets.NewQToolButton(widget)
	topButton.SetPopupMode(widgets.QToolButton__InstantPopup)
	topButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	topButton.SetMinimumSize2(45, 26)
	topButton.SetText("Top")

	yearButton := widgets.NewQToolButton(widget)
	yearButton.SetPopupMode(widgets.QToolButton__InstantPopup)
	yearButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	yearButton.SetMinimumSize2(45, 26)
	yearButton.SetText("Year")

	popularButton := widgets.NewQToolButton(widget)
	popularButton.SetPopupMode(widgets.QToolButton__InstantPopup)
	popularButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	popularButton.SetMinimumSize2(45, 26)
	popularButton.SetText("Popular")

	topRatedButton := widgets.NewQToolButton(widget)
	topRatedButton.SetPopupMode(widgets.QToolButton__InstantPopup)
	topRatedButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	topRatedButton.SetMinimumSize2(45, 26)
	topRatedButton.SetText("Top Rated")

	byGenreButton := widgets.NewQToolButton(widget)
	byGenreButton.SetPopupMode(widgets.QToolButton__InstantPopup)
	byGenreButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	byGenreButton.SetMinimumSize2(45, 26)
	byGenreButton.SetText("Genre")

	topMenu := widgets.NewQMenu(widget)
	action := topMenu.AddAction("Movies")
	action.SetData(core.NewQVariant7(bukanir.CategoryMovies))
	action = topMenu.AddAction("HD Movies")
	action.SetData(core.NewQVariant7(bukanir.CategoryHDmovies))
	action = topMenu.AddAction("TV Shows")
	action.SetData(core.NewQVariant7(bukanir.CategoryTV))
	action = topMenu.AddAction("HD TV Shows")
	action.SetData(core.NewQVariant7(bukanir.CategoryHDTV))
	topButton.SetMenu(topMenu)

	yearMenu := widgets.NewQMenu(widget)
	yearButton.SetMenu(yearMenu)

	currYear := time.Now().Year()
	for i := currYear; i > 1937; i-- {
		action := yearMenu.AddAction(strconv.Itoa(i))
		action.SetData(core.NewQVariant7(i))
	}

	popularMenu := widgets.NewQMenu(widget)
	popularButton.SetMenu(popularMenu)

	topRatedMenu := widgets.NewQMenu(widget)
	topRatedButton.SetMenu(topRatedMenu)

	byGenreMenu := widgets.NewQMenu(widget)
	byGenreButton.SetMenu(byGenreMenu)

	hlayout := widgets.NewQHBoxLayout()
	hlayout.SetSpacing(5)
	hlayout.SetContentsMargins(5, 5, 5, 5)

	hlayout.AddWidget(lineInput, 0, 0)
	hlayout.AddWidget(searchButton, 0, 0)
	hlayout.AddWidget(refreshButton, 0, 0)
	hlayout.AddWidget(topButton, 0, 0)
	hlayout.AddWidget(yearButton, 0, 0)
	hlayout.AddWidget(popularButton, 0, 0)
	hlayout.AddWidget(topRatedButton, 0, 0)
	hlayout.AddWidget(byGenreButton, 0, 0)
	hlayout.AddSpacerItem(widgets.NewQSpacerItem(40, 20, widgets.QSizePolicy__Expanding, widgets.QSizePolicy__Preferred))
	hlayout.AddWidget(logButton, 0, 0)
	hlayout.AddWidget(settingsButton, 0, 0)
	hlayout.AddWidget(aboutButton, 0, 0)

	layout := widgets.NewQVBoxLayout()
	layout.AddLayout(hlayout, 0)
	layout.SetSpacing(6)
	layout.SetContentsMargins(0, 0, 0, 0)

	widget.SetLayout(layout)

	toolbar := &Toolbar{NewObject(parent), widget, searchButton, refreshButton, logButton, settingsButton, aboutButton,
		lineInput, topButton, yearButton, popularButton, topRatedButton, byGenreButton}

	toolbar.ConnectFinished2(func(data string) {
		var d []bukanir.TItem
		err := json.Unmarshal([]byte(data), &d)
		if err != nil {
			log.Printf("ERROR: Unmarshal: %s\n", err.Error())
			return
		}

		popularMenu.AddSection(" Movies ")
		for _, p := range d {
			if p.Title != "" && p.Year != "" {
				text := fmt.Sprintf("%s (%s)", p.Title, p.Year)
				a := popularMenu.AddAction(text)
				a.SetData(core.NewQVariant14(p.Title))
			} else {
				popularMenu.AddSection(" TV Shows ")
			}
		}

	})

	toolbar.ConnectFinished3(func(data string) {
		var d []bukanir.TItem
		err := json.Unmarshal([]byte(data), &d)
		if err != nil {
			log.Printf("ERROR: Unmarshal: %s\n", err.Error())
			return
		}

		topRatedMenu.AddSection(" Movies ")
		for _, t := range d {
			if t.Title != "" && t.Year != "" {
				text := fmt.Sprintf("%s (%s)", t.Title, t.Year)
				a := topRatedMenu.AddAction(text)
				a.SetData(core.NewQVariant14(t.Title))
			} else {
				topRatedMenu.AddSection(" TV Shows ")
			}
		}
	})

	toolbar.ConnectFinished4(func(data string) {
		var d []bukanir.TGenre
		err := json.Unmarshal([]byte(data), &d)
		if err != nil {
			log.Printf("ERROR: Unmarshal: %s\n", err.Error())
			return
		}

		for _, t := range d {
			if t.Id != 0 && t.Name != "" {
				a := byGenreMenu.AddAction(t.Name)
				a.SetData(core.NewQVariant7(t.Id))
			}
		}
	})

	return toolbar
}

func (t *Toolbar) SetEnabled(enabled bool) {
	t.Input.SetEnabled(enabled)
	t.Search.SetEnabled(enabled)
	t.Refresh.SetEnabled(enabled)
	t.Top.SetEnabled(enabled)
	t.Year.SetEnabled(enabled)
	t.Popular.SetEnabled(enabled)
	t.TopRated.SetEnabled(enabled)
	t.Genre.SetEnabled(enabled)
}

func (t *Toolbar) Complete(w *Window, data string) {
	var c []bukanir.TItem
	err := json.Unmarshal([]byte(data), &c)
	if err != nil {
		log.Printf("ERROR: Unmarshal: %s\n", err.Error())
		return
	}

	list := make([]string, 0)
	for _, a := range c {
		if !inSlice(a.Title, list) {
			list = append(list, a.Title)
		}
	}

	w.Model.SetStringList(list)
	w.Completer.Complete(core.NewQRect())
}

type Log struct {
	*Object2
	*widgets.QDialog

	TextEdit *widgets.QPlainTextEdit
}

func NewLog(parent *widgets.QWidget) *Log {
	dialog := widgets.NewQDialog(parent, 0)
	dialog.SetWindowTitle("Log")
	dialog.Resize2(700, 520)

	textEdit := widgets.NewQPlainTextEdit(parent)
	textEdit.SetReadOnly(true)
	textEdit.SetSizePolicy2(widgets.QSizePolicy__Preferred, widgets.QSizePolicy__MinimumExpanding)

	buttonBox := widgets.NewQDialogButtonBox3(widgets.QDialogButtonBox__Close, dialog)
	buttonBox.ConnectRejected(func() { dialog.Close() })

	vlayout := widgets.NewQVBoxLayout()
	vlayout.AddWidget(textEdit, 1, 0)
	vlayout.AddWidget(buttonBox, 0, 0)

	dialog.SetLayout(vlayout)

	return &Log{NewObject2(parent), dialog, textEdit}
}

func NewAbout(parent *widgets.QWidget) *widgets.QDialog {
	dialog := widgets.NewQDialog(parent, 0)
	dialog.SetWindowTitle("About")
	dialog.Resize2(450, 190)

	textBrowser := widgets.NewQTextBrowser(dialog)
	textBrowser.SetOpenExternalLinks(true)
	textBrowser.Append("<center>Bukanir " + bukanir.Version + "</center>")
	textBrowser.Append("<center><a href=\"https://bukanir.com\">https://bukanir.com</a></center>")
	textBrowser.Append("<br/><center>Author: Milan Nikolić (gen2brain)</center>")
	textBrowser.Append("<center>This program is released under the terms of the</center>")
	textBrowser.Append("<center><a href=\"http://www.gnu.org/licenses/gpl-3.0.txt\">GNU General Public License version 3</a></center><br/>")

	label := widgets.NewQLabel(dialog, 0)
	label.SetPixmap(gui.NewQPixmap5(":/qml/images/bukanir.png", "PNG", core.Qt__AutoColor))

	buttonBox := widgets.NewQDialogButtonBox3(widgets.QDialogButtonBox__Close|widgets.QDialogButtonBox__Help, dialog)
	buttonBox.ConnectRejected(func() { dialog.Close() })
	buttonBox.ConnectHelpRequested(func() { NewHelp(dialog.QWidget_PTR()).Show() })

	hlayout := widgets.NewQHBoxLayout()
	hlayout.AddWidget(label, 0, 0)
	hlayout.AddWidget(textBrowser, 0, 0)

	vlayout := widgets.NewQVBoxLayout()
	vlayout.AddLayout(hlayout, 0)
	vlayout.AddWidget(buttonBox, 0, 0)

	dialog.SetLayout(vlayout)

	return dialog
}

func NewHelp(parent *widgets.QWidget) *widgets.QDialog {
	dialog := widgets.NewQDialog(parent, 0)
	dialog.SetWindowTitle("Shortcuts (mpv)")
	dialog.Resize2(400, 350)

	font := gui.NewQFont()
	font.SetFamily("Monospace")
	font.SetFixedPitch(true)
	font.SetPointSize(10)

	textBrowser := widgets.NewQTextBrowser(dialog)
	textBrowser.SetFont(font)

	textBrowser.Append("<ul type=\"none\"><li><b>p</b>\t\tPause/playback mode</li>")
	textBrowser.Append("<li><b>f</b>\t\tToggle fullscreen</li>")
	textBrowser.Append("<li><b>m</b>\t\tMute/unmute audio</li>")
	textBrowser.Append("<li><b>A</b>\t\tCycle aspect ratio</li>")
	textBrowser.Append("<br/>")
	textBrowser.Append("<li><b>v</b>\t\tShow/hide subtitles</li>")
	textBrowser.Append("<li><b>j/J</b>\t\tNext/previous subtitle</li>")
	textBrowser.Append("<li><b>r/t</b>\t\tMove subtitles up / down</li>")
	textBrowser.Append("<li><b>z/x</b>\t\tIncrease/decrease subtitle delay</li>")
	textBrowser.Append("<br/>")
	textBrowser.Append("<li><b>ctrl++</b>\t\tIncrease audio delay</li>")
	textBrowser.Append("<li><b>ctrl+-</b>\t\tDecrease audio delay</li>")
	textBrowser.Append("<br/>")
	textBrowser.Append("<li><b>RIGHT/LEFT</b>\t\tSeek 5 seconds</li>")
	textBrowser.Append("<li><b>UP/DOWN</b>\t\tSeek 60 seconds</li>")
	textBrowser.Append("<br/>")
	textBrowser.Append("<li><b>1/2</b>\t\tDecrease/increase contrast</li>")
	textBrowser.Append("<li><b>3/4</b>\t\tDecrease/increase brightness</li>")
	textBrowser.Append("<li><b>5/6</b>\t\tDecrease/increase gamma</li>")
	textBrowser.Append("<li><b>7/8</b>\t\tDecrease/increase saturation</li>")
	textBrowser.Append("<li><b>9/0</b>\t\tDecrease/increase audio volume</li></ul>")

	buttonBox := widgets.NewQDialogButtonBox3(widgets.QDialogButtonBox__Close, dialog)
	buttonBox.ConnectRejected(func() { dialog.Close() })

	vlayout := widgets.NewQVBoxLayout()
	vlayout.AddWidget(textBrowser, 0, 0)
	vlayout.AddWidget(buttonBox, 0, 0)

	dialog.SetLayout(vlayout)

	return dialog
}
