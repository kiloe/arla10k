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

build: bin/arla
	docker build -t arla/10k .

bin/arla: src/arla/querystore/postgres_runtime.go
	mkdir -p bin
	$(GO) build -o bin/arla arla

src/db/graphql.js:
	$(PEGJS) -e 'module.exports' < src/db/graphql.peg > $@

src/arla/querystore/postgres_runtime.go: src/db/graphql.js $(wildcard src/db/**/*)
	echo 'package querystore' > $@
	echo 'const postgresRuntimeScript = `' >> $@
	$(BROWSERIFY) src/db/index.js -t [ /usr/local/lib/node_modules/babelify --modules common ] | sed "s/\`/'/g" >> $@
	echo '`' >> $@
	cat -n $@

all: bin/arla

release: build
	docker push arla/10k

clean:
	rm -f src/arla/querystore/postgres_runtime.go
	rm -f src/arla/querystore/postgres_runtime.go.tmp
	rm -f src/db/graphql.js
	rm -f bin/arla

test: all
	$(GO) test -v arla/querystore
	$(GO) test -v arla/mutationstore
	$(GO) test -v arla/identstore

.PHONY: default build test release clean
