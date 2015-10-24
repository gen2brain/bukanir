#!/usr/bin/env python

import os
import sys
import shutil
import tarfile
import platform
import subprocess
from fnmatch import fnmatch
from os.path import join, dirname, realpath, basename
from distutils.core import setup, Command
from distutils.dep_util import newer
from distutils.command.build import build
from distutils.command.clean import clean
from distutils.dir_util import copy_tree

sys.path.insert(0, realpath("src"))
from bukanir import APP_VERSION
BASE_DIR = dirname(realpath(__file__))


class build_qt(Command):
    user_options = []

    def initialize_options(self):
        pass

    def finalize_options(self):
        pass

    def compile_ui(self, ui_file):
        from PyQt5 import uic
        py_file = os.path.splitext(ui_file)[0] + "_ui.py"
        if not newer(ui_file, py_file):
            return
        fp = open(py_file, "w")
        uic.compileUi(ui_file, fp, from_imports=True)
        fp.close()

    def compile_rc(self, qrc_file):
        import PyQt5
        py_file = os.path.splitext(qrc_file)[0] + "_rc.py"
        if not newer(qrc_file, py_file):
            return
        origpath = os.getenv("PATH")
        path = origpath.split(os.pathsep)
        path.append(dirname(PyQt5.__file__))
        os.putenv("PATH", os.pathsep.join(path))
        if subprocess.call(["pyrcc5", qrc_file, "-o", py_file]) > 0:
            self.warn("Unable to compile resource file %s" % qrc_file)
            if not os.path.exists(py_file):
                sys.exit(1)
        os.putenv('PATH', origpath)

    def run(self):
        basepath = join(dirname(__file__), 'src', 'bukanir', 'ui')
        for dirpath, dirs, filenames in os.walk(basepath):
            for filename in filenames:
                if filename.endswith('.ui'):
                    self.compile_ui(join(dirpath, filename))
                elif filename.endswith('.qrc'):
                    self.compile_rc(join(dirpath, filename))


class build_exe(Command):
    user_options = []
    dist_dir = join(BASE_DIR, "dist", "windows")

    def initialize_options(self):
        pass

    def finalize_options(self):
        pass

    def copy_files(self):
        dest_path = join(self.dist_dir, "bukanir")
        for file_name in ["AUTHORS", "COPYING", "README.md", "mpv.exe", "mpv.com"]:
            shutil.copy(join(BASE_DIR, file_name), dest_path)
        for dir_name in ["backend", "fonts", "mpv"]:
            copy_tree(join(BASE_DIR, dir_name), join(dest_path, dir_name))

        import PyQt5
        qt5_dir = dirname(PyQt5.__file__)
        qwindows = join(qt5_dir, "plugins", "platforms", "qwindows.dll")
        qwindows_dest = join(dest_path, "qt5_plugins", "platforms")
        if not os.path.exists(qwindows_dest):
            os.makedirs(qwindows_dest)
        shutil.copy(qwindows, qwindows_dest)

    def run_build_installer(self):
        iss_file = ""
        iss_in = join(self.dist_dir, "bukanir.iss.in")
        iss_out = join(self.dist_dir, "bukanir.iss")
        with open(iss_in, "r") as iss: data = iss.read()
        lines = data.split("\n")
        for line in lines:
            line = line.replace("{ICON}", realpath(join(self.dist_dir, "bukanir")))
            line = line.replace("{VERSION}", APP_VERSION)
            iss_file += line + "\n"
        with open(iss_out, "w") as iss: iss.write(iss_file)
        iscc = join(os.environ["ProgramFiles(x86)"], "Inno Setup 5", "ISCC.exe")
        subprocess.call([iscc, iss_out])

    def run(self):
        self.run_command("build_qt")
        set_rthook()
        run_build(self.dist_dir)
        self.copy_files()
        self.run_build_installer()


class build_bin(Command):
    user_options = []
    dist_dir = join(BASE_DIR, "dist", "linux")

    def initialize_options(self):
        pass

    def finalize_options(self):
        pass

    def copy_files(self):
        dest_path = join(self.dist_dir, "bukanir")
        if not os.path.exists(dest_path):
            os.mkdir(dest_path)
        for file_name in ["AUTHORS", "COPYING", "README.md"]:
            shutil.copy(join(BASE_DIR, file_name), dest_path)
        copy_tree(join(BASE_DIR, "xdg"), join(dest_path, "xdg"))
        shutil.move(join(self.dist_dir, "bukanir.bin"), join(dest_path, "bukanir"))

    def run_build_tarball(self):
        bin_dir = join(self.dist_dir, "bukanir")
        source_dir = "%s-%s" % (bin_dir, APP_VERSION)
        os.rename(bin_dir, source_dir)
        arch = platform.architecture()[0]
        output_file = join(self.dist_dir, "bukanir-%s-%s.tar.gz" % (APP_VERSION, arch))
        with tarfile.open(output_file, "w:gz") as tar:
            tar.add(source_dir, arcname=basename(source_dir))

    def run(self):
        self.run_command("build_qt")
        set_rthook()
        run_build(self.dist_dir)
        self.copy_files()
        self.run_build_tarball()


def run_build(dist_dir):
    import PyInstaller.building.build_main
    work_path = join(dist_dir, "build")
    spec_file = join(dist_dir, "bukanir.spec")
    os.environ["BASE_DIR"] = BASE_DIR
    os.environ["DIST_DIR"] = dist_dir
    opts = {"distpath": dist_dir, "workpath": work_path, "clean_build": True, "upx_dir": None}
    PyInstaller.building.build_main.main(None, spec_file, noconfirm=True, ascii=False, **opts)

def set_rthook():
    import PyInstaller
    hook_file = ""
    module_dir = dirname(PyInstaller.__file__)
    rthook = join(module_dir, "loader", "rthooks", "pyi_rth_qt5plugins.py")
    with open(rthook, "r") as hook: data = hook.read()
    if "import sip" not in data:
        lines = data.split("\n")
        for line in lines:
            hook_file += line + "\n"
            if "MEIPASS" in line:
                hook_file += "\nimport sip\n"
        with open(rthook, "w") as hook: hook.write(hook_file)


class clean_local(Command):
    pats = ['*.py[co]', '*_ui.py', '*_rc.py', '__pycache__']
    excludedirs = ['.git', 'build', 'dist']
    user_options = []

    def initialize_options(self):
        pass

    def finalize_options(self):
        pass

    def run(self):
        for e in self._walkpaths('.'):
            os.remove(e)

    def _walkpaths(self, path):
        for root, _dirs, files in os.walk(path):
            if any(root == join(path, e) or root.startswith(
                    join(path, e, '')) for e in self.excludedirs):
                continue
            for e in files:
                fpath = join(root, e)
                if any(fnmatch(fpath, p) for p in self.pats):
                    yield fpath


class mybuild(build):
    def run(self):
        self.run_command("build_qt")
        build.run(self)


class myclean(clean):
    def run(self):
        self.run_command("clean_local")
        clean.run(self)

cmdclass = {
    'build': mybuild,
    'build_qt': build_qt,
    'build_exe': build_exe,
    'build_bin': build_bin,
    'clean': myclean,
    'clean_local': clean_local
}

setup(
    name = "bukanir",
    version = APP_VERSION,
    description = "Bukanir streams movies from bittorrent magnet links",
    author = "Milan Nikolic",
    author_email = "gen2brain@gmail.com",
    license = "GNU GPLv3",
    url = "http://bukanir.com",
    packages = ["bukanir", "bukanir.ui"],
    package_dir = {"": "src"},
    scripts = ["bukanir", join("backend", "bukanir-http")],
    requires = ["PyQt5"],
    platforms = ["Linux", "Windows"],
    cmdclass = cmdclass,
    data_files = [
        ("share/pixmaps", ["xdg/bukanir.png"]),
        ("share/applications", ["xdg/bukanir.desktop"])
    ]
)
