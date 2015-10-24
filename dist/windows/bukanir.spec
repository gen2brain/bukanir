#-*- coding:utf-8 -*-

import os
from os.path import join

DIST_DIR = os.environ["DIST_DIR"]
BASE_DIR = os.environ["BASE_DIR"]

a = Analysis(
    [join(BASE_DIR, 'bukanir')],
    hiddenimports=['pickle', 'PyQt5.Qt'],
    pathex=[join(BASE_DIR, 'src')],
    cipher=None)

pyz = PYZ(
    a.pure,
    a.zipped_data,
    cipher=None)

exe = EXE(pyz,
	a.scripts,
	exclude_binaries=True,
	name=join(DIST_DIR, 'build', 'pyi.win32', 'bukanir', 'bukanir.exe'),
	debug=False,
	strip=None,
	upx=True,
	console=False,
	icon=join(DIST_DIR, 'bukanir.ico'))

coll = COLLECT(exe,
	a.binaries,
	a.zipfiles,
	a.datas,
	strip=None,
	upx=True,
	name='bukanir')
