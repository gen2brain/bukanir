//go:build media && !gl

package main

import "github.com/gen2brain/iup-go/iup"

func openGL() {}

func gpuEngine(string) bool {
	return false
}

func newVideo(string) iup.Ihandle {
	return iup.Video()
}
