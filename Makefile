SHELL          = /bin/bash

BUILD_IMAGE   = golang:1.7.5
PROJECT_NAME  = netPing

GROUP_NAME     = acs
TARGET         = netPing
IMAGE_NAME     = $(GROUP_NAME)/$(TARGET)
MAJOR_VERSION = $(shell cat VERSION)
DATE = $(shell date +%Y%m%d)

MAJOR_VERSION = $(shell cat VERSION)
GIT_VERSION   = $(shell git log -1 --pretty=format:%h)
GIT_NOTES     = $(shell git log -1 --oneline)

default: image

build:
	docker run --rm -v $(shell pwd):/go/src/github.com/wangforthinker/${PROJECT_NAME} -w /go/src/github.com/wangforthinker/${PROJECT_NAME} ${BUILD_IMAGE}  go build -a -v

image:
	docker run --rm -v $(shell pwd):/go/src/github.com/wangforthinker/${PROJECT_NAME} -w /go/src/github.com/wangforthinker/${PROJECT_NAME} ${BUILD_IMAGE}  go build -a -v
	docker build --rm -t ${IMAGE_NAME}:${MAJOR_VERSION}-${GIT_VERSION}-${DATE}  .