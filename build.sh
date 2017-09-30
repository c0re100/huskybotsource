#!/bin/bash

VERSION=2.8
BUILDDATE=$(date '+%d/%m/%Y %H:%M:%S %Z')

echo "Cross Compile: Windows Building"
export GOOS=windows
export GOARCH=amd64
go build -ldflags "-s -X main.version=$VERSION -X \"main.builddate=$BUILDDATE\"" tg.go

echo "Cross Compile: ARM Building"
export GOOS=linux
export GOARCH=arm
go build -ldflags "-s -X main.version=$VERSION -X \"main.builddate=$BUILDDATE\"" tg.go