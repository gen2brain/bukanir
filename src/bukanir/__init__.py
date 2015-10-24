#-*- coding:utf-8 -*-

import os
import sys

from bukanir.utils import which
from bukanir.logger import log

APP_NAME = "bukanir"
APP_VERSION = "1.9"

T2H_BIND = "127.0.0.1:5001"
HTTP_BIND = "127.0.0.1:7314"

T2H_URL = "http://%s" % T2H_BIND
HTTP_URL = "http://%s" % HTTP_BIND

DOWNLOAD_REQUIRED = 12


PLAYER = None
if sys.platform.startswith("linux"):
    if which("mpv"):
        PLAYER = [which("mpv"), "--fullscreen", "--quiet"]
        SUB = ["--sub-file"]
    elif which("mplayer"):
        PLAYER = [which("mplayer"), "-fs", "-quiet"]
        SUB = ["-sub"]
elif sys.platform == "win32":
    PLAYER = [os.path.join(os.getcwd(), "mpv.exe"), "--fullscreen", "--quiet"]
    SUB = ["--sub-file"]

if not PLAYER:
    log.fatal("This application needs mpv or MPlayer installed.")


T2H_BINARY = None
if sys.platform.startswith("linux"):
    if hasattr(sys, "_MEIPASS"):
        T2H_BINARY = os.path.join(sys._MEIPASS, "backend", "torrent2http")
    else:
        T2H_BINARY = which("torrent2http")
        if not T2H_BINARY:
            T2H_BINARY = os.path.join(os.getcwd(), "backend", "torrent2http")
elif sys.platform == "win32":
    if hasattr(sys, "_MEIPASS"):
        T2H_BINARY = os.path.join(sys._MEIPASS, "backend", "torrent2http.exe")
    else:
        T2H_BINARY = os.path.join(os.getcwd(), "backend", "torrent2http.exe")

if not T2H_BINARY:
    log.fatal("This application needs torrent2http binary.")


HTTP_BINARY = None
if sys.platform.startswith("linux"):
    if hasattr(sys, "_MEIPASS"):
        HTTP_BINARY = os.path.join(sys._MEIPASS, "backend", "bukanir-http")
    else:
        HTTP_BINARY = which("bukanir-http")
        if not HTTP_BINARY:
            HTTP_BINARY = os.path.join(os.getcwd(), "backend", "bukanir-http")
elif sys.platform == "win32":
    if hasattr(sys, "_MEIPASS"):
        HTTP_BINARY = os.path.join(sys._MEIPASS, "backend", "bukanir-http.exe")
    else:
        HTTP_BINARY = os.path.join(os.getcwd(), "backend", "bukanir-http.exe")

if not T2H_BINARY:
    log.fatal("This application needs bukanir-http binary.")


CACHE_DIR = os.path.expanduser(os.path.join("~", ".cache", APP_NAME))
if not os.path.isdir(CACHE_DIR):
    os.makedirs(CACHE_DIR)
