#!/bin/sh

set -e
set -o pipefail

# test everything except mocks and the main package
COVERPKG="$(go list ./... | egrep -v '(controlcenter|mock)$' | tr '\n' ',')"

export CGO_ENABLED=1

make mocks
go test ./... -coverpkg=$COVERPKG "$@"
