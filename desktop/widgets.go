package main

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/network"
	"github.com/therecipe/qt/widgets"

	"github.com/gen2brain/bukanir/lib"
)

//go:generate qtmoc
//go:generate qtrcc

// Object type
type Object struct {
	core.QObject

	_ func(value string) `signal:"finished"`
	_ func(value string) `signal:"finished2"`
	_ func(value string) `signal:"finished3"`
	_ func(value string) `signal:"finished4"`
}

// Object2 type
type Object2 struct {
	core.QObject

	_ func(value string) `signal:"valueChanged"`
}

// LabelStatus type
type LabelStatus struct {
	*Object2
	*widgets.QLabel
}

// List type
type List struct {
	*Object
	*widgets.QListWidget

	Started bool
}

// NewList returns new list
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
			font: 16px;
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

	return &List{NewObject(w), listWidget, false}
}

// Init initialize list
func (l *List) Init(manager *network.QNetworkAccessManager, data string) {
	var movies []bukanir.TMovie
	err := json.Unmarshal([]byte(data), &movies)
	if err != nil {
		log.Printf("ERROR: Unmarshal: %s\n", err.Error())
		return
	}

	if l.Count() > 0 {
		l.Clear()
	}

	for idx, m := range movies {
		item := NewListItem(l, m)
		l.InsertItem(idx, item)

		reply := manager.Get(network.NewQNetworkRequest(core.NewQUrl3(m.PosterLarge, core.QUrl__TolerantMode)))
		reply.ConnectFinished(func() {
			defer reply.DeleteLater()

			if reply.IsReadable() && reply.Error() == network.QNetworkReply__NoError {
				data := reply.ReadAll()
				if data != nil && data.ConstData() != "" {
					image := gui.NewQImage()
					ok := image.LoadFromData2(data, "JPG")
					if ok {
						//item.SetIcon(gui.NewQIcon2(pixmap.Scaled2(300, 450, core.Qt__KeepAspectRatio, core.Qt__SmoothTransformation)))
						item.SetIcon(gui.NewQIcon2(gui.QPixmap_FromImage(image, core.Qt__AutoColor)))
					}
					image.DestroyQImage()
				}
			}
		})
	}

	l.SetFocus2()
	l.SetCurrentRow(0)
}

// ListItem type
type ListItem struct {
	*Object
	*widgets.QListWidgetItem
}

// NewListItem returns new list item
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

	item := widgets.NewQListWidgetItem(l, 0)
	item.SetSizeHint(core.NewQSize2(270, 450))

	item.SetText(fmt.Sprintf("%s\n%s", m.Title, desc))

	movie, _ := json.Marshal(m)
	item.SetData(int(core.Qt__UserRole), core.NewQVariant14(string(movie[:])))

	return &ListItem{NewObject(l), item}
}

// Summary type
type Summary struct {
	*Object
	*widgets.QFrame

	Poster  *widgets.QLabel
	Watch   *widgets.QPushButton
	Trailer *widgets.QPushButton

	Cast     *widgets.QButtonGroup
	Director *widgets.QPushButton

	Video  string
	ImdbId string

	Player *Player

	Started        bool
	TorrentStarted bool

	layoutCast       *widgets.QHBoxLayout
	labelCast        *widgets.QLabel
	labelDirector    *widgets.QLabel
	labelGenre       *widgets.QLabel
	labelOverview    *widgets.QLabel
	labelRatingYear  *widgets.QLabel
	labelRuntimeSize *widgets.QLabel
	labelRelease     *widgets.QLabel
	labelTagline     *widgets.QLabel
	labelTitle       *widgets.QLabel
	labelTmdb        *widgets.QLabel
	labelTmdbLogo    *widgets.QLabel
	labelOpenSubs    *widgets.QLabel
}

// NewSummary returns new summary
func NewSummary(parent *widgets.QTabWidget) *Summary {
	frame := widgets.NewQFrame(parent, core.Qt__Widget)
	widget := widgets.NewQWidget(frame, 0)

	// Labels
	labelTitle := widgets.NewQLabel(widget, 0)
	labelTitle.SetWordWrap(true)
	labelTitle.SetStyleSheet("font: 20pt; font-weight: bold")
	labelTitle.SetMinimumWidth(475)
	labelTitle.SetSizePolicy2(widgets.QSizePolicy__MinimumExpanding, widgets.QSizePolicy__Preferred)

	labelRatingYear := widgets.NewQLabel(widget, 0)
	labelRatingYear.SetWordWrap(true)
	labelRatingYear.SetStyleSheet("font: 12pt")

	labelGenre := widgets.NewQLabel(widget, 0)
	labelGenre.SetWordWrap(true)
	labelGenre.SetStyleSheet("font: 12pt")

	labelRuntimeSize := widgets.NewQLabel(widget, 0)
	labelRuntimeSize.SetWordWrap(true)
	labelRuntimeSize.SetStyleSheet("font: 12pt")

	labelRelease := widgets.NewQLabel(widget, 0)
	labelRelease.SetWordWrap(true)
	labelRelease.SetStyleSheet("font: 10pt")

	labelTagline := widgets.NewQLabel(widget, 0)
	labelTagline.SetWordWrap(true)
	labelTagline.SetStyleSheet("font: 14pt; font-weight: bold")

	labelDirector := widgets.NewQLabel(widget, 0)
	labelDirector.SetWordWrap(true)
	labelDirector.SetStyleSheet("font: 12pt")

	labelCast := widgets.NewQLabel(widget, 0)
	labelCast.SetWordWrap(true)
	labelCast.SetStyleSheet("font: 12pt")

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
	labelOverview.SetStyleSheet("font: 12pt")
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

	layoutDirector := widgets.NewQHBoxLayout()
	layoutDirector.AddWidget(labelDirector, 0, 0)
	vlayout.AddLayout(layoutDirector, 0)

	layoutCast := widgets.NewQHBoxLayout()
	layoutCast.AddWidget(labelCast, 0, 0)
	vlayout.AddLayout(layoutCast, 0)

	vlayout.AddWidget(scrollArea, 0, 0)

	// Poster
	labelPoster := widgets.NewQLabel(widget, 0)
	labelPoster.SetSizePolicy2(widgets.QSizePolicy__MinimumExpanding, widgets.QSizePolicy__Preferred)
	labelPoster.SetScaledContents(true)
	labelPoster.SetMinimumSize2(300, 450)
	labelPoster.SetMaximumSize2(700, 1050)

	// Horizontal layout
	hlayout := widgets.NewQHBoxLayout()
	hlayout.SetSpacing(0)
	hlayout.SetContentsMargins(0, 0, 0, 0)

	hlayout.AddWidget(labelPoster, 0, 0)
	hlayout.AddLayout(vlayout, 0)

	// Tmdb/OpenSubs
	labelTmdb := widgets.NewQLabel(widget, 0)
	labelTmdb.SetStyleSheet("font: 8pt")

	labelTmdbLogo := widgets.NewQLabel(widget, 0)
	labelTmdbLogo.SetStyleSheet("font: 8pt")

	labelOpenSubs := widgets.NewQLabel(widget, 0)
	labelOpenSubs.SetStyleSheet("font: 8pt")

	// Tmdb/OpenSubs layout
	vlayout2 := widgets.NewQVBoxLayout()
	vlayout2.SetSpacing(0)
	vlayout2.SetContentsMargins(0, 0, 0, 0)

	vlayout2.AddWidget(labelTmdbLogo, 0, 0)
	vlayout2.AddWidget(labelTmdb, 0, 0)
	vlayout2.AddWidget(labelOpenSubs, 0, 0)

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
	buttonWatch.SetFocusPolicy(core.Qt__StrongFocus)
	buttonWatch.SetCursor(gui.NewQCursor2(core.Qt__PointingHandCursor))
	buttonWatch.SetText(tr("WATCH"))

	buttonTrailer := widgets.NewQPushButton(widget)
	buttonTrailer.SetSizePolicy2(widgets.QSizePolicy__Minimum, widgets.QSizePolicy__Fixed)
	buttonTrailer.SetStyleSheet(stylesheet)
	buttonTrailer.SetFocusPolicy(core.Qt__StrongFocus)
	buttonTrailer.SetCursor(gui.NewQCursor2(core.Qt__PointingHandCursor))
	buttonTrailer.SetText(tr("TRAILER"))

	buttonGroup := widgets.NewQButtonGroup(widget)

	buttonDirector := widgets.NewQPushButton(widget)
	buttonDirector.SetSizePolicy2(widgets.QSizePolicy__Minimum, widgets.QSizePolicy__Fixed)
	buttonDirector.SetCursor(gui.NewQCursor2(core.Qt__PointingHandCursor))
	layoutDirector.AddWidget(buttonDirector, 0, core.Qt__AlignLeft)
	layoutDirector.AddSpacerItem(widgets.NewQSpacerItem(20, 20, widgets.QSizePolicy__MinimumExpanding, widgets.QSizePolicy__Preferred))

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
	buttonDirector.SetVisible(false)

	return &Summary{
		NewObject(parent), frame, labelPoster, buttonWatch, buttonTrailer, buttonGroup, buttonDirector, "", "", nil, false, false,
		layoutCast, labelCast, labelDirector, labelGenre, labelOverview, labelRatingYear, labelRuntimeSize,
		labelRelease, labelTagline, labelTitle, labelTmdb, labelTmdbLogo, labelOpenSubs,
	}
}

// Init initialize summary
func (l *Summary) Init(m bukanir.TMovie, data string) {
	stylesheet := `
		QPushButton {
			min-width: 5em;
			padding: 6px;
		}

		QPushButton:pressed {
			border-style: inset;
		}

		QPushButton:disabled {
			background-color: gray;
		}
	`

	var s bukanir.TSummary
	err := json.Unmarshal([]byte(data), &s)
	if err != nil {
		log.Printf("ERROR: Unmarshal: %s\n", err.Error())
		return
	}

	l.Video = s.Video
	l.ImdbId = s.ImdbId

	if len(s.CastIds) > 4 {
		s.CastIds = s.CastIds[0:4]
	}

	for n := 0; n <= 4; n++ {
		item := l.layoutCast.TakeAt(1).Widget()
		if item != nil {
			item.DeleteLater()
		}
	}

	for n, _ := range s.CastIds {
		b := widgets.NewQPushButton2(s.Cast[n], l)
		b.SetStyleSheet(stylesheet)
		b.SetCursor(gui.NewQCursor2(core.Qt__PointingHandCursor))
		l.Cast.AddButton(b, n)
		l.layoutCast.AddWidget(b, 0, core.Qt__AlignLeft)
	}

	l.layoutCast.AddSpacerItem(widgets.NewQSpacerItem(20, 20, widgets.QSizePolicy__MinimumExpanding, widgets.QSizePolicy__Preferred))

	l.Director.SetText(s.Director)
	l.Director.SetProperty("id", core.NewQVariant7(s.DirectorId))
	l.Director.SetStyleSheet(stylesheet)

	director := ""
	if s.Director != "" {
		director = "<em>Director:</em> "
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
	tmdbText := tr("This product uses the TMDb API but is not endorsed or certified by TMDb.")
	openSubsText := tr("Subtitles are from opensubtitles.org, podnapisi.net and subscene.com.")

	l.labelCast.SetText("<em>Cast:</em> ")
	l.labelDirector.SetText(director)
	l.labelGenre.SetText(genre)
	l.labelOverview.SetText(s.Overview)
	l.labelRatingYear.SetText(rating + desc)
	l.labelRuntimeSize.SetText(runtime + m.SizeHuman)
	l.labelRelease.SetText(release)
	l.labelTagline.SetText(s.TagLine)
	l.labelTitle.SetText(m.Title)
	l.labelTmdbLogo.SetPixmap(tmdb)
	l.labelTmdb.SetText(tmdbText)
	l.labelOpenSubs.SetText(openSubsText)

	if s.TagLine == "" {
		l.labelTagline.SetEnabled(false)
	}

	if s.Director != "" {
		l.Director.SetVisible(true)
	}

	l.Watch.SetVisible(true)
	l.Trailer.SetVisible(true)

	if l.Video == "" {
		l.Trailer.SetVisible(false)
	}
}

// Toolbar type
type Toolbar struct {
	*Object
	*widgets.QWidget

	Search   *widgets.QToolButton
	Refresh  *widgets.QToolButton
	Log      *widgets.QToolButton
	Settings *widgets.QToolButton
	About    *widgets.QToolButton

	Input    *widgets.QLineEdit
	Media    *widgets.QToolButton
	SortBy   *widgets.QToolButton
	Top      *widgets.QToolButton
	Year     *widgets.QToolButton
	Popular  *widgets.QToolButton
	TopRated *widgets.QToolButton
	Genre    *widgets.QToolButton
}

// NewToolbar returns new toolbar
func NewToolbar(parent *widgets.QWidget) *Toolbar {
	widget := widgets.NewQWidget(parent, 0)
	widget.SetStyleSheet("QMenu {font-size: 11px;} QMenu::item {color: #000000;}")

	lineInput := widgets.NewQLineEdit(widget)
	lineInput.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	lineInput.SetMinimumWidth(200)
	lineInput.SetPlaceholderText(tr("Search"))

	searchButton := widgets.NewQToolButton(widget)
	searchButton.SetIcon(gui.NewQIcon5(":/qml/images/search.png"))
	searchButton.SetIconSize(core.NewQSize2(20, 20))
	searchButton.SetMinimumSize2(25, 25)
	searchButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	searchButton.SetToolTip(tr("Search"))

	refreshButton := widgets.NewQToolButton(widget)
	refreshButton.SetIcon(gui.NewQIcon5(":/qml/images/refresh.png"))
	refreshButton.SetIconSize(core.NewQSize2(20, 20))
	refreshButton.SetMinimumSize2(25, 25)
	refreshButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	refreshButton.SetToolTip("Refresh")

	logButton := widgets.NewQToolButton(widget)
	logButton.SetIcon(gui.NewQIcon5(":/qml/images/log.png"))
	logButton.SetIconSize(core.NewQSize2(20, 20))
	logButton.SetMinimumSize2(25, 25)
	logButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	logButton.SetToolTip(tr("Log"))

	settingsButton := widgets.NewQToolButton(widget)
	settingsButton.SetIcon(gui.NewQIcon5(":/qml/images/settings.png"))
	settingsButton.SetIconSize(core.NewQSize2(20, 20))
	settingsButton.SetMinimumSize2(25, 25)
	settingsButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	settingsButton.SetToolTip(tr("Settings"))

	aboutButton := widgets.NewQToolButton(widget)
	aboutButton.SetIcon(gui.NewQIcon5(":/qml/images/bukanir-gray.png"))
	aboutButton.SetIconSize(core.NewQSize2(20, 20))
	aboutButton.SetMinimumSize2(25, 25)
	aboutButton.SetToolTip(tr("About"))

	mediaButton := widgets.NewQToolButton(widget)
	mediaButton.SetPopupMode(widgets.QToolButton__InstantPopup)
	mediaButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	mediaButton.SetMinimumSize2(47, 27)
	mediaButton.SetText(tr("Media"))

	sortByButton := widgets.NewQToolButton(widget)
	sortByButton.SetPopupMode(widgets.QToolButton__InstantPopup)
	sortByButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	sortByButton.SetMinimumSize2(47, 27)
	sortByButton.SetText(tr("Sort By"))

	topButton := widgets.NewQToolButton(widget)
	topButton.SetPopupMode(widgets.QToolButton__InstantPopup)
	topButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	topButton.SetMinimumSize2(47, 27)
	topButton.SetText(tr("Top"))

	yearButton := widgets.NewQToolButton(widget)
	yearButton.SetPopupMode(widgets.QToolButton__InstantPopup)
	yearButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	yearButton.SetMinimumSize2(47, 27)
	yearButton.SetText(tr("Year"))

	popularButton := widgets.NewQToolButton(widget)
	popularButton.SetPopupMode(widgets.QToolButton__InstantPopup)
	popularButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	popularButton.SetMinimumSize2(47, 27)
	popularButton.SetText(tr("Popular"))

	topRatedButton := widgets.NewQToolButton(widget)
	topRatedButton.SetPopupMode(widgets.QToolButton__InstantPopup)
	topRatedButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	topRatedButton.SetMinimumSize2(47, 27)
	topRatedButton.SetText(tr("Top Rated"))

	byGenreButton := widgets.NewQToolButton(widget)
	byGenreButton.SetPopupMode(widgets.QToolButton__InstantPopup)
	byGenreButton.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	byGenreButton.SetMinimumSize2(47, 27)
	byGenreButton.SetText(tr("Genre"))

	mediaMenu := widgets.NewQMenu(widget)
	mediaActionGroup := widgets.NewQActionGroup(widget)

	act1 := widgets.NewQAction2(tr("All"), widget)
	act1.SetCheckable(true)
	act1.SetChecked(true)

	act2 := widgets.NewQAction2(tr("Movies"), widget)
	act2.SetCheckable(true)

	act3 := widgets.NewQAction2(tr("Episodes"), widget)
	act3.SetCheckable(true)

	mediaActionGroup.AddAction(act1)
	mediaActionGroup.AddAction(act2)
	mediaActionGroup.AddAction(act3)

	mediaMenu.AddActions([]*widgets.QAction{act1, act2, act3})
	mediaButton.SetMenu(mediaMenu)

	sortByMenu := widgets.NewQMenu(widget)
	sortByActionGroup := widgets.NewQActionGroup(widget)

	act1 = widgets.NewQAction2(tr("Seeders"), widget)
	act1.SetCheckable(true)
	act1.SetChecked(true)

	act2 = widgets.NewQAction2(tr("Episodes"), widget)
	act2.SetCheckable(true)

	sortByActionGroup.AddAction(act1)
	sortByActionGroup.AddAction(act2)

	sortByMenu.AddActions([]*widgets.QAction{act1, act2})
	sortByButton.SetMenu(sortByMenu)

	topMenu := widgets.NewQMenu(widget)
	action := topMenu.AddAction(tr("Movies"))
	action.SetData(core.NewQVariant7(bukanir.CategoryMovies))
	action = topMenu.AddAction(tr("HD Movies"))
	action.SetData(core.NewQVariant7(bukanir.CategoryHDmovies))
	action = topMenu.AddAction(tr("TV Shows"))
	action.SetData(core.NewQVariant7(bukanir.CategoryTV))
	action = topMenu.AddAction(tr("HD TV Shows"))
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
	hlayout.AddWidget(mediaButton, 0, 0)
	hlayout.AddWidget(sortByButton, 0, 0)
	hlayout.AddSpacerItem(widgets.NewQSpacerItem(30, 20, widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Preferred))
	hlayout.AddWidget(topButton, 0, 0)
	hlayout.AddWidget(yearButton, 0, 0)
	hlayout.AddWidget(popularButton, 0, 0)
	hlayout.AddWidget(topRatedButton, 0, 0)
	hlayout.AddWidget(byGenreButton, 0, 0)
	hlayout.AddSpacerItem(widgets.NewQSpacerItem(30, 20, widgets.QSizePolicy__Expanding, widgets.QSizePolicy__Preferred))
	hlayout.AddWidget(logButton, 0, 0)
	hlayout.AddWidget(settingsButton, 0, 0)
	hlayout.AddWidget(aboutButton, 0, 0)

	layout := widgets.NewQVBoxLayout()
	layout.AddLayout(hlayout, 0)
	layout.SetSpacing(5)
	layout.SetContentsMargins(0, 0, 0, 0)

	widget.SetLayout(layout)

	toolbar := &Toolbar{NewObject(parent), widget, searchButton, refreshButton, logButton, settingsButton, aboutButton,
		lineInput, mediaButton, sortByButton, topButton, yearButton, popularButton, topRatedButton, byGenreButton}

	toolbar.ConnectFinished2(func(data string) {
		var d []bukanir.TItem
		err := json.Unmarshal([]byte(data), &d)
		if err != nil {
			log.Printf("ERROR: Unmarshal: %s\n", err.Error())
			return
		}

		popularMenu.AddSection(tr(" Movies "))
		for _, p := range d {
			if p.Title != "" && p.Year != "" {
				text := fmt.Sprintf("%s (%s)", p.Title, p.Year)
				a := popularMenu.AddAction(text)
				a.SetData(core.NewQVariant14(p.Title))
			} else {
				popularMenu.AddSection(tr(" TV Shows "))
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
				topRatedMenu.AddSection(tr(" TV Shows "))
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

// SetEnabled sets elements enabled property
func (t *Toolbar) SetEnabled(enabled bool) {
	t.Input.SetEnabled(enabled)
	t.Search.SetEnabled(enabled)
	t.Refresh.SetEnabled(enabled)
	t.Media.SetEnabled(enabled)
	t.SortBy.SetEnabled(enabled)
	t.Top.SetEnabled(enabled)
	t.Year.SetEnabled(enabled)
	t.Popular.SetEnabled(enabled)
	t.TopRated.SetEnabled(enabled)
	t.Genre.SetEnabled(enabled)
}

// Complete completes search queries
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

// Log type
type Log struct {
	*Object2
	*widgets.QDialog

	TextEdit *widgets.QPlainTextEdit
}

// NewLog returns new log
func NewLog(parent *widgets.QWidget) *Log {
	dialog := widgets.NewQDialog(parent, 0)
	dialog.SetWindowTitle(tr("Log"))
	dialog.Resize2(700, 520)

	textEdit := widgets.NewQPlainTextEdit(parent)
	textEdit.SetReadOnly(true)
	textEdit.SetSizePolicy2(widgets.QSizePolicy__Preferred, widgets.QSizePolicy__MinimumExpanding)

	buttonBox := widgets.NewQDialogButtonBox3(widgets.QDialogButtonBox__Close, dialog)
	buttonBox.ConnectRejected(func() { dialog.Close() })

	buttonBox.Button(widgets.QDialogButtonBox__Close).SetText(tr("Close"))

	vlayout := widgets.NewQVBoxLayout()
	vlayout.AddWidget(textEdit, 1, 0)
	vlayout.AddWidget(buttonBox, 0, 0)

	dialog.SetLayout(vlayout)

	return &Log{NewObject2(parent), dialog, textEdit}
}

// NewAbout returns new about
func NewAbout(parent *widgets.QWidget) *widgets.QDialog {
	dialog := widgets.NewQDialog(parent, 0)
	dialog.SetWindowTitle(tr("About"))
	dialog.Resize2(450, 250)

	textBrowser := widgets.NewQTextBrowser(dialog)
	textBrowser.SetOpenExternalLinks(true)
	textBrowser.Append("<center>Bukanir " + bukanir.Version + "</center>")
	textBrowser.Append("<center><a href=\"https://github.com/gen2brain/bukanir\">https://github.com/gen2brain/bukanir</a></center>")
	textBrowser.Append("<br/><center>Author: Milan Nikolić (gen2brain)</center>")
	textBrowser.Append("<center>This program is released under the terms of the</center>")
	textBrowser.Append("<center><a href=\"http://www.gnu.org/licenses/gpl-3.0.txt\">GNU General Public License version 3</a></center><br/>")

	label := widgets.NewQLabel(dialog, 0)
	label.SetPixmap(gui.NewQPixmap5(":/qml/images/bukanir.png", "PNG", core.Qt__AutoColor))

	buttonBox := widgets.NewQDialogButtonBox3(widgets.QDialogButtonBox__Close|widgets.QDialogButtonBox__Help, dialog)
	buttonBox.ConnectRejected(func() { dialog.Close() })
	buttonBox.ConnectHelpRequested(func() { NewHelp(dialog.QWidget_PTR()).Show() })

	buttonBox.Button(widgets.QDialogButtonBox__Close).SetText(tr("Close"))
	buttonBox.Button(widgets.QDialogButtonBox__Help).SetText(tr("Help"))

	hlayout := widgets.NewQHBoxLayout()
	hlayout.AddWidget(label, 0, 0)
	hlayout.AddWidget(textBrowser, 0, 0)

	vlayout := widgets.NewQVBoxLayout()
	vlayout.AddLayout(hlayout, 0)
	vlayout.AddWidget(buttonBox, 0, 0)

	dialog.SetLayout(vlayout)

	return dialog
}

// NewHelp returns new help
func NewHelp(parent *widgets.QWidget) *widgets.QDialog {
	dialog := widgets.NewQDialog(parent, 0)
	dialog.SetWindowTitle(tr("Shortcuts (mpv)"))
	dialog.Resize2(400, 650)

	font := gui.NewQFont()
	font.SetFamily("Monospace")
	font.SetFixedPitch(true)
	font.SetPointSize(10)
	defer font.DestroyQFont()

	textBrowser := widgets.NewQTextBrowser(dialog)
	textBrowser.SetFont(font)

	textBrowser.Append("<ul type=\"none\"><li><b>p</b>  -  Pause/playback mode</li>")
	textBrowser.Append("<li><b>f</b>  -  Toggle fullscreen</li>")
	textBrowser.Append("<li><b>m</b>  -  Mute/unmute audio</li>")
	textBrowser.Append("<li><b>A</b>  -  Cycle aspect ratio</li>")
	textBrowser.Append("<br/>")
	textBrowser.Append("<li><b>v</b>  -  Show/hide subtitles</li>")
	textBrowser.Append("<li><b>j/J</b>  -  Next/previous subtitle</li>")
	textBrowser.Append("<li><b>r/t</b>  -  Move subtitles up / down</li>")
	textBrowser.Append("<li><b>z/x</b>  -  Increase/decrease subtitle delay</li>")
	textBrowser.Append("<br/>")
	textBrowser.Append("<li><b>ctrl++</b>  -  Increase audio delay</li>")
	textBrowser.Append("<li><b>ctrl+-</b>  -  Decrease audio delay</li>")
	textBrowser.Append("<br/>")
	textBrowser.Append("<li><b>Right/Left</b>  -  Seek 5 seconds</li>")
	textBrowser.Append("<li><b>Up/Down</b>  -  Seek 60 seconds</li>")
	textBrowser.Append("<br/>")
	textBrowser.Append("<li><b>1/2</b>  -  Decrease/increase contrast</li>")
	textBrowser.Append("<li><b>3/4</b>  -  Decrease/increase brightness</li>")
	textBrowser.Append("<li><b>5/6</b>  -  Decrease/increase gamma</li>")
	textBrowser.Append("<li><b>7/8</b>  -  Decrease/increase saturation</li>")
	textBrowser.Append("<li><b>9/0</b>  -  Decrease/increase audio volume</li></ul>")

	buttonBox := widgets.NewQDialogButtonBox3(widgets.QDialogButtonBox__Close, dialog)
	buttonBox.ConnectRejected(func() { dialog.Close() })

	buttonBox.Button(widgets.QDialogButtonBox__Close).SetText(tr("Close"))

	vlayout := widgets.NewQVBoxLayout()
	vlayout.AddWidget(textBrowser, 0, 0)
	vlayout.AddWidget(buttonBox, 0, 0)

	dialog.SetLayout(vlayout)

	return dialog
}
