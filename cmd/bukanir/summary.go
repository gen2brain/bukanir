//go:build media

package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/gen2brain/bukanir"
	"github.com/gen2brain/iup-go/iup"
)

const (
	busyBG = "96 96 96"
	busyFG = "56 56 56"
)

// openSummary opens a detail tab for a movie and loads its metadata.
func openSummary(m bukanir.TMovie) {
	if isShowEntry(m) {
		openShow(m)
		return
	}
	holder := iup.Vbox(loadingBox("Loading")).SetAttributes("EXPAND=YES")
	holder.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(ih iup.Ihandle, _ string, _ int, p any) int {
		if r, ok := p.(summaryResult); ok {
			showSummary(ih, r.movie, r.summary)
		}
		return iup.DEFAULT
	}))
	showDetails(m.Title, holder)
	setTabTip(holder, movieTip(m, nil))

	reload := func(ctx context.Context) {
		loadInTab(holder, func() {
			s, err := client.Summary(ctx, m.ID, m.Category, m.Season, m.Episode)
			if err != nil {
				iup.PostMessage(iup.GetHandle("status"), "error: "+err.Error(), 0, nil)
				return
			}
			movie := m
			if movie.Magnet == "" && !movie.Category.IsTV() {
				movie, _ = findTorrent(ctx, m)
			}
			iup.PostMessage(holder, "", 0, summaryResult{movie, s})
		})
	}
	setReloader(holder, reload)
	reload(context.Background())
}

// summaryResult is a title's details with the torrent it will play.
type summaryResult struct {
	movie   bukanir.TMovie
	summary bukanir.TSummary
}

// summaryView holds the widgets of a details view for the shell to lay out.
type summaryView struct {
	movie    bukanir.TMovie
	summary  bukanir.TSummary
	top      []iup.Ihandle
	wrapped  []iup.Ihandle
	rows     []iup.Ihandle
	actions  []iup.Ihandle
	overview string
}

// showSummary renders the movie detail layout into holder.
func showSummary(holder iup.Ihandle, m bukanir.TMovie, s bukanir.TSummary) {
	clearChildren(holder)
	v := summaryView{movie: m, summary: s, overview: s.Overview}

	title := iup.Label(plainText(s.Title)).SetAttributes("FONTSTYLE=Bold, FONTSIZE=19, WORDWRAP=YES, EXPAND=HORIZONTAL")
	v.wrapped = append(v.wrapped, title)

	metaParts := []string{fmt.Sprintf("%.1f/10", s.Rating), fmt.Sprintf("%d", s.Year)}
	if r := runtimeStr(s.Runtime); r != "" {
		metaParts = append(metaParts, r)
	}
	if q := qualityLabel(m.Quality); q != "" {
		metaParts = append(metaParts, q)
	}
	if m.SizeHuman != "" {
		metaParts = append(metaParts, m.SizeHuman)
	}
	meta := iup.Label(strings.Join(metaParts, "   "))
	genres := setThemed(iup.Label(plainText(strings.Join(s.Genres, ", "))), "FGCOLOR", roleDim)

	watch := accentButton("&WATCH")
	playButton(watch, func() { onWatch(m, s, holder, watch) })
	watchActions[holder] = func() {
		if watch.GetAttribute("ACTIVE") != "NO" {
			onWatch(m, s, holder, watch)
		}
	}
	v.actions = []iup.Ihandle{watch}
	if s.TrailerKey != "" {
		trailer := accentButton("&TRAILER")
		playButton(trailer, func() { onTrailer(s, holder, trailer) })
		v.actions = append(v.actions, trailer)
	}

	v.top = []iup.Ihandle{title, meta, genres}
	if m.Release != "" {
		l := setThemed(iup.Label("Torrent: "+plainText(m.Release)).SetAttributes("WORDWRAP=YES, EXPAND=HORIZONTAL"), "FGCOLOR", roleSubtle)
		v.top = append(v.top, l)
		v.wrapped = append(v.wrapped, l)
	}
	if m.FileName != "" {
		l := setThemed(iup.Label("File: "+plainText(m.FileName)).SetAttributes("WORDWRAP=YES, EXPAND=HORIZONTAL"), "FGCOLOR", roleSubtle)
		v.top = append(v.top, l)
		v.wrapped = append(v.wrapped, l)
	}
	if s.Tagline != "" {
		l := iup.Label(plainText(s.Tagline)).SetAttributes("FONTSTYLE=Italic, FONTSIZE=13, WORDWRAP=YES, EXPAND=HORIZONTAL")
		v.top = append(v.top, l)
		v.wrapped = append(v.wrapped, l)
	}
	if d := directorRow(s); d != 0 {
		v.rows = append(v.rows, d)
	}
	if c := castRow(s); c != 0 {
		v.rows = append(v.rows, c)
	}
	v.top = append(v.top, v.rows...)

	content := layoutSummary(holder, v)
	iup.Append(holder, content)
	iup.Map(content)
	iup.Refresh(holder)
	for _, r := range v.rows {
		centerCaption(r)
	}
	iup.Refresh(holder)

	setTabTip(holder, movieTip(m, &s))
}

// favoriteToggle makes star add or remove m from favorites and keeps its icon in sync.
func favoriteToggle(holder, star iup.Ihandle, m bukanir.TMovie) {
	if old, ok := favoriteButtons[holder]; ok {
		delete(monoButtons, old.button)
	}
	setFavoriteIcon(star, isFavorite(m))
	star.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		toggleFavorite(m)
		return iup.DEFAULT
	}))
	favoriteActions[holder] = func() { toggleFavorite(m) }
	favoriteButtons[holder] = favoriteButton{star, m}
}

// movieTip is a details tab tooltip: the title and year, then the facts of the details page once s is loaded.
func movieTip(m bukanir.TMovie, s *bukanir.TSummary) string {
	head := m.Title
	if m.Year > 0 {
		head = fmt.Sprintf("%s (%d)", m.Title, m.Year)
	}
	if m.Category.IsTV() && !isShowEntry(m) {
		head += fmt.Sprintf(" S%02dE%02d", m.Season, m.Episode)
	}
	var facts []string
	if s != nil && s.Rating > 0 {
		facts = append(facts, fmt.Sprintf("%.1f/10", s.Rating))
	}
	if s != nil {
		if r := runtimeStr(s.Runtime); r != "" {
			facts = append(facts, r)
		}
	}
	if q := qualityLabel(m.Quality); q != "" {
		facts = append(facts, q)
	}
	if m.SizeHuman != "" {
		facts = append(facts, m.SizeHuman)
	}
	lines := []string{head}
	if len(facts) > 0 {
		lines = append(lines, strings.Join(facts, "   "))
	}
	if s != nil && len(s.Genres) > 0 {
		lines = append(lines, strings.Join(s.Genres, ", "))
	}
	if s != nil && s.Director != "" {
		lines = append(lines, "Director: "+s.Director)
	}
	return strings.Join(lines, "\n")
}

func directorRow(s bukanir.TSummary) iup.Ihandle {
	if s.Director == "" {
		return 0
	}
	id, name := s.DirectorID, s.Director
	link := linkButton(name, func() {
		openBrowse(name, func(ctx context.Context) ([]bukanir.TMovie, error) {
			return client.Crew(ctx, id, 0)
		})
	})
	return fieldRow(fieldLabel("Director:"), link)
}

func castRow(s bukanir.TSummary) iup.Ihandle {
	items := []iup.Ihandle{fieldLabel("Cast:")}
	for i := 0; i < len(s.Cast) && i < 6; i++ {
		name := s.Cast[i]
		id := 0
		if i < len(s.CastIDs) {
			id = s.CastIDs[i]
		}
		items = append(items, linkButton(name, func() {
			openBrowse(name, func(ctx context.Context) ([]bukanir.TMovie, error) {
				return client.Cast(ctx, id, 0)
			})
		}))
	}
	if len(items) == 1 {
		return 0
	}
	return fieldRow(items...)
}

func fieldRow(items ...iup.Ihandle) iup.Ihandle {
	return iup.MultiBox(items...).SetAttributes("ORIENTATION=HORIZONTAL, GAPHORIZ=6, GAPVERT=4")
}

// centerCaption pads a mapped field row's caption so its text centers on the first name button.
func centerCaption(row iup.Ihandle) {
	var lw, lh, bw, bh int
	fmt.Sscanf(iup.GetChild(row, 0).GetAttribute("NATURALSIZE"), "%dx%d", &lw, &lh)
	fmt.Sscanf(iup.GetChild(row, 1).GetAttribute("NATURALSIZE"), "%dx%d", &bw, &bh)
	if bh > lh {
		iup.GetChild(row, 0).SetAttribute("PADDING", fmt.Sprintf("0x%d", (bh-lh)/2))
	}
}

func fieldLabel(text string) iup.Ihandle {
	return setThemed(iup.Label(text), "FGCOLOR", roleDim)
}

// openBrowse opens a new grid tab populated by load, from a link in the content.
func openBrowse(title string, load func(context.Context) ([]bukanir.TMovie, error)) {
	openBrowseSorted(title, sortBy, false, load)
}

// openRootBrowse is openBrowse from the menus or the search, which replaces the main grid on mobile.
func openRootBrowse(title string, load func(context.Context) ([]bukanir.TMovie, error)) {
	openBrowseSorted(title, sortBy, true, load)
}

// openBrowseSorted is openBrowse with the tab's initial sort order.
func openBrowseSorted(title, by string, root bool, load func(context.Context) ([]bukanir.TMovie, error)) {
	holder := gridTab()
	setSortable(holder, by)
	showList(title, holder, root)
	reload := func(ctx context.Context) {
		loadInTab(holder, func() {
			movies, err := load(ctx)
			if err != nil {
				iup.PostMessage(iup.GetHandle("status"), "error: "+err.Error(), 0, nil)
				return
			}
			iup.PostMessage(holder, "", 0, movies)
		})
	}
	setReloader(holder, reload)
	reload(context.Background())
}

func accentButton(label string) iup.Ihandle {
	return iup.Button(label).SetAttributes(`PADDING=22x8, FLAT=YES, FGCOLOR="0 0 0"`).SetAttribute("BGCOLOR", accentColor)
}

func playButton(b iup.Ihandle, onClick func()) {
	b.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		onClick()
		return iup.DEFAULT
	}))
}

// disablePlayButton greys and blocks a play button, saving colors for enable.
func disablePlayButton(b iup.Ihandle) {
	b.SetAttribute("SAVEDBG", b.GetAttribute("BGCOLOR"))
	b.SetAttribute("SAVEDFG", b.GetAttribute("FGCOLOR"))
	b.SetAttribute("BGCOLOR", busyBG)
	b.SetAttribute("FGCOLOR", busyFG)
	b.SetAttribute("ACTIVE", "NO")
}

func linkButton(label string, onClick func()) iup.Ihandle {
	b := iup.Button(plainText(label)).SetAttributes("FLAT=YES")
	b.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		onClick()
		return iup.DEFAULT
	}))
	return b
}

func runtimeStr(min int) string {
	if min <= 0 {
		return ""
	}
	return fmt.Sprintf("%dmin", min)
}
