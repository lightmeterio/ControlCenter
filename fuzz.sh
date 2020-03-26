#!/bin/bash

set -e

export GO111MODULE=off

go-fuzz-build
go-fuzz
