#
# Makefile for building the container image
#
default: all

SHELL := /bin/bash
PWD := $(shell pwd)
BASE := arla/base
CONF := src/arla/querystore/conf
JS := src/arla/querystore/js
SQL := src/arla/querystore/sql
RUN := docker run --rm -i -v $(PWD):/app -w /app -v $(PWD)/$(CONF):/etc/postgresql/9.4/main
GO := $(RUN) --entrypoint /usr/bin/go -e GOPATH=/app $(BASE)
BROWSERIFY := $(RUN) --entrypoint /usr/local/bin/browserify $(BASE)
PEGJS := $(RUN) --entrypoint /usr/local/bin/pegjs $(BASE)

build: bin/arla
	docker build -t arla/10k .

bin/arla: src/arla/querystore/postgres_init.go
	mkdir -p bin
	$(GO) build -o bin/arla arla

$(JS)/graphql.js: $(JS)/graphql.peg
	$(PEGJS) -e 'module.exports' < $< > $@

$(SQL)/02_js.sql: $(JS)/index.js $(JS)/graphql.js
	$(BROWSERIFY) $< -t [ /usr/local/lib/node_modules/babelify --modules common ] >> $@

src/arla/querystore/postgres_init.go: $(SQL)/02_js.sql $(wildcard $(SQL)/*.sql)
	echo 'package querystore' > $@
	echo 'const postgresInitScript = `' >> $@
	cat $(SQL)/*.sql | sed "s/\`/'/g" >> $@
	echo '`' >> $@

all: bin/arla

release: build
	docker push arla/10k

clean:
	rm -f bin/arla
	rm -f src/arla/querystore/postgres_init.go
	rm -f $(SQL)/02_js.sql
	rm -f $(JS)/graphql.js

test: all
	$(GO) test -v arla/querystore
	$(GO) test -v arla/mutationstore

.PHONY: default build test release clean
