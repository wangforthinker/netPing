SHELL          = /bin/bash

BUILD_IMAGE   = golang:1.7.5
PROJECT_NAME  = netPing

GROUP_NAME     = acs
TARGET         = netping
IMAGE_NAME     = $(GROUP_NAME)/$(TARGET)
MAJOR_VERSION = 1.0.0
DATE = $(shell date +%Y%m%d)

MAJOR_VERSION = $(shell cat VERSION)
GIT_VERSION   = $(shell git log -1 --pretty=format:%h)
GIT_NOTES     = $(shell git log -1 --oneline)

default: image

build:
	docker run --rm -v $(shell pwd):/go/src/github.com/wangforthinker/${PROJECT_NAME} -w /go/src/github.com/wangforthinker/${PROJECT_NAME} ${BUILD_IMAGE} make build-local

build-local:
	go get -v github.com/gorilla/context
	go build -a -v


image:
	make build
	docker build --rm -t ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION}-${DATE}  .