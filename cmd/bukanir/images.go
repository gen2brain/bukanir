//go:build media

package main

import (
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"math"
	"strconv"

	"github.com/gen2brain/folio/raster"
	"github.com/gen2brain/folio/svg"
	"github.com/gen2brain/iup-go/iup"
)

// spinnerFrameCount and spinnerFrameTime set the loading spinner animation.
const (
	spinnerFrameCount = 12
	spinnerFrameTime  = 80
)

// loadingFrames holds the spinner frame image names shared by loading boxes and tab indicators.
var loadingFrames []string

// loadLoadingAnimation registers the spinner frames and the "loading-anim" element once.
func loadLoadingAnimation() {
	anim := iup.User().SetHandle("loading-anim")
	anim.SetAttribute("FRAMETIME", strconv.Itoa(spinnerFrameTime))
	for i := 0; i < spinnerFrameCount; i++ {
		name := fmt.Sprintf("load-%d", i)
		scaledImage(spinnerGlyph(screenPixels(tabIconSize), float64(i)/spinnerFrameCount)).SetHandle(name)
		iup.Append(anim, iup.GetHandle(name))
		loadingFrames = append(loadingFrames, name)
	}
}

// loadingBox returns a centered animated spinner with a caption.
func loadingBox(text string) iup.Ihandle {
	if len(loadingFrames) == 0 {
		return placeholder(text)
	}
	spinner := iup.AnimatedLabel(iup.GetHandle("loading-anim"))
	spinner.SetAttribute("START", "YES")

	caption := setThemed(iup.Label(text), "FGCOLOR", roleSubtle)
	row := iup.Hbox(iup.Fill(), spinner, caption, iup.Fill()).SetAttributes("ALIGNMENT=ACENTER, GAP=8")
	return iup.Vbox(iup.Fill(), row, iup.Fill()).SetAttributes("EXPAND=YES")
}

const tabIconSize = 20

// tabSpinFrames holds the registered image names for the tab loading spinner.
var tabSpinFrames []string

// registerTabIndicators registers the tab-indicator images (spinner, play, pause).
func registerTabIndicators() {
	scaledImage(playGlyph(screenPixels(tabIconSize))).SetHandle("tab-play")
	scaledImage(pauseGlyph(screenPixels(tabIconSize))).SetHandle("tab-pause")
	tabSpinFrames = loadingFrames
}

// iconSize is the uniform toolbar/status icon size.
const iconSize = 24

//go:embed images/bukanir.svg
var svgLogo []byte

// svgTor is the Tor onion icon (The Tor Project, CC BY 3.0 US).
//
//go:embed images/tor.svg
var svgTor []byte

//go:embed images/tmdb.svg
var svgTMDB []byte

// tmdbHeight is the height of the TMDB logo.
const tmdbHeight = 14

// logoLargeSize is the size of the logo in the About dialog.
const logoLargeSize = 128

// iconPaths holds the mono icons as SVG path data on a 24x24 grid (Material Design icons, Apache 2.0).
var iconPaths = map[string]string{
	"refresh":           "M17.65 6.35C16.2 4.9 14.21 4 12 4c-4.42 0-7.99 3.58-7.99 8s3.57 8 7.99 8c3.73 0 6.84-2.55 7.73-6h-2.08c-.82 2.33-3.04 4-5.65 4-3.31 0-6-2.69-6-6s2.69-6 6-6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35z",
	"settings":          "M12 10c-1.1 0-2 .9-2 2s.9 2 2 2 2-.9 2-2-.9-2-2-2zm7-7H5c-1.11 0-2 .9-2 2v14c0 1.1.89 2 2 2h14c1.11 0 2-.9 2-2V5c0-1.1-.89-2-2-2zm-1.75 9c0 .23-.02.46-.05.68l1.48 1.16c.13.11.17.3.08.45l-1.4 2.42c-.09.15-.27.21-.43.15l-1.74-.7c-.36.28-.76.51-1.18.69l-.26 1.85c-.03.17-.18.3-.35.3h-2.8c-.17 0-.32-.13-.35-.29l-.26-1.85c-.43-.18-.82-.41-1.18-.69l-1.74.7c-.16.06-.34 0-.43-.15l-1.4-2.42c-.09-.15-.05-.34.08-.45l1.48-1.16c-.03-.23-.05-.46-.05-.69 0-.23.02-.46.05-.68l-1.48-1.16c-.13-.11-.17-.3-.08-.45l1.4-2.42c.09-.15.27-.21.43-.15l1.74.7c.36-.28.76-.51 1.18-.69l.26-1.85c.03-.17.18-.3.35-.3h2.8c.17 0 .32.13.35.29l.26 1.85c.43.18.82.41 1.18.69l1.74-.7c.16-.06.34 0 .43.15l1.4 2.42c.09.15.05.34-.08.45l-1.48 1.16c.03.23.05.46.05.69z",
	"log":               "M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm1 15h-2v-6h2v6zm0-8h-2V7h2v2z",
	"star":              "M12 17.27L18.18 21l-1.64-7.03L22 9.24l-7.19-.61L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21z",
	"star-outline":      "M22 9.24l-7.19-.62L12 2 9.19 8.63 2 9.24l5.46 4.73L5.82 21 12 17.27 18.18 21l-1.63-7.03L22 9.24zM12 15.4l-3.76 2.27 1-4.28-3.32-2.88 4.38-.38L12 6.1l1.71 4.04 4.38.38-3.32 2.88 1 4.28L12 15.4z",
	"player-play":       "M8 5v14l11-7z",
	"player-pause":      "M6 19h4V5H6v14zm8-14v14h4V5h-4z",
	"player-fullscreen": "M7 14H5v5h5v-2H7v-3zm-2-4h2V7h3V5H5v5zm12 7h-3v2h5v-5h-2v3zM14 5v2h3v3h2V5h-5z",
	"player-subtitles":  "M20 4H4c-1.1 0-2 .9-2 2v12c0 1.1.9 2 2 2h16c1.1 0 2-.9 2-2V6c0-1.1-.9-2-2-2zM4 12h4v2H4v-2zm10 6H4v-2h10v2zm6 0h-4v-2h4v2zm0-4H10v-2h10v2z",
	"player-volume":     "M3 9v6h4l5 5V4L7 9H3zm13.5 3c0-1.77-1.02-3.29-2.5-4.03v8.05c1.48-.73 2.5-2.25 2.5-4.02zM14 3.23v2.06c2.89.86 5 3.54 5 6.71s-2.11 5.85-5 6.71v2.06c4.01-.91 7-4.49 7-8.77s-2.99-7.86-7-8.77z",
	"player-volume-off": "M16.5 12c0-1.77-1.02-3.29-2.5-4.03v2.21l2.45 2.45c.03-.2.05-.41.05-.63zm2.5 0c0 .94-.2 1.82-.54 2.64l1.51 1.51C20.63 14.91 21 13.5 21 12c0-4.28-2.99-7.86-7-8.77v2.06c2.89.86 5 3.54 5 6.71zM4.27 3L3 4.27 7.73 9H3v6h4l5 5v-6.73l4.25 4.25c-.67.52-1.42.93-2.25 1.18v2.06c1.38-.31 2.63-.95 3.69-1.81L19.73 21 21 19.73l-9-9L4.27 3zM12 4L9.91 6.09 12 8.18V4z",
}

// iconGrid is the size of the grid the icon paths are drawn on.
const iconGrid = 24

// svgColor formats c as an SVG fill color with its opacity.
func svgColor(c color.Color) string {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	return fmt.Sprintf(`"#%02x%02x%02x" fill-opacity="%.3f"`, n.R, n.G, n.B, float64(n.A)/255)
}

// svgStroke formats c as an SVG stroke color with its opacity.
func svgStroke(c color.Color) string {
	n := color.NRGBAModel.Convert(c).(color.NRGBA)
	return fmt.Sprintf(`"#%02x%02x%02x" stroke-opacity="%.3f"`, n.R, n.G, n.B, float64(n.A)/255)
}

// drawIcon renders the named icon at size pixels in color c.
func drawIcon(name string, size int, c color.Color) *image.RGBA {
	doc := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d"><path fill=%s d="%s"/></svg>`, iconGrid, iconGrid, svgColor(c), iconPaths[name])
	return renderSVG([]byte(doc), size, size)
}

// spinnerGlyph draws a faint accent ring with a bright rounded arc rotated by turn (0..1).
func spinnerGlyph(size int, turn float64) *image.RGBA {
	c := float64(size) / 2
	width := float64(size) / 7
	r := c - 1 - width/2
	a0 := 2*math.Pi*turn - math.Pi/2
	a1 := a0 + 2*math.Pi/3
	faint := color.NRGBA{accentRGBA.R, accentRGBA.G, accentRGBA.B, 70}
	doc := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d" fill="none" stroke-width="%g">`+
		`<circle cx="%g" cy="%g" r="%g" stroke=%s/>`+
		`<path d="M%g %gA%g %g 0 0 1 %g %g" stroke=%s stroke-linecap="round"/></svg>`,
		size, size, width,
		c, c, r, svgStroke(faint),
		c+r*math.Cos(a0), c+r*math.Sin(a0), r, r, c+r*math.Cos(a1), c+r*math.Sin(a1), svgStroke(accentRGBA))
	return renderSVG([]byte(doc), size, size)
}

// px rounds a fraction of size to a whole pixel, so straight edges stay sharp.
func px(size int, f float64) float64 {
	return math.Round(float64(size) * f)
}

// playGlyph draws a right-pointing accent triangle on a transparent square.
func playGlyph(size int) *image.RGBA {
	left, top, bottom, tip := px(size, 0.3), px(size, 0.2), px(size, 0.8), px(size, 0.8)
	doc := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d"><path fill=%s d="M%g %gL%g %gL%g %gZ"/></svg>`,
		size, size, svgColor(accentRGBA), left, top, tip, (top+bottom)/2, left, bottom)
	return renderSVG([]byte(doc), size, size)
}

// pauseGlyph draws two vertical accent bars on a transparent square.
func pauseGlyph(size int) *image.RGBA {
	top, bottom, width := px(size, 0.22), px(size, 0.78), px(size, 0.2)
	doc := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 %d %d"><g fill=%s>`+
		`<rect x="%g" y="%g" width="%g" height="%g"/><rect x="%g" y="%g" width="%g" height="%g"/></g></svg>`,
		size, size, svgColor(accentRGBA), px(size, 0.25), top, width, bottom-top, px(size, 0.55), top, width, bottom-top)
	return renderSVG([]byte(doc), size, size)
}

// monoButtons maps toolbar buttons to their drawn icon, recolored to the dialog text color.
var monoButtons = map[iup.Ihandle]string{}

// tintedName returns the image name of icon tinted for the current text color, registering it once.
func tintedName(icon string) string {
	fg := globalColor("DLGFGCOLOR")
	name := fmt.Sprintf("%s-%02x%02x%02x", icon, fg.R, fg.G, fg.B)
	if iup.GetHandle(name) == 0 && iconPaths[icon] != "" {
		scaledImage(drawIcon(icon, screenPixels(iconSize), fg)).SetHandle(name)
	}
	return name
}

// retintIcons switches the toolbar buttons to icons tinted for the current text color.
func retintIcons() {
	for b, icon := range monoButtons {
		b.SetAttribute("IMAGE", tintedName(icon))
	}
}

// loadIcons renders the embedded SVG drawings into named IUP images.
func loadIcons() {
	size := screenPixels(iconSize)

	scaledImage(renderSVG(svgTor, size, size)).SetHandle("tor")
	scaledImage(grayIcon(renderSVG(svgTor, size, size))).SetHandle("tor-off")
	scaledImage(fadeIcon(renderSVG(svgTor, size, size), 0.4)).SetHandle("tor-starting")
	scaledImage(reddenIcon(grayIcon(renderSVG(svgTor, size, size)))).SetHandle("tor-failed")

	iup.ImageFromImage(renderSVG(svgLogo, 48, 48)).SetHandle("appicon")
	scaledImage(renderSVG(svgLogo, size, size)).SetHandle("logo")
	scaledImage(grayIcon(renderSVG(svgLogo, size, size))).SetHandle("logo-gray")
	large := screenPixels(logoLargeSize)
	scaledImage(renderSVG(svgLogo, large, large)).SetHandle("logo-large")

	tmdbH := screenPixels(tmdbHeight)
	scaledImage(renderSVG(svgTMDB, 0, tmdbH)).SetHandle("tmdb")
}

// renderSVG draws the SVG data centered in a w by h image, keeping its aspect ratio.
// A zero side is taken from the drawing's aspect ratio.
func renderSVG(data []byte, w, h int) *image.RGBA {
	d, err := svg.Load(data)
	if err != nil {
		return image.NewRGBA(image.Rect(0, 0, max(w, 1), max(h, 1)))
	}
	defer d.Close()

	p, err := d.Page(0)
	if err != nil {
		return image.NewRGBA(image.Rect(0, 0, max(w, 1), max(h, 1)))
	}

	b := p.Bounds()
	bw, bh := b.X1-b.X0, b.Y1-b.Y0
	if w <= 0 {
		w = int(math.Round(float64(bw * float32(h) / bh)))
	}
	if h <= 0 {
		h = int(math.Round(float64(bh * float32(w) / bw)))
	}

	dst := image.NewRGBA(image.Rect(0, 0, w, h))
	s := min(float32(w)/bw, float32(h)/bh)
	m := raster.Matrix{A: s, D: s, E: (float32(w)-bw*s)/2 - b.X0*s, F: (float32(h)-bh*s)/2 - b.Y0*s}
	_ = p.RenderTo(dst, m, &svg.Options{Alpha: true})
	return dst
}

// grayIcon replaces the colors of img with their luminance.
func grayIcon(img *image.RGBA) *image.RGBA {
	for i := 0; i < len(img.Pix); i += 4 {
		y := uint8((299*uint32(img.Pix[i]) + 587*uint32(img.Pix[i+1]) + 114*uint32(img.Pix[i+2])) / 1000)
		img.Pix[i], img.Pix[i+1], img.Pix[i+2] = y, y, y
	}
	return img
}

// fadeIcon scales the alpha of img by f.
func fadeIcon(img *image.RGBA, f float64) *image.RGBA {
	for i := 3; i < len(img.Pix); i += 4 {
		for j := i - 3; j <= i; j++ {
			img.Pix[j] = uint8(float64(img.Pix[j]) * f)
		}
	}
	return img
}

// reddenIcon keeps the red channel of img and darkens green and blue.
func reddenIcon(img *image.RGBA) *image.RGBA {
	for i := 0; i < len(img.Pix); i += 4 {
		img.Pix[i+1] = img.Pix[i+1] / 3
		img.Pix[i+2] = img.Pix[i+2] / 3
	}
	return img
}

// screenScale returns the number of screen pixels in a logical unit.
func screenScale() float64 {
	if s, err := strconv.ParseFloat(iup.GetGlobal("SCREENSCALE"), 64); err == nil && s > 0 {
		return s
	}
	return 1
}

// screenPixels returns the number of screen pixels in size logical units.
func screenPixels(size int) int {
	return int(math.Round(float64(size) * screenScale()))
}

// scaledImage creates an IUP image from img drawn at the screen scale.
func scaledImage(img image.Image) iup.Ihandle {
	return iup.ImageFromImage(img).SetAttribute("SCALE", strconv.FormatFloat(screenScale(), 'g', -1, 64))
}
