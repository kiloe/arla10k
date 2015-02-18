#
# Makefile for building the container image
#
default: build

SHELL := /bin/bash


build:
	docker build -t arla .

enter: build
	docker run -it --rm --name arla_test arla /bin/bash

test: build
	docker run -it --rm -p 3000:3000 --name arla_test -e AUTH_SECRET=testing arla

release: test
	docker push arla

clean:
	docker rmi arla
	docker rm arla_test

.PHONY: default build test release clean enter
