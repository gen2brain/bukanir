//go:build media

package main

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/gen2brain/bukanir"
	"github.com/gen2brain/iup-go/iup"
)

// menuSectionLimit caps items per Movies/Episodes submenu.
const menuSectionLimit = 20

// suggestLabelMax caps the visible length of a suggested title.
const suggestLabelMax = 60

// suggestTitles holds the plain titles behind the current suggestion list.
var suggestTitles []string

// watchActions starts the current detail tab's playback on Enter, keyed by the tab.
var watchActions = map[iup.Ihandle]func(){}

// statusRow is one status bar; every screen has its own, and all of them show the same state.
type statusRow struct{ label, progress, tor iup.Ihandle }

var statusRows []statusRow

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

// mediaMenu / sortMenu are radio-style filter menus; each item keeps its value in FILTER.
func mediaMenu() iup.Ihandle {
	opts := []struct{ label, val string }{{"All", "all"}, {"Movies", "movies"}, {"Episodes", "episodes"}}
	var items []iup.Ihandle
	for _, o := range opts {
		mi := menuItemAction(o.label, func() {
			media = o.val
			showFilters()
		})
		mi.SetAttribute("FILTER", o.val)
		if media == o.val {
			mi.SetAttribute("VALUE", "ON")
		}
		items = append(items, mi)
	}
	return iup.Menu(items...).SetAttribute("RADIO", "YES")
}

func sortMenu() iup.Ihandle {
	opts := []struct{ label, val string }{{"Seeders", "seeders"}, {"Episodes", "episodes"}}
	var items []iup.Ihandle
	for _, o := range opts {
		mi := menuItemAction(o.label, func() {
			sortBy = o.val
			showFilters()
			resortCurrentTab()
		})
		mi.SetAttribute("FILTER", o.val)
		if sortBy == o.val {
			mi.SetAttribute("VALUE", "ON")
		}
		items = append(items, mi)
	}
	return iup.Menu(items...).SetAttribute("RADIO", "YES")
}

func titleCase(s string) string {
	if s == "" {
		return ""
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

// popularMenu / topRatedMenu / genreMenu build their menus from background data.
func popularMenu() iup.Ihandle  { return titleMenu(cachedPopular()) }
func topRatedMenu() iup.Ihandle { return titleMenu(cachedTopRated()) }

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
			openRootBrowse(name, func(ctx context.Context) ([]bukanir.TMovie, error) {
				return client.Genre(ctx, id, 0)
			})
		}))
	}
	return iup.Menu(items...)
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

// onSuggestions shows the autocomplete results for the current query.
func onSuggestions(items []bukanir.TItem) {
	suggestTitles = suggestTitles[:0]
	labels := make([]string, len(items))
	for i, it := range items {
		suggestTitles = append(suggestTitles, it.Title)
		labels[i] = suggestLabel(it)
	}
	showSuggestions(labels)
}

// clearList removes every item from a List.
func clearList(list iup.Ihandle) {
	for n := list.GetInt("COUNT"); n >= 1; n-- {
		list.SetAttribute("REMOVEITEM", strconv.Itoa(n))
	}
}

// fillList replaces the items of a List.
func fillList(list iup.Ihandle, items []string) {
	clearList(list)
	for i, s := range items {
		iup.SetAttributeId(list, "", i+1, s)
	}
}

func iconButton(icon string, onClick func()) iup.Ihandle {
	b := iup.Button("").SetAttributes("FLAT=YES")
	if _, ok := iconPaths[icon]; ok {
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

// statusBar shows a status message, a progress bar, and the Tor status icon.
// The first one owns the "status", "progress" and "torStatus" handles that goroutines post to.
func statusBar() iup.Ihandle {
	status := iup.Label("Ready").SetAttributes("EXPAND=HORIZONTAL, ELLIPSIS=YES, RASTERSIZE=1x")
	tor := iup.Label("").SetAttributes("IMAGE=tor-off, TIP=\"Tor is off\"")
	progress := iup.ProgressBar().SetAttributes("RASTERSIZE=180x16, VISIBLE=NO").SetAttribute("FGCOLOR", accentColor)
	if len(statusRows) == 0 {
		status.SetHandle("status")
		status.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(_ iup.Ihandle, s string, _ int, _ any) int {
			setStatus(s)
			return iup.DEFAULT
		}))
		tor.SetHandle("torStatus")
		tor.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(onTorDone))
		progress.SetHandle("progress")
		progress.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(_ iup.Ihandle, s string, i int, _ any) int {
			setProgress(s != "hide", float64(i)/1000)
			return iup.DEFAULT
		}))
	} else {
		first := statusRows[0]
		status.SetAttribute("TITLE", first.label.GetAttribute("TITLE"))
		progress.SetAttribute("VISIBLE", first.progress.GetAttribute("VISIBLE"))
		progress.SetAttribute("VALUE", first.progress.GetAttribute("VALUE"))
		tor.SetAttribute("IMAGE", first.tor.GetAttribute("IMAGE"))
		tor.SetAttribute("TIP", first.tor.GetAttribute("TIP"))
	}
	row := statusRow{status, progress, tor}
	statusRows = append(statusRows, row)
	box := iup.Hbox(status, progress, tor).SetAttributes("GAP=8, ALIGNMENT=ACENTER, MARGIN=6x4")
	box.SetCallback("DESTROY_CB", iup.DestroyFunc(func(iup.Ihandle) int {
		statusRows = slices.DeleteFunc(statusRows, func(r statusRow) bool { return r == row })
		return iup.DEFAULT
	}))
	return box
}

func setStatus(s string) {
	for _, r := range statusRows {
		r.label.SetAttribute("TITLE", plainText(s))
	}
}

// setProgress shows the status bar progress at frac (0..1), or hides it.
func setProgress(show bool, frac float64) {
	for _, r := range statusRows {
		r.progress.SetAttribute("VISIBLE", yesNo(show))
		if show {
			r.progress.SetAttribute("VALUE", strconv.FormatFloat(frac, 'f', 3, 64))
		}
	}
}

// plainText doubles "&" so IUP shows it instead of taking it as a mnemonic.
func plainText(s string) string {
	return strings.ReplaceAll(s, "&", "&&")
}

// onBusMessage handles the messages every shell receives: "run", "suggest" and "enable".
func onBusMessage(msg string, _ int, p any) int {
	switch msg {
	case "run":
		if fn, ok := p.(func()); ok {
			fn()
		}
	case "suggest":
		if items, ok := p.([]bukanir.TItem); ok {
			onSuggestions(items)
		}
	case "enable":
		if h, ok := p.(iup.Ihandle); ok && viewAlive(h) {
			h.SetAttribute("ACTIVE", "YES")
			if bg := h.GetAttribute("SAVEDBG"); bg != "" {
				h.SetAttribute("BGCOLOR", bg)
			}
			if fg := h.GetAttribute("SAVEDFG"); fg != "" {
				h.SetAttribute("FGCOLOR", fg)
			}
		}
	}
	return iup.DEFAULT
}

// forgetView stops a closing view's playbacks and drops its state.
func forgetView(view iup.Ihandle) {
	stopSessions(view)
	delete(reloaders, view)
	delete(tabSorts, view)
	delete(watchActions, view)
	forgetFavoritesTab(view)
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

// tabActivity counts a tab's in-progress playbacks by phase.
type tabActivity struct{ loading, playing, paused int }

func (a *tabActivity) add(kind string, delta int) {
	switch kind {
	case "load":
		a.loading += delta
	case "play":
		a.playing += delta
	case "pause":
		a.paused += delta
	}
}

// loadInTab runs fetch in the background while tab shows the loading spinner.
func loadInTab(tab iup.Ihandle, fetch func()) {
	markTab(tab, "load", +1)
	go func() {
		defer markTab(tab, "load", -1)
		fetch()
	}()
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

func childPos(parent, child iup.Ihandle) int {
	for i := 0; i < iup.GetChildCount(parent); i++ {
		if iup.GetChild(parent, i) == child {
			return i
		}
	}
	return -1
}

func isChildOf(parent, child iup.Ihandle) bool {
	return childPos(parent, child) >= 0
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
