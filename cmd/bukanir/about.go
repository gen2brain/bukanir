package main

import (
	"github.com/gen2brain/iup-go/iup"
)

const (
	projectURL = "https://github.com/gen2brain/bukanir"
	licenseURL = "https://www.gnu.org/licenses/gpl-3.0.txt"
	tmdbURL    = "https://www.themoviedb.org"
)

// showAbout opens the modal About dialog with a Help button for the shortcuts.
func showAbout() {
	logo := iup.Label("").SetAttribute("IMAGE", "logo-large")
	text := iup.Vbox(
		iup.Label(appName+" "+appVersion).SetAttributes("FONTSTYLE=Bold, FONTSIZE=14"),
		link(projectURL, projectURL),
		iup.Label("Author: Milan Nikolić (gen2brain)"),
		iup.Label("This program is released under the terms of the"),
		link(licenseURL, "GNU General Public License version 3"),
	).SetAttributes("GAP=6")

	tmdb := iup.Hbox(
		iup.Label("").SetAttribute("IMAGE", "tmdb"),
		iup.Vbox(
			iup.Label("This product uses the TMDB API but is not endorsed or certified by TMDB."),
			link(tmdbURL, tmdbURL),
		).SetAttributes("GAP=2"),
	).SetAttributes("GAP=12, ALIGNMENT=ACENTER")

	help := iup.Button("Help").SetAttributes("PADDING=14x4")
	closeBtn := iup.Button("Close").SetAttributes("PADDING=14x4")
	body := iup.Vbox(
		iup.Hbox(logo, text).SetAttributes("GAP=16, ALIGNMENT=ACENTER"),
		iup.Label("").SetAttribute("SEPARATOR", "HORIZONTAL"),
		tmdb,
		iup.Hbox(iup.Fill(), help, closeBtn).SetAttributes("GAP=8"),
	).SetAttributes("NMARGIN=14x14, GAP=14")
	dlg := iup.Dialog(body).SetAttributes(map[string]string{
		"TITLE": "About", "DIALOGFRAME": "YES", "MINBOX": "NO", "MAXBOX": "NO",
	})
	iup.SetAttributeHandle(dlg, "PARENTDIALOG", iup.GetHandle("dlg"))
	iup.SetAttributeHandle(dlg, "DEFAULTESC", closeBtn)

	closeBtn.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int { return iup.CLOSE }))
	help.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		showShortcuts(dlg)
		return iup.DEFAULT
	}))

	iup.Popup(dlg, iup.CENTER, iup.CENTER)
	iup.Destroy(dlg)
}

// showShortcuts opens the modal keyboard shortcuts dialog.
func showShortcuts(parent iup.Ihandle) {
	app := frame("Bukanir",
		keyRows(
			"Enter", "Open the selected title, watch on a details tab",
			"Arrows", "Move the selection",
			"Alt+W", "Watch",
			"Alt+T", "Trailer",
			"Ctrl+Tab", "Next tab",
			"Ctrl+Shift+Tab", "Previous tab",
			"Ctrl+W", "Close the tab",
			"Ctrl+D", "Add or remove the title from favorites on a details tab",
		),
	)
	player := frame("Player (mpv)",
		keyRows(
			"Space, p", "Pause / resume",
			"f", "Toggle fullscreen",
			"m", "Mute / unmute audio",
			"A", "Cycle aspect ratio",
			"q", "Quit",
			"v", "Show / hide subtitles",
			"j / J", "Next / previous subtitle",
			"r / t", "Move subtitles up / down",
			"z / x", "Decrease / increase subtitle delay",
			"Ctrl++ / Ctrl+-", "Increase / decrease audio delay",
			"Right / Left", "Seek 5 seconds",
			"Up / Down", "Seek 60 seconds",
			"1 / 2", "Decrease / increase contrast",
			"3 / 4", "Decrease / increase brightness",
			"5 / 6", "Decrease / increase gamma",
			"7 / 8", "Decrease / increase saturation",
			"9 / 0", "Decrease / increase volume",
		),
	)

	closeBtn := iup.Button("Close").SetAttributes("PADDING=14x4")
	closeBtn.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int { return iup.CLOSE }))
	body := iup.Vbox(
		iup.Hbox(app, player).SetAttributes("GAP=10, ALIGNMENT=ATOP"),
		iup.Hbox(iup.Fill(), closeBtn),
	).SetAttributes("NMARGIN=10x10, GAP=10")
	dlg := iup.Dialog(body).SetAttributes(map[string]string{
		"TITLE": "Shortcuts", "DIALOGFRAME": "YES", "MINBOX": "NO", "MAXBOX": "NO",
	})
	iup.SetAttributeHandle(dlg, "PARENTDIALOG", parent)
	iup.SetAttributeHandle(dlg, "DEFAULTESC", closeBtn)
	iup.Popup(dlg, iup.CENTER, iup.CENTER)
	iup.Destroy(dlg)
}

func link(url, title string) iup.Ihandle {
	return iup.Link(url, title).SetAttribute("FGCOLOR", accentColor)
}

func keyRows(pairs ...string) iup.Ihandle {
	cells := make([]iup.Ihandle, 0, len(pairs))
	for i := 0; i+1 < len(pairs); i += 2 {
		cells = append(cells, iup.Label(pairs[i]).SetAttributes("FONTSTYLE=Bold"), iup.Label(pairs[i+1]))
	}
	return iup.GridBox(cells...).SetAttributes(
		"ORIENTATION=HORIZONTAL, NUMDIV=2, GAPLIN=4, GAPCOL=14, ALIGNMENTLIN=ACENTER, SIZELIN=-1, SIZECOL=-1",
	)
}
