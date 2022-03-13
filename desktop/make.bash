#!/usr/bin/env bash

export GO111MODULE=off

mkdir -p build
#go generate && qtminimal

# linux/amd64
TOOLCHAIN="/usr/x86_64-pc-linux-gnu-static"
INCPATH="$TOOLCHAIN/usr/include"
PLUGPATH="$TOOLCHAIN/usr/plugins"
export CC=gcc
export CXX=g++
export LD=ld
export AR=ar
export PKG_CONFIG_PATH="$TOOLCHAIN/usr/lib64/pkgconfig"
export PKG_CONFIG_LIBDIR="$TOOLCHAIN/usr/lib64/pkgconfig"
export LIBRARY_PATH="$TOOLCHAIN/usr/lib64:$TOOLCHAIN/lib64"

CGO_CFLAGS="-Wno-unused-parameter -Wno-unused-variable -Wno-return-type -Wno-deprecated-declarations -Wno-strict-aliasing" \
CGO_CXXFLAGS="-I$TOOLCHAIN/usr/include -I$INCPATH -I$INCPATH/QtCore -I$INCPATH/QtGui -I$INCPATH/QtWidgets -I$INCPATH/QtNetwork -I$INCPATH/libtorrent -Wno-unused-parameter -Wno-unused-variable -Wno-return-type -Wno-deprecated-declarations -Wno-strict-aliasing" \
CGO_CXXFLAGS="$CGO_CXXFLAGS -pipe -O2 -std=gnu++11 -Wall -W -D_REENTRANT -DQT_NO_DEBUG -DQT_CORE_LIB -DQT_GUI_LIB -DQT_WIDGETS_LIB -DQT_NETWORK_LIB -fPIC" \
CGO_LDFLAGS="-L$TOOLCHAIN/usr/lib64 -L$TOOLCHAIN/lib64 -L/lib/x86_64-linux-gnu -L$PLUGPATH/platforms -L$PLUGPATH/generic -L$PLUGPATH/imageformats -L$PLUGPATH/styles" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Core -lpthread -lz -lpcre2-16 -ldouble-conversion -lm -ldl -lrt -lssl -lcrypto" \
CGO_LDFLAGS="$CGO_LDFLAGS -ljpeg -lQt5Network -lQt5Widgets -lQt5Gui -lQt5Core -lpthread -lpng -lharfbuzz -lz -lGL" \
CGO_LDFLAGS="$CGO_LDFLAGS -lmpv -lasound -lGL -lEGL -lGL -lSDL2 -ljpeg -lass -lharfbuzz -lfreetype -luchardet -lavformat -lswscale -lavdevice -lavfilter -lavcodec -lavutil -lvorbis -logg -ldl -lm -lrt" \
CGO_LDFLAGS="$CGO_LDFLAGS -lswresample -lavresample -lpostproc -lluajit-5.1 -lEGL -lGLESv2 -lX11 -lXext -lXinerama -lXrandr -lXss -lz -lxcb -lXau -lXdmcp -lXfixes" \
CGO_LDFLAGS="$CGO_LDFLAGS -lfontconfig -lexpat -lfreetype -lfribidi -lstdc++ -lbz2 -lpng16 -luuid -ltorrent-rasterbar -lboost_system -lssl -lcrypto -lz" \
CGO_LDFLAGS="$CGO_LDFLAGS -lqxcb -lqwayland-egl -lqcleanlooksstyle -lqplastiquestyle -lqgif -lqjpeg -lqevdevkeyboardplugin -lqevdevmouseplugin -lQt5Widgets -lQt5XcbQpa -lQt5WaylandClient -lQt5EglSupport -lQt5EdidSupport -lQt5InputSupport -lQt5DeviceDiscoverySupport -lQt5ThemeSupport -lQt5FontDatabaseSupport" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5ServiceSupport -lQt5EventDispatcherSupport -lX11-xcb -lXi -lxcb-render -lxcb-render-util -lXv -lSM -lICE" \
CGO_LDFLAGS="$CGO_LDFLAGS -lXxf86vm -lXcursor -lxcb -lxcb-image -lxcb-icccm -lxcb-sync -lxcb-xfixes -lxcb-shm -lxcb-randr -lxcb-shape -lxcb-keysyms" \
CGO_CFLAGS="-Wno-unused-parameter -Wno-unused-variable -Wno-return-type -Wno-deprecated-declarations -Wno-strict-aliasing" \
CGO_LDFLAGS="$CGO_LDFLAGS -lxcb-xinerama -lxcb-xkb -lxcb-util -lxkbcommon-x11 -lxkbcommon -lfontconfig -lexpat -lfreetype -luuid -ldl -lXrender -lXext -lX11 -lxcb -lXau -lXdmcp -lm -lxml2 -lwayland-client -lwayland-cursor -lwayland-egl -lffi" \
CGO_LDFLAGS="$CGO_LDFLAGS -lEGL -lQt5Gui -ljpeg -lpng -lharfbuzz -lfreetype -lz -lbz2 -lQt5XkbCommonSupport -lGL -lQt5Core -lpthread -lGL -lstdc++" \
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -tags 'static minimal wayland' -o build/bukanir.linux.amd64 -v -x -ldflags "-linkmode external -s -w"

# linux/amd64 static
TOOLCHAIN="/usr/x86_64-pc-linux-musl"
INCPATH="$TOOLCHAIN/usr/include"
PLUGPATH="$TOOLCHAIN/usr/plugins"
export CC=x86_64-pc-linux-musl-gcc
export CXX=x86_64-pc-linux-musl-g++
export LD=x86_64-pc-linux-musl-ld
export AR=x86_64-pc-linux-musl-ar
export PKG_CONFIG_PATH="$TOOLCHAIN/usr/lib/pkgconfig"
export PKG_CONFIG_LIBDIR="$TOOLCHAIN/usr/lib/pkgconfig"
export LIBRARY_PATH="$TOOLCHAIN/usr/lib:$TOOLCHAIN/lib64"

CGO_CFLAGS="-Wno-unused-parameter -Wno-unused-variable -Wno-return-type -Wno-deprecated-declarations -Wno-strict-aliasing" \
CGO_CXXFLAGS="-I$TOOLCHAIN/usr/include -I$INCPATH -I$INCPATH/QtCore -I$INCPATH/QtGui -I$INCPATH/QtWidgets -I$INCPATH/QtNetwork -I$INCPATH/libtorrent -Wno-unused-parameter -Wno-unused-variable -Wno-return-type -Wno-deprecated-declarations -Wno-strict-aliasing" \
CGO_CXXFLAGS="$CGO_CXXFLAGS -pipe -O2 -std=gnu++11 -Wall -W -D_REENTRANT -DQT_NO_DEBUG -DQT_CORE_LIB -DQT_GUI_LIB -DQT_WIDGETS_LIB -DQT_NETWORK_LIB -fPIC" \
CGO_LDFLAGS="-L$TOOLCHAIN/usr/lib -L$TOOLCHAIN/lib64 -L$PLUGPATH/platforms -L$PLUGPATH/generic -L$PLUGPATH/imageformats -L$PLUGPATH/styles" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Core -lpthread -lz -lpcre2-16 -ldouble-conversion -lm -ldl -lrt -lssl -lcrypto" \
CGO_LDFLAGS="$CGO_LDFLAGS -ljpeg -lQt5Network -lQt5Widgets -lQt5Gui -lQt5Core -lpthread -lpng -lharfbuzz -lz" \
CGO_LDFLAGS="$CGO_LDFLAGS -lmpv -lasound -lSDL2 -ljpeg -lass -lharfbuzz -lfreetype -luchardet -lavformat -lswscale -lavdevice -lavfilter -lavcodec -lavutil -ldl -lm -lrt" \
CGO_LDFLAGS="$CGO_LDFLAGS -lswresample -lavresample -lpostproc -lluajit-5.1 -lX11 -lXext -lXinerama -lXrandr -lXss -lz -lxcb -lXau -lXdmcp -lXfixes" \
CGO_LDFLAGS="$CGO_LDFLAGS -lfontconfig -lexpat -lfreetype -lfribidi -lbz2 -lpng16 -luuid -ltorrent-rasterbar -lboost_system -lssl -lcrypto -lz" \
CGO_LDFLAGS="$CGO_LDFLAGS -lqxcb -lqcleanlooksstyle -lqplastiquestyle -lqgif -lqjpeg -lqevdevkeyboardplugin -lqevdevmouseplugin -lQt5Widgets -lQt5XcbQpa -lQt5EdidSupport -lQt5InputSupport -lQt5DeviceDiscoverySupport -lQt5ThemeSupport -lQt5FontDatabaseSupport" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5ServiceSupport -lQt5EventDispatcherSupport -lX11-xcb -lxcb-render -lxcb-render-util -lXv" \
CGO_LDFLAGS="$CGO_LDFLAGS -lXxf86vm -lxcb -lxcb-image -lxcb-icccm -lxcb-sync -lxcb-xfixes -lxcb-shm -lxcb-randr -lxcb-shape -lxcb-keysyms" \
CGO_LDFLAGS="$CGO_LDFLAGS -lxcb-xinerama -lxcb-xkb -lxcb-util -lxkbcommon-x11 -lxkbcommon -lfontconfig -lexpat -lfreetype -ldl -lXrender -lXext -lX11 -lxcb -lXau -lXdmcp -lm -lxml2 -lffi" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Gui -ljpeg -lpng -lharfbuzz -lfreetype -lz -lbz2 -lQt5XkbCommonSupport -lQt5Core -lpthread -lstdc++" \
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -trimpath -tags 'static minimal' -o build/bukanir.linux.amd64.static -v -x -ldflags "-linkmode external -s -w '-extldflags=-static'"

# windows/386
TOOLCHAIN="/usr/i686-w64-mingw32"
INCPATH="$TOOLCHAIN/usr/include"
PLUGPATH="$TOOLCHAIN/usr/plugins"
export CC="i686-w64-mingw32-gcc"
export CXX="i686-w64-mingw32-g++"
export LD="i686-w64-mingw32-ld"
export AR="i686-w64-mingw32-ar"
export PKG_CONFIG="/usr/bin/i686-w64-mingw32-pkg-config"
export PKG_CONFIG_PATH="$TOOLCHAIN/usr/lib/pkgconfig:$TOOLCHAIN/usr/lib/pkgconfig"
export PKG_CONFIG_LIBDIR="$TOOLCHAIN/usr/lib/pkgconfig:$TOOLCHAIN/usr/lib/pkgconfig"

CGO_CFLAGS="-Wno-unused-parameter -Wno-unused-variable -Wno-return-type" \
CGO_CXXFLAGS="-I$INCPATH -I$INCPATH/QtCore -I$INCPATH/QtGui -I$INCPATH/QtWidgets -I$INCPATH/QtNetwork -I$INCPATH/libtorrent -Wno-unused-parameter -Wno-unused-variable -Wno-return-type" \
CGO_CXXFLAGS="$CGO_CXXFLAGS -pipe -O2 -std=gnu++11 -Wall -W -D_REENTRANT -DQT_NO_DEBUG -DQT_CORE_LIB -DQT_GUI_LIB -DQT_WIDGETS_LIB -DQT_NETWORK_LIB" \
CGO_LDFLAGS="-L$TOOLCHAIN/usr/lib -L$TOOLCHAIN/lib -L$PLUGPATH/platforms -L$PLUGPATH/generic -L$PLUGPATH/imageformats -L$PLUGPATH/styles" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Core -lz -lpcre2-16 -ldouble-conversion -lole32 -luuid -lws2_32 -ladvapi32 -lshell32 -luser32 -lkernel32 -lmpr" \
CGO_LDFLAGS="$CGO_LDFLAGS -ljpeg -lQt5Network -lQt5Widgets -lQt5Gui -lws2_32 -lpng -lqtharfbuzz -lz -lopengl32 -lcomdlg32 -loleaut32 -limm32 -lwinmm -lglu32 -lopengl32 -liphlpapi -ldnsapi" \
CGO_LDFLAGS="$CGO_LDFLAGS -lgdi32 -lole32 -luuid -lws2_32 -ladvapi32 -lshell32 -luser32 -lkernel32 -lmpr  -ldouble-conversion -lqtpcre -lqtharfbuzz" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Network -lQt5Widgets -lQt5Gui -lQt5Core -lfontconfig -lexpat -lfreetype -lbz2 -lpng16 -lgdi32" \
CGO_LDFLAGS="$CGO_LDFLAGS -ltorrent-rasterbar -lssl -lcrypto -lgdi32 -lcrypt32 -lboost_system -lws2_32 -lmswsock -lgdi32" \
CGO_LDFLAGS="$CGO_LDFLAGS -lqwindows -lqwindowsvistastyle -lqgif -lqjpeg -lgdi32 -limm32 -loleaut32 -lwinmm -luxtheme -ldwmapi -lQt5Widgets -lQt5EventDispatcherSupport -lQt5ThemeSupport -lQt5DeviceDiscoverySupport -lQt5FontDatabaseSupport -lQt5WindowsUIAutomationSupport -lfontconfig -lfreetype -lQt5Gui -lole32 -ljpeg -lpng -lqtharfbuzz -lz -lbz2 -lopengl32 -lQt5Core -lopengl32 -ldouble-conversion -lole32 -lgdi32 -lnetapi32 -lversion -luserenv -lwtsapi32" \
CGO_ENABLED=1 GOOS=windows GOARCH=386 go build -trimpath -tags 'static minimal' -o build/bukanir.windows.386.exe -v -x -buildmode=exe -ldflags "-H=windowsgui -linkmode external -s -w '-extldflags=-static'"

# windows/amd64
TOOLCHAIN="/usr/x86_64-w64-mingw32"
INCPATH="$TOOLCHAIN/usr/include"
PLUGPATH="$TOOLCHAIN/usr/plugins"
export CC="x86_64-w64-mingw32-gcc"
export CXX="x86_64-w64-mingw32-g++"
export LD="x86_64-w64-mingw32-ld"
export AR="x86_64-w64-mingw32-ar"
export PKG_CONFIG="/usr/bin/x86_64-w64-mingw32-pkg-config"
export PKG_CONFIG_PATH="$TOOLCHAIN/usr/lib/pkgconfig:$TOOLCHAIN/usr/lib/pkgconfig"
export PKG_CONFIG_LIBDIR="$TOOLCHAIN/usr/lib/pkgconfig:$TOOLCHAIN/usr/lib/pkgconfig"

CGO_CFLAGS="-Wno-unused-parameter -Wno-unused-variable -Wno-return-type" \
CGO_CXXFLAGS="-I$INCPATH -I$INCPATH/QtCore -I$INCPATH/QtGui -I$INCPATH/QtWidgets -I$INCPATH/QtNetwork -I$INCPATH/libtorrent -Wno-unused-parameter -Wno-unused-variable -Wno-return-type" \
CGO_CXXFLAGS="$CGO_CXXFLAGS -pipe -O2 -std=gnu++11 -Wall -W -D_REENTRANT -DQT_NO_DEBUG -DQT_CORE_LIB -DQT_GUI_LIB -DQT_WIDGETS_LIB -DQT_NETWORK_LIB" \
CGO_LDFLAGS="-L$TOOLCHAIN/usr/lib -L$TOOLCHAIN/lib -L$PLUGPATH/platforms -L$PLUGPATH/generic -L$PLUGPATH/imageformats -L$PLUGPATH/styles" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Core -lz -lpcre2-16 -ldouble-conversion -lole32 -luuid -lws2_32 -ladvapi32 -lshell32 -luser32 -lkernel32 -lmpr" \
CGO_LDFLAGS="$CGO_LDFLAGS -ljpeg -lQt5Network -lQt5Widgets -lQt5Gui -lws2_32 -lpng -lqtharfbuzz -lz -lopengl32 -lcomdlg32 -loleaut32 -limm32 -lwinmm -lglu32 -lopengl32 -liphlpapi -ldnsapi" \
CGO_LDFLAGS="$CGO_LDFLAGS -lgdi32 -lole32 -luuid -lws2_32 -ladvapi32 -lshell32 -luser32 -lkernel32 -lmpr  -ldouble-conversion -lqtpcre2 -lqtharfbuzz" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Network -lQt5Widgets -lQt5Gui -lQt5Core -lfontconfig -lexpat -lfreetype -lbz2 -lpng16 -lgdi32" \
CGO_LDFLAGS="$CGO_LDFLAGS -ltorrent-rasterbar -lssl -lcrypto -lgdi32 -lcrypt32 -lboost_system -lws2_32 -lmswsock -lgdi32" \
CGO_LDFLAGS="$CGO_LDFLAGS -lqwindows -lqwindowsvistastyle -lqgif -lqjpeg -lgdi32 -limm32 -loleaut32 -lwinmm -luxtheme -ldwmapi -lQt5Widgets -lQt5EventDispatcherSupport -lQt5ThemeSupport -lQt5DeviceDiscoverySupport -lQt5FontDatabaseSupport -lQt5WindowsUIAutomationSupport -lfontconfig -lfreetype -lQt5Gui -lole32 -ljpeg -lpng -lqtharfbuzz -lz -lbz2 -lopengl32 -lQt5Core -lopengl32 -ldouble-conversion -lole32 -lgdi32 -lnetapi32 -lversion -luserenv -lwtsapi32 -lsetupapi" \
CGO_ENABLED=1 GOOS=windows GOARCH=amd64 go build -trimpath -tags 'static minimal' -o build/bukanir.windows.amd64.exe -v -x -buildmode=exe -ldflags "-H=windowsgui -linkmode external -s -w '-extldflags=-static'"

# darwin/amd64
TOOLCHAIN="/usr/x86_64-apple-darwin"
INCPATH="$TOOLCHAIN/SDK/MacOSX12.1.sdk/usr/include"
PLUGPATH="$TOOLCHAIN/SDK/MacOSX12.1.sdk/usr/plugins"
export CC=x86_64-apple-darwin21.1-clang
export CXX=x86_64-apple-darwin21.1-clang++
export LD=x86_64-apple-darwin21.1-ld
export AR=x86_64-apple-darwin21.1-ar
export PKG_CONFIG="x86_64-apple-darwin21.1-pkg-config"
export PKG_CONFIG_PATH="$TOOLCHAIN/SDK/MacOSX12.1.sdk/usr/lib/pkgconfig"
export PKG_CONFIG_LIBDIR="$TOOLCHAIN/SDK/MacOSX12.1.sdk/usr/lib/pkgconfig"
export LIBRARY_PATH="$TOOLCHAIN/SDK/MacOSX12.1.sdk/usr/lib:$TOOLCHAIN/macports/pkgs/opt/local/lib"
export PATH=${PATH}:/usr/x86_64-apple-darwin/bin

CGO_LDFLAGS="$CGO_LDFLAGS -framework Foundation -framework AudioToolbox -framework CoreAudio -framework AVFoundation -framework CoreVideo -framework CoreMedia -framework CoreGraphics -framework IOSurface -framework VideoToolbox -framework Carbon -framework QuartzCore" \
CGO_CFLAGS="-Wno-unused-parameter -Wno-unused-variable -Wno-return-type -Wno-deprecated-declarations -Wno-strict-aliasing -mmacosx-version-min=10.13" \
CGO_CXXFLAGS="-I$TOOLCHAIN/macports/pkgs/opt/local/include -I$INCPATH -I$INCPATH/QtCore -I$INCPATH/QtGui -I$INCPATH/QtWidgets -I$INCPATH/QtNetwork -I$INCPATH/libtorrent -Wno-unused-parameter -Wno-unused-variable -Wno-return-type -Wno-deprecated-declarations -Wno-strict-aliasing" \
CGO_CXXFLAGS="$CGO_CXXFLAGS -pipe -O2 -std=gnu++11 -Wall -W -D_REENTRANT -DQT_NO_DEBUG -DQT_CORE_LIB -DQT_GUI_LIB -DQT_WIDGETS_LIB -DQT_NETWORK_LIB -fPIC" \
CGO_LDFLAGS="-L$TOOLCHAIN/SDK/MacOSX12.1.sdk/usr/lib -L$TOOLCHAIN/macports/pkgs/opt/local/lib -L$PLUGPATH/platforms -L$PLUGPATH/printsupport -L$PLUGPATH/generic -L$PLUGPATH/imageformats -L$PLUGPATH/styles -mmacosx-version-min=10.13" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Core -lpthread -lz -lpcre2-16 -ldouble-conversion -lm -ldl -lssl -lcrypto" \
CGO_LDFLAGS="$CGO_LDFLAGS -ljpeg -lQt5Network -lQt5Widgets -lQt5Gui -lQt5Core -lpthread -lpng -lharfbuzz -lz" \
CGO_LDFLAGS="$CGO_LDFLAGS -framework DiskArbitration -framework IOKit -lm -framework AppKit -framework Security -framework ApplicationServices -framework CoreServices -framework CoreFoundation -framework Foundation -framework SystemConfiguration" \
CGO_LDFLAGS="$CGO_LDFLAGS -lmpv -lSDL2 -ljpeg -lass -liconv -lharfbuzz -lfreetype -luchardet -lavformat -lswscale -lavdevice -lavfilter -lavcodec -lavutil -ldl -lm -lswresample -lavresample -lpostproc -lluajit-5.1 -lz" \
CGO_LDFLAGS="$CGO_LDFLAGS -framework Foundation -framework AudioToolbox -framework CoreAudio -framework AVFoundation -framework CoreVideo -framework CoreMedia -framework CoreGraphics -framework IOSurface -framework VideoToolbox -framework Carbon -framework QuartzCore" \
CGO_LDFLAGS="$CGO_LDFLAGS -lfontconfig -liconv -lexpat -lintl -lfreetype -lbrotlicommon -lbrotlidec -lfribidi -lbz2 -lpng16 -luuid -ltorrent-rasterbar -lboost_system -lssl -lcrypto -lz" \
CGO_LDFLAGS="$CGO_LDFLAGS -lqcocoa -lcocoaprintersupport -lqmacstyle -lqgif -lqjpeg -lQt5MacExtras -lQt5EdidSupport -lQt5DeviceDiscoverySupport -lQt5ThemeSupport -lQt5FontDatabaseSupport -lQt5GraphicsSupport -lQt5PrintSupport -lQt5ClipboardSupport -lQt5AccessibilitySupport -lQt5ServiceSupport -lQt5EventDispatcherSupport" \
CGO_LDFLAGS="$CGO_LDFLAGS -lfontconfig -liconv -lexpat -lintl -lcups -lfreetype -ldl -lm -lxml2 -lffi" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Gui -ljpeg -lpng -lharfbuzz -lfreetype -framework CoreGraphics -framework OpenGL -framework AGL -lz -lbz2 -lQt5Core -lpthread -lc++" \
CGO_ENABLED=1 GOOS=darwin GOARCH=amd64 go build -trimpath -tags 'static minimal' -o build/bukanir.darwin.amd64 -v -x -ldflags "-linkmode external -s -w '-extldflags=-mmacosx-version-min=10.13'"

# darwin/arm64
TOOLCHAIN="/usr/aarch64-apple-darwin"
INCPATH="$TOOLCHAIN/SDK/MacOSX12.1.sdk/usr/include"
PLUGPATH="$TOOLCHAIN/SDK/MacOSX12.1.sdk/usr/plugins"
export CC=aarch64-apple-darwin21.1-clang
export CXX=aarch64-apple-darwin21.1-clang++
export LD=aarch64-apple-darwin21.1-ld
export AR=aarch64-apple-darwin21.1-ar
export PKG_CONFIG="aarch64-apple-darwin21.1-pkg-config"
export PKG_CONFIG_PATH="$TOOLCHAIN/SDK/MacOSX12.1.sdk/usr/lib/pkgconfig"
export PKG_CONFIG_LIBDIR="$TOOLCHAIN/SDK/MacOSX12.1.sdk/usr/lib/pkgconfig"
export LIBRARY_PATH="$TOOLCHAIN/SDK/MacOSX12.1.sdk/usr/lib:$TOOLCHAIN/macports/pkgs/opt/local/lib"
export PATH=${PATH}:/usr/aarch64-apple-darwin/bin

CGO_CFLAGS="-Wno-unused-parameter -Wno-unused-variable -Wno-return-type -Wno-deprecated-declarations -Wno-strict-aliasing -mmacosx-version-min=10.13" \
CGO_CXXFLAGS="-I$TOOLCHAIN/macports/pkgs/opt/local/include -I$INCPATH -I$INCPATH/QtCore -I$INCPATH/QtGui -I$INCPATH/QtWidgets -I$INCPATH/QtNetwork -I$INCPATH/libtorrent -Wno-unused-parameter -Wno-unused-variable -Wno-return-type -Wno-deprecated-declarations -Wno-strict-aliasing" \
CGO_CXXFLAGS="$CGO_CXXFLAGS -pipe -O2 -std=gnu++11 -Wall -W -D_REENTRANT -DQT_NO_DEBUG -DQT_CORE_LIB -DQT_GUI_LIB -DQT_WIDGETS_LIB -DQT_NETWORK_LIB -fPIC" \
CGO_LDFLAGS="-L$TOOLCHAIN/SDK/MacOSX12.1.sdk/usr/lib -L$TOOLCHAIN/macports/pkgs/opt/local/lib -L$PLUGPATH/platforms -L$PLUGPATH/printsupport -L$PLUGPATH/generic -L$PLUGPATH/imageformats -L$PLUGPATH/styles -mmacosx-version-min=10.13" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Core -lpthread -lz -lpcre2-16 -ldouble-conversion -lm -ldl -lssl -lcrypto" \
CGO_LDFLAGS="$CGO_LDFLAGS -ljpeg -lQt5Network -lQt5Widgets -lQt5Gui -lQt5Core -lpthread -lpng -lharfbuzz -lz" \
CGO_LDFLAGS="$CGO_LDFLAGS -framework DiskArbitration -framework IOKit -lm -framework AppKit -framework Security -framework ApplicationServices -framework CoreServices -framework CoreFoundation -framework Foundation -framework SystemConfiguration" \
CGO_LDFLAGS="$CGO_LDFLAGS -lmpv -lSDL2 -ljpeg -lass -liconv -lharfbuzz -lfreetype -luchardet -lavformat -lswscale -lavdevice -lavfilter -lavcodec -lavutil -ldl -lm -lswresample -lavresample -lpostproc -lluajit-5.1 -lz" \
CGO_LDFLAGS="$CGO_LDFLAGS -framework Foundation -framework AudioToolbox -framework CoreAudio -framework AVFoundation -framework CoreVideo -framework CoreMedia -framework CoreGraphics -framework IOSurface -framework VideoToolbox -framework Carbon -framework QuartzCore" \
CGO_LDFLAGS="$CGO_LDFLAGS -lfontconfig -liconv -lexpat -lintl -lfreetype -lbrotlicommon -lbrotlidec -lfribidi -lbz2 -lpng16 -luuid -ltorrent-rasterbar -lboost_system -lssl -lcrypto -lz" \
CGO_LDFLAGS="$CGO_LDFLAGS -lqcocoa -lcocoaprintersupport -lqmacstyle -lqgif -lqjpeg -lQt5MacExtras -lQt5EdidSupport -lQt5DeviceDiscoverySupport -lQt5ThemeSupport -lQt5FontDatabaseSupport -lQt5GraphicsSupport -lQt5PrintSupport -lQt5ClipboardSupport -lQt5AccessibilitySupport -lQt5ServiceSupport -lQt5EventDispatcherSupport" \
CGO_LDFLAGS="$CGO_LDFLAGS -lfontconfig -liconv -lexpat -lintl -lcups -lfreetype -ldl -lm -lxml2 -lffi" \
CGO_LDFLAGS="$CGO_LDFLAGS -lQt5Gui -ljpeg -lpng -lharfbuzz -lfreetype -framework CoreGraphics -framework OpenGL -framework AGL -lz -lbz2 -lQt5Core -lpthread -lc++" \
CGO_ENABLED=1 GOOS=darwin GOARCH=arm64 go build -trimpath -tags 'static minimal' -o build/bukanir.darwin.arm64 -v -x -ldflags "-linkmode external -s -w '-extldflags=-mmacosx-version-min=10.13'"
