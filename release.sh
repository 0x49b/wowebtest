#!/bin/bash

echo "build windows"
GOOS=windows GOARCH=amd64 go build -o ./bin/wt-win.exe

echo "build linux"
GOOS=linux GOARCH=amd64 go build -o ./bin/wt-linux

echo "build mac"
GOOS=darwin GOARCH=amd64 go build -o ./bin/wt-mac