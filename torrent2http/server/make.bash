#!/usr/bin/env bash


export PATH=/opt/android-toolchain-arm/bin:/opt/android-toolchain-arm64/bin:/opt/android-toolchain-x86/bin:/opt/android-toolchain-amd64/bin:${PATH}

mkdir -p build


CHROOT="/home/milann/chroot"
export CC=gcc CXX=g++
export PKG_CONFIG_PATH="$CHROOT/usr/lib/pkgconfig"
export PKG_CONFIG_LIBDIR="$CHROOT/usr/lib/pkgconfig"
export LIBRARY_PATH="$CHROOT/usr/lib:$CHROOT/lib"
CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o build/torrent2http.linux.amd64 -v -x
strip build/torrent2http.linux.amd64


#CHROOT="/home/milann/chroot"
#export CC=gcc CXX=g++
#export PKG_CONFIG_PATH="$CHROOT/usr/lib32/pkgconfig"
#export PKG_CONFIG_LIBDIR="$CHROOT/usr/lib32/pkgconfig"
#export LIBRARY_PATH="$CHROOT/usr/lib32:$CHROOT/lib32"
#CGO_ENABLED=1 GOOS=linux GOARCH=386 go build -o build/torrent2http.linux.386 -v -x
#strip build/torrent2http.linux.386


#MINGW="/usr/i686-w64-mingw32"
#export CC="i686-w64-mingw32-gcc" CXX="i686-w64-mingw32-g++"
#export PKG_CONFIG="/usr/bin/i686-w64-mingw32-pkg-config"
#export PKG_CONFIG_PATH="$MINGW/usr/lib/pkgconfig:$MINGW/usr/lib/pkgconfig"
#export PKG_CONFIG_LIBDIR="$MINGW/usr/lib/pkgconfig:$MINGW/usr/lib/pkgconfig"
#CC_FOR_TARGET="i686-w64-mingw32-gcc" CXX_FOR_TARGET="i686-w64-mingw32-g++" \
#CGO_ENABLED=1 GOOS=windows GOARCH=386 go build -o build/torrent2http.exe -v -x -ldflags -H=windowsgui
#i686-w64-mingw32-strip build/torrent2http.exe


ANDROID="/opt/android-toolchain-arm7"
export CC=arm-linux-androideabi-gcc CXX=arm-linux-androideabi-g++
export PKG_CONFIG_PATH=$ANDROID/lib/pkgconfig
export PKG_CONFIG_LIBDIR=$ANDROID/lib/pkgconfig
CGO_LDFLAGS="-L$ANDROID/lib" \
CGO_CFLAGS="-I$ANDROID/include" \
CGO_CXXFLAGS="-I$ANDROID/include" \
CC_FOR_TARGET=arm-linux-androideabi-gcc CXX_FOR_TARGET=arm-linux-androideabi-g++ \
CGO_ENABLED=1 GOOS=android GOARCH=arm go build -o build/torrent2http.android.arm -v -x
/opt/android-toolchain-arm/arm-linux-androideabi/bin/strip build/torrent2http.android.arm


ANDROID="/opt/android-toolchain-arm64"
export CC=aarch64-linux-android-gcc CXX=aarch64-linux-android-g++
export PKG_CONFIG_PATH=$ANDROID/lib/pkgconfig
export PKG_CONFIG_LIBDIR=$ANDROID/lib/pkgconfig
CGO_LDFLAGS="-L$ANDROID/lib" \
CGO_CFLAGS="-I$ANDROID/include" \
CGO_CXXFLAGS="-I$ANDROID/include" \
CC_FOR_TARGET=aarch64-linux-android-gcc CXX_FOR_TARGET=aarch64-linux-android-g++ \
CGO_ENABLED=1 GOOS=android GOARCH=arm64 go build -o build/torrent2http.android.arm64 -v -x
/opt/android-toolchain-arm64/aarch64-linux-android/bin/strip build/torrent2http.android.arm64

ANDROID="/opt/android-toolchain-x86"
export CC=i686-linux-android-gcc CXX=i686-linux-android-g++
export PKG_CONFIG_PATH=$ANDROID/lib/pkgconfig
export PKG_CONFIG_LIBDIR=$ANDROID/lib/pkgconfig
CGO_LDFLAGS="-L$ANDROID/lib" \
CGO_CFLAGS="-I$ANDROID/include" \
CGO_CXXFLAGS="-I$ANDROID/include" \
CC_FOR_TARGET=i686-linux-android-gcc CXX_FOR_TARGET=i686-linux-android-g++ \
CGO_ENABLED=1 GOOS=android GOARCH=386 go build -o build/torrent2http.android.x86 -v -x
/opt/android-toolchain-x86/i686-linux-android/bin/strip build/torrent2http.android.x86
