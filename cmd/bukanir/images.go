//go:build media

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"strconv"

	"github.com/gen2brain/iup-go/iup"
	xdraw "golang.org/x/image/draw"
	"golang.org/x/image/vector"
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
		iup.ImageFromImage(spinnerGlyph(tabIconSize, float64(i)/spinnerFrameCount)).SetHandle(name)
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
	iup.ImageFromImage(playGlyph(tabIconSize)).SetHandle("tab-play")
	iup.ImageFromImage(pauseGlyph(tabIconSize)).SetHandle("tab-pause")
	tabSpinFrames = loadingFrames
}

// fillShape rasterizes the closed polygons with anti-aliasing onto a transparent square in color c.
func fillShape(size int, c color.Color, polys ...[][2]float32) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	z := vector.NewRasterizer(size, size)
	for _, poly := range polys {
		z.MoveTo(poly[0][0], poly[0][1])
		for _, p := range poly[1:] {
			z.LineTo(p[0], p[1])
		}
		z.ClosePath()
	}
	z.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{})
	return img
}

// arcBand returns the polygon of a ring segment from angle a0 to a1 (radians) between radii r0 and r1.
func arcBand(cx, cy, r0, r1, a0, a1 float64) [][2]float32 {
	steps := int(math.Ceil(math.Abs(a1-a0) / (math.Pi / 48)))
	poly := make([][2]float32, 0, 2*steps+2)
	for i := 0; i <= steps; i++ {
		a := a0 + (a1-a0)*float64(i)/float64(steps)
		poly = append(poly, [2]float32{float32(cx + r1*math.Cos(a)), float32(cy + r1*math.Sin(a))})
	}
	for i := steps; i >= 0; i-- {
		a := a0 + (a1-a0)*float64(i)/float64(steps)
		poly = append(poly, [2]float32{float32(cx + r0*math.Cos(a)), float32(cy + r0*math.Sin(a))})
	}
	return poly
}

// spinnerGlyph draws a faint accent ring with a bright rounded arc rotated by turn (0..1).
func spinnerGlyph(size int, turn float64) *image.RGBA {
	c := float64(size) / 2
	width := float64(size) / 7
	r1 := c - 1
	r0 := r1 - width
	img := fillShape(size, color.NRGBA{accentRGBA.R, accentRGBA.G, accentRGBA.B, 70}, arcBand(c, c, r0, r1, 0, 2*math.Pi))

	a0 := 2*math.Pi*turn - math.Pi/2
	a1 := a0 + 2*math.Pi/3
	mid := (r0 + r1) / 2
	capAt := func(a float64) [][2]float32 {
		return arcBand(c+mid*math.Cos(a), c+mid*math.Sin(a), 0, width/2, 0, 2*math.Pi)
	}
	arc := fillShape(size, accentRGBA, arcBand(c, c, r0, r1, a0, a1), capAt(a0), capAt(a1))
	draw.Draw(img, img.Bounds(), arc, image.Point{}, draw.Over)
	return img
}

// px rounds a fraction of size to a whole pixel, so straight edges stay sharp.
func px(size int, f float64) float32 {
	return float32(math.Round(float64(size) * f))
}

// playGlyph draws a right-pointing accent triangle on a transparent square.
func playGlyph(size int) *image.RGBA {
	left, top, bottom, tip := px(size, 0.3), px(size, 0.2), px(size, 0.8), px(size, 0.8)
	return fillShape(size, accentRGBA, [][2]float32{{left, top}, {tip, (top + bottom) / 2}, {left, bottom}})
}

// pauseGlyph draws two vertical accent bars on a transparent square.
func pauseGlyph(size int) *image.RGBA {
	top, bottom, width := px(size, 0.22), px(size, 0.78), px(size, 0.2)
	bar := func(x0 float32) [][2]float32 {
		return [][2]float32{{x0, top}, {x0 + width, top}, {x0 + width, bottom}, {x0, bottom}}
	}
	return fillShape(size, accentRGBA, bar(px(size, 0.25)), bar(px(size, 0.55)))
}

// iconSize is the uniform toolbar/status icon size.
const iconSize = 22

//go:embed images/refresh.png
var iconRefresh []byte

//go:embed images/settings.png
var iconSettings []byte

//go:embed images/bukanir.png
var iconLogo []byte

//go:embed images/bukanir-gray.png
var iconLogoGray []byte

//go:embed images/tor.png
var iconTor []byte

//go:embed images/tor-off.png
var iconTorOff []byte

//go:embed images/log.png
var iconLog []byte

//go:embed images/star.png
var iconStar []byte

//go:embed images/star-outline.png
var iconStarOutline []byte

//go:embed images/tmdb.png
var iconTMDB []byte

//go:embed images/player-play.png
var iconPlayerPlay []byte

//go:embed images/player-pause.png
var iconPlayerPause []byte

//go:embed images/player-fullscreen.png
var iconPlayerFullscreen []byte

//go:embed images/player-subtitles.png
var iconPlayerSubtitles []byte

//go:embed images/player-volume.png
var iconPlayerVolume []byte

//go:embed images/player-volume-off.png
var iconPlayerVolumeOff []byte

// monoIcons are recolored to the dialog text color; monoButtons maps toolbar buttons to their icon.
var (
	monoIcons = map[string]image.Image{
		"refresh":           nil,
		"settings":          nil,
		"log":               nil,
		"star":              nil,
		"star-outline":      nil,
		"player-play":       nil,
		"player-pause":      nil,
		"player-fullscreen": nil,
		"player-subtitles":  nil,
		"player-volume":     nil,
		"player-volume-off": nil,
	}
	monoButtons = map[iup.Ihandle]string{}
)

// tintedName returns the image name of icon tinted for the current text color, registering it once.
func tintedName(icon string) string {
	fg := globalColor("DLGFGCOLOR")
	name := fmt.Sprintf("%s-%02x%02x%02x", icon, fg.R, fg.G, fg.B)
	if iup.GetHandle(name) == 0 && monoIcons[icon] != nil {
		iup.ImageFromImage(tintIcon(monoIcons[icon], iconSize, fg)).SetHandle(name)
	}
	return name
}

// retintIcons switches the toolbar buttons to icons tinted for the current text color.
func retintIcons() {
	for b, icon := range monoButtons {
		b.SetAttribute("IMAGE", tintedName(icon))
	}
}

// loadIcons decodes the embedded PNGs into named IUP images.
func loadIcons() {
	mono := map[string][]byte{
		"refresh":           iconRefresh,
		"settings":          iconSettings,
		"log":               iconLog,
		"star":              iconStar,
		"star-outline":      iconStarOutline,
		"player-play":       iconPlayerPlay,
		"player-pause":      iconPlayerPause,
		"player-fullscreen": iconPlayerFullscreen,
		"player-subtitles":  iconPlayerSubtitles,
		"player-volume":     iconPlayerVolume,
		"player-volume-off": iconPlayerVolumeOff,
	}
	colored := map[string][]byte{
		"tor":     iconTor,
		"tor-off": iconTorOff,
	}
	for name, data := range mono {
		if img, err := png.Decode(bytes.NewReader(data)); err == nil {
			monoIcons[name] = img
		}
	}
	for name, data := range colored {
		if img, err := png.Decode(bytes.NewReader(data)); err == nil {
			iup.ImageFromImage(resizeIcon(img, iconSize)).SetHandle(name)
		}
	}
	if img, err := png.Decode(bytes.NewReader(iconTor)); err == nil {
		iup.ImageFromImage(fadeIcon(resizeIcon(img, iconSize), 0.4)).SetHandle("tor-starting")
	}
	if img, err := png.Decode(bytes.NewReader(iconTorOff)); err == nil {
		iup.ImageFromImage(reddenIcon(resizeIcon(img, iconSize))).SetHandle("tor-failed")
	}

	if img, err := png.Decode(bytes.NewReader(iconLogo)); err == nil {
		iup.ImageFromImage(resizeIcon(img, 48)).SetHandle("appicon")
		iup.ImageFromImage(resizeIcon(img, iconSize)).SetHandle("logo")
		iup.ImageFromImage(img).SetHandle("logo-large")
	}
	if img, err := png.Decode(bytes.NewReader(iconLogoGray)); err == nil {
		iup.ImageFromImage(resizeIcon(img, iconSize)).SetHandle("logo-gray")
	}
	if img, err := png.Decode(bytes.NewReader(iconTMDB)); err == nil {
		iup.ImageFromImage(img).SetHandle("tmdb")
	}
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

// resizeIcon scales src to a square of the given size.
func resizeIcon(src image.Image, size int) *image.RGBA {
	dst := image.NewRGBA(image.Rect(0, 0, size, size))
	xdraw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), xdraw.Over, nil)
	return dst
}

// tintIcon resizes src and recolors it to c, keeping the alpha channel.
func tintIcon(src image.Image, size int, c color.RGBA) image.Image {
	dst := resizeIcon(src, size)
	for i := 0; i < len(dst.Pix); i += 4 {
		dst.Pix[i], dst.Pix[i+1], dst.Pix[i+2] = c.R, c.G, c.B
	}
	return dst
}
