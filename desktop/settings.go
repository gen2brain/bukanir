package main

import (
	"encoding/json"
	"log"
	"strconv"
	"strings"

	"github.com/therecipe/qt/core"
	"github.com/therecipe/qt/gui"
	"github.com/therecipe/qt/widgets"

	"github.com/gen2brain/bukanir/lib/bukanir"
)

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
	DlRate          int
	UlRate          int
	Port            int
	TPBHost         string
	EZTVHost        string
	KeepFiles       bool
	DlPath          string
}

func NewSettings(parent *widgets.QWidget) *Settings {
	widget := widgets.NewQDialog(parent, 0)
	widget.SetWindowTitle("Settings")
	widget.Resize2(430, 645)

	// General
	groupGeneral := widgets.NewQGroupBox2("General", widget)

	labelLimit := widgets.NewQLabel2("Movies limit per tab", widget, 0)
	labelDays := widgets.NewQLabel2("Days to keep cache", widget, 0)

	comboLimit := widgets.NewQComboBox(widget)
	comboLimit.SetObjectName("comboLimit")
	comboLimit.AddItems([]string{"10", "30", "50", "70", "100"})

	comboDays := widgets.NewQComboBox(widget)
	comboDays.SetObjectName("comboDays")
	comboDays.AddItems([]string{"3", "7", "30", "90", "180"})

	generalLayout := widgets.NewQGridLayout2()
	generalLayout.AddWidget(labelLimit, 0, 0, 0)
	generalLayout.AddWidget(comboLimit, 0, 1, 0)
	generalLayout.AddWidget(labelDays, 1, 0, 0)
	generalLayout.AddWidget(comboDays, 1, 1, 0)

	groupGeneral.SetLayout(generalLayout)

	// Player
	groupPlayer := widgets.NewQGroupBox2("Player", widget)

	checkFullscreen := widgets.NewQCheckBox2("Fullscreen", widget)
	checkFullscreen.SetObjectName("checkFullscreen")
	checkFullscreen.SetToolTip("Fullscreen playback")

	checkStopScreensaver := widgets.NewQCheckBox2("Stop screensaver", widget)
	checkStopScreensaver.SetObjectName("checkStopScreensaver")
	checkStopScreensaver.SetToolTip("Turns off the screensaver (or screen blanker)")

	labelVolumeMax := widgets.NewQLabel2("Volume maximum", widget, 0)
	labelVolumeMax.SetToolTip("Set the maximum amplification level in percents")

	sliderVolumeMax := widgets.NewQSlider2(core.Qt__Horizontal, widget)
	sliderVolumeMax.SetObjectName("sliderVolumeMax")
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
	groupSubtitles := widgets.NewQGroupBox2("Subtitles", widget)
	groupSubtitles.SetObjectName("groupSubtitles")
	groupSubtitles.SetCheckable(true)

	labelLanguage := widgets.NewQLabel2("Language", widget, 0)
	labelLanguage.SetObjectName("labelLanguage")
	labelCodepage := widgets.NewQLabel2("Codepage", widget, 0)
	labelCodepage.SetObjectName("labelCodepage")
	labelScale := widgets.NewQLabel2("Scale", widget, 0)
	labelScale.SetObjectName("labelScale")
	labelColor := widgets.NewQLabel2("Text color", widget, 0)
	labelColor.SetObjectName("labelColor")

	comboLanguage := widgets.NewQComboBox(widget)
	comboLanguage.SetObjectName("comboLanguage")
	comboLanguage.AddItems(strings.Split(bukanir.Languages(), ","))

	comboCodepage := widgets.NewQComboBox(widget)
	comboCodepage.SetObjectName("comboCodepage")
	comboCodepage.AddItems([]string{"Auto", "BIG-5", "ISO_8859-1", "ISO_8859-13", "ISO_8859-14", "ISO_8859-15", "ISO_8859-2", "ISO_8859-3", "ISO_8859-4", "ISO_8859-5", "ISO_8859-6", "ISO_8859-7", "ISO_8859-8", "ISO_8859-9", "KOI8-R", "KOI8-U", "SHIFT_JIS", "UTF-16", "UTF-8", "CP1250", "CP1251", "CP1253", "CP1256"})

	spinScale := widgets.NewQDoubleSpinBox(widget)
	spinScale.SetObjectName("spinScale")
	spinScale.SetToolTip("Factor for the text subtitle font size")
	spinScale.SetDecimals(1)
	spinScale.SetSingleStep(0.1)

	pushColor := widgets.NewQPushButton(widget)
	pushColor.SetObjectName("pushColor")
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
	groupTorrents := widgets.NewQGroupBox2("Torrents", widget)

	checkEncryption := widgets.NewQCheckBox2("Encryption", widget)
	checkEncryption.SetObjectName("checkEncryption")
	checkEncryption.SetToolTip("Protocol encryption (avoids ISP block)")

	checkKeepFiles := widgets.NewQCheckBox2("Keep files", widget)
	checkKeepFiles.SetObjectName("checkKeepFiles")
	checkKeepFiles.SetToolTip("Keep files after exiting")

	labelDlRate := widgets.NewQLabel2("Maximum download rate (KB/s)", widget, 0)
	labelUlRate := widgets.NewQLabel2("Maximum upload rate (KB/s)", widget, 0)
	labelPort := widgets.NewQLabel2("Port for incoming connections", widget, 0)
	labelTPB := widgets.NewQLabel2("TPB host", widget, 0)
	labelEZTV := widgets.NewQLabel2("EZTV host", widget, 0)

	comboDlRate := widgets.NewQComboBox(widget)
	comboDlRate.SetObjectName("comboDlRate")
	comboDlRate.AddItems([]string{"-1", "1", "10", "50", "100", "500", "1000", "5000", "10000"})

	comboUlRate := widgets.NewQComboBox(widget)
	comboUlRate.SetObjectName("comboUlRate")
	comboUlRate.AddItems([]string{"-1", "1", "10", "50", "100", "500", "1000", "5000", "10000"})

	spinPort := widgets.NewQSpinBox(widget)
	spinPort.SetObjectName("spinPort")
	spinPort.SetMinimum(6800)
	spinPort.SetMaximum(6999)

	comboTPB := widgets.NewQComboBox(widget)
	comboTPB.SetObjectName("comboTPB")
	comboTPB.AddItems(bukanir.TpbHosts)
	comboTPB.SetEditable(true)
	comboTPB.SetInsertPolicy(widgets.QComboBox__NoInsert)
	comboTPB.SetToolTip("TPB domain name, if empty it will be autodetected")

	comboEZTV := widgets.NewQComboBox(widget)
	comboEZTV.SetObjectName("comboEZTV")
	comboEZTV.AddItems(bukanir.EztvHosts)
	comboEZTV.SetEditable(true)
	comboEZTV.SetInsertPolicy(widgets.QComboBox__NoInsert)
	comboEZTV.SetToolTip("EZTV domain name, if empty it will be autodetected")

	lineDlPath := widgets.NewQLineEdit(widget)
	lineDlPath.SetObjectName("lineDlPath")
	lineDlPath.SetToolTip("Download directory")
	lineDlPath.SetPlaceholderText("Download directory")

	pushDlPath := widgets.NewQPushButton(widget)
	pushDlPath.SetObjectName("pushDlPath")
	pushDlPath.SetText("Browse...")

	torrentsLayout := widgets.NewQGridLayout2()
	torrentsLayout.AddWidget(checkEncryption, 0, 0, 0)
	torrentsLayout.AddWidget(labelDlRate, 1, 0, 0)
	torrentsLayout.AddWidget(comboDlRate, 1, 1, 0)
	torrentsLayout.AddWidget(labelUlRate, 2, 0, 0)
	torrentsLayout.AddWidget(comboUlRate, 2, 1, 0)
	torrentsLayout.AddWidget(labelPort, 3, 0, 0)
	torrentsLayout.AddWidget(spinPort, 3, 1, 0)
	torrentsLayout.AddWidget(labelTPB, 4, 0, 0)
	torrentsLayout.AddWidget(comboTPB, 4, 1, 0)
	torrentsLayout.AddWidget(labelEZTV, 5, 0, 0)
	torrentsLayout.AddWidget(comboEZTV, 5, 1, 0)
	torrentsLayout.AddWidget(checkKeepFiles, 6, 0, 0)
	torrentsLayout.AddWidget(pushDlPath, 6, 1, 0)
	torrentsLayout.AddWidget3(lineDlPath, 7, 0, 7, 2, 0)

	groupTorrents.SetLayout(torrentsLayout)

	// Close
	buttonBox := widgets.NewQDialogButtonBox3(widgets.QDialogButtonBox__Close, widget)
	buttonBox.SetObjectName("buttonBox")

	// Layout
	layout := widgets.NewQVBoxLayout()
	layout.AddWidget(groupGeneral, 0, 0)
	layout.AddWidget(groupPlayer, 0, 0)
	layout.AddWidget(groupSubtitles, 0, 0)
	layout.AddWidget(groupTorrents, 0, 0)
	layout.AddWidget(buttonBox, 0, 0)
	widget.SetLayout(layout)

	qsettings := core.NewQSettings("bukanir", "bukanir", parent)
	settings := &Settings{widget, qsettings, 0, 0, false, false, 0, false, "", "", 0, "", false, 0, 0, 0, "", "", false, ""}
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

func (s *Settings) ConnectSignals() {
	sliderVolumeMax := widgets.NewQSliderFromPointer(s.QWidget.FindChild("sliderVolumeMax", core.Qt__FindChildrenRecursively))
	groupSubtitles := widgets.NewQGroupBoxFromPointer(s.QWidget.FindChild("groupSubtitles", core.Qt__FindChildrenRecursively))
	comboLanguage := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboLanguage", core.Qt__FindChildrenRecursively))
	comboCodepage := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboCodepage", core.Qt__FindChildrenRecursively))
	labelLanguage := widgets.NewQLabelFromPointer(s.QWidget.FindChild("labelLanguage", core.Qt__FindChildrenRecursively))
	labelCodepage := widgets.NewQLabelFromPointer(s.QWidget.FindChild("labelCodepage", core.Qt__FindChildrenRecursively))
	labelScale := widgets.NewQLabelFromPointer(s.QWidget.FindChild("labelScale", core.Qt__FindChildrenRecursively))
	spinScale := widgets.NewQDoubleSpinBoxFromPointer(s.QWidget.FindChild("spinScale", core.Qt__FindChildrenRecursively))
	labelColor := widgets.NewQLabelFromPointer(s.QWidget.FindChild("labelColor", core.Qt__FindChildrenRecursively))
	pushColor := widgets.NewQPushButtonFromPointer(s.QWidget.FindChild("pushColor", core.Qt__FindChildrenRecursively))
	buttonBox := widgets.NewQDialogButtonBoxFromPointer(s.QWidget.FindChild("buttonBox", core.Qt__FindChildrenRecursively))
	checkKeepFiles := widgets.NewQCheckBoxFromPointer(s.QWidget.FindChild("checkKeepFiles", core.Qt__FindChildrenRecursively))
	lineDlPath := widgets.NewQLineEditFromPointer(s.QWidget.FindChild("lineDlPath", core.Qt__FindChildrenRecursively))
	pushDlPath := widgets.NewQPushButtonFromPointer(s.QWidget.FindChild("pushDlPath", core.Qt__FindChildrenRecursively))

	sliderVolumeMax.ConnectValueChanged(func(value int) {
		sliderVolumeMax.SetToolTip(strconv.Itoa(value) + "%")
	})

	groupSubtitles.ConnectClicked(func(checked bool) {
		comboLanguage.SetEnabled(checked)
		comboCodepage.SetEnabled(checked)
		labelCodepage.SetEnabled(checked)
		labelLanguage.SetEnabled(checked)
		labelScale.SetEnabled(checked)
		spinScale.SetEnabled(checked)
		labelColor.SetEnabled(checked)
		pushColor.SetEnabled(checked)
	})

	pushColor.ConnectClicked(func(checked bool) {
		dialog := widgets.NewQColorDialog(s.QDialog)
		dialog.SetCurrentColor(pushColor.Palette().Color2(gui.QPalette__Button))

		dialog.ConnectColorSelected(func(color *gui.QColor) {
			pushColor.Palette().SetColor2(gui.QPalette__Button, color)
		})

		dialog.Show()
	})

	checkKeepFiles.ConnectClicked(func(checked bool) {
		lineDlPath.SetEnabled(checked)
		pushDlPath.SetEnabled(checked)
	})

	pushDlPath.ConnectClicked(func(checked bool) {
		dialog := widgets.NewQFileDialog(s.QDialog, core.Qt__Dialog)
		dialog.SetWindowTitle("Download path")
		dialog.SetFileMode(widgets.QFileDialog__Directory)
		dialog.SetOption(widgets.QFileDialog__ShowDirsOnly, true)

		if dialog.Exec() == int(widgets.QDialog__Accepted) {
			files := dialog.SelectedFiles()
			if len(files) > 0 {
				path := files[0]
				if path != "" {
					lineDlPath.SetText(path)
				}
			}
		}
	})

	buttonBox.ConnectRejected(func() {
		s.Save()
		s.Sync()
		s.QDialog.Close()
	})
}

func (s *Settings) Set() {
	comboLimit := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboLimit", core.Qt__FindChildrenRecursively))
	comboDays := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboDays", core.Qt__FindChildrenRecursively))
	checkFullscreen := widgets.NewQCheckBoxFromPointer(s.QWidget.FindChild("checkFullscreen", core.Qt__FindChildrenRecursively))
	checkStopScreensaver := widgets.NewQCheckBoxFromPointer(s.QWidget.FindChild("checkStopScreensaver", core.Qt__FindChildrenRecursively))
	sliderVolumeMax := widgets.NewQSliderFromPointer(s.QWidget.FindChild("sliderVolumeMax", core.Qt__FindChildrenRecursively))
	groupSubtitles := widgets.NewQGroupBoxFromPointer(s.QWidget.FindChild("groupSubtitles", core.Qt__FindChildrenRecursively))
	comboLanguage := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboLanguage", core.Qt__FindChildrenRecursively))
	comboCodepage := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboCodepage", core.Qt__FindChildrenRecursively))
	spinScale := widgets.NewQDoubleSpinBoxFromPointer(s.QWidget.FindChild("spinScale", core.Qt__FindChildrenRecursively))
	pushColor := widgets.NewQPushButtonFromPointer(s.QWidget.FindChild("pushColor", core.Qt__FindChildrenRecursively))
	checkEncryption := widgets.NewQCheckBoxFromPointer(s.QWidget.FindChild("checkEncryption", core.Qt__FindChildrenRecursively))
	comboDlRate := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboDlRate", core.Qt__FindChildrenRecursively))
	comboUlRate := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboUlRate", core.Qt__FindChildrenRecursively))
	spinPort := widgets.NewQSpinBoxFromPointer(s.QWidget.FindChild("spinPort", core.Qt__FindChildrenRecursively))
	comboTPB := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboTPB", core.Qt__FindChildrenRecursively))
	comboEZTV := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboEZTV", core.Qt__FindChildrenRecursively))
	checkKeepFiles := widgets.NewQCheckBoxFromPointer(s.QWidget.FindChild("checkKeepFiles", core.Qt__FindChildrenRecursively))
	lineDlPath := widgets.NewQLineEditFromPointer(s.QWidget.FindChild("lineDlPath", core.Qt__FindChildrenRecursively))
	pushDlPath := widgets.NewQPushButtonFromPointer(s.QWidget.FindChild("pushDlPath", core.Qt__FindChildrenRecursively))

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
	dlrate := s.Value("dlrate", core.NewQVariant7(-1))
	ulrate := s.Value("ulrate", core.NewQVariant7(-1))
	port := s.Value("port", core.NewQVariant7(6881))
	tpbHost := s.Value("tpbhost", core.NewQVariant14("thepiratebay.org"))
	eztvHost := s.Value("eztvhost", core.NewQVariant14("eztv.ag"))
	keep := s.Value("keep", core.NewQVariant7(0))
	dlpath := s.Value("dlpath", core.NewQVariant14(""))

	comboLimit.SetCurrentText(limit.ToString())
	comboDays.SetCurrentText(days.ToString())
	checkFullscreen.SetChecked(fullscreen.ToBool())
	checkStopScreensaver.SetChecked(stopscreensaver.ToBool())
	sliderVolumeMax.SetValue(volumemax.ToInt(false))
	sliderVolumeMax.SetToolTip(strconv.Itoa(volumemax.ToInt(false)) + "%")
	groupSubtitles.SetChecked(subtitles.ToBool())
	comboLanguage.SetCurrentText(language.ToString())
	comboCodepage.SetCurrentText(codepage.ToString())
	spinScale.SetValue(float64(subscale.ToFloat(false)))
	pushColor.Palette().SetColor2(gui.QPalette__Button, gui.NewQColor6(subcolor.ToString()))
	checkEncryption.SetChecked(encryption.ToBool())
	comboDlRate.SetCurrentText(dlrate.ToString())
	comboUlRate.SetCurrentText(ulrate.ToString())
	spinPort.SetValue(port.ToInt(false))
	comboTPB.SetCurrentText(tpbHost.ToString())
	comboEZTV.SetCurrentText(eztvHost.ToString())
	checkKeepFiles.SetChecked(keep.ToBool())
	lineDlPath.SetText(dlpath.ToString())

	lineDlPath.SetEnabled(keep.ToBool())
	pushDlPath.SetEnabled(keep.ToBool())

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
	s.DlRate = dlrate.ToInt(false)
	s.UlRate = ulrate.ToInt(false)
	s.Port = port.ToInt(false)
	s.TPBHost = tpbHost.ToString()
	s.EZTVHost = eztvHost.ToString()
	s.KeepFiles = keep.ToBool()
	s.DlPath = dlpath.ToString()
}

func (s *Settings) Save() {
	comboLimit := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboLimit", core.Qt__FindChildrenRecursively))
	comboDays := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboDays", core.Qt__FindChildrenRecursively))
	checkFullscreen := widgets.NewQCheckBoxFromPointer(s.QWidget.FindChild("checkFullscreen", core.Qt__FindChildrenRecursively))
	checkStopScreensaver := widgets.NewQCheckBoxFromPointer(s.QWidget.FindChild("checkStopScreensaver", core.Qt__FindChildrenRecursively))
	sliderVolumeMax := widgets.NewQSliderFromPointer(s.QWidget.FindChild("sliderVolumeMax", core.Qt__FindChildrenRecursively))
	groupSubtitles := widgets.NewQGroupBoxFromPointer(s.QWidget.FindChild("groupSubtitles", core.Qt__FindChildrenRecursively))
	comboLanguage := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboLanguage", core.Qt__FindChildrenRecursively))
	comboCodepage := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboCodepage", core.Qt__FindChildrenRecursively))
	spinScale := widgets.NewQDoubleSpinBoxFromPointer(s.QWidget.FindChild("spinScale", core.Qt__FindChildrenRecursively))
	pushColor := widgets.NewQPushButtonFromPointer(s.QWidget.FindChild("pushColor", core.Qt__FindChildrenRecursively))
	checkEncryption := widgets.NewQCheckBoxFromPointer(s.QWidget.FindChild("checkEncryption", core.Qt__FindChildrenRecursively))
	comboDlRate := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboDlRate", core.Qt__FindChildrenRecursively))
	comboUlRate := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboUlRate", core.Qt__FindChildrenRecursively))
	spinPort := widgets.NewQSpinBoxFromPointer(s.QWidget.FindChild("spinPort", core.Qt__FindChildrenRecursively))
	comboTPB := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboTPB", core.Qt__FindChildrenRecursively))
	comboEZTV := widgets.NewQComboBoxFromPointer(s.QWidget.FindChild("comboEZTV", core.Qt__FindChildrenRecursively))
	checkKeepFiles := widgets.NewQCheckBoxFromPointer(s.QWidget.FindChild("checkKeepFiles", core.Qt__FindChildrenRecursively))
	lineDlPath := widgets.NewQLineEditFromPointer(s.QWidget.FindChild("lineDlPath", core.Qt__FindChildrenRecursively))

	limit := comboLimit.CurrentText()
	days := comboDays.CurrentText()
	fullscreen := checkFullscreen.IsChecked()
	stopscreensaver := checkStopScreensaver.IsChecked()
	volumemax := sliderVolumeMax.Value()
	subtitles := groupSubtitles.IsChecked()
	subcolor := pushColor.Palette().Color2(gui.QPalette__Button).Name()
	language := comboLanguage.CurrentText()
	codepage := comboCodepage.CurrentText()
	subscale := spinScale.Value()
	encryption := checkEncryption.IsChecked()
	dlrate := comboDlRate.CurrentText()
	ulrate := comboUlRate.CurrentText()
	port := spinPort.Value()
	tpbHost := comboTPB.CurrentText()
	eztvHost := comboEZTV.CurrentText()
	keep := checkKeepFiles.IsChecked()
	dlpath := lineDlPath.Text()

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
	s.DlRate = core.NewQVariant14(dlrate).ToInt(false)
	s.UlRate = core.NewQVariant14(ulrate).ToInt(false)
	s.Port = core.NewQVariant7(port).ToInt(false)
	s.TPBHost = core.NewQVariant14(tpbHost).ToString()
	s.EZTVHost = core.NewQVariant14(eztvHost).ToString()
	s.KeepFiles = core.NewQVariant11(keep).ToBool()
	s.DlPath = core.NewQVariant14(dlpath).ToString()
}

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
	c.TorrentConnectBoost = 500
	c.ConnectionSpeed = 500
	c.PeerConnectTimeout = 2
	c.RequestTimeout = 2
	c.MaxDownloadRate = s.DlRate
	c.MaxUploadRate = s.UlRate
	c.MinReconnectTime = 60
	c.MaxFailCount = 3
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
