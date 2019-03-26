#!/bin/sh

export MOV2GPXPATH=github.com/clarified/mov2gps/go/mov2gpx
export TMPTARGET=/tmp/build/

if [ ! -e $TMPTARGET ] ; then mkdir /tmp/build/ ; fi

cp README.gz LICENSE ../doc/mov2gpx_man.pdf ../doc/mov2gpx.1  $TMPTARGET

cd $TMPTARGET

GOOS=windows GOARCH=386 go build -ldflags=-w -o mov2gpx.exe $MOV2GPXPATH
# Build zip
zip mov2gpx_windows_386.zip README.gz LICENSE mov2gpx_man.pdf mov2gpx.exe
rm mov2gpx.exe

GOOS=windows GOARCH=amd64 go build -ldflags=-w -o mov2gpx.exe $MOV2GPXPATH
# Build zip
zip mov2gpx_windows_amd64.zip README.gz LICENSE mov2gpx_man.pdf mov2gpx.exe
rm mov2gpx.exe

GOOS=linux GOARCH=arm go build -ldflags=-w -o mov2gpx $MOV2GPXPATH
# Build tar
tar -cJf mov2gpx_linux_arm.tar.xz README.gz LICENSE mov2gpx.1 mov2gpx
rm mov2gpx

GOOS=linux GOARCH=arm64 go build -ldflags=-w -o mov2gpx $MOV2GPXPATH
# Build tar
tar -cJf mov2gpx_linux_arm64.tar.xz README.gz LICENSE mov2gpx.1 mov2gpx
rm mov2gpx

GOOS=linux GOARCH=amd64 go build -ldflags=-w -o mov2gpx $MOV2GPXPATH
# Build tar
tar -cJf mov2gpx_linux_amd64.tar.xz README.gz LICENSE mov2gpx.1 mov2gpx
rm mov2gpx

GOOS=linux GOARCH=386 GO386=387 go build -ldflags=-w -o mov2gpx $MOV2GPXPATH
# Build tar
tar -cJf mov2gpx_linux_386.tar.xz README.gz LICENSE mov2gpx.1 mov2gpx
rm mov2gpx

GOOS=linux GOARCH=386 GO386=sse2 go build -ldflags=-w -o mov2gpx $MOV2GPXPATH
# Build tar
tar -cJf mov2gpx_linux_386_sse2.tar.xz README.gz LICENSE mov2gpx.1 mov2gpx
rm mov2gpx

echo "?mv files from " $TMPTARGET '?'

