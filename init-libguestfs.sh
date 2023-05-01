#!/bin/bash -eu
MAJOR=1
MINOR=46
PATCH=0
VERSION=${MAJOR}.${MINOR}.${PATCH}
BASENAME=libguestfs-${VERSION}

rm -fr /tmp/${BASENAME}*
curl -Lv https://download.libguestfs.org/${MAJOR}.${MINOR}-stable/${BASENAME}.tar.gz -o /tmp/${BASENAME}.tar.gz
tar xvfz /tmp/${BASENAME}.tar.gz -C /tmp

cd /tmp/${BASENAME}
./configure
make

cd -

cp -a /tmp/libguestfs-1.46.0/lib/.libs/libguestfs.so* /usr/lib/x86_64-linux-gnu/
