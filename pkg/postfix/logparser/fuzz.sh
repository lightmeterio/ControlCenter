#!/bin/bash

set -e

export GO111MODULE=off

go run github.com/dvyukov/go-fuzz/go-fuzz-build
go run github.com/dvyukov/go-fuzz/go-fuzz
