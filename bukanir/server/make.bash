#!/usr/bin/env bash

mkdir -p build

CGO_ENABLED=1 GOOS=linux GOARCH=amd64 go build -o build/bukanir-http.linux.amd64
strip build/bukanir-http.linux.amd64

CGO_ENABLED=1 GOOS=linux GOARCH=386 go build -o build/bukanir-http.linux.386
strip build/bukanir-http.linux.386

CGO_ENABLED=1 GOOS=windows GOARCH=386 go build -o build/bukanir-http.exe -a -installsuffix cgo -ldflags -H=windowsgui
