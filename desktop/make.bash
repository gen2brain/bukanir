#!/usr/bin/env bash

mkdir -p build
#go generate && qtminimal

# linux/amd64
CHROOT="$HOME/chroot"
INCPATH="$CHROOT/usr/include/qt5"
PLUGPATH="$CHROOT/usr/lib/qt5/plugins"
export CC=gcc CXX=g++
export PKG_CONFIG_PATH="$CHROOT/usr/lib/pkgconfig"
export PKG_CONFIG_LIBDIR="$CHROOT/usr/lib/pkgconfig"
export LIBRARY_PATH="$CHROOT/usr/lib:$CHROOT/lib"

CGO_CFLAGS="-Wno-unused-parameter -Wno-unused-variable -Wno-return-type" \
CGO_CXXFLAGS="-I$CHROOT/usr/include -I$INCPATH -I$INCPATH/QtCore -I$INCPATH/QtGui -I$INCPATH/QtWidgets -I$INCPATH/QtNetwork -I$INCPATH/libtorrent -Wno-unused-parameter -Wno-unused-variable -Wno-return-type" \
CGO_CXXFLAGS="$CGO_CXXFLAGS -pipe -O2 -std=gnu++11 -Wall -W -D_REENTRANT -DQT_NO_DEBUG -DQT_DBUS_LIB -DQT_CORE_LIB -DQT_GUI_LIB -DQT_WIDGETS_LIB -DQT_NETWORK_LIB -fPIC" \
CGO_LDFLAGS="-L$CHROOT/usr/lib -L$CHROOT/lib -L$PLUGPATH/platforms -L$PLUGPATH/generic -L$PLUGPATH/imageformats" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Core -lpthread -lz -lpcre16 -ldouble-conversion -lm -ldl -lgthread-2.0 -lglib-2.0 -lrt" \
CGO_LDFLAGS="$CGO_LDFLAGS -ljpeg -lQt5Network -lQt5Widgets -lQt5Gui -lQt5Core -lpthread -lpng -lharfbuzz -lz -lGL" \
CGO_LDFLAGS="$CGO_LDFLAGS -lmpv -lasound -lGL -lEGL -lGL -lSDL2 -ljpeg -lass -lharfbuzz -lfreetype -luchardet -lavformat -lswscale -lavdevice -lavfilter -lavcodec -lavutil -ldl -lm -lrt" \
CGO_LDFLAGS="$CGO_LDFLAGS -lswresample -lavresample -lpostproc -lluajit-5.1 -lEGL -lGLESv2 -lX11 -lXext -lXinerama -lXrandr -lXss -lz -lxcb -lXau -lXdmcp -lXfixes" \
CGO_LDFLAGS="$CGO_LDFLAGS -lfontconfig -lexpat -lfreetype -lfribidi -lbz2 -lpng16 -ltorrent-rasterbar -lboost_system -lssl -lcrypto -lz" \
CGO_LDFLAGS="$CGO_LDFLAGS -lvpx -lvorbisenc -lvorbis -logg -ltheoraenc -ltheoradec -logg -lmp3lame -lfdk-aac -lx264 -lx265" \
CGO_LDFLAGS="$CGO_LDFLAGS -lqxcb -lqgif -lqjpeg -lqevdevkeyboardplugin -lqevdevmouseplugin -lQt5XcbQpa -lQt5InputSupport -lQt5DeviceDiscoverySupport -lQt5ThemeSupport -lQt5GlxSupport -lQt5FontDatabaseSupport" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5ServiceSupport -lQt5EventDispatcherSupport -lQt5DBus -lX11-xcb -lXi -lxcb-render -lxcb-render-util -lXv -lSM -lICE -lbsd" \
CGO_LDFLAGS="$CGO_LDFLAGS -lXxf86vm -lXcursor -ldbus-1 -lxcb -lxcb-image -lxcb-icccm -lxcb-sync -lxcb-xfixes -lxcb-shm -lxcb-randr -lxcb-shape -lxcb-keysyms" \
CGO_LDFLAGS="$CGO_LDFLAGS -lxcb-xinerama -lxcb-xkb -lxcb-util -lxcb-glx -lxkbcommon-x11 -lxkbcommon  -lfontconfig -lexpat -lfreetype -ldl -lXrender -lXext -lX11 -lm" \
CGO_LDFLAGS="$CGO_LDFLAGS -ludev -lmtdev -lEGL -lQt5Gui -ljpeg -lpng -lharfbuzz -lfreetype -lz -lbz2 -lGL -lQt5DBus -lQt5Core -lpthread -lGL" \
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -tags 'static minimal' -o build/bukanir.amd64 -v -x -ldflags "-linkmode external -s -w"


# windows/386
#MINGW="/usr/i686-w64-mingw32"
#INCPATH="$MINGW/usr/include"
#PLUGPATH="$MINGW/usr/plugins"
#export CC="i686-w64-mingw32-gcc" CXX="i686-w64-mingw32-g++"
#export PKG_CONFIG="/usr/bin/i686-w64-mingw32-pkg-config"
#export PKG_CONFIG_PATH="$MINGW/usr/lib/pkgconfig:$MINGW/usr/lib/pkgconfig"
#export PKG_CONFIG_LIBDIR="$MINGW/usr/lib/pkgconfig:$MINGW/usr/lib/pkgconfig"

##CGO_LDFLAGS="$CGO_LDFLAGS -lmpv -lopengl32 -lSDL2 -ljpeg -lass -lharfbuzz -luchardet -lavformat -lswscale -lavdevice -llua -lversion -lgdi32 -lopengl32 -lshlwapi -lvfw32 -lwinmm -lstrmiids -lole32 -loleaut32" \
##CGO_LDFLAGS="$CGO_LDFLAGS -lavfilter -lavcodec -lavutil -lswresample -lavresample -lpostproc -lavrt -lsecur32 -lcomctl32 -liconv -luuid -ladvapi32 -lole32 -loleaut32 -lshell32 -lws2_32 -ldwmapi -lz" \
##CGO_LDFLAGS="$CGO_LDFLAGS -lvpx -lvorbisenc -lvorbis -logg -ltheoraenc -ltheoradec -logg -lmp3lame -lfdk-aac -lx264" \
#CGO_CFLAGS="-Wno-unused-parameter -Wno-unused-variable" \
#CGO_CXXFLAGS="-I$INCPATH -I$INCPATH/QtCore -I$INCPATH/QtGui -I$INCPATH/QtWidgets -I$INCPATH/QtNetwork -I$INCPATH/libtorrent -Wno-unused-parameter -Wno-unused-variable" \
#CGO_CXXFLAGS="$CGO_CXXFLAGS -pipe -O2 -std=gnu++11 -Wall -W -D_REENTRANT -DQT_NO_DEBUG -DQT_CORE_LIB -DQT_GUI_LIB -DQT_WIDGETS_LIB -DQT_NETWORK_LIB" \
#CGO_LDFLAGS="-L$MINGW/usr/lib -L$MINGW/lib -L$PLUGPATH/platforms -L$PLUGPATH/generic" \
#CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Core -lz -lpcre16 -ldouble-conversion -lole32 -luuid -lws2_32 -ladvapi32 -lshell32 -luser32 -lkernel32 -lmpr" \
#CGO_LDFLAGS="$CGO_LDFLAGS -ljpeg -lQt5Network -lQt5Widgets -lQt5Gui -lws2_32 -lpng -lharfbuzz -lz -lopengl32 -lcomdlg32 -loleaut32 -limm32 -lwinmm -lglu32 -lopengl32 -liphlpapi -ldnsapi" \
#CGO_LDFLAGS="$CGO_LDFLAGS -lgdi32 -lole32 -luuid -lws2_32 -ladvapi32 -lshell32 -luser32 -lkernel32 -lmpr  -ldouble-conversion -lqtpcre -lqtharfbuzzng" \
#CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Network -lQt5Widgets -lQt5Gui -lQt5Core -lfontconfig -lexpat -lfreetype -lbz2 -lpng16 -lgdi32" \
#CGO_LDFLAGS="$CGO_LDFLAGS -ltorrent-rasterbar -lssl -lcrypto -lgdi32 -lcrypt32 -lboost_system -lws2_32 -lmswsock -lgdi32" \
#CGO_LDFLAGS="$CGO_LDFLAGS -lqwindows  -lgdi32 -limm32 -loleaut32 -lwinmm -lQt5PlatformSupport -lfontconfig -lfreetype -lQt5Gui -lole32 -ljpeg -lpng -lharfbuzz -lz -lbz2 -lopengl32 -lQt5Core -lopengl32 -ldouble-conversion -lole32 -lgdi32" \
#CC_FOR_TARGET="i686-w64-mingw32-gcc" CXX_FOR_TARGET="i686-w64-mingw32-g++" \
#CGO_ENABLED=1 GOOS=windows GOARCH=386 go build -tags 'static minimal' -o build/bukanir.exe -v -x -ldflags "-H=windowsgui -linkmode external -s -w"
