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

all:
	$(error Use make (dev|release|static_release) instead)

race:
	./tools/go_test.sh -race

# TODO: add vue as as a build dependency once we adopt the vue.js UI.
BUILD_DEPENDENCIES = go gcc ragel
K := $(foreach exec,$(BUILD_DEPENDENCIES),\
      $(if $(shell which $(exec)),nothing here,$(error "Build dependency program $(exec) could not be found in PATH. Check README.md for more info")))

dev_headless_pre_build: mocks translations swag static_www postfix_parser domain_mapping_list po2go

dev_pre_build: npminstall mocks translations frontend_root swag static_www postfix_parser domain_mapping_list po2go

pre_build: npminstall translations frontend_root static_www postfix_parser domain_mapping_list po2go

pre_release: pre_build recommendation_release

dev_bin: dev_pre_build recommendation_dev
	go build -tags="dev include no_postgres no_mysql no_clickhouse no_mssql" -o "lightmeter" -ldflags "${BUILD_INFO_FLAGS}"

dev_headless_bin: dev_headless_pre_build recommendation_dev
	go build -tags="dev include no_postgres no_mysql no_clickhouse no_mssql" -o "lightmeter" -ldflags "${BUILD_INFO_FLAGS}"

release_bin: pre_release
	go build -tags="release include no_postgres no_mysql no_clickhouse no_mssql" -o "lightmeter" -ldflags "${BUILD_INFO_FLAGS}"

static_release_bin: pre_release
	go build -tags="release include no_postgres no_mysql no_clickhouse no_mssql" -o "lightmeter" -ldflags \
		"${BUILD_INFO_FLAGS} -linkmode external -extldflags '-static' -s -w" -a -v

static_www:
	go generate -tags="include" gitlab.com/lightmeter/controlcenter/staticdata

domain_mapping_list: domainmapping/generated_list.go

domainmapping/generated_list.go: domainmapping/mapping.json
	go generate gitlab.com/lightmeter/controlcenter/domainmapping

recommendation_dev:
	go generate -tags="dev" gitlab.com/lightmeter/controlcenter/recommendation

recommendation_release:
	go generate -tags="release" gitlab.com/lightmeter/controlcenter/recommendation

mocks: postfix_parser dashboard_mock insights_mock

dashboard_mock:
	go generate -tags="dev" gitlab.com/lightmeter/controlcenter/dashboard

insights_mock:
	go generate -tags="dev" gitlab.com/lightmeter/controlcenter/insights/core

po2go:
	go generate -tags="dev" gitlab.com/lightmeter/controlcenter/po

go2po:
	sh ./tools/go2poutil.sh

swag:
	go generate -tags="dev" gitlab.com/lightmeter/controlcenter/api
	cp api/docs/swagger.json www/api.json

clean: clean_binaries clean_swag clean_staticdata clean_mocks clean_postfix_parser
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

make testlocal:
	./tools/go_test_local.sh

postfix_parser:
	go generate gitlab.com/lightmeter/controlcenter/pkg/postfix/logparser/rawparser

clean_postfix_parser:
	@rm -vf pkg/postfix/logparser/rawparser/*.gen.go

npminstall:
	sh ./frontend/controlcenter/npm_install.sh

frontend_root:
	sh ./frontend/controlcenter/root_build.sh

# TODO: remove git as a build dependency (not used on release tarballs)
# TODO: consider to use git as dep in container
restore:
	git checkout  ./frontend/controlcenter/package-lock.json
	git checkout  ./frontend/controlcenter/src/translation/translations.json
	git clean -fdx ./www
	git checkout ./www

release: release_bin restore

dev: dev_bin restore

devheadless: dev_headless_bin restore

static_release: static_release_bin restore

#template from https://github.com/Polyconseil/vue-gettext/blob/master/Makefile

NODE_BINDIR = ./frontend/controlcenter/node_modules/.bin
export PATH := $(NODE_BINDIR):$(PATH)
LOGNAME ?= $(shell logname)

# adding the name of the user's login name to the template file, so that
# on a multi-user system several users can run this without interference
TEMPLATE_POT ?= ./po/template.pot

# Where to find input files (it can be multiple paths).
INPUT_FILES = ./frontend/controlcenter/src

# Where to write the files generated by this makefile.
OUTPUT_DIR = ./po

TRANSLATION_OUTPUT = ./frontend/controlcenter/src/translation/translations.json

# Available locales for the app.
# TODO: Consider to use new approch to retrieve contry codes
LOCALES = en de pt_BR pl

# Name of the generated .po files for each available locale.
LOCALE_FILES ?= $(patsubst %,$(OUTPUT_DIR)/%/LC_MESSAGES/app.po,$(LOCALES))

GETTEXT_SOURCES ?= $(shell find $(INPUT_FILES) -name '*.jade' -o -name '*.html' -o -name '*.js' -o -name '*.vue' 2> /dev/null)

cleantemplatepot:
	rm -f $(TEMPLATE_POT)

makemessages: $(TEMPLATE_POT)

# Create a main .pot template, then generate .po files for each available language.
# Thanx to Systematic: https://github.com/Polyconseil/systematic/blob/866d5a/mk/main.mk#L167-L183
$(TEMPLATE_POT): $(GETTEXT_SOURCES)
# `dir` is a Makefile built-in expansion function which extracts the directory-part of `$@`.
# `$@` is a Makefile automatic variable: the file name of the target of the rule.
# => `mkdir -p /tmp/`
	mkdir -p $(dir $@)
# Extract gettext strings from templates files and create a POT dictionary template.
	gettext-extract --quiet  --output $@ $(GETTEXT_SOURCES)
# Generate .po files for each available language.
	@for lang in $(LOCALES); do \
		export PO_FILE=$(OUTPUT_DIR)/$$lang/LC_MESSAGES/app.po; \
		mkdir -p $$(dirname $$PO_FILE); \
		if [ -f $$PO_FILE ]; then  \
			echo "msgmerge --update $$PO_FILE $@"; \
			msgmerge --lang=$$lang --update $$PO_FILE $@ || break ;\
		else \
			msginit --no-translator --locale=$$lang --input=$@ --output-file=$$PO_FILE || break ; \
			msgattrib --no-wrap --no-obsolete -o $$PO_FILE $$PO_FILE || break; \
		fi; \
	done;

translations:
	gettext-compile --output $(TRANSLATION_OUTPUT) $(LOCALE_FILES) || true