package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/gif"
	"image/png"

	"github.com/gen2brain/iup-go/iup"
	xdraw "golang.org/x/image/draw"
)

//go:embed images/loading.gif
var loadingGIF []byte

// loadingFrames holds the spinner frame image names shared by loading boxes and tab indicators.
var loadingFrames []string

// loadLoadingAnimation registers the spinner GIF frames and the "loading-anim" element once.
func loadLoadingAnimation() {
	g, err := gif.DecodeAll(bytes.NewReader(loadingGIF))
	if err != nil || len(g.Image) == 0 {
		return
	}
	anim := iup.User().SetHandle("loading-anim")
	anim.SetAttribute("FRAMETIME", "90")
	canvas := image.NewRGBA(image.Rect(0, 0, g.Config.Width, g.Config.Height))
	for i, frame := range g.Image {
		draw.Draw(canvas, frame.Bounds(), frame, frame.Bounds().Min, draw.Over)
		snap := image.NewRGBA(canvas.Bounds())
		copy(snap.Pix, canvas.Pix)
		name := fmt.Sprintf("load-%d", i)
		iup.ImageFromImage(snap).SetHandle(name)
		iup.Append(anim, iup.GetHandle(name))
		loadingFrames = append(loadingFrames, name)
		if i < len(g.Disposal) && g.Disposal[i] == gif.DisposalBackground {
			draw.Draw(canvas, frame.Bounds(), image.Transparent, image.Point{}, draw.Src)
		}
	}
}

// loadingBox returns a centered animated spinner with a caption.
func loadingBox(text string) iup.Ihandle {
	if len(loadingFrames) == 0 {
		return placeholder(text)
	}
	spinner := iup.AnimatedLabel(iup.GetHandle("loading-anim"))
	spinner.SetAttribute("START", "YES")

	caption := iup.Label(text).SetAttribute("FGCOLOR", subtleColor)
	row := iup.Hbox(iup.Fill(), spinner, caption, iup.Fill()).SetAttributes("ALIGNMENT=ACENTER, GAP=8")
	return iup.Vbox(iup.Fill(), row, iup.Fill()).SetAttributes("EXPAND=YES")
}

const tabIconSize = 20

// tabSpinFrames holds the registered image names for the tab loading spinner.
var tabSpinFrames []string

// registerTabIndicators registers the tab-indicator images (blank, spinner, play, pause).
func registerTabIndicators() {
	iup.ImageFromImage(image.NewRGBA(image.Rect(0, 0, tabIconSize, tabIconSize))).SetHandle("tab-blank")
	iup.ImageFromImage(playGlyph(tabIconSize)).SetHandle("tab-play")
	iup.ImageFromImage(pauseGlyph(tabIconSize)).SetHandle("tab-pause")
	tabSpinFrames = loadingFrames
}

// playGlyph draws a right-pointing accent triangle on a transparent square.
func playGlyph(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	c := accentRGBA
	left, right, mid := size/4, size-size/4, size/2
	for x := left; x < right; x++ {
		half := (right - x) * (mid - size/6) / (right - left)
		for y := mid - half; y <= mid+half; y++ {
			img.SetRGBA(x, y, c)
		}
	}
	return img
}

// pauseGlyph draws two vertical accent bars on a transparent square.
func pauseGlyph(size int) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	c := accentRGBA
	top, bot, barW, gap := size/4, size-size/4, size/6, size/8
	bar := func(x0 int) {
		for x := x0; x < x0+barW; x++ {
			for y := top; y < bot; y++ {
				img.SetRGBA(x, y, c)
			}
		}
	}
	bar(size/2 - gap - barW)
	bar(size/2 + gap)
	return img
}

// iconSize is the uniform toolbar/status icon size.
const iconSize = 22

//go:embed images/search.png
var iconSearch []byte

//go:embed images/refresh.png
var iconRefresh []byte

//go:embed images/settings.png
var iconSettings []byte

//go:embed images/bukanir.png
var iconAbout []byte

//go:embed images/tor.png
var iconTor []byte

//go:embed images/tor-off.png
var iconTorOff []byte

//go:embed images/log.png
var iconLog []byte

// loadIcons decodes the embedded PNGs into named IUP images.
func loadIcons() {
	mono := map[string][]byte{
		"search":   iconSearch,
		"refresh":  iconRefresh,
		"settings": iconSettings,
		"log":      iconLog,
	}
	colored := map[string][]byte{
		"tor":     iconTor,
		"tor-off": iconTorOff,
	}
	fg := globalColor("DLGFGCOLOR")
	for name, data := range mono {
		if img, err := png.Decode(bytes.NewReader(data)); err == nil {
			iup.ImageFromImage(tintIcon(img, iconSize, fg)).SetHandle(name)
		}
	}
	for name, data := range colored {
		if img, err := png.Decode(bytes.NewReader(data)); err == nil {
			iup.ImageFromImage(resizeIcon(img, iconSize)).SetHandle(name)
		}
	}

	if img, err := png.Decode(bytes.NewReader(iconAbout)); err == nil {
		iup.ImageFromImage(resizeIcon(img, 48)).SetHandle("appicon")
	}
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
