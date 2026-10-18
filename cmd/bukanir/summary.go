package main

import (
	"context"
	"fmt"
	"image"
	"math"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/gen2brain/bukanir"
	"github.com/gen2brain/iup-go/iup"
)

const summaryPosterW, summaryPosterH = 600, 900

// openSummary opens a detail tab for a movie and loads its metadata.
func openSummary(m bukanir.TMovie) {
	holder := iup.Vbox(loadingBox("Loading")).SetAttributes("EXPAND=YES")
	movie := m
	holder.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(ih iup.Ihandle, _ string, _ int, p any) int {
		if s, ok := p.(bukanir.TSummary); ok {
			showSummary(ih, movie, s)
		}
		return iup.DEFAULT
	}))
	addTab(m.Title, holder)

	reload := func() {
		go func() {
			s, err := client.Summary(context.Background(), m.ID, m.Category, m.Season, m.Episode)
			if err != nil {
				iup.PostMessage(iup.GetHandle("status"), "error: "+err.Error(), 0, nil)
				return
			}
			iup.PostMessage(holder, "", 0, s)
		}()
	}
	setReloader(holder, reload)
	reload()
}

// showSummary renders the movie detail layout into holder.
func showSummary(holder iup.Ihandle, m bukanir.TMovie, s bukanir.TSummary) {
	clearChildren(holder)

	poster := summaryPoster()

	title := iup.Label(s.Title).SetAttributes("FONTSTYLE=Bold, FONTSIZE=19, WORDWRAP=YES, EXPAND=HORIZONTAL")

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
	genres := setThemed(iup.Label(strings.Join(s.Genres, ", ")), "FGCOLOR", roleDim)

	overview := iup.Text().SetAttributes("MULTILINE=YES, READONLY=YES, WORDWRAP=YES, BORDER=NO, SCROLLBAR=VERTICAL, EXPAND=YES")
	setThemed(overview, "BGCOLOR", roleDialogBg)
	overview.SetAttribute("VALUE", s.Overview)

	watch := accentButton("&WATCH")
	playButton(watch, func() { onWatch(m, s, holder, watch) })
	watchActions[holder] = func() {
		if watch.GetAttribute("ACTIVE") != "NO" {
			onWatch(m, s, holder, watch)
		}
	}
	actions := []iup.Ihandle{watch}
	if s.TrailerKey != "" {
		trailer := accentButton("&TRAILER")
		playButton(trailer, func() { onTrailer(s, holder, trailer) })
		actions = append(actions, trailer)
	}

	top := []iup.Ihandle{title, meta, genres}
	if m.Release != "" {
		top = append(top, setThemed(iup.Label("Torrent: "+m.Release).SetAttributes("WORDWRAP=YES, EXPAND=HORIZONTAL"), "FGCOLOR", roleSubtle))
	}
	if m.FileName != "" {
		top = append(top, setThemed(iup.Label("File: "+m.FileName).SetAttributes("WORDWRAP=YES, EXPAND=HORIZONTAL"), "FGCOLOR", roleSubtle))
	}
	if s.Tagline != "" {
		top = append(top, iup.Label(s.Tagline).SetAttributes("FONTSTYLE=Italic, FONTSIZE=13, WORDWRAP=YES, EXPAND=HORIZONTAL"))
	}
	var rows []iup.Ihandle
	if d := directorRow(s); d != 0 {
		rows = append(rows, d)
	}
	if c := castRow(s); c != 0 {
		rows = append(rows, c)
	}
	top = append(top, rows...)

	topBox := iup.Vbox(top...).SetAttributes("GAP=8, EXPAND=HORIZONTAL")
	buttons := iup.Hbox(actions...).SetAttributes("GAP=8, EXPAND=HORIZONTAL")

	right := iup.Vbox(topBox, overview, buttons).SetAttributes("GAP=8, EXPAND=YES, NMARGIN=16x14")

	split := iup.Split(poster, right)
	split.SetAttributes("ORIENTATION=VERTICAL, VALUE=330, BARSIZE=4, MINMAX=200:560")
	iup.Append(holder, split)
	iup.Map(split)
	iup.Refresh(holder)
	for _, r := range rows {
		centerCaption(r)
	}
	iup.Refresh(holder)

	go loadPosterSized(posterPref(m), poster, summaryPosterW, summaryPosterH)
}

var summaryPosterSeq atomic.Int64

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
	for i := 0; i < len(s.Cast) && i < 5; i++ {
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

// openBrowse opens a new grid tab populated by load.
func openBrowse(title string, load func(context.Context) ([]bukanir.TMovie, error)) {
	holder := gridTab()
	addTab(title, holder)
	reload := func() {
		go func() {
			movies, err := load(context.Background())
			if err != nil {
				iup.PostMessage(iup.GetHandle("status"), "error: "+err.Error(), 0, nil)
				return
			}
			iup.PostMessage(holder, "", 0, movies)
		}()
	}
	setReloader(holder, reload)
	reload()
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

const (
	busyBG = "96 96 96"
	busyFG = "56 56 56"
)

// disablePlayButton greys and blocks a play button, saving colors for enable.
func disablePlayButton(b iup.Ihandle) {
	b.SetAttribute("SAVEDBG", b.GetAttribute("BGCOLOR"))
	b.SetAttribute("SAVEDFG", b.GetAttribute("FGCOLOR"))
	b.SetAttribute("BGCOLOR", busyBG)
	b.SetAttribute("FGCOLOR", busyFG)
	b.SetAttribute("ACTIVE", "NO")
}

func linkButton(label string, onClick func()) iup.Ihandle {
	b := iup.Button(label).SetAttributes("FLAT=YES")
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

func posterPref(m bukanir.TMovie) string {
	if m.PosterXLarge != "" {
		return m.PosterXLarge
	}
	return m.PosterLarge
}
