#!/bin/sh

set -e
set -vx
set -o pipefail

TAGS="${1:-dev}"

if [ "$#" -gt 0 ]; then
  shift 1
fi

if [ "$GOOS" = windows ]; then
  OUTPUT="lightmeter.exe"
else
  OUTPUT="lightmeter"
fi

function tag_name() {
  git describe --tags --exact-match 2>/dev/null
}

function branch_name() {
  git symbolic-ref -q --short HEAD
}

function commit_name() {
  git rev-parse --short HEAD
}

function gen_version() {
cat << EOF
// +build release

package version
const (
  Version = "$(cat VERSION.txt)"
)
EOF
}

function gen_build_info() {
cat << EOF
package version
const (
  Commit = "$(commit_name)"
  TagOrBranch = "$(tag_name || branch_name)"
)
EOF
}

gen_version > version/version_release.go

if [ -d .git ]; then
  gen_build_info > version/build_info.go
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
