#!/bin/sh

set -e

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

go test ./... -coverpkg=$COVERPKG "$@"
