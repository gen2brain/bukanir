//go:build media

package main

import (
	"context"
	"fmt"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/gen2brain/bukanir"
	"github.com/gen2brain/iup-go/iup"
)

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

func menuItemAction(label string, fn func()) iup.Ihandle {
	return iup.MenuItem(label).SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		fn()
		return iup.DEFAULT
	}))
}

func disabledMenuItem(label string) iup.Ihandle {
	return iup.MenuItem(label).SetAttributes("ACTIVE=NO")
}

func topMenu() iup.Ihandle {
	var items []iup.Ihandle
	for _, t := range []string{"Movies", "HD Movies", "TV Shows", "HD TV Shows"} {
		text := t
		items = append(items, menuItemAction(text, func() { onTop(text) }))
	}
	return iup.Menu(items...)
}

// mediaMenu / sortMenu are radio-style filter menus.
func mediaMenu() iup.Ihandle {
	opts := []struct{ label, val string }{{"All", "all"}, {"Movies", "movies"}, {"Episodes", "episodes"}}
	var items []iup.Ihandle
	for _, o := range opts {
		mi := menuItemAction(o.label, func() {
			media = o.val
			setButtonTitle("btnMedia", mediaLabel())
		})
		if media == o.val {
			mi.SetAttribute("VALUE", "ON")
		}
		items = append(items, mi)
	}
	return iup.Menu(items...)
}

func sortMenu() iup.Ihandle {
	opts := []struct{ label, val string }{{"Seeders", "seeders"}, {"Episodes", "episodes"}}
	var items []iup.Ihandle
	for _, o := range opts {
		mi := menuItemAction(o.label, func() {
			sortBy = o.val
			setButtonTitle("btnSort", sortLabel())
			resortCurrentTab()
		})
		if sortBy == o.val {
			mi.SetAttribute("VALUE", "ON")
		}
		items = append(items, mi)
	}
	return iup.Menu(items...)
}

// setButtonTitle changes a toolbar button's title and lays the dialog out again for its new width.
func setButtonTitle(handle, title string) {
	btn := iup.GetHandle(handle)
	btn.SetAttribute("TITLE", title)
	iup.Refresh(btn)
}

func mediaLabel() string { return "Media: " + titleCase(media) + " ▾" }
func sortLabel() string  { return "Sort: " + titleCase(sortBy) + " ▾" }

func titleCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// popularMenu / topRatedMenu / genreMenu build their menus from background data.
func popularMenu() iup.Ihandle  { return titleMenu(cachedPopular()) }
func topRatedMenu() iup.Ihandle { return titleMenu(cachedTopRated()) }

// menuSectionLimit caps items per Movies/Episodes submenu.
const menuSectionLimit = 20

// titleMenu splits titles into "Movies" and "Episodes" submenus.
func titleMenu(items []bukanir.TItem) iup.Ihandle {
	if len(items) == 0 {
		return iup.Menu(disabledMenuItem("(loading)"))
	}
	var movies, tv []iup.Ihandle
	for _, it := range items {
		if it.Media == "tv" {
			if len(tv) < menuSectionLimit {
				tv = append(tv, titleMenuItem(it))
			}
		} else if len(movies) < menuSectionLimit {
			movies = append(movies, titleMenuItem(it))
		}
	}
	var sections []iup.Ihandle
	if len(movies) > 0 {
		sections = append(sections, iup.Submenu("Movies", iup.Menu(movies...)))
	}
	if len(tv) > 0 {
		sections = append(sections, iup.Submenu("Episodes", iup.Menu(tv...)))
	}
	return iup.Menu(sections...)
}

func titleMenuItem(it bukanir.TItem) iup.Ihandle {
	label := it.Title
	if it.Year > 0 {
		label = fmt.Sprintf("%s (%d)", it.Title, it.Year)
	}
	title := it.Title
	return menuItemAction(label, func() { runSearch(title) })
}

func genreMenu() iup.Ihandle {
	genres := cachedGenres()
	if len(genres) == 0 {
		return iup.Menu(disabledMenuItem("(loading)"))
	}
	var items []iup.Ihandle
	for _, g := range genres {
		id, name := g.ID, g.Name
		items = append(items, menuItemAction(name, func() {
			openBrowse(name, func(ctx context.Context) ([]bukanir.TMovie, error) {
				return client.Genre(ctx, id, 0)
			})
		}))
	}
	return iup.Menu(items...)
}

// suggestTitles holds the plain titles behind the current suggestion list.
var suggestTitles []string

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
	list.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(onSuggestions))
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

// suggestLabelMax caps the visible length of a suggested title.
const suggestLabelMax = 60

// suggestLabel formats a suggestion as "Title (Year)", marking TV shows.
func suggestLabel(it bukanir.TItem) string {
	title := it.Title
	if r := []rune(title); len(r) > suggestLabelMax {
		title = strings.TrimSpace(string(r[:suggestLabelMax-1])) + "…"
	}
	if it.Media == "tv" {
		return fmt.Sprintf("%s (%d, TV)", title, it.Year)
	}
	return fmt.Sprintf("%s (%d)", title, it.Year)
}

// onSuggestions fills the floating suggestion list and positions it under the search field.
func onSuggestions(ih iup.Ihandle, _ string, _ int, p any) int {
	items, _ := p.([]bukanir.TItem)
	items = slices.DeleteFunc(slices.Clone(items), func(it bukanir.TItem) bool {
		return (media == "movies" && it.Media != "movie") || (media == "episodes" && it.Media != "tv")
	})
	if len(items) > autocompleteLimit {
		items = items[:autocompleteLimit]
	}
	search := iup.GetHandle("search")
	if len(items) == 0 || iup.GetFocus() != search {
		hideSuggestions()
		suggestTitles = nil
		return iup.DEFAULT
	}
	suggestTitles = suggestTitles[:0]
	clearList(ih)
	for i, it := range items {
		suggestTitles = append(suggestTitles, it.Title)
		iup.SetAttributeId(ih, "", i+1, suggestLabel(it))
	}
	ih.SetAttribute("VALUE", "0")

	var sx, sy, w, h int
	fmt.Sscanf(search.GetAttribute("POSITION"), "%d,%d", &sx, &sy)
	fmt.Sscanf(search.GetAttribute("RASTERSIZE"), "%dx%d", &w, &h)
	ih.SetAttribute("POSITION", fmt.Sprintf("%d,%d", sx, sy+h))
	ih.SetAttribute("MINSIZE", fmt.Sprintf("%dx0", w))
	iup.Refresh(ih)
	iup.Show(ih)
	return iup.DEFAULT
}

// acceptSuggestion searches for the chosen suggestion by its underlying title.
func acceptSuggestion(item int) {
	if item < 1 || item > len(suggestTitles) {
		return
	}
	title := suggestTitles[item-1]
	hideSuggestions()
	iup.GetHandle("search").SetAttribute("VALUE", title)
	runSearch(title)
}

func hideSuggestions() { iup.Hide(iup.GetHandle("suggest")) }

// clearList removes every item from a List.
func clearList(list iup.Ihandle) {
	for n := list.GetInt("COUNT"); n >= 1; n-- {
		list.SetAttribute("REMOVEITEM", strconv.Itoa(n))
	}
}

func iconButton(icon string, onClick func()) iup.Ihandle {
	b := iup.Button("").SetAttributes("FLAT=YES")
	if _, ok := monoIcons[icon]; ok {
		monoButtons[b] = icon
		icon = tintedName(icon)
	}
	b.SetAttribute("IMAGE", icon)
	b.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		onClick()
		return iup.DEFAULT
	}))
	return b
}

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

// yearMenu is a native menu of years grouped into per-decade submenus down to 1970.
func yearMenu() iup.Ihandle {
	cur := currentYear()
	var decades []iup.Ihandle
	for d := (cur / 10) * 10; d >= 1970; d -= 10 {
		var yrs []iup.Ihandle
		for y := d + 9; y >= d && y >= 1970; y-- {
			if y > cur {
				continue
			}
			text := strconv.Itoa(y)
			yrs = append(yrs, menuItemAction(text, func() { runSearch(text) }))
		}
		if len(yrs) == 0 {
			continue
		}
		decades = append(decades, iup.Submenu(fmt.Sprintf("%ds", d), iup.Menu(yrs...)))
	}
	return iup.Menu(decades...)
}

// statusBar shows a status message, a progress bar, and the Tor status icon.
func statusBar() iup.Ihandle {
	status := iup.Label("Ready").SetHandle("status").SetAttributes("EXPAND=HORIZONTAL, ELLIPSIS=YES, RASTERSIZE=1x")
	status.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(ih iup.Ihandle, s string, _ int, _ any) int {
		ih.SetAttribute("TITLE", plainText(s))
		return iup.DEFAULT
	}))
	tor := iup.Label("").SetHandle("torStatus").SetAttributes("IMAGE=tor-off, TIP=\"Tor is off\"")
	tor.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(onTorDone))
	progress := iup.ProgressBar().SetHandle("progress").SetAttributes("RASTERSIZE=180x16, VISIBLE=NO").SetAttribute("FGCOLOR", accentColor)
	progress.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(ih iup.Ihandle, s string, i int, _ any) int {
		if s == "hide" {
			ih.SetAttribute("VISIBLE", "NO")
			return iup.DEFAULT
		}
		ih.SetAttribute("VISIBLE", "YES")
		ih.SetAttribute("VALUE", strconv.FormatFloat(float64(i)/1000, 'f', 3, 64))
		return iup.DEFAULT
	}))
	return iup.Hbox(
		status,
		progress,
		tor,
	).SetAttributes("GAP=8, ALIGNMENT=ACENTER, MARGIN=6x4")
}

func setStatus(s string) {
	iup.GetHandle("status").SetAttribute("TITLE", plainText(s))
}

// plainText doubles "&" so IUP shows it instead of taking it as a mnemonic.
func plainText(s string) string {
	return strings.ReplaceAll(s, "&", "&&")
}

// watchActions starts the current detail tab's playback on Enter, keyed by the tab.
var watchActions = map[iup.Ihandle]func(){}

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

// content is the tabbed browsing area; the first tab is the "Top" landing grid.
func content() iup.Ihandle {
	top := gridTab().SetHandle("content0").SetAttribute("TABTITLE", "Top HD Movies")
	setReloader(top, startup)
	setSortable(top, sortBy)
	tabs := iup.Tabs(top).SetHandle("tabs").SetAttributes("EXPAND=YES, SHOWCLOSE=YES, ALLOWREORDER=YES")
	tabs.SetCallback("TABCLOSE_CB", iup.TabCloseFunc(func(ih iup.Ihandle, pos int) int {
		child := iup.GetChild(ih, pos)
		stopSessions(child)
		delete(reloaders, child)
		delete(tabSorts, child)
		delete(watchActions, child)
		forgetFavoritesTab(child)
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

// gridTab creates a tab container that renders a movie grid from PostMessage.
func gridTab() iup.Ihandle {
	holder := iup.Vbox(loadingBox("Loading")).SetAttributes("EXPAND=YES")
	holder.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(ih iup.Ihandle, _ string, _ int, p any) int {
		if movies, ok := p.([]bukanir.TMovie); ok {
			showSorted(ih, movies)
		}
		return iup.DEFAULT
	}))
	return holder
}

// placeholder is a centered, dimmed, expanding label for an empty content area.
func placeholder(text string) iup.Ihandle {
	return setThemed(iup.Label(text).SetAttributes("EXPAND=YES, ALIGNMENT=ACENTER:ACENTER"), "FGCOLOR", roleSubtle)
}

// clearChildren detaches and destroys all children of h.
func clearChildren(h iup.Ihandle) {
	for iup.GetChildCount(h) > 0 {
		c := iup.GetChild(h, 0)
		iup.Detach(c)
		iup.Destroy(c)
	}
	releaseImages()
}

// onTabMessage dispatches tabs-area messages: "gc", "enable", "load", "play".
func onTabMessage(tabs iup.Ihandle, msg string, delta int, p any) int {
	if fn, ok := p.(func()); ok && msg == "run" {
		fn()
		return iup.DEFAULT
	}
	if msg == "gc" {
		updateNoTabs()
		releaseImages()
		runtime.GC()
		return iup.DEFAULT
	}
	h, ok := p.(iup.Ihandle)
	if !ok {
		return iup.DEFAULT
	}
	switch msg {
	case "close":
		if isDescendant(tabs, h) {
			selectNeighborTab(tabs, childPos(tabs, h))
			stopSessions(h)
			delete(reloaders, h)
			delete(tabSorts, h)
			delete(watchActions, h)
			forgetFavoritesTab(h)
			iup.Destroy(h)
			updateNoTabs()
			releaseImages()
			runtime.GC()
		}
	case "enable":
		if isDescendant(tabs, h) {
			h.SetAttribute("ACTIVE", "YES")
			if bg := h.GetAttribute("SAVEDBG"); bg != "" {
				h.SetAttribute("BGCOLOR", bg)
			}
			if fg := h.GetAttribute("SAVEDFG"); fg != "" {
				h.SetAttribute("FGCOLOR", fg)
			}
		}
	case "load", "play", "pause":
		updateTabActivity(tabs, h, msg, delta)
	}
	return iup.DEFAULT
}

// tabActivity counts a tab's in-progress playbacks by phase.
type tabActivity struct{ loading, playing, paused int }

var (
	tabActs   = map[iup.Ihandle]*tabActivity{}
	spinIndex int
	spinTimer iup.Ihandle
	spinning  bool
)

// markTab adjusts a tab's activity from any goroutine: kind is "load" or "play".
func markTab(tab iup.Ihandle, kind string, delta int) {
	iup.PostMessage(iup.GetHandle("tabs"), kind, delta, tab)
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
	switch kind {
	case "load":
		act.loading += delta
	case "play":
		act.playing += delta
	case "pause":
		act.paused += delta
	}
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

func childPos(parent, child iup.Ihandle) int {
	for i := 0; i < iup.GetChildCount(parent); i++ {
		if iup.GetChild(parent, i) == child {
			return i
		}
	}
	return -1
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
		spinTimer.SetAttribute("TIME", "90")
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

// tabPhase tracks one playback's contribution to a tab's indicator.
type tabPhase struct {
	tab    iup.Ihandle
	state  string
	paused bool
}

func beginPlayback(tab iup.Ihandle) *tabPhase {
	markTab(tab, "load", +1)
	return &tabPhase{tab: tab, state: "load"}
}

func (p *tabPhase) started() {
	if p.state != "load" {
		return
	}
	markTab(p.tab, "load", -1)
	markTab(p.tab, "play", +1)
	p.state = "play"
}

func (p *tabPhase) setPaused(paused bool) {
	if p.state != "play" || p.paused == paused {
		return
	}
	p.paused = paused
	if paused {
		markTab(p.tab, "pause", +1)
	} else {
		markTab(p.tab, "pause", -1)
	}
}

func (p *tabPhase) end() {
	if p.paused {
		markTab(p.tab, "pause", -1)
		p.paused = false
	}
	markTab(p.tab, p.state, -1)
}

func isChildOf(parent, child iup.Ihandle) bool {
	for i := 0; i < iup.GetChildCount(parent); i++ {
		if iup.GetChild(parent, i) == child {
			return true
		}
	}
	return false
}

// isDescendant reports whether h appears anywhere in parent's subtree.
func isDescendant(parent, h iup.Ihandle) bool {
	for i := 0; i < iup.GetChildCount(parent); i++ {
		c := iup.GetChild(parent, i)
		if c == h || isDescendant(c, h) {
			return true
		}
	}
	return false
}
