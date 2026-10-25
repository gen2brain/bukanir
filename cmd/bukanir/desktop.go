//go:build media && !android && !ios

package main

import (
	"fmt"
	"image"
	"math"
	"runtime"
	"strconv"
	"sync/atomic"

	"github.com/gen2brain/iup-go/iup"
)

const (
	gridGap    = 30
	gridMargin = 20
)

const (
	posterW = 200
	posterH = 300
)

const summaryPosterW, summaryPosterH = 600, 900

var summaryPosterSeq atomic.Int64

// phone selects the layouts for a narrow touch screen.
const phone = false

var (
	tabActs   = map[iup.Ihandle]*tabActivity{}
	spinIndex int
	spinTimer iup.Ihandle
	spinning  bool
)

// mainDialog creates the main window: toolbar, tabbed content, and status bar.
func mainDialog() iup.Ihandle {
	dlg := iup.Dialog(layout()).SetAttributes(map[string]string{
		"TITLE":      appName,
		"ICON":       "appicon",
		"RASTERSIZE": "1000x720",
		"SHRINK":     "YES",
	}).SetHandle("dlg")
	dlg.SetCallback("K_ANY", iup.KAnyFunc(onKey))
	// A MultiBox reports its wrapped height one layout pass late; refresh again after a resize.
	dlg.SetCallback("RESIZE_CB", iup.ResizeFunc(func(ih iup.Ihandle, _, _ int) int {
		iup.Refresh(ih)
		return iup.DEFAULT
	}))
	return dlg
}

// exitApp ends the main loop once quitting has cleaned up.
func exitApp() { iup.ExitLoop() }

// showMainDialog maps the main window, widens it to fit the toolbar, and shows it centered.
func showMainDialog(dlg iup.Ihandle) {
	iup.Map(dlg)
	fitDialogWidth(dlg)
	iup.ShowXY(dlg, iup.CENTER, iup.CENTER)
}

// fitDialogWidth widens dlg when its content needs more than the initial width.
func fitDialogWidth(dlg iup.Ihandle) {
	iup.Refresh(dlg)
	var nw, nh, cw, ch, rw, rh int
	fmt.Sscanf(iup.GetChild(dlg, 0).GetAttribute("NATURALSIZE"), "%dx%d", &nw, &nh)
	fmt.Sscanf(dlg.GetAttribute("CLIENTSIZE"), "%dx%d", &cw, &ch)
	fmt.Sscanf(dlg.GetAttribute("RASTERSIZE"), "%dx%d", &rw, &rh)
	if cw > 0 && nw > cw {
		dlg.SetAttribute("RASTERSIZE", fmt.Sprintf("%dx%d", rw+nw-cw, rh))
	}
}

// layout builds the root widget tree: toolbar, tabbed content, and status bar.
func layout() iup.Ihandle {
	return iup.Vbox(
		toolbar(),
		iup.Label("").SetAttributes("SEPARATOR=HORIZONTAL"),
		content(),
		placeholder("Search for a title or pick one from the menus").SetHandle("noTabs").SetAttributes("VISIBLE=NO, FLOATING=IGNORE"),
		iup.Label("").SetAttributes("SEPARATOR=HORIZONTAL"),
		statusBar(),
		suggestList(),
	).SetAttributes("MARGIN=5x5, GAP=4")
}

// toolbar is the top row: search box, refresh, filter/list menu buttons, gear menu.
func toolbar() iup.Ihandle {
	return iup.Hbox(
		searchBox(),
		iconButton("refresh", onRefreshClick).SetAttributes("TIP=Refresh"),
		menuButton(mediaLabel(), "btnMedia", mediaMenu),
		menuButton(sortLabel(), "btnSort", sortMenu),
		menuButton("Top ▾", "", topMenu),
		menuButton("Year ▾", "", yearMenu),
		menuButton("Popular ▾", "", popularMenu),
		menuButton("Top Rated ▾", "", topRatedMenu),
		menuButton("Genre ▾", "", genreMenu),
		iup.Fill(),
		iconButton("star", openFavorites).SetAttributes("TIP=Favorites"),
		iconButton("log", showLogView).SetAttributes("TIP=Log"),
		iconButton("settings", onSettings).SetAttributes("TIP=Settings"),
		logoButton(),
	).SetAttributes("GAP=3, ALIGNMENT=ACENTER")
}

// menuButton is a text toolbar button that pops the menu returned by build.
func menuButton(label, handle string, build func() iup.Ihandle) iup.Ihandle {
	btn := iup.Button(label).SetAttributes("PADDING=10x4")
	if handle != "" {
		btn.SetHandle(handle)
	}
	btn.SetCallback("ACTION", popMenuOn(build))
	return btn
}

// popMenuOn returns an ACTION callback that pops build's menu below the button.
func popMenuOn(build func() iup.Ihandle) iup.ActionFunc {
	return iup.ActionFunc(func(ih iup.Ihandle) int {
		menu := build()
		if menu == 0 {
			return iup.DEFAULT
		}
		_, x, y := iup.GetInt2(ih, "SCREENPOSITION")
		var w, h int
		fmt.Sscanf(ih.GetAttribute("RASTERSIZE"), "%dx%d", &w, &h)
		iup.Popup(menu, x, y+h)
		iup.Destroy(menu)
		return iup.DEFAULT
	})
}

// showFilters shows the media filter and the sort order on their toolbar buttons.
func showFilters() {
	setButtonTitle("btnMedia", mediaLabel())
	setButtonTitle("btnSort", sortLabel())
}

// setButtonTitle changes a toolbar button's title and lays the dialog out again for its new width.
func setButtonTitle(handle, title string) {
	btn := iup.GetHandle(handle)
	if btn.GetAttribute("TITLE") == title {
		return
	}
	btn.SetAttribute("TITLE", title)
	iup.Refresh(btn)
}

func mediaLabel() string { return "Media: " + titleCase(media) + " ▾" }
func sortLabel() string  { return "Sort: " + titleCase(sortBy) + " ▾" }

// menusLoaded is called on the main loop when background menu data arrives; the toolbar menus are built when opened.
func menusLoaded() {}

// searchBox is the query field driving the autocomplete list; it keeps the focus while the list is open.
func searchBox() iup.Ihandle {
	search := iup.Text().SetAttributes("CUEBANNER=Search, VISIBLECOLUMNS=18").SetHandle("search")
	search.SetCallback("VALUECHANGED_CB", iup.ValueChangedFunc(func(ih iup.Ihandle) int {
		onSearchChanged(ih.GetAttribute("VALUE"))
		return iup.DEFAULT
	}))
	search.SetCallback("KILLFOCUS_CB", iup.KillFocusFunc(func(iup.Ihandle) int {
		hideSuggestions()
		return iup.DEFAULT
	}))
	search.SetCallback("K_ANY", iup.KAnyFunc(func(_ iup.Ihandle, c int) int {
		list := iup.GetHandle("suggest")
		open := list.GetAttribute("VISIBLE") == "YES"
		switch c {
		case iup.K_CR:
			if open && list.GetInt("VALUE") > 0 {
				acceptSuggestion(list.GetInt("VALUE"))
			} else {
				onSearchClick()
			}
			return iup.IGNORE
		case iup.K_DOWN, iup.K_UP:
			if open {
				moveSuggestion(list, c == iup.K_DOWN)
				return iup.IGNORE
			}
		case iup.K_ESC:
			if open {
				hideSuggestions()
				return iup.IGNORE
			}
		}
		return iup.CONTINUE
	}))
	return search
}

func onSearchClick() { runSearch(iup.GetHandle("search").GetAttribute("VALUE")) }

// endSearch hides the suggestions and clears the search field once a search runs.
func endSearch() {
	hideSuggestions()
	iup.GetHandle("search").SetAttribute("VALUE", "")
}

// moveSuggestion highlights the next or previous suggestion, passing through none at either end.
func moveSuggestion(list iup.Ihandle, down bool) {
	n, v := list.GetInt("COUNT"), list.GetInt("VALUE")
	if down {
		v++
	} else {
		v--
	}
	if v > n {
		v = 0
	} else if v < 0 {
		v = n
	}
	list.SetAttribute("VALUE", strconv.Itoa(v))
}

// suggestList is the autocomplete list, a floating child shown under the search field.
func suggestList() iup.Ihandle {
	list := iup.List().SetHandle("suggest").SetAttributes("VISIBLE=NO, FLOATING=YES, ZORDER=TOP, CANFOCUS=NO")
	list.SetCallback("MOTION_CB", iup.MotionFunc(func(ih iup.Ihandle, x, y int, _ string) int {
		if pos := iup.ConvertXYToPos(ih, x, y); pos > 0 && pos != ih.GetInt("VALUE") {
			ih.SetAttribute("VALUE", strconv.Itoa(pos))
		}
		return iup.DEFAULT
	}))
	list.SetCallback("BUTTON_CB", iup.ButtonFunc(func(ih iup.Ihandle, button, pressed, x, y int, _ string) int {
		if button == iup.BUTTON1 && pressed == 0 {
			if pos := iup.ConvertXYToPos(ih, x, y); pos > 0 {
				acceptSuggestion(pos)
			}
		}
		return iup.DEFAULT
	}))
	return list
}

// showSuggestions fills the floating suggestion list and positions it under the search field.
func showSuggestions(labels []string) {
	list := iup.GetHandle("suggest")
	search := iup.GetHandle("search")
	if len(labels) == 0 || iup.GetFocus() != search {
		hideSuggestions()
		suggestTitles = nil
		return
	}
	fillList(list, labels)
	list.SetAttribute("VALUE", "0")

	var sx, sy, w, h int
	fmt.Sscanf(search.GetAttribute("POSITION"), "%d,%d", &sx, &sy)
	fmt.Sscanf(search.GetAttribute("RASTERSIZE"), "%dx%d", &w, &h)
	list.SetAttribute("POSITION", fmt.Sprintf("%d,%d", sx, sy+h))
	list.SetAttribute("MINSIZE", fmt.Sprintf("%dx0", w))
	iup.Refresh(list)
	iup.Show(list)
}

// acceptSuggestion fills the search field with the chosen title; Enter then runs the search.
func acceptSuggestion(item int) {
	if item < 1 || item > len(suggestTitles) {
		return
	}
	title := suggestTitles[item-1]
	acGen.Add(1)
	hideSuggestions()
	search := iup.GetHandle("search")
	search.SetAttribute("VALUE", title)
	search.SetAttribute("CARETPOS", strconv.Itoa(len([]rune(title))))
}

func hideSuggestions() { iup.Hide(iup.GetHandle("suggest")) }

// logoButton opens About; the grey logo turns to the colored one under the pointer.
func logoButton() iup.Ihandle {
	b := iconButton("logo-gray", showAbout).SetAttributes("TIP=About")
	b.SetCallback("ENTERWINDOW_CB", iup.EnterWindowFunc(func(ih iup.Ihandle) int {
		ih.SetAttribute("IMAGE", "logo")
		return iup.DEFAULT
	}))
	b.SetCallback("LEAVEWINDOW_CB", iup.LeaveWindowFunc(func(ih iup.Ihandle) int {
		ih.SetAttribute("IMAGE", "logo-gray")
		return iup.DEFAULT
	}))
	return b
}

// onKey handles global shortcuts: cycle tabs (Ctrl+Tab), close (Ctrl+W / Ctrl+F4),
// watch (Enter on a detail tab). Ctrl+letter arrives uppercased with the Ctrl bit, so match K_W not K_w.
func onKey(_ iup.Ihandle, c int) int {
	switch c {
	case iup.K_CR:
		if cls := iup.GetClassName(iup.GetFocus()); cls == "text" || cls == "list" {
			return iup.CONTINUE
		}
		if f := watchActions[currentTab()]; f != nil {
			f()
			return iup.IGNORE
		}
	case iup.XKeyCtrl(iup.K_TAB):
		cycleTab(1)
		return iup.IGNORE
	case iup.XKeyCtrl(iup.XKeyShift(iup.K_TAB)):
		cycleTab(-1)
		return iup.IGNORE
	case iup.XKeyCtrl(iup.K_W), iup.XKeyCtrl(iup.K_F4):
		closeCurrentTab()
		return iup.IGNORE
	case iup.XKeyCtrl(iup.K_D):
		if f := favoriteActions[currentTab()]; f != nil {
			f()
			return iup.IGNORE
		}
	}
	return iup.CONTINUE
}

func closeCurrentTab() {
	tabs := iup.GetHandle("tabs")
	if iup.GetChildCount(tabs) == 0 {
		return
	}
	iup.PostMessage(tabs, "close", 0, currentTab()) // defer destroy out of the key callback
}

func cycleTab(delta int) {
	tabs := iup.GetHandle("tabs")
	n := iup.GetChildCount(tabs)
	if n <= 1 {
		return
	}
	selectTab(tabs, (tabs.GetInt("VALUEPOS")+delta+n)%n)
}

func currentTab() iup.Ihandle {
	tabs := iup.GetHandle("tabs")
	return iup.GetChild(tabs, tabs.GetInt("VALUEPOS"))
}

// selectTab selects the tab at pos and shows its order on the Sort button.
func selectTab(tabs iup.Ihandle, pos int) {
	tabs.SetAttribute("VALUEPOS", strconv.Itoa(pos))
	syncSortButton(iup.GetChild(tabs, pos))
}

// focusView switches to an open view.
func focusView(view iup.Ihandle) {
	tabs := iup.GetHandle("tabs")
	selectTab(tabs, iup.GetChildPos(tabs, view))
}

// content is the tabbed browsing area; the first tab is the "Top" landing grid.
func content() iup.Ihandle {
	top := gridTab().SetHandle("content0").SetAttribute("TABTITLE", "Top HD Movies")
	setReloader(top, startup)
	setSortable(top, sortBy)
	tabs := iup.Tabs(top).SetHandle("tabs").SetAttributes("EXPAND=YES, SHOWCLOSE=YES, ALLOWREORDER=YES")
	tabs.SetCallback("TABCLOSE_CB", iup.TabCloseFunc(func(ih iup.Ihandle, pos int) int {
		child := iup.GetChild(ih, pos)
		forgetView(child)
		selectNeighborTab(ih, pos)
		iup.PostMessage(ih, "gc", 0, nil) // GC after the page is destroyed
		return iup.CONTINUE
	}))
	tabs.SetCallback("TABCHANGE_CB", iup.TabChangeFunc(func(_, tab, _ iup.Ihandle) int {
		syncSortButton(tab)
		return iup.DEFAULT
	}))
	tabs.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(onTabMessage))
	return tabs
}

// updateNoTabs shows the placeholder in place of the tab strip while no tab is open.
func updateNoTabs() {
	tabs := iup.GetHandle("tabs")
	empty := iup.GetHandle("noTabs")
	none := iup.GetChildCount(tabs) == 0
	if (empty.GetAttribute("VISIBLE") == "YES") == none {
		return
	}
	show := func(ih iup.Ihandle, on bool) {
		ih.SetAttribute("FLOATING", map[bool]string{true: "NO", false: "IGNORE"}[on])
		ih.SetAttribute("VISIBLE", yesNo(on))
	}
	show(tabs, !none)
	show(empty, none)
	iup.Refresh(tabs)
}

// showList opens a grid view as a new tab.
func showList(name string, child iup.Ihandle, _ bool) { addTab(name, child) }

// showDetails opens a details view as a new tab.
func showDetails(name string, child iup.Ihandle) { addTab(name, child) }

// addTab appends child as a new tab, maps it, and switches to it.
func addTab(name string, child iup.Ihandle) {
	tabs := iup.GetHandle("tabs")
	child.SetAttribute("TABTITLE", plainText(elide(name, 16)))
	iup.Append(tabs, child)
	updateNoTabs()
	iup.Map(child)
	pos := iup.GetChildCount(tabs) - 1
	iup.Refresh(tabs)
	tabs.SetAttribute("TABTIP"+strconv.Itoa(pos), name)
	selectTab(tabs, pos)
}

// viewAlive reports whether h is still inside an open tab.
func viewAlive(h iup.Ihandle) bool {
	return isDescendant(iup.GetHandle("tabs"), h)
}

// onTabMessage dispatches tabs-area messages: "close", "gc" and the shared bus messages.
func onTabMessage(tabs iup.Ihandle, msg string, delta int, p any) int {
	switch msg {
	case "gc":
		updateNoTabs()
		releaseImages()
		runtime.GC()
		return iup.DEFAULT
	case "close":
		if h, ok := p.(iup.Ihandle); ok && isDescendant(tabs, h) {
			selectNeighborTab(tabs, childPos(tabs, h))
			forgetView(h)
			iup.Destroy(h)
			updateNoTabs()
			releaseImages()
			runtime.GC()
		}
		return iup.DEFAULT
	case "load", "play", "pause":
		if h, ok := p.(iup.Ihandle); ok {
			updateTabActivity(tabs, h, msg, delta)
		}
		return iup.DEFAULT
	}
	return onBusMessage(msg, delta, p)
}

// bus is the element that receives the messages background goroutines post to the main loop.
func bus() iup.Ihandle { return iup.GetHandle("tabs") }

// markTab adjusts a tab's activity from any goroutine: kind is "load", "play" or "pause".
func markTab(tab iup.Ihandle, kind string, delta int) {
	iup.PostMessage(bus(), kind, delta, tab)
}

// updateTabActivity applies an activity delta and repaints the tab's image.
func updateTabActivity(tabs, tab iup.Ihandle, kind string, delta int) {
	if !isChildOf(tabs, tab) {
		delete(tabActs, tab)
		syncSpinner()
		return
	}
	act := tabActs[tab]
	if act == nil {
		act = &tabActivity{}
		tabActs[tab] = act
	}
	act.add(kind, delta)
	switch {
	case act.paused > 0:
		setTabImage(tabs, tab, "tab-pause")
	case act.playing > 0:
		setTabImage(tabs, tab, "tab-play")
	case act.loading > 0:
		setTabImage(tabs, tab, spinImage())
	default:
		setTabImage(tabs, tab, "")
		delete(tabActs, tab)
	}
	syncSpinner()
}

// setTabImage updates a tab's image, addressing the tab by position.
func setTabImage(tabs, tab iup.Ihandle, img string) {
	if pos := childPos(tabs, tab); pos >= 0 {
		iup.SetAttributeId(tabs, "TABIMAGE", pos, img)
	}
}

// setTabTip sets a tab's tooltip, addressing the tab by position.
func setTabTip(tab iup.Ihandle, tip string) {
	tabs := iup.GetHandle("tabs")
	if pos := childPos(tabs, tab); pos >= 0 {
		iup.SetAttributeId(tabs, "TABTIP", pos, tip)
	}
}

// selectNeighborTab moves the selection off the current tab at pos before it is removed: to the previous tab, or the next one for the first.
func selectNeighborTab(tabs iup.Ihandle, pos int) {
	if pos < 0 || tabs.GetInt("VALUEPOS") != pos || iup.GetChildCount(tabs) < 2 {
		return
	}
	next := pos - 1
	if pos == 0 {
		next = 1
	}
	selectTab(tabs, next)
}

func spinImage() string {
	if len(tabSpinFrames) == 0 {
		return ""
	}
	return tabSpinFrames[spinIndex]
}

// syncSpinner runs a single shared timer while any tab is loading.
func syncSpinner() {
	need := false
	for _, a := range tabActs {
		if a.loading > 0 && a.playing == 0 {
			need = true
			break
		}
	}
	switch {
	case need && !spinning:
		spinTimer = iup.Timer()
		spinTimer.SetAttribute("TIME", strconv.Itoa(spinnerFrameTime))
		spinTimer.SetCallback("ACTION_CB", iup.TimerActionFunc(onSpinTick))
		spinTimer.SetAttribute("RUN", "YES")
		spinning = true
	case !need && spinning:
		spinTimer.SetAttribute("RUN", "NO")
		iup.Destroy(spinTimer)
		spinning = false
	}
}

// onSpinTick advances the spinner frame on every loading tab.
func onSpinTick(iup.Ihandle) int {
	if len(tabSpinFrames) == 0 {
		return iup.DEFAULT
	}
	spinIndex = (spinIndex + 1) % len(tabSpinFrames)
	frame := tabSpinFrames[spinIndex]
	tabs := iup.GetHandle("tabs")
	for tab, a := range tabActs {
		if a.loading > 0 && a.playing == 0 {
			setTabImage(tabs, tab, frame)
		}
	}
	return iup.DEFAULT
}

// layoutSummary places a details view: the poster on the left, the facts, the overview and the buttons on the right.
func layoutSummary(holder iup.Ihandle, v summaryView) iup.Ihandle {
	poster := summaryPoster()

	overview := iup.Text().SetAttributes("MULTILINE=YES, READONLY=YES, WORDWRAP=YES, BORDER=NO, SCROLLBAR=VERTICAL, EXPAND=YES")
	setThemed(overview, "BGCOLOR", roleDialogBg)
	overview.SetAttribute("VALUE", v.overview)

	star := iup.Button("Favorite").SetAttributes("PADDING=22x8")
	if v.movie.Category.IsTV() {
		star.SetAttribute("TIP", "Add or remove the show from favorites (Ctrl+D)")
	} else {
		star.SetAttribute("TIP", "Add or remove the movie from favorites (Ctrl+D)")
	}
	favoriteToggle(holder, star, v.movie)
	actions := append(v.actions, iup.Fill(), star)

	topBox := iup.Vbox(v.top...).SetAttributes("GAP=8, EXPAND=HORIZONTAL")
	buttons := iup.Hbox(actions...).SetAttributes("GAP=8, EXPAND=HORIZONTAL")

	right := iup.Vbox(topBox, overview, buttons).SetAttributes("GAP=8, EXPAND=YES, NMARGIN=16x14")

	split := iup.Split(poster, right)
	split.SetAttributes("ORIENTATION=VERTICAL, VALUE=330, BARSIZE=4, MINMAX=200:560")

	m := v.movie
	go loadPosterSized(poster, summaryPosterW, summaryPosterH, m.PosterXLarge, m.PosterLarge, m.PosterMedium)
	return split
}

// summaryPoster is a canvas that paints the poster scaled to fit, so it tracks the window size.
func summaryPoster() iup.Ihandle {
	c := iup.Canvas().SetAttributes("EXPAND=YES, BORDER=NO")
	c.SetAttribute("POSTERIMG", "sposter"+strconv.FormatInt(summaryPosterSeq.Add(1), 10))
	c.SetCallback("ACTION", iup.ActionFunc(drawPoster))
	trackTarget(c, "POSTERIMG")
	c.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(ih iup.Ihandle, _ string, _ int, p any) int {
		img, ok := p.(image.Image)
		if !ok {
			return iup.DEFAULT
		}
		name := ih.GetAttribute("POSTERIMG")
		iup.Destroy(iup.GetHandle(name))
		iup.ImageFromImage(img).SetHandle(name)
		iup.Update(ih)
		return iup.DEFAULT
	}))
	return c
}

// drawPoster paints the poster scaled to fit on the dialog background.
func drawPoster(ih iup.Ihandle) int {
	iup.DrawBegin(ih)
	defer iup.DrawEnd(ih)

	cw, ch := iup.DrawGetSize(ih)
	ih.SetAttribute("DRAWSTYLE", "FILL")
	ih.SetAttribute("DRAWCOLOR", iup.GetGlobal("DLGBGCOLOR"))
	iup.DrawRectangle(ih, 0, 0, cw, ch)

	name := ih.GetAttribute("POSTERIMG")
	iw, iih, _ := iup.DrawGetImageInfo(name)
	if iw <= 0 || iih <= 0 {
		return iup.DEFAULT
	}
	const margin = 18
	s := math.Min(float64(cw-2*margin)/float64(iw), float64(ch-2*margin)/float64(iih))
	if s <= 0 {
		return iup.DEFAULT
	}
	dw, dh := int(float64(iw)*s), int(float64(iih)*s)
	x, y := (cw-dw)/2, (ch-dh)/2
	iup.DrawImage(ih, name, x, y, dw, dh)
	return iup.DEFAULT
}

// settingsBody shows one settings page at a time, picked from a sidebar.
func settingsBody(pageList []settingsPage) iup.Ihandle {
	sidebar := iup.List().SetAttributes(map[string]string{
		"SPACING": "4", "VISIBLECOLUMNS": "10", "EXPAND": "VERTICAL",
		"VISIBLELINES": strconv.Itoa(len(pageList)),
	})
	title := iup.Label("").SetAttributes("FONTSTYLE=Bold, FONTSIZE=16, EXPAND=HORIZONTAL")
	summary := iup.Label("").SetAttributes(`FGCOLOR="128 128 128", EXPAND=HORIZONTAL`)
	bodies := make([]iup.Ihandle, len(pageList))
	for i, pg := range pageList {
		sidebar.SetAttribute(strconv.Itoa(i+1), pg.name)
		bodies[i] = pg.body
	}
	pages := iup.Zbox(bodies...).SetAttributes("ALIGNMENT=NW, EXPAND=YES")
	showPage := func(i int) {
		settingsPageIndex = i
		pages.SetAttribute("VALUEPOS", strconv.Itoa(i))
		title.SetAttribute("TITLE", pageList[i].name)
		summary.SetAttribute("TITLE", pageList[i].summary)
	}
	sidebar.SetCallback("ACTION", iup.ListActionFunc(func(_ iup.Ihandle, _ string, item, state int) int {
		if state == 1 {
			showPage(item - 1)
		}
		return iup.DEFAULT
	}))
	if settingsPageIndex >= len(pageList) {
		settingsPageIndex = 0
	}
	sidebar.SetAttribute("VALUE", strconv.Itoa(settingsPageIndex+1))
	showPage(settingsPageIndex)
	content := iup.Vbox(title, summary, pages).SetAttributes("GAP=8, NMARGIN=12x0, EXPAND=YES")
	return iup.Hbox(sidebar, iup.Label("").SetAttribute("SEPARATOR", "VERTICAL"), content).SetAttributes("GAP=8")
}
