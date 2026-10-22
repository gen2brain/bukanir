//go:build media

package main

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/png"
	"strconv"

	"github.com/gen2brain/iup-go/iup"
	xdraw "golang.org/x/image/draw"
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

// iconSize is the uniform toolbar/status icon size.
const iconSize = 24

//go:embed images/bukanir.png
var iconLogo []byte

//go:embed images/bukanir-gray.png
var iconLogoGray []byte

//go:embed images/tor.png
var iconTor []byte

//go:embed images/tor-off.png
var iconTorOff []byte

//go:embed images/tmdb.png
var iconTMDB []byte

// monoButtons maps toolbar buttons to their drawn icon, recolored to the dialog text color.
var monoButtons = map[iup.Ihandle]string{}

// tintedName returns the image name of icon tinted for the current text color, registering it once.
func tintedName(icon string) string {
	fg := globalColor("DLGFGCOLOR")
	name := fmt.Sprintf("%s-%02x%02x%02x", icon, fg.R, fg.G, fg.B)
	if iup.GetHandle(name) == 0 && iconPaths[icon] != "" {
		iup.ImageFromImage(drawIcon(icon, iconSize, fg)).SetHandle(name)
	}
	return name
}

// retintIcons switches the toolbar buttons to icons tinted for the current text color.
func retintIcons() {
	for b, icon := range monoButtons {
		b.SetAttribute("IMAGE", tintedName(icon))
	}
}

// loadIcons decodes the embedded colored PNGs into named IUP images.
func loadIcons() {
	colored := map[string][]byte{
		"tor":     iconTor,
		"tor-off": iconTorOff,
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
