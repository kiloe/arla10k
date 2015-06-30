#
# Makefile for building arla
# Note: This is usually called as part of the docker container build
#
default: bin/arla

GO = CGO_ENABLED=0 GOPATH=`pwd` go
SHELL := /bin/bash

bin/arla: src/arla/querystore/postgres_runtime.go
	$(GO) version
	$(GO) build -o bin/arla arla

src/arla/querystore/postgres_runtime.go: src/db/graphql.js
	echo 'package querystore' > $@
	echo 'const postgresRuntimeScript = `' >> $@
	browserify src/db/index.js -t [ /usr/local/lib/node_modules/babelify --modules common ] | sed "s/\`/'/g" >> $@
	echo '`' >> $@
	cat -n $@

src/db/graphql.js:
	pegjs -e 'module.exports' < src/db/graphql.peg > $@

install: bin/arla
	cp bin/arla /usr/bin/
	cp src/db/pg_hba.conf /etc/postgresql/9.4/main/
	cp src/db/postgresql.conf /etc/postgresql/9.4/main/
	mkdir -p /var/state

clean:
	rm -f bin/arla
	rm -f src/arla/querystore/postgres_runtime.go
	rm -f src/db/graphql.js

.PHONY: default clean install
