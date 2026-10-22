//go:build media

package main

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"regexp"
	"strconv"

	"golang.org/x/image/vector"
)

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

var pathToken = regexp.MustCompile(`[A-Za-z]|-?(?:\d*\.\d+|\d+\.?)(?:[eE][-+]?\d+)?`)

// tracePath adds the SVG path d to z, scaled from the icon grid by s.
func tracePath(z *vector.Rasterizer, d string, s float32) {
	tokens := pathToken.FindAllString(d, -1)
	var cmd, last byte
	var x, y, startX, startY, ctrlX, ctrlY float32
	num := func(i *int) float32 {
		v, _ := strconv.ParseFloat(tokens[*i], 32)
		*i++
		return float32(v)
	}
	reflect := func(curves byte) (float32, float32) {
		if last == curves || last == curves+1 {
			return 2*x - ctrlX, 2*y - ctrlY
		}
		return x, y
	}
	for i := 0; i < len(tokens); {
		if c := tokens[i][0]; (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z') {
			cmd = c
			i++
			if cmd == 'Z' || cmd == 'z' {
				z.ClosePath()
				x, y = startX, startY
				last = 'z'
				continue
			}
		}
		ox, oy := float32(0), float32(0)
		if cmd >= 'a' {
			ox, oy = x, y
		}
		op := cmd | 0x20
		switch op {
		case 'm':
			x, y = ox+num(&i), oy+num(&i)
			startX, startY = x, y
			z.MoveTo(x*s, y*s)
			cmd--
		case 'l':
			x, y = ox+num(&i), oy+num(&i)
			z.LineTo(x*s, y*s)
		case 'h':
			x = ox + num(&i)
			z.LineTo(x*s, y*s)
		case 'v':
			y = oy + num(&i)
			z.LineTo(x*s, y*s)
		case 'c', 's':
			var x1, y1 float32
			if op == 'c' {
				x1, y1 = ox+num(&i), oy+num(&i)
			} else {
				x1, y1 = reflect('c')
			}
			ctrlX, ctrlY = ox+num(&i), oy+num(&i)
			x, y = ox+num(&i), oy+num(&i)
			z.CubeTo(x1*s, y1*s, ctrlX*s, ctrlY*s, x*s, y*s)
			op = 'c'
		case 'q', 't':
			if op == 'q' {
				ctrlX, ctrlY = ox+num(&i), oy+num(&i)
			} else {
				ctrlX, ctrlY = reflect('q')
			}
			x, y = ox+num(&i), oy+num(&i)
			z.QuadTo(ctrlX*s, ctrlY*s, x*s, y*s)
			op = 'q'
		default:
			i++
		}
		last = op
	}
}

// drawIcon rasterizes the named icon at size pixels in color c.
func drawIcon(name string, size int, c color.Color) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	z := vector.NewRasterizer(size, size)
	tracePath(z, iconPaths[name], float32(size)/iconGrid)
	z.Draw(img, img.Bounds(), image.NewUniform(c), image.Point{})
	return img
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
