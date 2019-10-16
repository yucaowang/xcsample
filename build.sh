#!/bin/bash

export GO111MODULE=on
export GOFLAGS=-mod=vendor

go build --buildmode=plugin -o plugins/crypto/crypto-default.so.1.0.0 xcsample/client/crypto

go build -o xcsample xcsample/samples