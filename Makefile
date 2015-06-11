#
# Makefile for building the container image
#
default: build

SHELL := /bin/bash
PWD := $(shell pwd)

build:
	docker build -t arla/10k .

test: build
	docker run -it \
		--rm -p 3001:3001 \
		--name arla_test \
		-v $(PWD)/test-app:/var/lib/arla/app \
		-e AUTH_SECRET=testing \
		-e DEBUG=true \
		arla/10k -test

release: test
	docker push arla/10k

clean:
	docker rm  arla_test || echo 'ok'
	docker rmi arla || echo 'ok'

.PHONY: default build test release clean enter
