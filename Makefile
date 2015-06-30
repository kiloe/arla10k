#
# Makefile for building the container image
#
default: all

SHELL := /bin/bash
PWD := $(shell pwd)
BASE := arla/base
RUN := docker run --rm -i -v $(PWD):/app -w /app -v $(PWD)/src/db/:/etc/postgresql/9.4/main 
GO := $(RUN) --entrypoint /usr/bin/go -e GOPATH=/app $(BASE)
BROWSERIFY := $(RUN) --entrypoint /usr/local/bin/browserify $(BASE)
PEGJS := $(RUN) --entrypoint /usr/local/bin/pegjs $(BASE)

build:
	docker build -t arla/10k .

src/db/graphql.js:
	$(PEGJS) -e 'module.exports' < src/db/graphql.peg > $@

src/arla/querystore/postgres_runtime.go: src/db/graphql.js $(wildcard src/db/**/*)
	echo 'package querystore' > $@
	echo 'const postgresRuntimeScript = `' >> $@
	$(BROWSERIFY) src/db/index.js -t [ /usr/local/lib/node_modules/babelify --modules common ] | sed "s/\`/'/g" >> $@
	echo '`' >> $@
	cat -n $@

release: test
	docker push arla/10k

clean:
	rm -f src/arla/querystore/postgres_runtime.go
	rm -f src/arla/querystore/postgres_runtime.go.tmp
	rm -f src/db/graphql.js

all:

test: src/arla/querystore/postgres_runtime.go
	$(GO) test -v arla/querystore

.PHONY: test2 default build test release clean enter
