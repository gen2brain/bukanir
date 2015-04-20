#!/usr/bin/env bash

mkdir -p build

GOOS=linux GOARCH=amd64 go build -o build/bukanir-http.linux.amd64 bukanir.go
strip build/bukanir-http.linux.amd64

GOOS=linux GOARCH=386 go build -o build/bukanir-http.linux.386 crtaci.go
strip build/bukanir-http.linux.386
