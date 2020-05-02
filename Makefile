# vim: noexpandtab

export GO111MODULE = on
export CGO_ENABLED = 1

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

all: dev

dev: mocks swag
	go build -tags="dev" -o "lightmeter"

release: static_www
	go build -tags="release" -o "lightmeter" -ldflags \
		"-X ${PACKAGE_VERSION}.Commit=${GIT_COMMIT} -X ${PACKAGE_VERSION}.TagOrBranch=${GIT_BRANCH} -X ${PACKAGE_VERSION}.Version=${APP_VERSION}"

static_release: static_www 
	go build -tags="release" -o "lightmeter" -ldflags \
		"-X ${PACKAGE_VERSION}.Commit=${GIT_COMMIT} -X ${PACKAGE_VERSION}.TagOrBranch=${GIT_BRANCH} -X ${PACKAGE_VERSION}.Version=${APP_VERSION} -linkmode external -extldflags '-static' -s -w" -a -v

static_www:
	go generate -tags="release" gitlab.com/lightmeter/controlcenter/staticdata

mocks:
	go generate -tags="dev" gitlab.com/lightmeter/controlcenter/dashboard

swag:
	go run github.com/swaggo/swag/cmd/swag init --generalInfo api/http.go
	cp docs/swagger.json www/api.json

clean: clean_binaries clean_swag clean_staticdata clean_mocks

clean_binaries:
	rm -f lightmeter

clean_staticdata:
	rm -f staticdata/http_vfsdata.go

clean_swag:
	rm -f docs/docs.go docs/swagger.json docs/swagger.yaml www/api.json

clean_mocks:
	rm -f dashboard/mock/dashboard_mock.go

