# -*- coding: utf-8 -*-

from PyQt5.QtCore import QSettings
from PyQt5.QtWidgets import QDialog, QFileDialog

from bukanir.ui.settings_ui import Ui_Settings


class Settings(QDialog, Ui_Settings):

    def __init__(self, parent):
        QDialog.__init__(self, parent)
        self.parent = parent
        self.setupUi(self)

        self.q = QSettings("bukanir", "bukanir")
        self.q.setDefaultFormat(QSettings.IniFormat)

        self.checkKeepFiles.clicked.connect(self.on_keep_clicked)
        self.pushDlPath.clicked.connect(self.on_dlpath_clicked)

        self.set()

    def showEvent(self, event):
        self.set()

    def closeEvent(self, event):
        self.save()
        self.q.sync()

    def set_enabled(self, enabled):
        self.lineDlPath.setEnabled(enabled)
        self.pushDlPath.setEnabled(enabled)
        self.labelDlPath.setEnabled(enabled)

    def on_keep_clicked(self, state):
        self.set_enabled(state)

    def on_dlpath_clicked(self):
        dialog = QFileDialog()
        dialog.setFileMode(QFileDialog.Directory)
        path = dialog.getExistingDirectory(
            self, "Download path", self.lineDlPath.text(), QFileDialog.ShowDirsOnly)
        if path:
            self.lineDlPath.setText(path)

    def set(self):
        self.limit = self.q.value("limit", 30)
        self.days = self.q.value("days", 90)
        self.language = self.q.value("language", "English")
        self.codepage = str(self.q.value("codepage", "auto")).lower()
        self.dlrate = self.q.value("dlrate", -1)
        self.ulrate = self.q.value("ulrate", -1)
        self.port = self.q.value("port", 6881)

        self.keep = bool(int(self.q.value("keep", 0)))
        self.dlpath = self.q.value("dlpath", "")
        self.encryption = self.q.value("encryption", 1)
        enc = False if self.encryption==2 else True 

        self.comboLimit.setCurrentText(str(self.limit))
        self.comboDays.setCurrentText(str(self.days))
        self.comboLanguage.setCurrentText(self.language)
        self.comboCodepage.setCurrentText(self.codepage)
        self.comboDlRate.setCurrentText(str(self.dlrate))
        self.comboUlRate.setCurrentText(str(self.ulrate))
        self.spinPort.setValue(int(self.port))

        self.checkEncryption.setChecked(enc)
        self.checkKeepFiles.setChecked(self.keep)
        self.lineDlPath.setText(self.dlpath)
        self.set_enabled(self.keep)

    def save(self):
        self.limit = self.comboLimit.currentText()
        self.days = self.comboDays.currentText()
        self.language = self.comboLanguage.currentText()
        self.codepage = self.comboCodepage.currentText()
        self.dlrate = self.comboDlRate.currentText()
        self.ulrate = self.comboUlRate.currentText()
        self.port = self.spinPort.value()

        self.keep = self.checkKeepFiles.isChecked()
        self.dlpath = self.lineDlPath.text()
        self.encryption = 1 if self.checkEncryption.isChecked() else 2

        self.q.setValue("limit", int(self.limit))
        self.q.setValue("days", int(self.days))
        self.q.setValue("language", self.language)
        self.q.setValue("codepage", self.codepage)
        self.q.setValue("dlrate", int(self.dlrate))
        self.q.setValue("ulrate", int(self.ulrate))
        self.q.setValue("port", int(self.port))

        self.q.setValue("encryption", int(self.encryption))
        self.q.setValue("keep", int(self.keep))
        self.q.setValue("dlpath", self.dlpath)
