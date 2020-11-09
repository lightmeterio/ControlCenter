#!/bin/sh

# test everything except mocks and the main package
COVERPKG="$(go list ./... | egrep -v '(/examples/|/po/|/tools/|(controlcenter|lightmeter|mock)$)' | tr '\n' ',')"

echo "Clean golang cache"
go clean -cache

echo "Generate new mocks for test env"
make mocks > /dev/null

echo "Run all tests and create a coverage profile for each package"
go test -v ./... -race -coverpkg=$COVERPKG -failfast "$@"
