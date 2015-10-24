#-*- coding:utf-8 -*-

import time
import subprocess

import requests
from PyQt5.QtCore import QThread, pyqtSignal

from bukanir.logger import log
from bukanir.opts import VERBOSE
from bukanir import T2H_URL, T2H_BIND, T2H_BINARY


class Torrent(QThread):

    started = pyqtSignal()

    def __init__(self, parent=None, magnet=None):
        QThread.__init__(self, parent)
        self.proc = None
        self.parent = parent
        self.magnet = magnet

    def run(self):
        self.proc_open()
        self.started.emit()

    def proc_open(self):
        if self.parent.settings.keep and self.parent.settings.dlpath:
            dlpath = self.parent.settings.dlpath
        else:
            dlpath = self.parent.tmpdir

        cmd = [T2H_BINARY, "-bind", T2H_BIND, "-dl-path", dlpath, "-uri", self.magnet,
               "-encryption", str(self.parent.settings.encryption), "-dl-rate", str(self.parent.settings.dlrate),
               "-ul-rate", str(self.parent.settings.ulrate), "-listen-port", str(self.parent.settings.port)]
        if self.parent.settings.keep and self.parent.settings.dlpath:
            cmd += ["-keep-complete"]
        if VERBOSE:
            cmd += ["-verbose"]
            log.info(" ".join(cmd))
        self.proc = subprocess.Popen(cmd)

    def stop(self):
        try:
            if not self.proc.poll():
                requests.get("%s/shutdown" % T2H_URL)
                time.sleep(0.5)
            self.proc.terminate()
        except:
            pass
