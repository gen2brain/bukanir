//go:build media && (android || ios)

package main

import (
	"fmt"
	"os"
	"runtime"
	"slices"

	"github.com/gen2brain/iup-go/iup"
)

const (
	gridGap    = 12
	gridMargin = 12
)

// summaryMargin is the margin around a details view.
const summaryMargin = 12

// dialogIcon is empty: an icon takes title space in the app bar.
const dialogIcon = ""

// playerVolumeControls is off: the volume keys set and mute the volume.
const playerVolumeControls = false

// playerSeekRow puts the seek bar and the time on their own row above the player buttons.
const playerSeekRow = true

// playerVideoSize is a small natural size; the video expands over the room the controls leave.
const playerVideoSize = "160x90"

// narrowScreen stacks layouts vertically: settings labels above their controls, the About contents in one column.
const narrowScreen = true

// shortcutsHelp is off: a phone has no keyboard shortcuts to show.
const shortcutsHelp = false

// keepFilesSetting is off: there is no download folder to choose.
const keepFilesSetting = false

// singleTapOpens opens a grid title on a single tap.
const singleTapOpens = true

// posterW and posterH fit two grid columns across the narrow side of the screen; sizePosters sets them.
var posterW, posterH = 200, 300

var (
	screens               []iup.Ihandle
	wrapLabels            = map[iup.Ihandle][]iup.Ihandle{}
	mediaItems, sortItems iup.Ihandle
	searchDlg, searchList iup.Ihandle
)

// mainDialog creates the main screen: one grid that the drawer entries and the search replace, and the status bar.
func mainDialog() iup.Ihandle {
	sizePosters()
	top := gridTab().SetHandle("content0")
	setReloader(top, startup)
	setSortable(top, sortBy)
	views := iup.Vbox(top).SetHandle("views")
	dlg := iup.Dialog(iup.Vbox(
		views,
		iup.Label("").SetAttributes("SEPARATOR=HORIZONTAL"),
		statusBar(),
	)).SetAttributes(map[string]string{
		"TITLE":  "Top HD Movies",
		"SHRINK": "YES",
	}).SetHandle("dlg")
	dlg.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(_ iup.Ihandle, msg string, delta int, p any) int {
		return onBusMessage(msg, delta, p)
	}))
	iup.SetAttributeHandle(dlg, "MENU", appMenu())
	iup.SetAttributeHandle(dlg, "DRAWER", drawerMenu())
	return dlg
}

func showMainDialog(dlg iup.Ihandle) { iup.Show(dlg) }

// narrowWidth is the screen width less a margin on each side.
func narrowWidth(margin int) int {
	var w, h int
	fmt.Sscanf(iup.GetGlobal("SCREENSIZE"), "%dx%d", &w, &h)
	return w - 2*margin
}

// exitApp ends the process once quitting has cleaned up.
func exitApp() { os.Exit(0) }

// sizePosters fits two poster columns across the narrow side of the screen.
func sizePosters() {
	var w, h int
	fmt.Sscanf(iup.GetGlobal("SCREENSIZE"), "%dx%d", &w, &h)
	if w <= 0 || h <= 0 {
		return
	}
	posterW = (min(w, h) - 2*gridMargin - gridGap) / 2
	posterH = posterW * 3 / 2
}

// appMenu is the app bar: Search and Refresh as icons, the media filter and the sort order in the overflow menu.
func appMenu() iup.Ihandle {
	mediaItems = mediaMenu()
	sortItems = sortMenu()
	return iup.Menu(
		iconMenuItem("Search", "search", openSearch),
		iconMenuItem("Refresh", "refresh", onRefreshClick),
		iup.Submenu("Media", mediaItems),
		iup.Submenu("Sort", sortItems),
	)
}

// drawerMenu is the navigation drawer with the lists and the app pages.
func drawerMenu() iup.Ihandle {
	section := func(title string, menu iup.Ihandle) iup.Ihandle {
		return iup.Submenu(title, menu).SetAttribute("EXPANDABLE", "YES")
	}
	return iup.Menu(
		section("Top", topMenu()),
		section("Year", yearMenu()),
		section("Popular", popularMenu()),
		section("Top Rated", topRatedMenu()),
		section("Genre", genreMenu()),
		iup.MenuSeparator(),
		iconMenuItem("Favorites", "star", openFavorites),
		iconMenuItem("Settings", "settings", onSettings),
		iconMenuItem("Log", "log", showLogView),
		menuItemAction("About", showAbout),
	)
}

// iconMenuItem is a menu item with a mono icon recolored to the text color.
func iconMenuItem(label, icon string, fn func()) iup.Ihandle {
	mi := menuItemAction(label, fn)
	monoButtons[mi] = icon
	return mi.SetAttribute("IMAGE", tintedName(icon))
}

// menusLoaded rebuilds the drawer once background menu data arrives.
func menusLoaded() {
	dlg := iup.GetHandle("dlg")
	old := iup.GetAttributeHandle(dlg, "DRAWER")
	iup.SetAttributeHandle(dlg, "DRAWER", drawerMenu())
	if old != 0 {
		forgetMenuIcons(old)
		iup.Destroy(old)
	}
}

// forgetMenuIcons drops the icon items of a menu that is about to be destroyed.
func forgetMenuIcons(menu iup.Ihandle) {
	for i := 0; i < iup.GetChildCount(menu); i++ {
		delete(monoButtons, iup.GetChild(menu, i))
	}
}

// showFilters checks the media filter and the sort order in the overflow menu.
func showFilters() {
	check := func(menu iup.Ihandle, val string) {
		for i := 0; i < iup.GetChildCount(menu); i++ {
			if mi := iup.GetChild(menu, i); mi.GetAttribute("FILTER") == val && mi.GetAttribute("VALUE") != "ON" {
				mi.SetAttribute("VALUE", "ON")
			}
		}
	}
	check(mediaItems, media)
	check(sortItems, sortBy)
}

// openSearch opens the search screen: a query field and its suggestions; a tap on one searches it.
func openSearch() {
	if searchDlg != 0 {
		return
	}
	text := iup.Text().SetAttributes("CUEBANNER=Search, EXPAND=HORIZONTAL")
	text.SetCallback("VALUECHANGED_CB", iup.ValueChangedFunc(func(ih iup.Ihandle) int {
		onSearchChanged(ih.GetAttribute("VALUE"))
		return iup.DEFAULT
	}))
	text.SetCallback("K_ANY", iup.KAnyFunc(func(ih iup.Ihandle, c int) int {
		if c == iup.K_CR {
			runSearch(ih.GetAttribute("VALUE"))
			return iup.IGNORE
		}
		return iup.CONTINUE
	}))
	searchList = iup.List().SetAttributes("EXPAND=YES")
	searchList.SetCallback("ACTION", iup.ListActionFunc(func(_ iup.Ihandle, _ string, item, state int) int {
		if state == 1 && item >= 1 && item <= len(suggestTitles) {
			runSearch(suggestTitles[item-1])
		}
		return iup.DEFAULT
	}))
	searchDlg = iup.Dialog(iup.Vbox(text, searchList).SetAttributes("NMARGIN=12x12, GAP=8")).SetAttributes(map[string]string{
		"TITLE":  "Search",
		"SHRINK": "YES",
	})
	searchDlg.SetCallback("CLOSE_CB", iup.CloseFunc(func(ih iup.Ihandle) int {
		closeSearch()
		return iup.IGNORE
	}))
	iup.Show(searchDlg)
	iup.SetFocus(text)
}

// closeSearch drops pending suggestions and destroys the search screen.
func closeSearch() {
	if searchDlg == 0 {
		return
	}
	acGen.Add(1)
	iup.Destroy(searchDlg)
	searchDlg, searchList = 0, 0
	suggestTitles = nil
}

// endSearch closes the search screen once a search runs, outside the callback that ran it.
func endSearch() {
	if searchDlg != 0 {
		runOnMain(closeSearch)
	}
}

func showSuggestions(labels []string) {
	if searchList != 0 {
		fillList(searchList, labels)
	}
}

func hideSuggestions() {
	if searchList != 0 {
		clearList(searchList)
	}
	suggestTitles = nil
}

// currentTab is the grid on the main screen.
func currentTab() iup.Ihandle { return iup.GetChild(iup.GetHandle("views"), 0) }

func focusView(iup.Ihandle) {}

// showList shows a grid: one opened from the menus or the search replaces the main grid, one opened from a page gets its own screen.
func showList(name string, child iup.Ihandle, root bool) {
	if !root {
		pushScreen(name, child)
		return
	}
	views := iup.GetHandle("views")
	if old := iup.GetChild(views, 0); old != 0 {
		forgetView(old)
		iup.Detach(old)
		iup.Destroy(old)
	}
	iup.Append(views, child)
	iup.Map(child)
	iup.Refresh(views)
	iup.GetHandle("dlg").SetAttribute("TITLE", plainText(name))
	syncSortButton(child)
	releaseImages()
	runtime.GC()
}

// showDetails shows a details view on its own screen.
func showDetails(name string, child iup.Ihandle) { pushScreen(name, child) }

// pushScreen shows child on a new screen with its own status bar; Back closes it.
func pushScreen(name string, child iup.Ihandle) {
	dlg := iup.Dialog(iup.Vbox(
		child,
		iup.Label("").SetAttributes("SEPARATOR=HORIZONTAL"),
		statusBar(),
	)).SetAttributes(map[string]string{
		"TITLE":  plainText(name),
		"SHRINK": "YES",
	})
	dlg.SetCallback("CLOSE_CB", iup.CloseFunc(func(ih iup.Ihandle) int {
		closeScreen(ih, child)
		return iup.IGNORE
	}))
	dlg.SetCallback("RESIZE_CB", iup.ResizeFunc(func(_ iup.Ihandle, w, _ int) int {
		fitWrapped(child, w)
		return iup.DEFAULT
	}))
	screens = append(screens, dlg)
	iup.Show(dlg)
}

// closeScreen stops the playbacks of a screen's view and destroys the screen.
func closeScreen(dlg, child iup.Ihandle) {
	forgetView(child)
	delete(wrapLabels, child)
	screens = slices.DeleteFunc(screens, func(s iup.Ihandle) bool { return s == dlg })
	if menu := iup.GetAttributeHandle(dlg, "MENU"); menu != 0 {
		forgetMenuIcons(menu)
	}
	iup.Destroy(dlg)
	releaseImages()
	runtime.GC()
}

// fitWrapped gives the wrapping labels of a view the width of its screen.
func fitWrapped(view iup.Ihandle, w int) {
	labels := wrapLabels[view]
	if len(labels) == 0 || w <= 0 {
		return
	}
	size := fmt.Sprintf("%dx", w-2*summaryMargin)
	for _, l := range labels {
		l.SetAttribute("RASTERSIZE", size)
	}
	iup.Refresh(view)
}

// viewAlive reports whether h is still on an open screen.
func viewAlive(h iup.Ihandle) bool {
	if isDescendant(iup.GetHandle("views"), h) {
		return true
	}
	for _, s := range screens {
		if isDescendant(s, h) {
			return true
		}
	}
	return false
}

// bus is the element that receives the messages background goroutines post to the main loop.
func bus() iup.Ihandle { return iup.GetHandle("dlg") }

func markTab(iup.Ihandle, string, int) {}

func setTabTip(iup.Ihandle, string) {}

// layoutSummary stacks a details view in a vertical scroll: the poster, the facts, the buttons, the overview; Favorite is in the app bar.
func layoutSummary(holder iup.Ihandle, v summaryView) iup.Ihandle {
	m := v.movie
	poster := posterView(posterW, posterH, nil, nil, nil)
	go loadPosterSized(poster, posterW, posterH, m.PosterXLarge, m.PosterLarge, m.PosterMedium)

	overview := iup.Label(plainText(v.overview)).SetAttributes("WORDWRAP=YES")
	wrapLabels[holder] = append(v.wrapped, overview)

	star := iup.MenuItem("Favorite")
	favoriteToggle(holder, star, m)
	if dlg := iup.GetDialog(holder); dlg != 0 {
		old := iup.GetAttributeHandle(dlg, "MENU")
		iup.SetAttributeHandle(dlg, "MENU", iup.Menu(star))
		if old != 0 {
			forgetMenuIcons(old)
			iup.Destroy(old)
		}
	}

	items := []iup.Ihandle{iup.Hbox(iup.Fill(), poster, iup.Fill())}
	items = append(items, v.top...)
	items = append(items, iup.Hbox(v.actions...).SetAttributes("GAP=8"), overview)
	body := iup.Vbox(items...).SetAttributes(fmt.Sprintf("GAP=8, NMARGIN=%dx%d", summaryMargin, summaryMargin))
	_, w, _ := iup.GetInt2(iup.GetDialog(holder), "CLIENTSIZE")
	fitWrapped(holder, w)
	return iup.ScrollBox(body).SetAttributes("SCROLLBAR=VERTICAL")
}

// settingsBody stacks every settings page in one vertical scroll.
func settingsBody(pageList []settingsPage) iup.Ihandle {
	var items []iup.Ihandle
	for _, pg := range pageList {
		title := iup.Label(pg.name).SetAttributes("FONTSTYLE=Bold, FONTSIZE=16")
		summary := setThemed(iup.Label(pg.summary), "FGCOLOR", roleSubtle)
		items = append(items, iup.Vbox(title, summary, pg.body).SetAttributes("GAP=6"))
	}
	return iup.ScrollBox(iup.Vbox(items...).SetAttributes("GAP=16")).SetAttributes("SCROLLBAR=VERTICAL")
}
