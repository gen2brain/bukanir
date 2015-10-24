#-*- coding:utf-8 -*-

import requests
from requests.utils import quote
from PyQt5.QtCore import QThread, pyqtSignal

from bukanir import HTTP_URL, CACHE_DIR
from bukanir.utils import get_json


class Client(QThread):

    finished = pyqtSignal()

    def __init__(self, parent=None):
        QThread.__init__(self, parent)
        self.parent = parent
        self.mode = "search"
        self.last_mode = None
        self.results = []
        self.movie = None
        self.summary = None
        self.category = "201"
        self.refresh = False
        self.query = None

    def run(self):
        if self.mode == "top":
            self.results = self.get_top()
        elif self.mode == "search":
            self.results = self.get_search()
        elif self.mode == "summary":
            self.results = self.get_summary()
        elif self.mode == "subtitles":
            self.results = self.get_subtitles()
        if self.mode == "top" or self.mode == "search":
            self.last_mode = self.mode
        self.finished.emit()

    def get_top(self):
        url = "%s/category?c=%s&t=%s" % (HTTP_URL, self.category, quote(CACHE_DIR))
        if int(self.parent.settings.limit) != -1:
            url += "&l=" + str(self.parent.settings.limit)
        if self.refresh:
            url += "&f=1"
            self.refresh = False
        return get_json(url)

    def get_search(self):
        url = "%s/search?q=%s&t=%s" % (HTTP_URL, quote(self.query), quote(CACHE_DIR))
        if int(self.parent.settings.limit) != -1:
            url += "&l=" + str(self.parent.settings.limit)
        if self.refresh:
            url += "&f=1"
            self.refresh = False
        return get_json(url)

    def get_summary(self):
        url = "%s/summary?i=%s&c=%s&s=%s&e=%s" % (
            HTTP_URL, self.movie["id"], self.movie["category"], self.movie["season"], self.movie["episode"])
        return get_json(url)

    def get_subtitles(self):
        url = "%s/subtitle?m=%s&y=%s&r=%s&l=%s&c=%s&s=%s&e=%s&i=%s" % (
            HTTP_URL, self.movie["title"], self.movie["year"], self.movie["release"], self.parent.settings.language,
            self.movie["category"], self.movie["season"], self.movie["episode"], self.summary["imdbId"])
        subs = get_json(url)

        subtitles = []
        subs_length = len(subs)
        if subs_length >= 3:
            for s in subs[:3]:
                url = "%s/unzipsubtitle?u=%s&d=%s" % (HTTP_URL, s["downloadLink"], self.parent.tmpdir)
                r = requests.get(url)
                if r.status_code == 200:
                    subtitles.append(r.text)
        elif subs_length != 0:
            for s in subs[:subs_length]:
                url = "%s/unzipsubtitle?u=%s&d=%s" % (HTTP_URL, s["downloadLink"], self.parent.tmpdir)
                r = requests.get(url)
                if r.status_code == 200:
                    subtitles.append(r.text)
        return subtitles
