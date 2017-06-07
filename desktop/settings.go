package main

import (
	"encoding/json"
	"log"
	"runtime"
	"strconv"
	"strings"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"

	"github.com/gen2brain/bukanir/lib"
)

// Settings type
type Settings struct {
	*widgets.QDialog
	*core.QSettings

	Limit           int
	Days            int
	Fullscreen      bool
	StopScreensaver bool
	VolumeMax       int
	Subtitles       bool
	Language        string
	Codepage        string
	Scale           float64
	Color           string
	Encryption      bool
	Proxy           bool
	Blocklist       bool
	DlRate          int
	UlRate          int
	Port            int
	TPBHost         string
	EZTVHost        string
	KeepFiles       bool
	DlPath          string

	comboLimit           *widgets.QComboBox
	comboDays            *widgets.QComboBox
	checkFullscreen      *widgets.QCheckBox
	checkStopScreensaver *widgets.QCheckBox
	sliderVolumeMax      *widgets.QSlider
	groupSubtitles       *widgets.QGroupBox
	comboLanguage        *widgets.QComboBox
	comboCodepage        *widgets.QComboBox
	spinScale            *widgets.QDoubleSpinBox
	pushColor            *widgets.QPushButton
	checkEncryption      *widgets.QCheckBox
	checkProxy           *widgets.QCheckBox
	checkBlocklist       *widgets.QCheckBox
	comboDlRate          *widgets.QComboBox
	comboUlRate          *widgets.QComboBox
	spinPort             *widgets.QSpinBox
	comboTPB             *widgets.QComboBox
	comboEZTV            *widgets.QComboBox
	checkKeepFiles       *widgets.QCheckBox
	lineDlPath           *widgets.QLineEdit
	pushDlPath           *widgets.QPushButton
	labelLanguage        *widgets.QLabel
	labelCodepage        *widgets.QLabel
	labelScale           *widgets.QLabel
	labelColor           *widgets.QLabel
	buttonBox            *widgets.QDialogButtonBox
}

// NewSettings returns new settings
func NewSettings(parent *widgets.QWidget) *Settings {
	widget := widgets.NewQDialog(parent, 0)
	widget.SetWindowTitle("Settings")
	widget.Resize2(430, 645)

	// General
	groupGeneral := widgets.NewQGroupBox2(tr("General"), widget)

	labelLimit := widgets.NewQLabel2(tr("Movies limit per tab"), widget, 0)
	labelDays := widgets.NewQLabel2(tr("Days to keep cache"), widget, 0)

	comboLimit := widgets.NewQComboBox(widget)
	comboLimit.AddItems([]string{"10", "30", "50", "70", "100"})

	comboDays := widgets.NewQComboBox(widget)
	comboDays.AddItems([]string{"3", "7", "30", "90", "180"})

	generalLayout := widgets.NewQGridLayout2()
	generalLayout.AddWidget(labelLimit, 0, 0, 0)
	generalLayout.AddWidget(comboLimit, 0, 1, 0)
	generalLayout.AddWidget(labelDays, 1, 0, 0)
	generalLayout.AddWidget(comboDays, 1, 1, 0)

	groupGeneral.SetLayout(generalLayout)

	// Player
	groupPlayer := widgets.NewQGroupBox2(tr("Player"), widget)

	checkFullscreen := widgets.NewQCheckBox2(tr("Fullscreen"), widget)
	checkFullscreen.SetToolTip(tr("Fullscreen playback"))

	checkStopScreensaver := widgets.NewQCheckBox2(tr("Stop screensaver"), widget)
	checkStopScreensaver.SetToolTip(tr("Turns off the screensaver (or screen blanker)"))

	labelVolumeMax := widgets.NewQLabel2(tr("Volume maximum"), widget, 0)
	labelVolumeMax.SetToolTip(tr("Set the maximum amplification level in percents"))

	sliderVolumeMax := widgets.NewQSlider2(core.Qt__Horizontal, widget)
	sliderVolumeMax.SetTickPosition(widgets.QSlider__NoTicks)
	sliderVolumeMax.SetMinimum(100)
	sliderVolumeMax.SetMaximum(200)

	playerLayout := widgets.NewQGridLayout2()
	playerLayout.AddWidget(checkFullscreen, 0, 0, 0)
	playerLayout.AddWidget(checkStopScreensaver, 1, 0, 0)
	playerLayout.AddWidget(labelVolumeMax, 2, 0, 0)
	playerLayout.AddWidget(sliderVolumeMax, 2, 1, 0)

	groupPlayer.SetLayout(playerLayout)

	// Subtitles
	groupSubtitles := widgets.NewQGroupBox2(tr("Subtitles"), widget)
	groupSubtitles.SetCheckable(true)

	labelLanguage := widgets.NewQLabel2(tr("Language"), widget, 0)
	labelCodepage := widgets.NewQLabel2(tr("Codepage"), widget, 0)
	labelScale := widgets.NewQLabel2(tr("Scale"), widget, 0)
	labelColor := widgets.NewQLabel2(tr("Text color"), widget, 0)

	comboLanguage := widgets.NewQComboBox(widget)
	comboLanguage.AddItems(strings.Split(bukanir.Languages(), ","))

	comboCodepage := widgets.NewQComboBox(widget)
	comboCodepage.AddItems([]string{"Auto", "BIG-5", "ISO_8859-1", "ISO_8859-13", "ISO_8859-14", "ISO_8859-15",
		"ISO_8859-2", "ISO_8859-3", "ISO_8859-4", "ISO_8859-5", "ISO_8859-6", "ISO_8859-7", "ISO_8859-8",
		"ISO_8859-9", "KOI8-R", "KOI8-U", "SHIFT_JIS", "UTF-16", "UTF-8", "CP1250", "CP1251", "CP1253", "CP1256"})

	spinScale := widgets.NewQDoubleSpinBox(widget)
	spinScale.SetToolTip(tr("Factor for the text subtitle font size"))
	spinScale.SetDecimals(1)
	spinScale.SetSingleStep(0.1)

	pushColor := widgets.NewQPushButton(widget)
	pushColor.SetAutoFillBackground(true)
	pushColor.SetSizePolicy2(widgets.QSizePolicy__Fixed, widgets.QSizePolicy__Fixed)
	pushColor.SetFlat(true)

	subLayout := widgets.NewQGridLayout2()
	subLayout.AddWidget(labelLanguage, 0, 0, 0)
	subLayout.AddWidget(comboLanguage, 0, 1, 0)
	subLayout.AddWidget(labelCodepage, 1, 0, 0)
	subLayout.AddWidget(comboCodepage, 1, 1, 0)
	subLayout.AddWidget(labelScale, 2, 0, 0)
	subLayout.AddWidget(spinScale, 2, 1, 0)
	subLayout.AddWidget(labelColor, 3, 0, 0)
	subLayout.AddWidget(pushColor, 3, 1, 0)

	groupSubtitles.SetLayout(subLayout)

	// Torrents
	groupTorrents := widgets.NewQGroupBox2(tr("Torrents"), widget)

	checkEncryption := widgets.NewQCheckBox2(tr("Encryption"), widget)
	checkEncryption.SetToolTip(tr("Protocol encryption (avoids ISP block)"))

	checkProxy := widgets.NewQCheckBox2(tr("Use proxy"), widget)
	checkProxy.SetToolTip(tr("Use Tor socks5 proxy to connect to peers"))

	checkBlocklist := widgets.NewQCheckBox2(tr("Use blocklist"), widget)
	checkBlocklist.SetToolTip(tr("Prevents connecting to IPs that presumably belong to anti-piracy outfits"))

	checkKeepFiles := widgets.NewQCheckBox2(tr("Keep files"), widget)
	checkKeepFiles.SetToolTip(tr("Keep files after exiting"))

	labelDlRate := widgets.NewQLabel2(tr("Maximum download rate (KB/s)"), widget, 0)
	labelUlRate := widgets.NewQLabel2(tr("Maximum upload rate (KB/s)"), widget, 0)
	labelPort := widgets.NewQLabel2(tr("Port for incoming connections"), widget, 0)
	labelTPB := widgets.NewQLabel2(tr("TPB host"), widget, 0)
	labelEZTV := widgets.NewQLabel2(tr("EZTV host"), widget, 0)

	comboDlRate := widgets.NewQComboBox(widget)
	comboDlRate.AddItems([]string{"-1", "1", "10", "50", "100", "500", "1000", "5000", "10000"})

	comboUlRate := widgets.NewQComboBox(widget)
	comboUlRate.AddItems([]string{"-1", "1", "10", "50", "100", "500", "1000", "5000", "10000"})

	spinPort := widgets.NewQSpinBox(widget)
	spinPort.SetMinimum(6800)
	spinPort.SetMaximum(6999)

	comboTPB := widgets.NewQComboBox(widget)

	comboTPB.AddItems([]string{bukanir.TpbTor})
	comboTPB.AddItems(bukanir.TpbHosts)
	comboTPB.SetEditable(true)
	comboTPB.SetInsertPolicy(widgets.QComboBox__NoInsert)
	comboTPB.SetToolTip(tr("TPB domain name, if empty it will be autodetected"))

	comboEZTV := widgets.NewQComboBox(widget)
	comboEZTV.AddItems(bukanir.EztvHosts)
	comboEZTV.SetEditable(true)
	comboEZTV.SetInsertPolicy(widgets.QComboBox__NoInsert)
	comboEZTV.SetToolTip(tr("EZTV domain name, if empty it will be autodetected"))

	lineDlPath := widgets.NewQLineEdit(widget)
	lineDlPath.SetToolTip(tr("Download directory"))
	lineDlPath.SetPlaceholderText(tr("Download directory"))

	pushDlPath := widgets.NewQPushButton(widget)
	pushDlPath.SetText(tr("Browse..."))

	torrentsLayout := widgets.NewQGridLayout2()
	torrentsLayout.AddWidget(checkEncryption, 0, 0, 0)
	torrentsLayout.AddWidget(checkProxy, 1, 0, 0)
	torrentsLayout.AddWidget(checkBlocklist, 2, 0, 0)
	torrentsLayout.AddWidget(labelDlRate, 3, 0, 0)
	torrentsLayout.AddWidget(comboDlRate, 3, 1, 0)
	torrentsLayout.AddWidget(labelUlRate, 4, 0, 0)
	torrentsLayout.AddWidget(comboUlRate, 4, 1, 0)
	torrentsLayout.AddWidget(labelPort, 5, 0, 0)
	torrentsLayout.AddWidget(spinPort, 5, 1, 0)
	torrentsLayout.AddWidget(labelTPB, 6, 0, 0)
	torrentsLayout.AddWidget(comboTPB, 6, 1, 0)
	torrentsLayout.AddWidget(labelEZTV, 7, 0, 0)
	torrentsLayout.AddWidget(comboEZTV, 7, 1, 0)
	torrentsLayout.AddWidget(checkKeepFiles, 8, 0, 0)
	torrentsLayout.AddWidget(pushDlPath, 8, 1, 0)
	torrentsLayout.AddWidget3(lineDlPath, 9, 0, 9, 2, 0)

	groupTorrents.SetLayout(torrentsLayout)

	// Close
	buttonBox := widgets.NewQDialogButtonBox3(widgets.QDialogButtonBox__Close, widget)
	buttonBox.Button(widgets.QDialogButtonBox__Close).SetText(tr("Close"))

	// Layout
	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(groupGeneral, 0, 0)
	layout.AddWidget(groupPlayer, 0, 0)
	layout.AddWidget(groupSubtitles, 0, 0)
	layout.AddWidget(groupTorrents, 0, 0)
	layout.AddWidget(buttonBox, 0, 0)
	widget.SetLayout(layout)

	qsettings := core.NewQSettings("bukanir", "bukanir", parent)

	settings := &Settings{
		widget, qsettings,
		0, 0, false, false, 0, false, "", "", 0, "", false, false, false, 0, 0, 0, "", "", false, "",
		comboLimit, comboDays, checkFullscreen, checkStopScreensaver, sliderVolumeMax, groupSubtitles, comboLanguage, comboCodepage,
		spinScale, pushColor, checkEncryption, checkProxy, checkBlocklist, comboDlRate, comboUlRate, spinPort, comboTPB, comboEZTV, checkKeepFiles, lineDlPath, pushDlPath,
		labelLanguage, labelCodepage, labelScale, labelColor, buttonBox,
	}

	settings.Set()

	// Show event
	var filterObject = core.NewQObject(parent)
	filterObject.ConnectEventFilter(func(watched *core.QObject, event *core.QEvent) bool {
		if event.Type() == core.QEvent__Show {
			settings.Set()
			return true
		}
		return false
	})

	widget.InstallEventFilter(filterObject)

	return settings
}

// ConnectSignals connects signals
func (s *Settings) ConnectSignals() {
	s.sliderVolumeMax.ConnectValueChanged(func(value int) {
		s.sliderVolumeMax.SetToolTip(strconv.Itoa(value) + "%")
	})

	s.groupSubtitles.ConnectClicked(func(checked bool) {
		s.comboLanguage.SetEnabled(checked)
		s.comboCodepage.SetEnabled(checked)
		s.labelCodepage.SetEnabled(checked)
		s.labelLanguage.SetEnabled(checked)
		s.labelScale.SetEnabled(checked)
		s.spinScale.SetEnabled(checked)
		s.labelColor.SetEnabled(checked)
		s.pushColor.SetEnabled(checked)
	})

	s.pushColor.ConnectClicked(func(checked bool) {
		dialog := widgets.NewQColorDialog(s.QDialog)
		dialog.SetCurrentColor(s.pushColor.Palette().Color2(gui.QPalette__Button))

		dialog.ConnectColorSelected(func(color *gui.QColor) {
			s.pushColor.Palette().SetColor2(gui.QPalette__Button, color)
		})

		dialog.Show()
	})

	s.checkKeepFiles.ConnectClicked(func(checked bool) {
		s.lineDlPath.SetEnabled(checked)
		s.pushDlPath.SetEnabled(checked)
	})

	s.pushDlPath.ConnectClicked(func(checked bool) {
		dialog := widgets.NewQFileDialog(s.QDialog, core.Qt__Dialog)
		dialog.SetWindowTitle(tr("Download path"))
		dialog.SetFileMode(widgets.QFileDialog__Directory)
		dialog.SetOption(widgets.QFileDialog__ShowDirsOnly, true)

		if dialog.Exec() == int(widgets.QDialog__Accepted) {
			files := dialog.SelectedFiles()
			if len(files) > 0 {
				path := files[0]
				if path != "" {
					s.lineDlPath.SetText(path)
				}
			}
		}
	})

	s.buttonBox.ConnectRejected(func() {
		s.Save()
		s.Sync()
		s.QDialog.Close()
	})
}

// Set sets values
func (s *Settings) Set() {

	limit := s.Value("limit", core.NewQVariant7(30))
	days := s.Value("days", core.NewQVariant7(90))
	fullscreen := s.Value("fullscreen", core.NewQVariant7(0))
	stopscreensaver := s.Value("stopscreensaver", core.NewQVariant7(1))
	volumemax := s.Value("volumemax", core.NewQVariant7(130))
	subtitles := s.Value("subtitles", core.NewQVariant7(1))
	language := s.Value("language", core.NewQVariant14("English"))
	codepage := s.Value("codepage", core.NewQVariant14("auto"))
	subscale := s.Value("subscale", core.NewQVariant12(1.0))
	subcolor := s.Value("subcolor", core.NewQVariant14("#FFFF00"))
	encryption := s.Value("encryption", core.NewQVariant7(1))

	var proxy *core.QVariant
	if !bukanir.TorRunning() {
		s.checkProxy.SetChecked(false)
		s.checkProxy.SetEnabled(false)

		s.SetValue("proxy", core.NewQVariant11(false))
		proxy = s.Value("proxy", core.NewQVariant7(0))
	} else {
		proxy = s.Value("proxy", core.NewQVariant7(1))
	}

	blocklist := s.Value("blocklist", core.NewQVariant7(0))

	dlrate := s.Value("dlrate", core.NewQVariant7(-1))
	ulrate := s.Value("ulrate", core.NewQVariant7(-1))
	port := s.Value("port", core.NewQVariant7(6881))
	tpbHost := s.Value("tpbhost", core.NewQVariant14("thepiratebay.org"))
	eztvHost := s.Value("eztvhost", core.NewQVariant14("eztv.ag"))
	keep := s.Value("keep", core.NewQVariant7(0))
	dlpath := s.Value("dlpath", core.NewQVariant14(""))

	s.comboLimit.SetCurrentText(limit.ToString())
	s.comboDays.SetCurrentText(days.ToString())
	s.checkFullscreen.SetChecked(fullscreen.ToBool())
	s.checkStopScreensaver.SetChecked(stopscreensaver.ToBool())
	s.sliderVolumeMax.SetValue(volumemax.ToInt(false))
	s.sliderVolumeMax.SetToolTip(strconv.Itoa(volumemax.ToInt(false)) + "%")
	s.groupSubtitles.SetChecked(subtitles.ToBool())
	s.comboLanguage.SetCurrentText(language.ToString())
	s.comboCodepage.SetCurrentText(codepage.ToString())
	s.spinScale.SetValue(float64(subscale.ToFloat(false)))
	s.pushColor.Palette().SetColor2(gui.QPalette__Button, gui.NewQColor6(subcolor.ToString()))
	s.checkEncryption.SetChecked(encryption.ToBool())
	s.checkProxy.SetChecked(proxy.ToBool())
	s.checkBlocklist.SetChecked(blocklist.ToBool())
	s.comboDlRate.SetCurrentText(dlrate.ToString())
	s.comboUlRate.SetCurrentText(ulrate.ToString())
	s.spinPort.SetValue(port.ToInt(false))
	s.comboTPB.SetCurrentText(tpbHost.ToString())
	s.comboEZTV.SetCurrentText(eztvHost.ToString())
	s.checkKeepFiles.SetChecked(keep.ToBool())
	s.lineDlPath.SetText(dlpath.ToString())

	s.lineDlPath.SetEnabled(keep.ToBool())
	s.pushDlPath.SetEnabled(keep.ToBool())

	s.Limit = limit.ToInt(false)
	s.Days = days.ToInt(false)
	s.Fullscreen = fullscreen.ToBool()
	s.StopScreensaver = stopscreensaver.ToBool()
	s.VolumeMax = volumemax.ToInt(false)
	s.Subtitles = subtitles.ToBool()
	s.Language = language.ToString()
	s.Codepage = strings.ToLower(codepage.ToString())
	s.Scale = float64(subscale.ToFloat(false))
	s.Color = subcolor.ToString()
	s.Encryption = encryption.ToBool()
	s.Proxy = proxy.ToBool()
	s.Blocklist = blocklist.ToBool()
	s.DlRate = dlrate.ToInt(false)
	s.UlRate = ulrate.ToInt(false)
	s.Port = port.ToInt(false)
	s.TPBHost = tpbHost.ToString()
	s.EZTVHost = eztvHost.ToString()
	s.KeepFiles = keep.ToBool()
	s.DlPath = dlpath.ToString()
}

// Save saves values
func (s *Settings) Save() {
	limit := s.comboLimit.CurrentText()
	days := s.comboDays.CurrentText()
	fullscreen := s.checkFullscreen.IsChecked()
	stopscreensaver := s.checkStopScreensaver.IsChecked()
	volumemax := s.sliderVolumeMax.Value()
	subtitles := s.groupSubtitles.IsChecked()
	subcolor := s.pushColor.Palette().Color2(gui.QPalette__Button).Name()
	language := s.comboLanguage.CurrentText()
	codepage := s.comboCodepage.CurrentText()
	subscale := s.spinScale.Value()
	encryption := s.checkEncryption.IsChecked()
	proxy := s.checkProxy.IsChecked()
	blocklist := s.checkBlocklist.IsChecked()
	dlrate := s.comboDlRate.CurrentText()
	ulrate := s.comboUlRate.CurrentText()
	port := s.spinPort.Value()
	tpbHost := s.comboTPB.CurrentText()
	eztvHost := s.comboEZTV.CurrentText()
	keep := s.checkKeepFiles.IsChecked()
	dlpath := s.lineDlPath.Text()

	s.SetValue("limit", core.NewQVariant14(limit))
	s.SetValue("days", core.NewQVariant14(days))
	s.SetValue("fullscreen", core.NewQVariant11(fullscreen))
	s.SetValue("stopscreensaver", core.NewQVariant11(stopscreensaver))
	s.SetValue("volumemax", core.NewQVariant7(volumemax))
	s.SetValue("subtitles", core.NewQVariant11(subtitles))
	s.SetValue("subcolor", core.NewQVariant14(subcolor))
	s.SetValue("language", core.NewQVariant14(language))
	s.SetValue("codepage", core.NewQVariant14(codepage))
	s.SetValue("subscale", core.NewQVariant12(subscale))
	s.SetValue("encryption", core.NewQVariant11(encryption))
	s.SetValue("proxy", core.NewQVariant11(proxy))
	s.SetValue("blocklist", core.NewQVariant11(blocklist))
	s.SetValue("dlrate", core.NewQVariant14(dlrate))
	s.SetValue("ulrate", core.NewQVariant14(ulrate))
	s.SetValue("port", core.NewQVariant7(port))
	s.SetValue("tpbhost", core.NewQVariant14(tpbHost))
	s.SetValue("eztvhost", core.NewQVariant14(eztvHost))
	s.SetValue("keep", core.NewQVariant11(keep))
	s.SetValue("dlpath", core.NewQVariant14(dlpath))

	s.Limit = core.NewQVariant14(limit).ToInt(false)
	s.Days = core.NewQVariant14(days).ToInt(false)
	s.Fullscreen = core.NewQVariant11(fullscreen).ToBool()
	s.StopScreensaver = core.NewQVariant11(stopscreensaver).ToBool()
	s.VolumeMax = core.NewQVariant7(volumemax).ToInt(false)
	s.Subtitles = core.NewQVariant11(subtitles).ToBool()
	s.Language = core.NewQVariant14(language).ToString()
	s.Codepage = strings.ToLower(core.NewQVariant14(codepage).ToString())
	s.Scale = float64(core.NewQVariant12(subscale).ToFloat(false))
	s.Color = core.NewQVariant14(subcolor).ToString()
	s.Encryption = core.NewQVariant11(encryption).ToBool()
	s.Proxy = core.NewQVariant11(proxy).ToBool()
	s.Blocklist = core.NewQVariant11(blocklist).ToBool()
	s.DlRate = core.NewQVariant14(dlrate).ToInt(false)
	s.UlRate = core.NewQVariant14(ulrate).ToInt(false)
	s.Port = core.NewQVariant7(port).ToInt(false)
	s.TPBHost = core.NewQVariant14(tpbHost).ToString()
	s.EZTVHost = core.NewQVariant14(eztvHost).ToString()
	s.KeepFiles = core.NewQVariant11(keep).ToBool()
	s.DlPath = core.NewQVariant14(dlpath).ToString()
}

// TorrentConfig returns torrent config
func (s *Settings) TorrentConfig(url string) string {
	c := &bukanir.TConfig{}
	c.Uri = url
	c.BindAddress = "127.0.0.1:5001"
	c.FileIndex = -1
	c.KeepFiles = s.KeepFiles && s.DlPath != ""
	c.UserAgent = "Bukanir " + bukanir.Version
	c.DhtRouters = "router.bittorrent.com:6881,router.utorrent.com:6881,dht.transmissionbt.com:6881,dht.aelitis.com:6881"
	c.Trackers = "udp://tracker.publicbt.com:80,udp://tracker.openbittorrent.com:80,udp://open.demonii.com:80,udp://tracker.istole.it:80,udp://tracker.coppersurfer.tk:80,udp://tracker.leechers-paradise.org:6969,udp://exodus.desync.com:6969,udp://tracker.pomf.se,udp://tracker.blackunicorn.xyz:6969,udp://pow7.com:80/announce"
	c.ListenPort = s.Port
	c.TorrentConnectBoost = 10 * runtime.NumCPU()
	c.ConnectionSpeed = 10 * runtime.NumCPU()
	c.PeerConnectTimeout = 3
	c.RequestTimeout = 3
	c.MaxDownloadRate = s.DlRate
	c.MaxUploadRate = s.UlRate
	c.MinReconnectTime = 60
	c.MaxFailCount = 3
	c.Proxy = s.Proxy
	c.ProxyHost = "127.0.0.1"
	c.ProxyPort = 9250
	c.Blocklist = s.Blocklist
	c.Verbose = true

	if s.KeepFiles && s.DlPath != "" {
		c.DownloadPath = s.DlPath
	} else {
		c.DownloadPath = tempDir
	}

	c.Encryption = 1
	if !s.Encryption {
		c.Encryption = 2
	}

	js, err := json.Marshal(c)
	if err != nil {
		log.Printf("ERROR: Marshal: %s\n", err.Error())
		return ""
	}

	return string(js[:])
}
