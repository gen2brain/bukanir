package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gen2brain/bukanir"
	"github.com/gen2brain/iup-go/iup"
)

// Settings holds user-configurable preferences, persisted as JSON.
type Settings struct {
	CacheDays  int    `json:"cacheDays"`
	Appearance string `json:"appearance"`

	Fullscreen      bool   `json:"fullscreen"`
	StopScreensaver bool   `json:"stopScreensaver"`
	VolumeMax       int    `json:"volumeMax"`
	MPVConfig       bool   `json:"mpvConfig"`
	MPVOptions      string `json:"mpvOptions"`

	SubtitlesEnabled bool    `json:"subtitlesEnabled"`
	SubLanguage      string  `json:"subLanguage"`
	SubScale         float64 `json:"subScale"`
	SubColor         string  `json:"subColor"`

	UseTor       bool   `json:"useTor"`
	TPBHost      string `json:"tpbHost"`
	YTSHost      string `json:"ytsHost"`
	EZTVHost     string `json:"eztvHost"`
	DownloadRate int    `json:"downloadRate"`
	UploadRate   int    `json:"uploadRate"`
	Port         int    `json:"port"`
	KeepFiles    bool   `json:"keepFiles"`
	DownloadDir  string `json:"downloadDir"`
}

// tpbOnion is ThePirateBay's onion address, offered as a Tor-only TPB host.
const tpbOnion = "piratebayo3klnzokct3wt5yyxb2vpebbuyjl7m623iaxmqhsd52coid.onion"

// eztvMirrors are EZTV hosts that serve the same API.
var eztvMirrors = []string{bukanir.DefaultEZTVHost, "eztv1.xyz", "eztv.wf", "eztv.tf", "eztv.yt"}

// defaultSettings returns the settings used on first run.
func defaultSettings() *Settings {
	return &Settings{
		CacheDays:        1,
		Appearance:       "dark",
		Fullscreen:       false,
		StopScreensaver:  true,
		VolumeMax:        130,
		SubtitlesEnabled: true,
		SubLanguage:      "en",
		SubScale:         1.0,
		SubColor:         "#FFFFFF",
		UseTor:           false,
		TPBHost:          bukanir.DefaultTPBHost,
		YTSHost:          bukanir.DefaultYTSHost,
		EZTVHost:         bukanir.DefaultEZTVHost,
		DownloadRate:     0,
		UploadRate:       0,
		Port:             0,
		KeepFiles:        false,
	}
}

// App folders under IUP's per-user directories, resolved once on the main thread by initDirs.
var cacheDir, configDir, tmpDir string

func initDirs() {
	cacheDir = filepath.Join(iup.GetGlobal("CACHEDIR"), "bukanir")
	configDir = filepath.Join(iup.GetGlobal("CONFIGDIR"), "bukanir")
	tmpDir = filepath.Join(iup.GetGlobal("TMPDIR"), "bukanir")
}

// settingsPath is the on-disk location of the settings file.
func settingsPath() string {
	return filepath.Join(configDir, "settings.json")
}

// loadSettings reads settings from disk, falling back to defaults.
func loadSettings() *Settings {
	s := defaultSettings()
	data, err := os.ReadFile(settingsPath())
	if err != nil {
		return s
	}
	json.Unmarshal(data, s)
	return s
}

// save writes the settings to disk, creating the config directory if needed.
func (s *Settings) save() error {
	path := settingsPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// toConfig builds the library Config from the settings.
func (s *Settings) toConfig() bukanir.Config {
	cfg := bukanir.DefaultConfig()
	if s.CacheDays > 0 {
		cfg.CacheTTL = time.Duration(s.CacheDays) * 24 * time.Hour
	} else {
		cfg.CacheTTL = 0
	}
	cfg.CacheDir = cacheDir
	cfg.Tor = s.UseTor
	cfg.TorLog = logs
	cfg.TPBHost = s.TPBHost
	cfg.YTSHost = s.YTSHost
	cfg.EZTVHost = s.EZTVHost
	return cfg
}

// toTorrentConfig builds the streaming TorrentConfig from the settings.
func (s *Settings) toTorrentConfig(dialer bukanir.PeerDialer) bukanir.TorrentConfig {
	return bukanir.TorrentConfig{
		DownloadDir:  s.DownloadDir,
		ListenPort:   s.Port,
		DownloadRate: s.DownloadRate,
		UploadRate:   s.UploadRate,
		KeepFiles:    s.KeepFiles,
		Dialer:       dialer,
	}
}

// appearanceChoices are the APPEARANCE values offered in the dialog, in list order.
var appearanceChoices = []struct{ Value, Name string }{
	{"dark", "Dark"}, {"light", "Light"}, {"system", "System"},
}

// appearance returns the APPEARANCE global value for the setting, dark when unset or unknown.
func (s *Settings) appearance() string {
	for _, a := range appearanceChoices {
		if a.Value == s.Appearance {
			return strings.ToUpper(a.Value)
		}
	}
	return "DARK"
}

// subtitleLanguage returns the configured subtitle language, or "en" if unset.
func (s *Settings) subtitleLanguage() string {
	if s.SubLanguage == "" {
		return "en"
	}
	return s.SubLanguage
}

// subtitleLanguages lists the languages offered in the settings dialog.
var subtitleLanguages = []struct{ Code, Name string }{
	{"ar", "Arabic"}, {"bg", "Bulgarian"}, {"cs", "Czech"}, {"da", "Danish"},
	{"de", "German"}, {"el", "Greek"}, {"en", "English"}, {"es", "Spanish"},
	{"fa", "Persian"}, {"fi", "Finnish"}, {"fr", "French"}, {"he", "Hebrew"},
	{"hi", "Hindi"}, {"hr", "Croatian"}, {"hu", "Hungarian"}, {"id", "Indonesian"},
	{"it", "Italian"}, {"ja", "Japanese"}, {"ko", "Korean"}, {"nl", "Dutch"},
	{"no", "Norwegian"}, {"pl", "Polish"}, {"pt", "Portuguese"}, {"ro", "Romanian"},
	{"ru", "Russian"}, {"sk", "Slovak"}, {"sl", "Slovenian"}, {"sr", "Serbian"},
	{"sv", "Swedish"}, {"th", "Thai"}, {"tr", "Turkish"}, {"uk", "Ukrainian"},
	{"vi", "Vietnamese"}, {"zh", "Chinese"},
}

// cacheDayChoices and rateChoices are the preset values offered in the dialog.
var (
	cacheDayChoices = []int{1, 3, 7, 30, 90, 180}
	rateChoices     = []int{0, 50, 100, 500, 1000, 2000, 5000, 10000}
)

// onSettings opens the modal settings dialog, applying changes on Save.
func onSettings() {
	s := settings
	formLabels = nil

	cacheDays := choiceList(cacheDayChoices, s.CacheDays, nil)
	appearance := iup.List().SetAttributes("DROPDOWN=YES")
	for i, a := range appearanceChoices {
		appearance.SetAttribute(strconv.Itoa(i+1), a.Name)
		if strings.EqualFold(a.Value, s.appearance()) {
			appearance.SetAttribute("VALUE", strconv.Itoa(i+1))
		}
	}
	appearance.SetCallback("ACTION", iup.ListActionFunc(func(_ iup.Ihandle, _ string, item, state int) int {
		if state == 1 {
			iup.SetGlobal("APPEARANCE", strings.ToUpper(appearanceChoices[item-1].Value))
		}
		return iup.DEFAULT
	}))

	fullscreen := toggle(s.Fullscreen)
	stopSaver := toggle(s.StopScreensaver)
	volume := iup.Val("HORIZONTAL").SetAttributes(map[string]string{
		"MIN": "100", "MAX": "200", "VALUE": strconv.Itoa(s.VolumeMax), "EXPAND": "HORIZONTAL",
	})
	volumeLabel := iup.Label(fmt.Sprintf("%d%%", s.VolumeMax)).SetAttributes("RASTERSIZE=44x")
	volume.SetCallback("VALUECHANGED_CB", iup.ValueChangedFunc(func(ih iup.Ihandle) int {
		v, _ := strconv.ParseFloat(ih.GetAttribute("VALUE"), 64)
		volumeLabel.SetAttribute("TITLE", fmt.Sprintf("%d%%", int(v)))
		return iup.DEFAULT
	}))

	mpvConfig := toggle(s.MPVConfig)
	mpvOptions := iup.Text().SetAttributes("VISIBLECOLUMNS=20, CUEBANNER=\"hwdec=auto vo=gpu\"")
	mpvOptions.SetAttribute("VALUE", s.MPVOptions)

	subsEnabled := toggle(s.SubtitlesEnabled)
	subLang := languageList(s.subtitleLanguage())
	subScale := scaleSpin(s.SubScale)
	subColor := colorButton(s.SubColor)

	useTor := toggle(s.UseTor)
	tpbHost := editableChoice(s.TPBHost, []string{bukanir.DefaultTPBHost, tpbOnion})
	ytsHost := editableChoice(s.YTSHost, []string{bukanir.DefaultYTSHost})
	eztvHost := editableChoice(s.EZTVHost, eztvMirrors)
	dlRate := choiceList(rateChoices, s.DownloadRate, rateLabel)
	ulRate := choiceList(rateChoices, s.UploadRate, rateLabel)
	port := iup.Text().SetAttributes(map[string]string{
		"SPIN": "YES", "SPINMIN": "0", "SPINMAX": "65535", "SPININC": "1",
		"VALUE": strconv.Itoa(s.Port), "VISIBLECOLUMNS": "8",
	})
	keepFiles := toggle(s.KeepFiles)
	dlDir := iup.Text().SetAttributes("VISIBLECOLUMNS=24")
	dlDir.SetAttribute("VALUE", s.DownloadDir)
	browse := iup.Button("Browse...").SetAttributes("PADDING=8x2")
	browse.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		if dir, ok := pickDirectory(dlDir.GetAttribute("VALUE")); ok {
			dlDir.SetAttribute("VALUE", dir)
		}
		return iup.DEFAULT
	}))
	dlRow := iup.Hbox(dlDir, browse).SetAttributes("ALIGNMENT=ACENTER, GAP=6")
	setActive(s.KeepFiles, dlDir, browse)
	keepFiles.SetCallback("ACTION", iup.ToggleActionFunc(func(_ iup.Ihandle, state int) int {
		setActive(state == 1, dlDir, browse)
		return iup.DEFAULT
	}))

	pageList := []settingsPage{
		{"General", "Appearance and cache.",
			page(
				frame("Interface", formRows("Appearance", appearance)),
				frame("Cache", formRows("Days to keep cache", cacheDays)),
			)},
		{"Player", "Playback window, volume and mpv.",
			page(
				frame("Playback", formRows(
					"Fullscreen", fullscreen,
					"Stop screensaver", stopSaver,
					"Maximum volume", iup.Hbox(volume, volumeLabel).SetAttributes("ALIGNMENT=ACENTER, GAP=6"),
				)),
				frame("mpv", formRows(
					"Load configuration files", mpvConfig,
					"Extra options", mpvOptions,
				)),
			)},
		{"Subtitles", "Subtitle download and style.",
			page(
				frame("Download", formRows(
					"Download subtitles", subsEnabled,
					"Language", subLang,
				)),
				frame("Style", formRows(
					"Font scale", subScale,
					"Text color", subColor,
				)),
			)},
		{"Network", "Tor and torrent sites.",
			page(
				frame("Tor", formRows("Route peers through Tor", useTor)),
				frame("Sources", formRows(
					"TPB host", tpbHost,
					"YTS host", ytsHost,
					"EZTV host", eztvHost,
				)),
			)},
		{"Torrents", "Transfer limits, peer port and files.",
			page(
				frame("Limits", formRows(
					"Max download rate", dlRate,
					"Max upload rate", ulRate,
					"Peer port (0 = auto)", port,
				)),
				frame("Files", formRows(
					"Keep downloaded files", keepFiles,
					"Download folder", dlRow,
				)),
			)},
	}

	sidebar := iup.List().SetAttributes(map[string]string{
		"SPACING": "4", "VISIBLECOLUMNS": "10", "EXPAND": "VERTICAL",
		"VISIBLELINES": strconv.Itoa(len(pageList)),
	})
	title := iup.Label("").SetAttributes("FONTSTYLE=Bold, FONTSIZE=16, EXPAND=HORIZONTAL")
	summary := iup.Label("").SetAttributes(`FGCOLOR="128 128 128", EXPAND=HORIZONTAL`)
	bodies := make([]iup.Ihandle, len(pageList))
	for i, pg := range pageList {
		sidebar.SetAttribute(strconv.Itoa(i+1), pg.name)
		bodies[i] = pg.body
	}
	pages := iup.Zbox(bodies...).SetAttributes("ALIGNMENT=NW, EXPAND=YES")
	showPage := func(i int) {
		settingsPageIndex = i
		pages.SetAttribute("VALUEPOS", strconv.Itoa(i))
		title.SetAttribute("TITLE", pageList[i].name)
		summary.SetAttribute("TITLE", pageList[i].summary)
	}
	sidebar.SetCallback("ACTION", iup.ListActionFunc(func(_ iup.Ihandle, _ string, item, state int) int {
		if state == 1 {
			showPage(item - 1)
		}
		return iup.DEFAULT
	}))
	if settingsPageIndex >= len(pageList) {
		settingsPageIndex = 0
	}
	sidebar.SetAttribute("VALUE", strconv.Itoa(settingsPageIndex+1))
	showPage(settingsPageIndex)

	saveBtn := iup.Button("Save").SetAttributes("PADDING=14x4")
	cancelBtn := iup.Button("Cancel").SetAttributes("PADDING=14x4")
	buttons := iup.Hbox(iup.Fill(), cancelBtn, saveBtn).SetAttributes("GAP=8")

	// The normalizer equalizes every label's width so control columns line up across pages.
	norm := iup.Normalizer(formLabels...)
	content := iup.Vbox(title, summary, pages).SetAttributes("GAP=8, NMARGIN=12x0, EXPAND=YES")
	columns := iup.Hbox(sidebar, iup.Label("").SetAttribute("SEPARATOR", "VERTICAL"), content).SetAttributes("GAP=8")
	body := iup.Vbox(norm, columns, buttons).SetAttributes("NMARGIN=10x10, GAP=10")
	dlg := iup.Dialog(body).SetAttributes(map[string]string{
		"TITLE": "Settings", "DIALOGFRAME": "YES", "MINBOX": "NO", "MAXBOX": "NO",
	})
	iup.SetAttributeHandle(dlg, "PARENTDIALOG", iup.GetHandle("dlg"))

	saved := false
	cancelBtn.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int { return iup.CLOSE }))
	saveBtn.SetCallback("ACTION", iup.ActionFunc(func(iup.Ihandle) int {
		next := *s
		next.CacheDays = cacheDayChoices[listIndex(cacheDays)]
		next.Appearance = appearanceChoices[listIndex(appearance)].Value
		next.Fullscreen = isOn(fullscreen)
		next.StopScreensaver = isOn(stopSaver)
		next.VolumeMax = atoiDefault(volume.GetAttribute("VALUE"), s.VolumeMax)
		next.MPVConfig = isOn(mpvConfig)
		next.MPVOptions = strings.TrimSpace(mpvOptions.GetAttribute("VALUE"))
		next.SubtitlesEnabled = isOn(subsEnabled)
		next.SubLanguage = subtitleLanguages[listIndex(subLang)].Code
		next.SubScale = parseFloatDefault(subScale.GetAttribute("VALUE"), s.SubScale)
		next.SubColor = subColor.GetAttribute("BUKANIRCOLOR")
		next.UseTor = isOn(useTor)
		next.DownloadRate = rateChoices[listIndex(dlRate)]
		next.UploadRate = rateChoices[listIndex(ulRate)]
		next.Port = atoiDefault(port.GetAttribute("VALUE"), s.Port)
		next.KeepFiles = isOn(keepFiles)
		next.DownloadDir = dlDir.GetAttribute("VALUE")
		next.TPBHost = strings.TrimSpace(choiceText(tpbHost))
		next.YTSHost = strings.TrimSpace(choiceText(ytsHost))
		next.EZTVHost = strings.TrimSpace(choiceText(eztvHost))

		applySettings(&next)
		saved = true
		return iup.CLOSE
	}))

	iup.Popup(dlg, iup.CENTER, iup.CENTER)
	if !saved && iup.GetGlobal("APPEARANCE") != s.appearance() {
		iup.SetGlobal("APPEARANCE", s.appearance())
	}
	iup.Destroy(dlg)
}

// applySettings persists the new settings and rebuilds the library client.
func applySettings(next *Settings) {
	*settings = *next
	iup.SetGlobal("APPEARANCE", settings.appearance())
	if err := settings.save(); err != nil {
		setStatus("Could not save settings: " + err.Error())
	}

	if nc, err := bukanir.New(context.Background(), settings.toConfig()); err == nil {
		old := client
		client = nc
		if old != nil {
			old.Close()
		}
		client.StartTor()
	} else {
		setStatus("Settings applied, client reload failed: " + err.Error())
	}

	watchTor(client)
	setStatus("Settings saved")
}

// watchTor shows c's Tor state now and again when its Tor client finishes starting.
func watchTor(c *bukanir.Client) {
	updateTorStatus()
	tor := iup.GetHandle("torStatus")
	go func() {
		<-c.TorDone()
		iup.PostMessage(tor, "", 0, c)
	}()
}

// onTorDone updates the Tor icon for the current client and reports a failed start.
func onTorDone(_ iup.Ihandle, _ string, _ int, p any) int {
	if c, ok := p.(*bukanir.Client); !ok || c != client {
		return iup.DEFAULT
	}
	updateTorStatus()
	if st, err := client.TorStatus(); st == bukanir.TorFailed {
		setStatus("Tor error: " + err.Error())
	}
	return iup.DEFAULT
}

func updateTorStatus() {
	icon, tip := "tor-off", "Tor is off"
	switch st, err := client.TorStatus(); st {
	case bukanir.TorStarting:
		icon, tip = "tor-starting", "Tor is starting"
	case bukanir.TorReady:
		icon, tip = "tor", "Tor is connected"
	case bukanir.TorFailed:
		icon, tip = "tor-failed", "Tor failed: "+err.Error()
	}
	tor := iup.GetHandle("torStatus")
	tor.SetAttribute("IMAGE", icon)
	tor.SetAttribute("TIP", plainText(tip))
}

// settingsPage is one sidebar entry of the settings dialog.
type settingsPage struct {
	name, summary string
	body          iup.Ihandle
}

// settingsPageIndex remembers the last shown settings page across openings.
var settingsPageIndex int

func page(frames ...iup.Ihandle) iup.Ihandle {
	return iup.Vbox(frames...).SetAttributes("GAP=10")
}

func frame(title string, children ...iup.Ihandle) iup.Ihandle {
	box := iup.Vbox(children...).SetAttributes("GAP=6")
	return iup.Frame(iup.Hbox(box, iup.Fill()).SetAttributes("NMARGIN=10x8")).SetAttribute("TITLE", title)
}

// formLabels collects every form label so a normalizer can align them.
var formLabels []iup.Ihandle

func formRows(pairs ...any) iup.Ihandle {
	cells := make([]iup.Ihandle, 0, len(pairs))
	for i := 0; i+1 < len(pairs); i += 2 {
		label := iup.Label(pairs[i].(string)).SetAttributes("ALIGNMENT=ALEFT:ACENTER")
		formLabels = append(formLabels, label)
		cells = append(cells, label, pairs[i+1].(iup.Ihandle))
	}
	// SIZECOL=1 sizes each line to its control's height so controls do not overlap.
	return iup.GridBox(cells...).SetAttributes(
		"ORIENTATION=HORIZONTAL, NUMDIV=2, GAPLIN=6, GAPCOL=10, ALIGNMENTLIN=ACENTER, SIZECOL=1, SIZELIN=-1",
	)
}

func toggle(on bool) iup.Ihandle {
	t := iup.Toggle("")
	if on {
		t.SetAttribute("VALUE", "ON")
	}
	return t
}

func isOn(t iup.Ihandle) bool { return t.GetAttribute("VALUE") == "ON" }

func yesNo(b bool) string {
	if b {
		return "YES"
	}
	return "NO"
}

func setActive(on bool, controls ...iup.Ihandle) {
	for _, c := range controls {
		c.SetAttribute("ACTIVE", yesNo(on))
	}
}

// scaleSpin is a spin box stepping a factor by tenths, shown with one decimal.
func scaleSpin(value float64) iup.Ihandle {
	t := iup.Text().SetAttributes(map[string]string{
		"SPIN": "YES", "SPINAUTO": "NO", "SPINMIN": "5", "SPINMAX": "30", "SPININC": "1",
		"SPINVALUE": strconv.Itoa(int(math.Round(value * 10))), "VISIBLECOLUMNS": "6",
	})
	t.SetAttribute("VALUE", strconv.FormatFloat(value, 'f', 1, 64))
	t.SetCallback("SPIN_CB", iup.SpinFunc(func(ih iup.Ihandle, pos int) int {
		ih.SetAttribute("VALUE", strconv.FormatFloat(float64(pos)/10, 'f', 1, 64))
		return iup.DEFAULT
	}))
	t.SetCallback("VALUECHANGED_CB", iup.ValueChangedFunc(func(ih iup.Ihandle) int {
		if f, err := strconv.ParseFloat(ih.GetAttribute("VALUE"), 64); err == nil {
			ih.SetAttribute("SPINVALUE", strconv.Itoa(int(math.Round(f*10))))
		}
		return iup.DEFAULT
	}))
	return t
}

// choiceList builds a dropdown of integer choices, selecting current.
func choiceList(choices []int, current int, label func(int) string) iup.Ihandle {
	list := iup.List().SetAttributes("DROPDOWN=YES, VISIBLEITEMS=10")
	sel := 1
	for i, v := range choices {
		text := strconv.Itoa(v)
		if label != nil {
			text = label(v)
		}
		list.SetAttribute(strconv.Itoa(i+1), text)
		if v == current {
			sel = i + 1
		}
	}
	list.SetAttribute("VALUE", strconv.Itoa(sel))
	return list
}

// editableChoice builds an editable dropdown offering options plus free-typed text.
func editableChoice(current string, options []string) iup.Ihandle {
	list := iup.List().SetAttributes("DROPDOWN=YES, EDITBOX=YES, VISIBLEITEMS=6, VISIBLECOLUMNS=18")
	for i, o := range options {
		list.SetAttribute(strconv.Itoa(i+1), o)
	}
	list.SetAttribute("VALUE", current)
	return list
}

func choiceText(list iup.Ihandle) string { return list.GetAttribute("VALUE") }

func languageList(code string) iup.Ihandle {
	list := iup.List().SetAttributes("DROPDOWN=YES, VISIBLEITEMS=14")
	sel := 1
	for i, l := range subtitleLanguages {
		list.SetAttribute(strconv.Itoa(i+1), l.Name)
		if l.Code == code {
			sel = i + 1
		}
	}
	list.SetAttribute("VALUE", strconv.Itoa(sel))
	return list
}

// listIndex returns the 0-based selected index of a dropdown list.
func listIndex(list iup.Ihandle) int {
	n, _ := strconv.Atoi(list.GetAttribute("VALUE"))
	if n < 1 {
		return 0
	}
	return n - 1
}

// colorButton is a swatch button that opens a color picker, storing "#RRGGBB".
func colorButton(hex string) iup.Ihandle {
	b := iup.Button("").SetAttributes("RASTERSIZE=48x20")
	setButtonColor(b, hex)
	b.SetCallback("ACTION", iup.ActionFunc(func(ih iup.Ihandle) int {
		r, g, bl := parseHexColor(ih.GetAttribute("BUKANIRCOLOR"))
		dlg := iup.ColorDlg().SetAttributes(map[string]string{"TITLE": "Text color", "SHOWHEX": "YES"})
		dlg.SetAttribute("VALUE", fmt.Sprintf("%d %d %d", r, g, bl))
		iup.SetAttributeHandle(dlg, "PARENTDIALOG", iup.GetDialog(ih))
		iup.Popup(dlg, iup.CENTER, iup.CENTER)
		if dlg.GetInt("STATUS") == 1 {
			if _, err := fmt.Sscanf(dlg.GetAttribute("VALUE"), "%d %d %d", &r, &g, &bl); err == nil {
				setButtonColor(ih, fmt.Sprintf("#%02X%02X%02X", r, g, bl))
			}
		}
		iup.Destroy(dlg)
		return iup.DEFAULT
	}))
	return b
}

func setButtonColor(b iup.Ihandle, hex string) {
	r, g, bl := parseHexColor(hex)
	b.SetAttribute("BUKANIRCOLOR", fmt.Sprintf("#%02X%02X%02X", r, g, bl))
	b.SetAttribute("BGCOLOR", fmt.Sprintf("%d %d %d", r, g, bl))
}

func pickDirectory(dir string) (string, bool) {
	dlg := iup.FileDlg().SetAttributes(map[string]string{"DIALOGTYPE": "DIR", "TITLE": "Download folder"})
	if dir != "" {
		dlg.SetAttribute("DIRECTORY", dir)
	}
	iup.SetAttributeHandle(dlg, "PARENTDIALOG", iup.GetHandle("dlg"))
	iup.Popup(dlg, iup.CENTER, iup.CENTER)
	defer iup.Destroy(dlg)
	if dlg.GetAttribute("STATUS") == "-1" {
		return "", false
	}
	return dlg.GetAttribute("VALUE"), true
}

func rateLabel(v int) string {
	if v == 0 {
		return "Unlimited"
	}
	return fmt.Sprintf("%d KiB/s", v)
}

func atoiDefault(s string, def int) int {
	if n, err := strconv.Atoi(s); err == nil {
		return n
	}
	return def
}

func parseFloatDefault(s string, def float64) float64 {
	if f, err := strconv.ParseFloat(s, 64); err == nil {
		return f
	}
	return def
}

// parseHexColor parses "#RRGGBB" (defaulting to white on error).
func parseHexColor(hex string) (r, g, b uint8) {
	var rr, gg, bb uint8
	if _, err := fmt.Sscanf(hex, "#%02x%02x%02x", &rr, &gg, &bb); err != nil {
		return 255, 255, 255
	}
	return rr, gg, bb
}
