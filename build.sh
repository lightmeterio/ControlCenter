#!/bin/sh

set -e
set -vx
set -o pipefail

TAGS="${1:-dev}"

shift 1

if [ "$GOOS" = windows ]; then
  OUTPUT="lightmeter.exe"
else
  OUTPUT="lightmeter"
fi

export GO111MODULE=on
export CGO_ENABLED=1
go mod download

# go get does not play well with modules :-(
GO111MODULE=off go get -v -u github.com/shurcooL/vfsgen

go run github.com/swaggo/swag/cmd/swag init --generalInfo api/http.go
cp docs/swagger.json www/api.json
go generate -tags="$TAGS" gitlab.com/lightmeter/controlcenter/dashboard
go generate -tags="$TAGS" gitlab.com/lightmeter/controlcenter/staticdata
go build -tags="$TAGS" -o "${OUTPUT}" "$@"
