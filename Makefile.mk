#
# Makefile for building arla
# Note: This is usually called as part of the docker container build
#
default: build

GO = CGO_ENABLED=0 GOPATH=`pwd` go
SHELL := /bin/bash
PLV8JS=$(wildcard src/db/*.js)

dist:
	mkdir -p dist/bin

dist/graphql.js: dist
	pegjs -e 'module.exports' < src/db/graphql.peg >> $@

js: dist/graphql.js
	cp src/db/*.js dist

dist/template.sql: js
	cp src/db/template.sql $@

dist/bin/initdb: dist/template.sql
	cp src/db/initdb.bash $@

dist/bin/init: dist/bin/initdb
	$(GO) get init
	$(GO) build -o $@ init

build: dist/bin/init

install: build
	mkdir -p /var/lib/arla/
	cp -r dist/* /var/lib/arla/
	cp src/db/pg_hba.conf /etc/postgresql/9.4/main/
	cp src/db/postgresql.conf /etc/postgresql/9.4/main/
	npm install --global ./src/api
	mkdir -p /var/lib/arla/data

clean:
	rm -rf ./dist

.PHONY: default build dist clean js
