//go:build media && gl

package main

import "github.com/gen2brain/iup-go/iup"

// openGL makes the OpenGL controls available; called before the media controls.
func openGL() {
	iup.GLCanvasOpen()
}

// gpuEngine reports whether engine is played by the OpenGL video control.
func gpuEngine(engine string) bool {
	return engine == "MPV"
}

// newVideo creates the video control that plays engine.
func newVideo(engine string) iup.Ihandle {
	if gpuEngine(engine) {
		if v := iup.GLVideo(); v != 0 {
			return v
		}
	}
	return iup.Video()
}
