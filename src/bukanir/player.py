#-*- coding:utf-8 -*-

import time
import subprocess

import requests
from PyQt5.QtCore import QThread, pyqtSignal

from bukanir import PLAYER, SUB, T2H_URL, HTTP_URL, DOWNLOAD_REQUIRED
from bukanir.opts import VERBOSE
from bukanir.logger import log
from bukanir.utils import get_json, kill_proc_tree


class Player(QThread):

    started = pyqtSignal(str)
    finished = pyqtSignal(int)

    def __init__(self, url=None, subtitles=None, parent=None):
        QThread.__init__(self, parent)
        self.parent = parent
        self.stopped = False
        self.subtitles = subtitles
        self.url = url
        self.proc = None
        self.title = None

    def run(self):
        self.stopped = False

        if self.url is None:
            if self.loop():
                files = get_json("%s/ls" % T2H_URL)["files"]
                file = max(files, key=lambda x: x["size"])
                self.url = file["url"]

        if self.url is not None and "youtube.com" in self.url:
            self.url = self.get_yt_url()

        if self.url:
            self.proc_open()
            time.sleep(3)
            self.started.emit(self.title)
            self.proc.wait()
            self.url,self.title,self.subtitles = None,None,None
            self.finished.emit(self.proc.returncode)

    def loop(self):
        if not self.wait():
            return False

        ready = False
        while not ready:
            if self.stopped:
               return False
            status = get_json("%s/status" % T2H_URL)
            if status:
                if self.parent:
                    self.parent.status_changed.emit(status)
                if status["state"] >= 3 and not ready:
                    downloaded = status["total_download"] / (1024 * 1024)
                    if downloaded >= DOWNLOAD_REQUIRED:
                        self.title = status["name"]
                        ready = True
                        break
            time.sleep(1)
        return True

    def wait(self):
        start = time.time()
        while(time.time() - start) < 20 and not self.stopped:
            try:
                get_json("%s/status" % T2H_URL)["state"]
                return True
            except:
                pass
            time.sleep(1)
        return False

    def proc_open(self):
        cmd = PLAYER + [u"%s" % self.url]

        if "mpv" in PLAYER[0]:
            cmd += ["--no-ytdl"]
            if not VERBOSE:
                cmd += ["--really-quiet"]
            if self.subtitles:
                for sub in self.subtitles:
                    cmd += SUB + [sub]
                if self.parent.settings.codepage != "auto":
                    cmd += ["--sub-codepage", self.parent.settings.codepage.lower()]
            if self.title:
                cmd += ["--title", u"%s" % self.title]
                cmd += ["--media-title", u"%s" % self.title]
        elif "mplayer" in PLAYER[0]:
            if not VERBOSE:
                cmd += ["-really-quiet"]
            if self.subtitles:
                cmd += SUB + [",".join(self.subtitles)]
                if self.parent.settings.codepage.lower() != "auto":
                    cmd += ["-sub-cp", self.parent.settings.codepage.lower()]
            if self.title:
                cmd += ["-title", u"%s" % self.title]

        if VERBOSE:
            log.info(subprocess.list2cmdline(cmd))

        self.proc = subprocess.Popen(cmd)

    def get_yt_url(self):
        url = "%s/trailer?i=%s" % (
            HTTP_URL, self.url.replace("https://www.youtube.com/watch?v=", ""))
        r = requests.get(url)
        if r.status_code == 200:
            return r.text
        return None

    def stop(self):
        self.stopped = True
        try:
            kill_proc_tree(self.proc.pid)
            self.proc.terminate()
        except:
            pass
