#!/bin/sh

set -e
set -vx

TAGS="${1:-dev}"

if [ "$GOOS" = windows ]; then
  OUTPUT="lightmeter.exe"
else
  OUTPUT="lightmeter"
fi

export GO111MODULE=on
export CGO_ENABLED=1
go mod download

(
  # Temporarily disable modules, as they do not play well with stuff in GOPATH :-(
  export GO111MODULE=off
  unset GOOS
  unset CC
  unset GOARCH
  go get -u github.com/shurcooL/vfsgen
  go generate -tags="$TAGS"
)

go build -tags="$TAGS" -o "${OUTPUT}"
