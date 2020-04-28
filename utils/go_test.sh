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

go generate gitlab.com/lightmeter/controlcenter/dashboard
go test ./... -coverpkg=$COVERPKG "$@"
