#!/bin/bash

set -e

NAME=snmp_subagent

export GOPATH=$BUILDDIR/go
PROJECTDIR=$GOPATH/src/gitlab.intraway.com/sentinel/sentinel-snmp-subagent

mkdir -p $PROJECTDIR
cp -r *.go snmp_subagent/ test $PROJECTDIR

git clone git@gitlab.intraway.com:golang/snmp-handler.git $GOPATH/src/gitlab.intraway.com/golang/snmp-handler

cd $PROJECTDIR
go get

echo "Testing..."
go test ./...

echo "Building..."
go build

echo "Packaging..."
DESTDIR=$PACKAGEDIR

mkdir -p $DESTDIR/bin
mkdir -p $DESTDIR/etc
mkdir -p $DESTDIR/doc
mkdir -p $DESTDIR/conf
mkdir -p $DESTDIR/data

chmod a+w $DESTDIR/data

BASEARCH=noarch
RELEASEVER=`cat /etc/yum/vars/releasever`
BINNAME="$NAME-$VERSION-el$RELEASEVER-$BASEARCH"

cp $PROJECTDIR/sentinel-snmp-subagent $DESTDIR/bin/$BINNAME
ln -s $BINNAME $DESTDIR/bin/$NAME

cp $ROOTDIR/conf/config.yaml.in $DESTDIR/conf/
cp $ROOTDIR/README.md $ROOTDIR/Changelog $DESTDIR/doc/
