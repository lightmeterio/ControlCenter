#!/bin/sh

set -e
set -o pipefail

COVERPKG="$(go list ./... | tr '\n' ',')"

export CGO_ENABLED=1

make mocks
go test ./... -coverpkg=$COVERPKG "$@"
