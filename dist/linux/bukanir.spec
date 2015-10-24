#-*- coding:utf-8 -*-

import os
from os.path import join

DIST_DIR = os.environ["DIST_DIR"]
BASE_DIR = os.environ["BASE_DIR"]

a = Analysis([join(BASE_DIR, 'bukanir')], pathex=[join(BASE_DIR, 'src')])
a.datas += [((join('backend', 'bukanir-http'), join('backend', 'bukanir-http'), 'DATA'))]
a.datas += [((join('backend', 'torrent2http'), join('backend', 'torrent2http'), 'DATA'))]

pyz = PYZ(a.pure)

exe = EXE(pyz,
    a.scripts,
    a.binaries,
    a.zipfiles,
    a.datas,
    name=join(DIST_DIR, 'bukanir', 'bukanir.bin'),
    debug=False,
    strip=True,
    upx=True,
    console=True)
