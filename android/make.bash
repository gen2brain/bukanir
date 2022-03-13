#!/usr/bin/env bash

export GO111MODULE=off

export PATH=/opt/android-toolchain-arm/bin:/opt/android-toolchain-aarch64/bin:${PATH}

mkdir -p build

ANDROID_TOOLCHAIN="/opt/android-toolchain-arm"
export CC=arm-linux-androideabi24-clang
export CXX=arm-linux-androideabi24-clang++
#export CC=arm-linux-androideabi-gcc
#export CXX=arm-linux-androideabi-g++
export PKG_CONFIG_PATH=$ANDROID_TOOLCHAIN/lib/pkgconfig
export PKG_CONFIG_LIBDIR=$ANDROID_TOOLCHAIN/lib/pkgconfig

CGO_CFLAGS="-I${ANDROID_TOOLCHAIN}/include/c++/4.9.x -I${ANDROID_TOOLCHAIN}/include" \
CGO_CXXFLAGS="-I${ANDROID_TOOLCHAIN}/include/c++/4.9.x -I${ANDROID_TOOLCHAIN}/include/c++/4.9.x/arm-linux-androideabi" \
CGO_LDFLAGS="-L${ANDROID_TOOLCHAIN}/lib" \
CGO_ENABLED=1 GOOS=android GOARCH=arm \
gomobile bind -v -x -o build/bukanir-arm7.aar -target android/arm -ldflags "-s -w" github.com/gen2brain/bukanir/lib


#ANDROID_TOOLCHAIN="/opt/android-toolchain-arm64"
#export CC=aarch64-linux-android-gcc
#export CXX=aarch64-linux-android-g++
#export PKG_CONFIG_PATH=$ANDROID_TOOLCHAIN/lib/pkgconfig
#export PKG_CONFIG_LIBDIR=$ANDROID_TOOLCHAIN/lib/pkgconfig

#CGO_CFLAGS="-I${ANDROID_TOOLCHAIN}/include/c++/4.9.x" \
#CGO_CXXFLAGS="-I${ANDROID_TOOLCHAIN}/include/c++/4.9.x -I${ANDROID_TOOLCHAIN}/include/c++/4.9.x/aarch64-linux-android" \
#CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
#gomobile bind -v -x -o build/bukanir-arm64.aar -target android/arm64 -ldflags "-s -w" github.com/gen2brain/bukanir/lib
