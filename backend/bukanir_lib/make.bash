#!/usr/bin/env bash

export ARM_CROSS_HOME=/opt/android-toolchain-arm
export X86_CROSS_HOME=/opt/android-toolchain-x86
export PATH=$ARM_CROSS_HOME/bin:$X86_CROSS_HOME/bin:$PATH

export CC=arm-linux-androideabi-gcc

mkdir -p build

CGO_ENABLED=1 GOOS=android GOARCH=arm GOARM=6 go build -o build/libgojni.so.6 -ldflags="-shared" .
$ARM_CROSS_HOME/arm-linux-androideabi/bin/strip build/libgojni.so.6
cp -f build/libgojni.so.6 ../../src/main/jniLibs/armeabi/libgojni.so

CGO_ENABLED=1 GOOS=android GOARCH=arm GOARM=7 go build -o build/libgojni.so.7 -ldflags="-shared" .
$ARM_CROSS_HOME/arm-linux-androideabi/bin/strip build/libgojni.so.7
cp -f build/libgojni.so.7 ../../src/main/jniLibs/armeabi-v7a/libgojni.so

#export CC=i686-linux-android-gcc

#CGO_ENABLED=1 GOOS=android GOARCH=386 go build -o build/libgojni.so.x86 -ldflags -extldflags="-shared" .
#$X86_CROSS_HOME/i686-linux-android/bin/strip build/libgojni.so.x86
#cp -f build/libgojni.so.x86 ../../src/main/jniLibs/x86/libgojni.so
