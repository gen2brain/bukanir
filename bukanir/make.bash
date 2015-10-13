#!/usr/bin/env bash

mkdir -p build

gomobile bind -v -x -o build/bukanir.aar -target android bukanir
