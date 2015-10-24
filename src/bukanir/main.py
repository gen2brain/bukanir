#-*- coding:utf-8 -*-

import json

from PyQt5.QtWebKitWidgets import QWebPage
from PyQt5.QtCore import Qt, pyqtSignal, QStringListModel, QUrl
from PyQt5.QtWidgets import QMainWindow, QApplication, QLabel, QProgressBar, QDialog, QCompleter
from PyQt5.QtGui import QMovie, QPalette, QColor
from PyQt5.QtNetwork import QNetworkAccessManager, QNetworkRequest

from bukanir.http import Http
from bukanir.client import Client
from bukanir.player import Player
from bukanir.torrent import Torrent
from bukanir.settings import Settings

from bukanir.ui.about_ui import Ui_AboutDialog
from bukanir.ui.mainwindow_ui import Ui_MainWindow

from bukanir import APP_VERSION, DOWNLOAD_REQUIRED, HTTP_URL
from bukanir.utils import get_status, get_view_html, get_summary_html


class MainWindow(QMainWindow, Ui_MainWindow):

    status_changed = pyqtSignal(dict)

    def __init__(self, tmpdir, optparse):
        QMainWindow.__init__(self, parent=None)
        self.setupUi(self)
        self.center_window()

        self.add_status()
        self.add_completer()
        self.set_loading()
        self.set_visible(combo=True, refresh=True)

        self.movie = {}
        self.summary = {}
        self.movies = []
        self.tmpdir = tmpdir

        self.http = Http(parent=self)
        self.client = Client(parent=self)
        self.player = Player(parent=self)
        self.torrent = Torrent(parent=self)
        self.settings = Settings(parent=self)

        self.webView.page().setLinkDelegationPolicy(QWebPage.DelegateAllLinks)
        self.webView.setContextMenuPolicy(Qt.NoContextMenu)

        self.connect_signals()

        self.http.start()

    def closeEvent(self, event):
        if self.player:
            self.player.stop()
        if self.http:
            self.http.stop()

    def add_status(self):
        self.labelStatus = QLabel()
        self.labelStatus.setIndent(2)
        self.progressBar = QProgressBar()
        palette = QPalette(self.progressBar.palette())
        palette.setColor(QPalette.Highlight, QColor(QColor(255, 167, 37)))
        self.progressBar.setPalette(palette)
        self.statusBar.addWidget(self.labelStatus)
        self.statusBar.addPermanentWidget(self.progressBar)

    def add_completer(self):
        self.completer = QCompleter()
        self.model = QStringListModel()
        self.completer.setCaseSensitivity(Qt.CaseInsensitive)
        self.completer.setMaxVisibleItems(15)
        self.completer.setModel(self.model)
        self.lineEdit.setCompleter(self.completer)

    def center_window(self):
        size = self.size()
        desktop = QApplication.desktop()
        width, height = size.width(), size.height()
        dwidth, dheight = desktop.width(), desktop.height()
        cw, ch = (dwidth/2)-(width/2), (dheight/2)-(height/2)
        self.move(cw, ch)

    def connect_signals(self):
        self.http.started.connect(self.on_http_started)
        self.torrent.started.connect(self.on_torrent_started)
        self.player.started.connect(self.on_player_started)
        self.player.finished.connect(self.on_player_finished)
        self.client.finished.connect(self.on_client_finished)
        self.status_changed.connect(self.on_status_changed)
        self.lineEdit.textEdited.connect(self.on_text_edited)

        self.backButton.clicked.connect(self.on_back_clicked)
        self.searchButton.clicked.connect(self.on_search_clicked)
        self.refreshButton.clicked.connect(self.on_refresh_clicked)
        self.settingsButton.clicked.connect(self.on_settings_clicked)
        self.aboutButton.clicked.connect(self.on_about_clicked)
        self.lineEdit.returnPressed.connect(self.on_search_clicked)
        self.progressBar.valueChanged.connect(self.on_progressbar_changed)
        self.comboBox.currentIndexChanged.connect(self.on_index_changed)
        self.webView.linkClicked.connect(self.on_link_clicked)

    def on_http_started(self):
        self.set_visible(loading=True, combo=True, refresh=True,
                         status="Downloading movies metadata...")
        self.client.mode = "top"
        self.client.start()
        self.lineEdit.setFocus(True)

    def on_torrent_started(self):
        self.client.mode = "subtitles"
        self.client.summary = self.summary
        self.client.tmpdir = self.tmpdir
        self.client.start()

    def on_player_started(self, title):
        self.labelStatus.setText("Playing: %s" % title)

    def on_about_clicked(self):
        About(self)

    def on_settings_clicked(self):
        self.settings.show()

    def on_search_clicked(self):
        self.set_visible(loading=True, combo=True, refresh=True,
                         status="Downloading movies metadata...")

        query = self.lineEdit.text().strip()
        if query:
            self.client.mode = "search"
            self.client.query = query
        else:
            self.client.mode = "top"
        self.client.start()

    def on_refresh_clicked(self):
        self.refreshButton.setDisabled(True)
        combo = True if self.client.last_mode == "top" else False
        self.set_visible(loading=True, combo=combo, refresh=True,
                         status="Downloading movies metadata...")

        self.client.mode = self.client.last_mode
        self.client.refresh = True
        self.client.start()

    def on_link_clicked(self, url):
        url = url.toString()
        if url.startswith("magnet"):
            self.player.stop()
            self.torrent.stop()

            self.labelLoading.setVisible(True)

            self.torrent.magnet = url
            self.torrent.start()
        elif url.startswith("http"):
            self.player.url = url
            self.player.title = "%s (%s) - Trailer" % (
                self.movie["title"], self.movie["year"])
            self.player.start()
        else:
            self.labelLoading.setVisible(True)

            movie = self.get_movie(url)
            self.client.mode = "summary"
            self.client.movie = movie
            self.client.start()

    def on_back_clicked(self):
        self.player.stop()
        self.torrent.stop()

        combo = True if self.client.last_mode == "top" else False
        self.set_visible(refresh=True, combo=combo)

        html = get_view_html(self.movies, self.movie["id"]+self.movie["seeders"])
        self.webView.setHtml(html)

    def on_text_edited(self, text):
        if len(text) < 3: return
        url = "%s/autocomplete?q=%s&l=10" % (HTTP_URL, text)
        manager = QNetworkAccessManager(self)
        manager.get(QNetworkRequest(QUrl(url)))
        manager.finished.connect(self.on_manager_finished)

    def on_manager_finished(self, reply):
        data = json.loads(bytes(reply.readAll()).decode())
        self.model.setStringList([item["title"] for item in data])
        self.completer.complete()

    def on_player_finished(self, ret):
        self.torrent.stop()
        self.set_visible(back=True, status="Stopped")

        html = get_summary_html(self.movie, self.summary)
        self.webView.setHtml(html)

    def on_client_finished(self):
        if self.client.mode == "top":
            self.movies = self.client.results

            self.refreshButton.setDisabled(False)
            self.set_visible(combo=True, refresh=True)

            if self.client.results:
                html = get_view_html(self.client.results)
                self.webView.setHtml(html)
        elif self.client.mode == "search":
            self.movies = self.client.results

            self.refreshButton.setDisabled(False)
            self.set_visible(refresh=True)

            if self.client.results:
                html = get_view_html(self.client.results)
                self.webView.setHtml(html)
        elif self.client.mode == "summary":
            self.movie = self.client.movie
            self.summary = self.client.results

            self.set_visible(back=True, status="Release: " + self.client.movie["release"])

            if self.client.results:
                html = get_summary_html(self.client.movie, self.client.results)
                self.webView.setHtml(html)
        elif self.client.mode == "subtitles":
            self.player.subtitles = self.client.results
            self.player.start()

    def on_index_changed(self, index):
        if index == 0:
            self.client.category = "201"
        elif index == 1:
            self.client.category = "207"
        elif index == 2:
            self.client.category = "205"
        elif index == 3:
            self.client.category = "208"

        self.set_visible(loading=True, combo=True, refresh=True,
                         status="Downloading movies metadata...")

        self.client.mode = "top"
        self.client.start()

    def on_status_changed(self, status):
        self.labelLoading.setVisible(True)
        self.progressBar.setVisible(False)

        state = status["state"]
        self.labelStatus.setText(status["state_str"]+"...")
        self.labelStatus.setVisible(True)

        if state >= 3:
            self.progressBar.setValue(0)
            self.progressBar.setVisible(True)

            downloaded = status["total_download"] / (1024 * 1024)
            percent = float(downloaded) / float(DOWNLOAD_REQUIRED) * 100
            self.progressBar.valueChanged.emit(int(percent))
            self.labelStatus.setText(get_status(status)[2])

            if downloaded >= DOWNLOAD_REQUIRED:
                self.labelLoading.setVisible(False)
                self.progressBar.setVisible(False)
                self.labelStatus.setText("Opening player...")

    def on_progressbar_changed(self, value):
        self.progressBar.setValue(value)

    def set_loading(self):
        movie = QMovie(":/images/loading.gif")
        self.labelLoading.setMovie(movie)
        self.labelLoading.setVisible(False)
        movie.start()

    def set_visible(self, status=None, loading=False, back=False,
                    progress=False, refresh=False, combo=False):
        if status:
            self.labelStatus.setText(status)
            self.labelStatus.setVisible(True)
        else:
            self.labelStatus.setVisible(False)

        self.labelLoading.setVisible(loading)
        self.backButton.setVisible(back)
        self.progressBar.setVisible(progress)
        self.refreshButton.setVisible(refresh)
        self.comboBox.setVisible(combo)

    def get_movie(self, url):
        for n, movie in enumerate(self.movies):
            if "movie%d-%s" % (n, movie["id"]) == url:
                return movie
        return None


class About(QDialog, Ui_AboutDialog):
    def __init__(self, parent):
        QDialog.__init__(self, parent)
        self.setupUi(self)
        self.setModal(True)
        html = self.textBrowser.toHtml()
        html = html.replace("APP_VERSION", APP_VERSION)
        self.textBrowser.setHtml(html)
        self.show()
