#!/bin/bash

ZMQ_VERSION=3.2.4
GO_VERSION=1.2.1

if hash go 2>/dev/null ; then
    echo "Skipping Go installation..."
else
    echo "Fetching Go..."
    curl -O -s "https://go.googlecode.com/files/go$GO_VERSION.linux-amd64.tar.gz"
    echo "Installing Go..."
    tar -C /usr/local -xzf "go$GO_VERSION.linux-amd64.tar.gz"
    echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
    echo 'export GOPATH=$HOME/workspace' >> /etc/profile
    source /etc/profile
fi

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
fi


echo "Fetching special dependencies..."
mkdir $HOME/workspace 2>/dev/null
cd $HOME/workspace
go get -tags zmq_3_x github.com/alecthomas/gozmq
