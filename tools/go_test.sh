#!/bin/sh

set -e
set -o pipefail

# workaround SQLite warning reported at:
# https://github.com/mattn/go-sqlite3/issues/803
export CGO_CFLAGS="-g -O2 -Wno-return-local-addr"

# test everything except mocks and the main package
COVERPKG="$(go list ./... | egrep -v '(/examples/|/po/|/tools/|(controlcenter|lightmeter|mock)$)' | tr '\n' ',')"

export CGO_ENABLED=1

make mocks > /dev/null
go test ./... -coverpkg=$COVERPKG "$@"
