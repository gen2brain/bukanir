//go:build media

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
