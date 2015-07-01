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
JS := src/arla/querystore/runtime/

build: bin/arla
	docker build -t arla/10k .

bin/arla: src/arla/querystore/postgres_runtime.go
	mkdir -p bin
	$(GO) build -o bin/arla arla

$(JS)/graphql.js: $(JS)/graphql.peg
	$(PEGJS) -e 'module.exports' < $< > $@

$(JS)/index.compiled.js: $(JS)/index.js $(JS)/graphql.js
	$(BROWSERIFY) $< -t [ /usr/local/lib/node_modules/babelify --modules common ] >> $@

src/arla/querystore/postgres_runtime.go:  $(JS)/runtime.js
	echo 'package querystore' > $@
	echo 'const postgresRuntimeScript = `' >> $@
	cat $(JS)/index.compiled.js  | sed "s/\`/'/g" >> $@
	echo '`' >> $@
	cat -n $@

all: bin/arla

release: build
	docker push arla/10k

clean:
	rm -f bin/arla
	rm -f src/arla/querystore/postgres_runtime.go
	rm -f $(JS)/index.compiled.js
	rm -f $(JS)/graphql.js

test: all
	$(GO) test -v arla/querystore
	$(GO) test -v arla/mutationstore
	$(GO) test -v arla/identstore

.PHONY: default build test release clean
