#!/usr/bin/env bash

mkdir -p build

CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o build/torrent2http.linux.amd64 -v -x
strip build/torrent2http.linux.amd64

#CGO_ENABLED=1 GOOS=linux GOARCH=386 go build -o build/torrent2http.linux.386 -v -x
#strip build/torrent2http.linux.386

#PKG_CONFIG_LIBDIR=/usr/i686-pc-mingw32/usr/lib/pkgconfig \
#CC=i686-pc-mingw32-gcc CXX=i686-pc-mingw32-g++ \
#CC_FOR_TARGET=i686-pc-mingw32-gcc CXX_FOR_TARGET=i686-pc-mingw32-g++ \
#CGO_ENABLED=1 GOOS=windows GOARCH=386 go build -o build/torrent2http.exe -ldflags -H=windowsgui
#i686-pc-mingw32-strip build/torrent2http.exe

PATH=${GOPATH}/pkg/gomobile/android-ndk-r10e/arm/bin:${PATH} \
PKG_CONFIG_LIBDIR=${GOPATH}/pkg/gomobile/android-ndk-r10e/arm/lib/pkgconfig \
CC=arm-linux-androideabi-gcc CXX=arm-linux-androideabi-g++ \
CC_FOR_TARGET=arm-linux-androideabi-gcc CXX_FOR_TARGET=arm-linux-androideabi-g++ \
CGO_ENABLED=1 GOOS=android GOARCH=arm go build -o build/torrent2http.android.arm -v -x
/opt/android-toolchain-arm/arm-linux-androideabi/bin/strip build/torrent2http.android.arm
