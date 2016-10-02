#!/usr/bin/env bash

mkdir -p build
go generate

# linux/amd64
CHROOT="$HOME/chroot"
INCPATH="$CHROOT/usr/include/qt5"
PLUGPATH="$CHROOT/usr/lib/qt5/plugins"
export CC=gcc CXX=g++
export PKG_CONFIG_PATH="$CHROOT/usr/lib/pkgconfig"
export PKG_CONFIG_LIBDIR="$CHROOT/usr/lib/pkgconfig"
export LIBRARY_PATH="$CHROOT/usr/lib:$CHROOT/lib"

CGO_CFLAGS="-Wno-unused-parameter -Wno-unused-variable" \
CGO_CXXFLAGS="-I$INCPATH -I$INCPATH/QtCore -I$INCPATH/QtGui -I$INCPATH/QtWidgets -I$INCPATH/QtUiTools -Wno-unused-parameter -Wno-unused-variable" \
CGO_CXXFLAGS="$CGO_CXXFLAGS -pipe -O2 -std=gnu++11 -Wall -W -D_REENTRANT -DQT_NO_DEBUG -DQT_CORE_LIB -DQT_GUI_LIB -DQT_WIDGETS_LIB -DQT_UITOOLS_LIB -fPIC" \
CGO_LDFLAGS="-L$CHROOT/usr/lib -L$CHROOT/lib -L$PLUGPATH/platforms -L$PLUGPATH/generic" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Core -lpthread -licui18n -licuuc -licudata -lz -lpcre16 -ldouble-conversion -lm -ldl -lgthread-2.0 -lglib-2.0 -lrt" \
CGO_LDFLAGS="$CGO_LDFLAGS -ljpeg -lQt5Widgets -lQt5Gui -lQt5Core -lpthread -lpng -lharfbuzz -lz -lGL" \
CGO_LDFLAGS="$CGO_LDFLAGS -lmpv -lasound -lGL -lEGL -lGL -ljpeg -lass -lharfbuzz -lavformat -lswscale -lavdevice -lavfilter -lavcodec -lavutil -ldl -lm -lrt -lswresample -lavresample -lpostproc -lluajit-5.1 -lEGL -lGLESv2 -lX11 -lXext -lXinerama -lXrandr -lXss -lz -lxcb -lXau -lXdmcp -lXfixes" \
CGO_LDFLAGS="$CGO_LDFLAGS -lfontconfig -lexpat -lfreetype -lfribidi -lbz2 -lpng16" \
CGO_LDFLAGS="$CGO_LDFLAGS -lvpx -lvorbisenc -lvorbis -logg -ltheoraenc -ltheoradec -logg -lmp3lame -lfdk-aac -lx264 -lx265 -lfaac" \
CGO_LDFLAGS="$CGO_LDFLAGS -lqxcb -lqevdevkeyboardplugin -lqevdevmouseplugin -lQt5XcbQpa -lQt5PlatformSupport -lQt5DBus -lX11-xcb -lXi -lxcb-render -lxcb-render-util -lXv -lSM -lICE -ldbus-1 -lxcb -lxcb-image -lxcb-icccm -lxcb-sync -lxcb-xfixes -lxcb-shm -lxcb-randr -lxcb-shape -lxcb-keysyms -lxcb-xinerama -lxcb-xkb -lxcb-util -lxcb-glx -lxkbcommon-x11 -lxkbcommon  -lfontconfig -lfreetype -ldl -lXrender -lXext -lX11 -lm -ludev -lmtdev -lEGL -lQt5Gui -ljpeg -lpng -lharfbuzz -lz -lbz2 -lGL -lQt5DBus -lQt5Core -lpthread -lGL" \
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags 'static minimal' -o build/bukanir.amd64 -v -x -ldflags "-s -w -extldflags=-Wl,--allow-multiple-definition"


# linux/386
CHROOT="$HOME/chroot"
INCPATH="$CHROOT/usr/include/qt5"
PLUGPATH="$CHROOT/usr/lib32/qt5/plugins"
export CC=gcc CXX=g++
export PKG_CONFIG_PATH="$CHROOT/usr/lib32/pkgconfig"
export PKG_CONFIG_LIBDIR="$CHROOT/usr/lib32/pkgconfig"
export LIBRARY_PATH="$CHROOT/usr/lib32:$CHROOT/lib32"

CGO_CFLAGS="-Wno-unused-parameter -Wno-unused-variable" \
CGO_CXXFLAGS="-I$INCPATH -I$INCPATH/QtCore -I$INCPATH/QtGui -I$INCPATH/QtWidgets -I$INCPATH/QtUiTools -Wno-unused-parameter -Wno-unused-variable" \
CGO_CXXFLAGS="$CGO_CXXFLAGS -pipe -O2 -std=gnu++11 -Wall -W -D_REENTRANT -DQT_NO_DEBUG -DQT_CORE_LIB -DQT_GUI_LIB -DQT_WIDGETS_LIB -DQT_UITOOLS_LIB -fPIC" \
CGO_LDFLAGS="-L$CHROOT/usr/lib -L$CHROOT/lib -L$PLUGPATH/platforms -L$PLUGPATH/generic" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Core -lpthread -licui18n -licuuc -licudata -lz -lpcre16 -ldouble-conversion -lm -ldl -lgthread-2.0 -lglib-2.0 -lrt" \
CGO_LDFLAGS="$CGO_LDFLAGS -ljpeg -lQt5Widgets -lQt5Gui -lQt5Core -lpthread -lpng -lharfbuzz -lz -lGL" \
CGO_LDFLAGS="$CGO_LDFLAGS -lmpv -lasound -lGL -lEGL -lGL -ljpeg -lass -lharfbuzz -lavformat -lswscale -lavdevice -lavfilter -lavcodec -lavutil -ldl -lm -lrt -lswresample -lavresample -lpostproc -lluajit-5.1 -lEGL -lGLESv2 -lX11 -lXext -lXinerama -lXrandr -lXss -lz -lxcb -lXau -lXdmcp -lXfixes" \
CGO_LDFLAGS="$CGO_LDFLAGS -lfontconfig -lexpat -lfreetype -lfribidi -lbz2 -lpng16" \
CGO_LDFLAGS="$CGO_LDFLAGS -lvpx -lvorbisenc -lvorbis -logg -ltheoraenc -ltheoradec -logg -lmp3lame -lfdk-aac -lx264 -lx265 -lfaac" \
CGO_LDFLAGS="$CGO_LDFLAGS -lqxcb -lqevdevkeyboardplugin -lqevdevmouseplugin -lQt5XcbQpa -lQt5PlatformSupport -lQt5DBus -lX11-xcb -lXi -lxcb-render -lxcb-render-util -lXv -lSM -lICE -ldbus-1 -lxcb -lxcb-image -lxcb-icccm -lxcb-sync -lxcb-xfixes -lxcb-shm -lxcb-randr -lxcb-shape -lxcb-keysyms -lxcb-xinerama -lxcb-xkb -lxcb-util -lxcb-glx -lxkbcommon-x11 -lxkbcommon  -lfontconfig -lfreetype -ldl -lXrender -lXext -lX11 -lm -ludev -lmtdev -lEGL -lQt5Gui -ljpeg -lpng -lharfbuzz -lz -lbz2 -lGL -lQt5DBus -lQt5Core -lpthread -lGL" \
CGO_ENABLED=1 GOOS=linux GOARCH=386 go build -o build/bukanir.386 -v -x


# windows/386
MINGW="/usr/i686-w64-mingw32"
INCPATH="$MINGW/usr/include"
PLUGPATH="$MINGW/usr/plugins"
export CC="i686-w64-mingw32-gcc" CXX="i686-w64-mingw32-g++"
export PKG_CONFIG="/usr/bin/i686-w64-mingw32-pkg-config"
export PKG_CONFIG_PATH="$MINGW/usr/lib/pkgconfig:$MINGW/usr/lib/pkgconfig"
export PKG_CONFIG_LIBDIR="$MINGW/usr/lib/pkgconfig:$MINGW/usr/lib/pkgconfig"

CGO_CFLAGS="-Wno-unused-parameter -Wno-unused-variable" \
CGO_CXXFLAGS="-I$INCPATH -I$INCPATH/QtCore -I$INCPATH/QtGui -I$INCPATH/QtWidgets  -I$INCPATH/QtUiTools -Wno-unused-parameter -Wno-unused-variable" \
CGO_CXXFLAGS="$CGO_CXXFLAGS -pipe -O2 -std=gnu++11 -Wall -W -D_REENTRANT -DQT_NO_DEBUG -DQT_CORE_LIB -DQT_GUI_LIB -DQT_WIDGETS_LIB -DQT_UITOOLS_LIB" \
CGO_LDFLAGS="-L$MINGW/usr/lib -L$MINGW/lib -L$PLUGPATH/platforms -L$PLUGPATH/generic" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5UiTools -lQt5Core -lz -lpcre16 -ldouble-conversion -lole32 -luuid -lws2_32 -ladvapi32 -lshell32 -luser32 -lkernel32 -lmpr" \
CGO_LDFLAGS="$CGO_LDFLAGS -ljpeg -lQt5Widgets -lQt5Gui -lws2_32 -lpng -lharfbuzz -lz -lopengl32 -lcomdlg32 -loleaut32 -limm32 -lwinmm -lglu32 -lopengl32" \
CGO_LDFLAGS="$CGO_LDFLAGS -lgdi32 -lole32 -luuid -lws2_32 -ladvapi32 -lshell32 -luser32 -lkernel32 -lmpr  -ldouble-conversion -lqtpcre -lqtharfbuzzng" \
CGO_LDFLAGS="$CGO_LDFLAGS -lmpv -lopengl32 -ljpeg -llua -lass -lharfbuzz -lavformat -lswscale -lavdevice -lgdi32 -lopengl32 -lshlwapi -lvfw32 -lwinmm -lstrmiids -lole32 -loleaut32" \
CGO_LDFLAGS="$CGO_LDFLAGS -lavfilter -lavcodec -lavutil -lswresample -lavresample -lpostproc -lavrt -lsecur32 -lcomctl32 -liconv -luuid -ladvapi32 -lole32 -loleaut32 -lshell32 -lws2_32 -ldwmapi -lz" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Widgets -lQt5Gui -lQt5Core -lfontconfig -lexpat -lfreetype -lbz2 -lpng16 -lgdi32" \
CGO_LDFLAGS="$CGO_LDFLAGS -lvpx -lvorbisenc -lvorbis -logg -ltheoraenc -ltheoradec -logg -lmp3lame -lfdk-aac -lx264" \
CGO_LDFLAGS="$CGO_LDFLAGS -lqwindows  -lgdi32 -limm32 -loleaut32 -lwinmm -lQt5PlatformSupport -lfontconfig -lfreetype -lQt5Gui -lole32 -ljpeg -lpng -lharfbuzz -lz -lbz2 -lopengl32 -lQt5Core -lopengl32 -ldouble-conversion -lole32 -lgdi32" \
CC_FOR_TARGET="i686-w64-mingw32-gcc" CXX_FOR_TARGET="i686-w64-mingw32-g++" \
CGO_ENABLED=1 GOOS=windows GOARCH=386 go build -tags 'static minimal' -o build/bukanir.exe -v -x -ldflags "-H=windowsgui -linkmode external -s -w -extldflags=-static"
