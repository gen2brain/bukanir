# -*- coding: utf-8 -*-

from optparse import OptionParser

from bukanir import APP_NAME, APP_VERSION

usage = 'usage: %prog <option>'
parser = OptionParser(usage=usage, version="%s %s" % (APP_NAME.title(), APP_VERSION))
parser.add_option("-v", "--verbose", action="store_true", dest="verbose", help="show verbose output")
opts, args = parser.parse_args()

VERBOSE = opts.verbose
