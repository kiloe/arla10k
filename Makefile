#
# Makefile for building the container image
#
default: all

# fix issue where removing containers breaks ci
ifeq ($(CIRCLECI),true)
RM :=
else
RM := --rm
endif

SHELL := /bin/bash
PWD := $(shell pwd)
IMAGE := arla/10k
CONF := src/arla/querystore/conf
JS := src/arla/querystore/js
SQL := src/arla/querystore/sql
RUN := docker run $(RM) -i -v $(PWD):/app -w /app -v $(PWD)/$(CONF):/etc/postgresql/9.4/main
GOPATH := $(PWD)
GO := GOPATH=$(GOPATH) go
BROWSERIFY := ./node_modules/.bin/browserify
PEGJS := ./node_modules/.bin/pegjs

build: bin/arla
	docker build -t $(IMAGE) .

bin/arla: src/arla/querystore/postgres_init.go
	mkdir -p bin
	CGO_ENABLED=0 $(GO) build -tags netgo -installsuffix netgo -o bin/arla arla

bin/test: bin/arla
	mkdir -p bin
	CGO_ENABLED=0 $(GO) test -c -o bin/test -v arla

node_modules:
	npm install

$(JS)/graphql.js: $(JS)/graphql.peg node_modules
	$(PEGJS) -e 'module.exports' < $< > $@

$(SQL)/02_js.sql: $(JS)/index.js $(JS)/graphql.js node_modules
	$(BROWSERIFY) $< -t [ babelify --modules common ] >> $@

src/arla/querystore/postgres_init.go: $(SQL)/02_js.sql $(wildcard $(SQL)/*.sql)
	echo 'package querystore' > $@
	echo 'const postgresInitScript = `' >> $@
	cat $(SQL)/*.sql | sed "s/\`/'/g" >> $@
	echo '`' >> $@

all: bin/arla

release: build
	docker push $(IMAGE)

clean:
	rm -rf node_modules
	rm -f bin/arla
	rm -f bin/test
	rm -f src/arla/querystore/postgres_init.go
	rm -f $(SQL)/02_js.sql
	rm -f $(JS)/graphql.js
	rm -rf pkg/
	rm -rf client/spec/dist
	rm -rf client/dist
	rm -f client-test.log
	docker rm -f 10k 2>/dev/null || true
	docker rmi -f $(IMAGE) 2>/dev/null || true

test-client: build
	docker rm -f 10k 2>/dev/null || true
	docker run -i --name 10k -i \
		-p 3030:80 \
		-v $(PWD)/test-app:/app \
		-w /app \
		-v $(PWD)/$(CONF):/etc/postgresql/9.4/main \
		$(IMAGE) \
			--secret=testing \
			--debug \
			--config-path=./config.js &
	(cd client && npm test) || (docker logs 10k &> client-test.log && false)
	docker rm -f 10k 2>/dev/null || true


test-server: bin/test build
	$(RUN) --entrypoint bin/test $(IMAGE)

test: test-server test-client

.PHONY: default build test test-client test-server release clean
