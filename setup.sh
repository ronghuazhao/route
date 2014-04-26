#!/bin/bash

ZMQ_VERSION=3.2.4
GO_VERSION=1.2.1

VAGRANTHOME=/home/vagrant

if hash go 2>/dev/null ; then
    echo "Skipping Go installation..."
else
    echo "Fetching Go..."
    curl -O -s "https://go.googlecode.com/files/go$GO_VERSION.linux-amd64.tar.gz"
    echo "Installing Go..."
    tar -C /usr/local -xzf "go$GO_VERSION.linux-amd64.tar.gz"
    rm -f "go$GO_VERSION.linux-amd64.tar.gz"
fi

echo "Setting GOPATH..."
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
echo "export GOPATH=$VAGRANTHOME/workspace" >> /etc/profile
source /etc/profile

echo "Installing dependencies..."
apt-get install -qq git mercurial subversion build-essential pkg-config redis-server >/dev/null 2>/dev/null


if ldconfig -p | grep zmq >/dev/null 2>/dev/null ; then
    echo "Skipping ZeroMQ installation..."
else
    echo "Fetching ZeroMQ..."
    curl -O -s "http://download.zeromq.org/zeromq-$ZMQ_VERSION.tar.gz"
    tar -xzf "zeromq-$ZMQ_VERSION.tar.gz"
    cd "zeromq-$ZMQ_VERSION"
    ./configure -q
    echo "Building ZeroMQ..."
    make -s >/dev/null
    make -s install >/dev/null
    ldconfig
    cd ..
    rm -f "zeromq-$ZMQ_VERSION.tar.gz"
    rm -rf "zeromq-$ZMQ_VERSION"
fi


echo "Building workspace..."
mkdir $VAGRANTHOME/workspace 2>/dev/null
cd $VAGRANTHOME/workspace

echo "Fetching special dependencies..."
go get -tags zmq_3_x github.com/alecthomas/gozmq

echo "Cleanup..."
chown -R vagrant:vagrant $VAGRANTHOME/workspace
