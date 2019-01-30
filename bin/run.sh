#!/bin/sh -e

# mkfs and mount local SSD on /var/scratch
mkfs -t ext4 -F /dev/disk/by-id/google-local-ssd-0
mkdir -p /var/scratch
mount -t ext4 /dev/disk/by-id/google-local-ssd-0 /var/scratch
chmod 1777 /var/scratch

GOPATH=$HOME/go
export GOPATH
PATH=/usr/local/go/bin:$GOPATH/bin:$PATH
export PATH

git clone https://github.com/cybozu-go/neco $HOME/go/src/github.com/cybozu-go/neco
cd $HOME/go/src/github.com/cybozu-go/neco
git checkout -qf release

cd dctest
cp /assets/cybozu-ubuntu-18.04-server-cloudimg-amd64.img .
export GO111MODULE=on
make setup
make placemat
sleep 3
make test-light
pmctl snapshot save init
exit $?
