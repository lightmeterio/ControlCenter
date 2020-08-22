# vim: noexpandtab

export GO111MODULE = on
export CGO_ENABLED = 1
export CGO_CFLAGS = -g -O2 -Wno-return-local-addr

PACKAGE_ROOT = gitlab.com/lightmeter/controlcenter
PACKAGE_VERSION = ${PACKAGE_ROOT}/version
APP_VERSION = `cat VERSION.txt`

ifneq ($(wildcard .git),)
	GIT_COMMIT = `git rev-parse --short HEAD`
	GIT_BRANCH = `git describe --tags --exact-match 2>/dev/null || git symbolic-ref -q --short HEAD`
else
	GIT_COMMIT = ""
	GIT_BRANCH = ""
endif

BUILD_INFO_FLAGS = -X ${PACKAGE_VERSION}.Commit=${GIT_COMMIT} -X ${PACKAGE_VERSION}.TagOrBranch=${GIT_BRANCH} -X ${PACKAGE_VERSION}.Version=${APP_VERSION}

all: dev

dev: mocks swag domain_mapping_list po2go
	go build -tags="dev" -o "lightmeter" -ldflags "${BUILD_INFO_FLAGS}"

release: static_www domain_mapping_list po2go
	go build -tags="release" -o "lightmeter" -ldflags "${BUILD_INFO_FLAGS}"

windows_release: static_www domain_mapping_list po2go
	CC=x86_64-w64-mingw32-gcc GOOS=windows GOARCH=amd64 go build -tags="release" -o "lightmeter.exe" -ldflags "${BUILD_INFO_FLAGS}"

static_release: static_www domain_mapping_list po2go
	go build -tags="release" -o "lightmeter" -ldflags \
		"${BUILD_INFO_FLAGS} -linkmode external -extldflags '-static' -s -w" -a -v

static_www:
	go generate -tags="release" gitlab.com/lightmeter/controlcenter/staticdata

domain_mapping_list: domainmapping/generated_list.go

domainmapping/generated_list.go: domainmapping/mapping.json
	go generate gitlab.com/lightmeter/controlcenter/domainmapping

mocks:
	go generate -tags="dev" gitlab.com/lightmeter/controlcenter/dashboard

po2go:
	go generate gitlab.com/lightmeter/controlcenter/po

code2po:
	go run tools/code2po/main.go -i www -o po/en/LC_MESSAGES/controlcenter.po
	go run tools/code2po/main.go -i www -pot -o po/controlcenter.pot

swag:
	go run github.com/swaggo/swag/cmd/swag init --generalInfo api/http.go
	cp docs/swagger.json www/api.json

clean: clean_binaries clean_swag clean_staticdata clean_mocks
	rm -f dependencies.svg

clean_binaries:
	rm -f lightmeter lightmeter.exe

clean_staticdata:
	rm -f staticdata/http_vfsdata.go

clean_swag:
	rm -f docs/docs.go docs/swagger.json docs/swagger.yaml www/api.json

clean_mocks:
	rm -f dashboard/mock/dashboard_mock.go

dependencies.svg: go.sum go.mod
	go mod graph | tools/gen_deps_graph.py | dot -Tsvg > dependencies.svg
