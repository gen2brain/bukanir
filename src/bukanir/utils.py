#-*- coding:utf-8 -*-

import os

import psutil
import requests
from PyQt5.QtCore import QFile

from bukanir.logger import log


def which(prog):
    def is_exe(fpath):
        return os.path.exists(fpath) and os.access(fpath, os.X_OK)
    fpath, fname = os.path.split(prog)
    if fpath:
        if is_exe(prog):
            return prog
    else:
        for path in os.environ["PATH"].split(os.pathsep):
            filename = os.path.join(path, prog)
            if is_exe(filename):
                return filename
    return None


def kill_proc_tree(pid, including_parent=True):
    parent = psutil.Process(pid)
    children = parent.children(recursive=True)
    for child in children:
        child.kill()
    if including_parent:
        parent.kill()


def read_resource(rc):
    from bukanir.ui import assets_rc
    fd = QFile(rc)
    fd.open(QFile.ReadOnly)
    data = bytes(fd.readAll()).decode()
    fd.close()
    return data


def get_status(status):
    return [
        status["name"],
        "%.2f%% %s" % (status["progress"] * 100, status["state_str"]),
        "D:%(download_rate).2fkB/s U:%(upload_rate).2fkB/s S:%(num_seeds)d (%(total_seeds)s) P:%(num_peers)d (%(total_peers)s)" % status,
    ]


def get_json(url):
    try:
        r = requests.get(url)
        if r.status_code == 200:
            return r.json()
    except Exception as err:
        log.warn("%s: %s" % (url, err))
    return None


def get_view_html(movies, mid=0):
    html = ""
    template = read_resource("://assets/view.html")
    for n, movie in enumerate(movies):
        if movie["category"] == 205 or movie["category"] == 208:
            desc = "S%02dE%02d" % (movie["season"], movie["episode"])
        else:
            desc = movie["year"]
        if movie["category"] == 207 or movie["category"] == 208:
            try:
                if movie["quality"]:
                    desc = "%s (%sp)" % (desc, movie["quality"])
            except KeyError:
                pass
        html += '''
    <div class="box">
        <div class="image" id="m%s">
            <a href="movie%d-%s"><img data-original="%s" width="200" height="300"/>
            <div class="text">%s<br/><small>%s</small></div></a>
        </div>
    </div>''' % (movie["id"]+movie["seeders"], n, movie["id"], movie["posterMedium"], movie["title"], desc)
    template = template.replace("{HTML}", html)
    template = template.replace("{ID}", str(mid))
    return template


def get_summary_html(movie, summary):
    if movie["category"] == 205 or movie["category"] == 208:
        desc = "S%02dE%02d" % (movie["season"], movie["episode"])
    else:
        desc = "(%s)" % movie["year"] if movie["year"] else ""

    rating = "%s/10" % summary["rating"] if summary["rating"] else ""
    runtime = "%smin / " % summary["runtime"] if summary["runtime"] else ""

    genre = ""
    if summary["genre"]:
        genre = ", ".join(summary["genre"])

    director = ""
    if summary["director"]:
        director = "<em>Director:</em> " + summary["director"]

    cast = ""
    cast_length = 0 if summary["cast"] is None else len(summary["cast"])
    if cast_length >= 4:
        cast = "<em>Cast:</em> " + ", ".join(summary["cast"][:4]) + "..."
    elif cast_length != 0:
        cast = "<em>Cast:</em> " + ", ".join(summary["cast"][:cast_length])

    template = read_resource("://assets/summary.html")

    if summary["video"]:
        video = '<a class="button right" href="https://www.youtube.com/watch?v=%s">TRAILER</a>' % summary["video"]
    else:
        video = ""
    template = template.replace("{VIDEO}", video)

    template = template.replace("{TITLE}", movie["title"])
    template = template.replace("{MAGNET}", movie["magnetLink"])
    template = template.replace("{POSTER}", movie["posterXLarge"])
    template = template.replace("{YEAR}", desc)
    template = template.replace("{RUNTIME}", runtime)
    template = template.replace("{SIZE}", movie["sizeHuman"])
    template = template.replace("{GENRE}", genre)
    template = template.replace("{RATING}", rating)
    template = template.replace("{TAGLINE}", summary["tagline"])
    template = template.replace("{DIRECTOR}", director)
    template = template.replace("{CAST}", cast)
    template = template.replace("{OVERVIEW}", summary["overview"])
    return template
