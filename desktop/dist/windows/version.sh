#!/bin/bash
VERSION=`cat ../../../lib/const.go | grep Version | awk -F' = ' '{print $2}' | tr -d '"'`
sed "s/{VERSION}/$VERSION/g" bukanir.iss.in > bukanir.iss
