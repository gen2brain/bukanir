#!/usr/bin/env bash

export PATH=/opt/android-toolchain-arm7/bin:/opt/android-toolchain-arm64/bin:/opt/android-toolchain-x86/bin:/opt/android-toolchain-x86_64/bin:${PATH}

mkdir -p build

gomobile bind -v -x -o build/bukanir.aar -target android/arm,android/arm64,android/386 -ldflags "-s -w -extldflags=-Wl,--allow-multiple-definition" bukanir 

#ANDROID="/opt/android-toolchain-arm7"
#export CC=arm-linux-androideabi-gcc CXX=arm-linux-androideabi-g++
#export PKG_CONFIG_PATH=$ANDROID/lib/pkgconfig
#export PKG_CONFIG_LIBDIR=$ANDROID/lib/pkgconfig

#CC=arm-linux-androideabi-gcc CXX=arm-linux-androideabi-g++ \
#CC_FOR_TARGET=arm-linux-androideabi-gcc CXX_FOR_TARGET=arm-linux-androideabi-g++ \
#CGO_ENABLED=1 GOOS=android GOARCH=arm \
#gomobile bind -v -x -o build/bukanir-arm7.aar -target android/arm -ldflags "-s -w -extldflags=-Wl,--allow-multiple-definition" bukanir 

#ANDROID="/opt/android-toolchain-arm64"
#export CC=aarch64-linux-android-gcc CXX=aarch64-linux-android-g++
#export PKG_CONFIG_PATH=$ANDROID/lib/pkgconfig
#export PKG_CONFIG_LIBDIR=$ANDROID/lib/pkgconfig

#CC=aarch64-linux-android-gcc CXX=aarch64-linux-android-g++ \
#CC_FOR_TARGET=aarch64-linux-android-gcc CXX_FOR_TARGET=aarch64-linux-android-g++ \
#CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
#gomobile bind -v -x -o build/bukanir-arm64.aar -target android/arm64 -ldflags "-s -w -extldflags=-Wl,--allow-multiple-definition" bukanir 

#ANDROID="/opt/android-toolchain-x86"
#export CC=i686-linux-android-gcc CXX=i686-linux-android-g++
#export PKG_CONFIG_PATH=$ANDROID/lib/pkgconfig
#export PKG_CONFIG_LIBDIR=$ANDROID/lib/pkgconfig

#CC=i686-linux-android-gcc CXX=i686-linux-android-g++ \
#CC_FOR_TARGET=i686-linux-android-gcc CXX_FOR_TARGET=i686-linux-android-g++ \
#CGO_ENABLED=1 GOOS=android GOARCH=arm64 \
#gomobile bind -v -x -o build/bukanir-x86.aar -target android/386 -ldflags "-s -w -extldflags=-Wl,--allow-multiple-definition" bukanir 
