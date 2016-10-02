#!/bin/sh

# For installation and usage, please refer to my blog post:
# http://derekstavis.github.io/posts/creating-a-installer-using-inno-setup-on-linux-and-mac-os-x/
#
# The MIT License (MIT)
#
# Copyright (c) 2014 Derek Willian Stavis
#
# Permission is hereby granted, free of charge, to any person obtaining a copy
# of this software and associated documentation files (the "Software"), to deal
# in the Software without restriction, including without limitation the rights
# to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
# copies of the Software, and to permit persons to whom the Software is
# furnished to do so, subject to the following conditions:
#
# The above copyright notice and this permission notice shall be included in
# all copies or substantial portions of the Software.
#
# THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
# IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
# FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
# AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
# LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
# OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
# THE SOFTWARE.


SCRIPTNAME=$1
INNO_BIN="Inno Setup 5/ISCC.exe"

# Check if variable is set
[ -z "$SCRIPTNAME" ] && { echo "Usage: $0 <SCRIPT_NAME>"; echo; exit 1; }

# Check if filename exist
[ ! -f "$SCRIPTNAME" ] && { echo "File not found. Aborting."; echo; exit 1; }

# Check if wine is present
command -v wine >/dev/null 2>&1 || { echo >&2 "I require wine but it's not installed. Aborting."; echo; exit 1; }

# Get Program Files path via wine command prompt
PROGRAMFILES=$(wine cmd /c 'echo %PROGRAMFILES%' 2>/dev/null)

# Translate windows path to absolute unix path
PROGFILES_PATH=$(winepath -u "${PROGRAMFILES}" 2>/dev/null)

# Get inno setup path
INNO_PATH="${PROGFILES_PATH%?}/${INNO_BIN}"

# Translate unix script path to windows path 
SCRIPTNAME=$(winepath -w "$SCRIPTNAME" 2> /dev/null)

# Check if Inno Setup is installed into wine
[ ! -f "$INNO_PATH" ] && { echo "Install Inno Setup 5 Quickstart before running this script."; echo; exit 1; }

# Compile!
WINEDLLOVERRIDES="mscoree,mshtml=" wine "$INNO_PATH" "$SCRIPTNAME"
