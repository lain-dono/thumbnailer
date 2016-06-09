#!/usr/bin/env bash

# auto exit on failures
set -e
set -o pipefail

PORT=5000
NAME=thumbnailer
ACI=$NAME-linux-amd64.aci
ACBUILD=acbuild
GOVERSION=go1.6.2.linux-amd64
ACBUILDVERSION=v0.3.0

export GOPATH=`pwd`

if [[ $EUID -ne 0 ]]; then
   echo "You must be a root" 2>&1
   exit 1
fi

if command -v go > /dev/null 2>&1; then
   echo "golang exists"
else
   echo "golang not exists"
   if [ ! -d "./go" ]; then
      echo "download go"
      wget https://storage.googleapis.com/golang/$GOVERSION.tar.gz
      tar xf $GOVERSION.tar.gz
      rm -f $GOVERSION.tar.gz
   fi
   export GOROOT=`pwd`/go
   export PATH=`pwd`/go:$PATH
fi

if command -v $ACBUILD > /dev/null 2>&1; then
   echo "acbuild exists"
else
   echo "acbuild not exists"
   if [ ! -f "./acbuild" ]; then
      echo "download acbuild"
      wget https://github.com/appc/acbuild/releases/download/$ACBUILDVERSION/acbuild.tar.gz
      tar xf acbuild.tar.gz
      rm -f acbuild.tar.gz
   fi
   ACBUILD=./acbuild
fi

if [ -f "./ffmpeg" ]; then
   echo "./ffmpeg exists"
else
   echo "./ffmpeg not exists"
   #wget http://johnvansickle.com/ffmpeg/builds/ffmpeg-git-64bit-static.tar.xz
   tar xvf ffmpeg-git-64bit-static.tar.xz -C `pwd` --strip=1 --wildcards '*ffmpeg'
   #rm -f ffmpeg-git-64bit-static.tar.xz
fi

echo "make tool"
go get github.com/nfnt/resize
CGO_ENABLED=0 go build -ldflags '-extldflags "-static"' -o $NAME main.go

strip -s $NAME

#file $NAME
#ldd $NAME

echo "make image"
rm -f $ACI
rm -rf .acbuild

$ACBUILD begin
$ACBUILD set-name $NAME
$ACBUILD copy ./ffmpeg /bin/ffmpeg
$ACBUILD copy $NAME /bin/$NAME
$ACBUILD set-exec /bin/$NAME
$ACBUILD port add www tcp $PORT
#$ACBUILD label add version $(VERSION) #xx
$ACBUILD label add arch amd64
$ACBUILD label add os linux
$ACBUILD annotation add authors "Lain-dono <lain.dono@gmail.com>"
$ACBUILD write $ACI
$ACBUILD end

echo "ok"
