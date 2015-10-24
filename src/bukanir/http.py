#-*- coding:utf-8 -*-

import time
import subprocess

import requests
from PyQt5.QtCore import QThread, pyqtSignal

from bukanir.opts import VERBOSE
from bukanir import HTTP_URL, HTTP_BIND, HTTP_BINARY
from bukanir.utils import get_json


class Http(QThread):

    started = pyqtSignal()

    def __init__(self, parent=None):
        QThread.__init__(self, parent)
        self.parent = parent
        self.proc = None

    def run(self):
        self.proc_open()
        if self.wait():
            self.started.emit()

    def wait(self):
        start = time.time()
        while(time.time() - start) < 10:
            try:
                if get_json("%s/status" % HTTP_URL):
                    return True
            except:
                pass
            time.sleep(1)
        return False

    def proc_open(self):
        cmd = [HTTP_BINARY, "-bind", HTTP_BIND]
        if VERBOSE:
            cmd += ["--verbose"]
        self.proc = subprocess.Popen(cmd)

    def stop(self):
        try:
            if not self.proc.poll():
                requests.get("%s/shutdown" % HTTP_URL)
            self.proc.terminate()
        except:
            pass
