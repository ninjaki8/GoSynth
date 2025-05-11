#!/bin/bash
set -e

echo "Building for Linux..."
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/goSynth-linux

echo "Building for Windows..."
GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/goSynth.exe

echo "Building for macOS..."
GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -ldflags="-s -w" -o build/goSynth-macos

echo "Done!"

