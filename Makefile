#!/bin/bash
export LANG=zh_CN.UTF-8

ENVARG=CGO_ENABLED=0 GOPROXY=https://goproxy.cn,direct
LINUXARG=GOOS=linux GOARCH=amd64
BUILDARG=-mod=mod -ldflags " -s -X main.buildTime=`date '+%Y-%m-%dT%H:%M:%S'` -X main.gitHash=`git symbolic-ref --short -q HEAD`:`git rev-parse --short HEAD`"

dep:
	cd src; ${ENVARG} go get ./...;

updep:
	cd src; ${ENVARG} go get -u ./...; go mod tidy;

remover:
	cd src/remover; ${ENVARG} go build ${BUILDARG} -o ../../bin/redis-remover *.go;

copyer:
	cd src/copyer; ${ENVARG} go build ${BUILDARG} -o ../../bin/redis-copyer *.go;
	
expirer:
	cd src/expirer; ${ENVARG} go build ${BUILDARG} -o ../../bin/redis-expirer *.go;
	
idler:
	cd src/idler; ${ENVARG} go build ${BUILDARG} -o ../../bin/redis-idler *.go; 
	
paser:
	cd src/paser; ${ENVARG} go build ${BUILDARG} -o ../../bin/redis-paser *.go; 

all: remover copyer expirer idler paser
	
linux_remover:
	cd src/remover; ${ENVARG} ${LINUXARG} go build ${BUILDARG} -o ../../lbin/redis-remover *.go;
	
linux_copyer:
	cd src/copyer; ${ENVARG} ${LINUXARG} go build ${BUILDARG} -o ../../lbin/redis-copyer *.go;
	
linux_expirer:
	cd src/expirer; ${ENVARG} ${LINUXARG} go build ${BUILDARG} -o ../../lbin/redis-expirer *.go;
	
linux_idler:
	cd src/idler; ${ENVARG} ${LINUXARG} go build ${BUILDARG} -o ../../lbin/redis-idler *.go;
	
linux_paser:
	cd src/paser; ${ENVARG} ${LINUXARG} go build ${BUILDARG} -o ../../lbin/redis-paser *.go;

linux_all: linux_remover linux_copyer linux_expirer linux_idler linux_paser

clean:
	rm -fr bin/*
	rm -fr lbin/*
	rm src/go.sum
