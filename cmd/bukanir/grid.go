//go:build media

package main

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"github.com/gen2brain/bukanir"
	"github.com/gen2brain/iup-go/iup"
)

const accentColor = "255 167 37"

const (
	roleDim colorRole = iota
	roleSubtle
	roleDialogBg
)

// posterSeq generates unique IUP image-handle names for posters.
var posterSeq atomic.Int64

// selectedTitle and selectedPoster are the highlighted grid cell's widgets.
var selectedTitle, selectedPoster iup.Ihandle

var accentRGBA = color.RGBA{255, 167, 37, 255}

var themed = map[iup.Ihandle]struct {
	attr string
	role colorRole
}{}

var (
	liveTargets   sync.Map
	pendingImages []string
)

type colorRole int

// roleColor resolves a color role for the current appearance.
func roleColor(r colorRole) string {
	dark := iup.GetGlobal("DARKMODE") == "YES"
	switch r {
	case roleDim:
		if dark {
			return "170 170 170"
		}
		return "120 120 120"
	case roleSubtle:
		if dark {
			return "150 150 150"
		}
		return "142 142 142"
	}
	return iup.GetGlobal("DLGBGCOLOR")
}

// setThemed sets attr to the role color and keeps it updated on appearance changes.
func setThemed(ih iup.Ihandle, attr string, role colorRole) iup.Ihandle {
	ih.SetAttribute(attr, roleColor(role))
	if _, ok := themed[ih]; !ok {
		ih.SetCallback("DESTROY_CB", iup.DestroyFunc(func(ih iup.Ihandle) int {
			delete(themed, ih)
			return iup.DEFAULT
		}))
	}
	themed[ih] = struct {
		attr string
		role colorRole
	}{attr, role}
	return ih
}

// onThemeChanged re-applies every color the app derived from the palette.
func onThemeChanged(iup.Ihandle, int) int {
	for ih, t := range themed {
		ih.SetAttribute(t.attr, roleColor(t.role))
	}
	retintIcons()
	liveTargets.Range(func(k, _ any) bool {
		ih := k.(iup.Ihandle)
		if img := ih.GetAttribute("IMAGE"); strings.HasPrefix(img, "poster-blank-") {
			var w, h int
			fmt.Sscanf(img, "poster-blank-%dx%d", &w, &h)
			ih.SetAttribute("IMAGE", blankPoster(w, h))
		}
		return true
	})
	return iup.DEFAULT
}

// globalColor parses an "R G B" global attribute into a color.
func globalColor(name string) color.RGBA {
	var r, g, b uint8
	fmt.Sscanf(iup.GetGlobal(name), "%d %d %d", &r, &g, &b)
	return color.RGBA{r, g, b, 255}
}

// trackTarget keeps ih in liveTargets until destroyed, then drops it as the selected cell and queues the images named by attrs for release.
func trackTarget(ih iup.Ihandle, attrs ...string) {
	liveTargets.Store(ih, struct{}{})
	ih.SetCallback("DESTROY_CB", iup.DestroyFunc(func(ih iup.Ihandle) int {
		liveTargets.Delete(ih)
		if ih == selectedPoster {
			selectedTitle, selectedPoster = 0, 0
		}
		for _, a := range attrs {
			if name := ih.GetAttribute(a); name != "" {
				pendingImages = append(pendingImages, name)
			}
		}
		return iup.DEFAULT
	}))
}

// targetAlive reports whether a loader may still post to ih.
func targetAlive(ih iup.Ihandle) bool {
	_, ok := liveTargets.Load(ih)
	return ok
}

// releaseImages destroys the images whose controls have been destroyed.
func releaseImages() {
	for _, name := range pendingImages {
		if h := iup.GetHandle(name); h != 0 {
			iup.Destroy(h)
		}
	}
	pendingImages = pendingImages[:0]
}

// selectCell highlights a cell as the current selection.
func selectCell(title, poster iup.Ihandle) {
	if selectedTitle != 0 && selectedTitle != title {
		selectedTitle.SetAttribute("FGCOLOR", nil)
	}
	if selectedPoster != 0 && selectedPoster != poster {
		setPosterSelected(selectedPoster, false)
	}
	title.SetAttribute("FGCOLOR", accentColor)
	setPosterSelected(poster, true)
	selectedTitle, selectedPoster = title, poster
}

// setPosterSelected swaps the poster label between its plain and bordered image.
func setPosterSelected(poster iup.Ihandle, on bool) {
	key := "BASEIMG"
	if on {
		key = "SELIMG"
	}
	poster.SetAttribute("SELECTED", map[bool]string{true: "YES", false: "NO"}[on])
	if img := poster.GetAttribute(key); img != "" {
		poster.SetAttribute("IMAGE", img)
		iup.Refresh(poster)
	}
}

// borderedPoster returns a copy of src with an accent border drawn along its edge.
func borderedPoster(src image.Image, thickness int) *image.RGBA {
	b := src.Bounds()
	dst := image.NewRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	c := accentRGBA
	for t := range thickness {
		x0, y0, x1, y1 := b.Min.X+t, b.Min.Y+t, b.Max.X-1-t, b.Max.Y-1-t
		for x := x0; x <= x1; x++ {
			dst.Set(x, y0, c)
			dst.Set(x, y1, c)
		}
		for y := y0; y <= y1; y++ {
			dst.Set(x0, y, c)
			dst.Set(x1, y, c)
		}
	}
	return dst
}

// showMovies replaces holder's content with a scrollable poster grid.
func showMovies(holder iup.Ihandle, movies []bukanir.TMovie) {
	clearChildren(holder)
	if len(movies) == 0 {
		text := holder.GetAttribute("EMPTYTEXT")
		if text == "" {
			text = "No results"
		}
		lbl := placeholder(text)
		iup.Append(holder, lbl)
		iup.Map(lbl)
		iup.Refresh(holder)
		return
	}

	cells := make([]iup.Ihandle, len(movies))
	targets := make([]iup.Ihandle, len(movies))
	for i, m := range movies {
		cells[i], targets[i] = posterCell(m)
	}
	grid := iup.MultiBox(cells...).SetAttributes(map[string]string{
		"ORIENTATION": "HORIZONTAL",
		"NGAPHORIZ":   strconv.Itoa(gridGap),
		"NGAPVERT":    strconv.Itoa(gridGap),
		"NMARGIN":     fmt.Sprintf("%dx%d", gridMargin, gridMargin),
	}).SetHandle("grid")

	scroll := iup.ScrollBox(grid).SetAttributes("EXPAND=YES, CANFOCUS=YES")
	scroll.SetCallback("RESIZE_CB", iup.ResizeFunc(func(ih iup.Ihandle, _, _ int) int {
		fitGrid(ih, grid)
		return iup.DEFAULT
	}))
	scroll.SetCallback("K_ANY", iup.KAnyFunc(func(_ iup.Ihandle, c int) int {
		if navGrid(scroll, grid, movies, c) {
			return iup.IGNORE
		}
		return iup.CONTINUE
	}))
	iup.Append(holder, scroll)
	iup.Map(scroll)
	iup.Refresh(holder)
	iup.Refresh(holder)
	fitGrid(scroll, grid)

	selectGridCell(scroll, grid, 0)
	iup.SetFocus(scroll)

	for i, m := range movies {
		go loadPoster(targets[i], m.PosterLarge, m.PosterXLarge, m.PosterMedium)
	}
}

// navGrid moves the selection by arrow keys and opens on Enter, reporting whether it handled the key.
func navGrid(scroll, grid iup.Ihandle, movies []bukanir.TMovie, key int) bool {
	n := len(movies)
	if n == 0 {
		return false
	}
	cur := gridIndexOf(grid, selectedPoster)
	switch key {
	case iup.K_CR:
		if cur >= 0 && cur < n {
			openSummary(movies[cur])
		}
		return true
	case iup.K_LEFT, iup.K_RIGHT, iup.K_UP, iup.K_DOWN:
	default:
		return false
	}
	if cur < 0 {
		selectGridCell(scroll, grid, 0)
		return true
	}
	next := cur
	switch key {
	case iup.K_LEFT:
		next--
	case iup.K_RIGHT:
		next++
	case iup.K_UP:
		next -= gridCols(grid)
	case iup.K_DOWN:
		next += gridCols(grid)
	}
	if next >= 0 && next < n {
		selectGridCell(scroll, grid, next)
	}
	return true
}

func gridIndexOf(grid, poster iup.Ihandle) int {
	if poster == 0 {
		return -1
	}
	for i := 0; i < iup.GetChildCount(grid); i++ {
		if iup.GetChild(iup.GetChild(grid, i), 0) == poster {
			return i
		}
	}
	return -1
}

// gridCols infers the column count from cell x positions (x resets on a new row).
func gridCols(grid iup.Ihandle) int {
	n := iup.GetChildCount(grid)
	if n <= 1 {
		return 1
	}
	prev := cellX(grid, 0)
	for i := 1; i < n; i++ {
		x := cellX(grid, i)
		if x <= prev {
			return i
		}
		prev = x
	}
	return n
}

func cellX(grid iup.Ihandle, i int) int {
	pos := iup.GetChild(grid, i).GetAttribute("POSITION")
	if k := strings.IndexByte(pos, ','); k > 0 {
		x, _ := strconv.Atoi(pos[:k])
		return x
	}
	return 0
}

func selectGridCell(scroll, grid iup.Ihandle, i int) {
	if i < 0 || i >= iup.GetChildCount(grid) {
		return
	}
	cell := iup.GetChild(grid, i)
	selectCell(iup.GetChild(cell, 1), iup.GetChild(cell, 0))
	scrollToShowCell(scroll, cell)
}

// fitGrid justifies grid until its layout settles.
func fitGrid(scroll, grid iup.Ihandle) {
	for range 3 {
		if !justifyGrid(scroll, grid) {
			return
		}
		iup.Refresh(scroll)
	}
}

// justifyGrid fits as many columns as scroll shows and spreads the spare width evenly over the gaps and margins.
func justifyGrid(scroll, grid iup.Ihandle) bool {
	var cellW, cellH int
	w := scroll.GetInt("DX")
	fmt.Sscanf(iup.GetChild(grid, 0).GetAttribute("NATURALSIZE"), "%dx%d", &cellW, &cellH)
	if w <= 0 || cellW <= 0 {
		return false
	}
	n := max(1, (w-2*gridMargin+gridGap)/(cellW+gridGap))
	extra := max(0, (w-n*cellW-(n-1)*gridGap-2*gridMargin)/(n+1))
	gap, margin := strconv.Itoa(gridGap+extra), fmt.Sprintf("%dx%d", gridMargin+extra, gridMargin)
	if grid.GetAttribute("NGAPHORIZ") == gap && grid.GetAttribute("NMARGIN") == margin {
		return false
	}
	grid.SetAttribute("NGAPHORIZ", gap)
	grid.SetAttribute("NMARGIN", margin)
	return true
}

// scrollToShowCell scrolls the grid only as far as needed for the cell to be fully visible.
func scrollToShowCell(scroll, cell iup.Ihandle) {
	_, _, top := iup.GetInt2(cell, "POSITION")
	var cellW, cellH int
	fmt.Sscanf(cell.GetAttribute("RASTERSIZE"), "%dx%d", &cellW, &cellH)
	bottom := top + cellH
	visible := scroll.GetInt("DY")
	posY := scroll.GetInt("POSY")
	switch {
	case top < 0:
		posY += top
	case bottom > visible:
		posY += bottom - visible
	default:
		return
	}
	scroll.SetAttribute("SCROLLTO", fmt.Sprintf("%d,%d", scroll.GetInt("POSX"), posY))
}

// posterCell builds a grid cell (poster + title + subtitle) and its image target.
func posterCell(m bukanir.TMovie) (cell, target iup.Ihandle) {
	titleAttr := fmt.Sprintf("ALIGNMENT=ACENTER, WORDWRAP=YES, RASTERSIZE=%dx", posterW)
	title := iup.Label(plainText(elide(m.Title, 40))).SetAttributes(titleAttr)
	sub := setThemed(iup.Label(movieSubtitle(m)).SetAttributes(fmt.Sprintf("ALIGNMENT=ACENTER, RASTERSIZE=%dx", posterW)), "FGCOLOR", roleSubtle)
	movie := m

	var poster iup.Ihandle
	poster = posterView(posterW, posterH,
		func() {
			selectCell(title, poster)
			cell := iup.GetParent(poster)
			scrollToShowCell(iup.GetParent(iup.GetParent(cell)), cell)
		},
		func() { selectCell(title, poster); openSummary(movie) },
		func() { favoriteMenu(movie) },
	)
	target = poster
	poster.SetAttribute("TIP", m.Title)
	title.SetAttribute("TIP", m.Title)

	cell = iup.Vbox(poster, title, sub).SetAttributes("ALIGNMENT=ACENTER, GAP=2, MARGIN=0x0")
	return cell, target
}

// posterView returns an image Label sized to its poster, filled via PostMessage; on a phone a release inside it opens.
func posterView(w, h int, onSelect, onOpen, onMenu func()) iup.Ihandle {
	name := "poster" + strconv.FormatInt(posterSeq.Add(1), 10)

	var poster iup.Ihandle
	switch {
	case phone && onOpen != nil:
		poster = iup.Label("").SetAttribute("IMAGE", blankPoster(w, h))
		poster.SetCallback("BUTTON_CB", iup.ButtonFunc(func(ih iup.Ihandle, button, pressed, x, y int, _ string) int {
			_, pw, ph := iup.GetInt2(ih, "RASTERSIZE")
			if button == iup.BUTTON1 && pressed == 0 && x >= 0 && y >= 0 && x < pw && y < ph {
				onOpen()
			}
			return iup.DEFAULT
		}))
	case onSelect != nil || onOpen != nil || onMenu != nil:
		poster = iup.Label("").SetAttribute("IMAGE", blankPoster(w, h))
		poster.SetCallback("BUTTON_CB", iup.ButtonFunc(func(_ iup.Ihandle, button, pressed, _, _ int, status string) int {
			if button == iup.BUTTON3 && pressed == 1 && onMenu != nil {
				if onSelect != nil {
					onSelect()
				}
				onMenu()
				return iup.DEFAULT
			}
			if button != iup.BUTTON1 || pressed != 1 {
				return iup.DEFAULT
			}
			if iup.IsDouble(status) {
				if onOpen != nil {
					onOpen()
				}
			} else if onSelect != nil {
				onSelect()
			}
			return iup.DEFAULT
		}))
	default:
		poster = iup.Label("").SetAttribute("IMAGE", blankPoster(w, h))
	}
	trackTarget(poster, "BASEIMG", "SELIMG")
	poster.SetCallback("POSTMESSAGE_CB", iup.PostMessageFunc(func(ih iup.Ihandle, _ string, _ int, p any) int {
		img, ok := p.(image.Image)
		if !ok {
			return iup.DEFAULT
		}
		iup.Destroy(iup.GetHandle(name))
		iup.Destroy(iup.GetHandle(name + "-sel"))
		iup.ImageFromImage(img).SetHandle(name)
		iup.ImageFromImage(borderedPoster(img, 4)).SetHandle(name + "-sel")
		ih.SetAttribute("BASEIMG", name)
		ih.SetAttribute("SELIMG", name+"-sel")
		if ih.GetAttribute("SELECTED") == "YES" {
			ih.SetAttribute("IMAGE", name+"-sel")
		} else {
			ih.SetAttribute("IMAGE", name)
		}
		iup.Refresh(ih)
		return iup.DEFAULT
	}))
	return poster
}

// blankPoster returns the name of a shared placeholder image of the given size in the current text background.
func blankPoster(w, h int) string {
	bg := globalColor("TXTBGCOLOR")
	name := fmt.Sprintf("poster-blank-%dx%d-%02x%02x%02x", w, h, bg.R, bg.G, bg.B)
	if iup.GetHandle(name) == 0 {
		m := image.NewRGBA(image.Rect(0, 0, w, h))
		draw.Draw(m, m.Bounds(), image.NewUniform(bg), image.Point{}, draw.Src)
		iup.ImageFromImage(m).SetHandle(name)
	}
	return name
}

func movieSubtitle(m bukanir.TMovie) string {
	s := strconv.Itoa(m.Year)
	if m.Category.IsTV() && !isShowEntry(m) {
		s = fmt.Sprintf("S%02dE%02d", m.Season, m.Episode)
	}
	if q := qualityLabel(m.Quality); q != "" {
		s += " (" + q + ")"
	}
	return s
}

// qualityLabel turns a release quality such as "1080" into "1080p".
func qualityLabel(quality string) string {
	if quality == "" {
		return ""
	}
	return quality + "p"
}

func elide(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return strings.TrimSpace(string(r[:n-1])) + "..."
}
