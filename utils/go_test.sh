#!/bin/sh

set -e
set -o pipefail

PACKAGES="
  data
  data/postfix
  logdb
  logeater
  staticdata
  workspace
  dashboard
  api
  util
  lmsqlite3
"

PREFIX="gitlab.com/lightmeter/controlcenter/"

COVERPKG=""

for P in $PACKAGES; do
  COVERPKG="$COVERPKG,$PREFIX$P"
done

export CGO_ENABLED=1

make mocks
go test ./... -coverpkg=$COVERPKG "$@"
