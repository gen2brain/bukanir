#-*- coding:utf-8 -*-

import logging

logging.getLogger("requests").setLevel(logging.WARNING)

ch = logging.StreamHandler()
ch.setLevel(logging.DEBUG)
ch.setFormatter(logging.Formatter("%(asctime)s %(message)s", "%Y/%m/%d %H:%M:%S"))

log = logging.getLogger("bukanir")
log.setLevel(logging.DEBUG)
log.addHandler(ch)
