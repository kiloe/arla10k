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
BASE := arla/base
IMAGE := arla/10k
CONF := src/arla/querystore/conf
JS := src/arla/querystore/js
SQL := src/arla/querystore/sql
RUN := docker run $(RM) -i -v $(PWD):/app -w /app -v $(PWD)/$(CONF):/etc/postgresql/9.4/main
GO := $(RUN) --entrypoint /usr/bin/go -e GOPATH=/app -e CGO_ENABLED=0 $(BASE)
DELETE := $(RUN) --entrypoint /bin/rm $(BASE)
BROWSERIFY := $(RUN) --entrypoint /usr/local/bin/browserify $(BASE)
PEGJS := $(RUN) --entrypoint /usr/local/bin/pegjs $(BASE)

build: bin/arla
	docker build -t $(IMAGE) .

bin/arla: src/arla/querystore/postgres_init.go
	mkdir -p bin
	$(GO) get arla
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
	docker push $(IMAGE)

clean:
	rm -f bin/arla
	rm -f src/arla/querystore/postgres_init.go
	rm -f $(SQL)/02_js.sql
	rm -f $(JS)/graphql.js
	rm -rf client/spec/dist
	rm -rf client/dist
	rm -f client-test.log
	#$(DELETE) -rf src/code.google.com/ src/github.com/ src/golang.org/
	docker rm -f 10k 2>/dev/null || true
	docker rmi -f $(IMAGE) 2>/dev/null || true

test-client: build
	docker rm -f 10k 2>/dev/null || true
	docker run -d --name 10k -i \
		-p 3000:80 \
		-v $(PWD)/test-app:/app \
		-w /app \
		-v $(PWD)/$(CONF):/etc/postgresql/9.4/main \
		$(IMAGE) \
			--secret=testing \
			--debug \
			--config-path=./config.js
	(cd client && npm test) || (docker logs 10k &> client-test.log && false)
	docker rm -f 10k 2>/dev/null || true


test-server: all
	$(GO) get -t arla
	$(GO) test -v arla

test: test-server test-client

.PHONY: default build test test-client test-server release clean
