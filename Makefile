#! /usr/bin/make
SHELL = /bin/bash
BUILDTIME = $(shell date -u --rfc-3339=seconds)
GITHASH = $(shell git describe --dirty --always --tags)
GITCOMMITNO = $(shell git rev-list --all --count)
SHORTBUILDTAG = v0.0.$(GITCOMMITNO)-$(GITHASH)
BUILDINFO = Build Time:$(BUILDTIME)
LDFLAGS = -X 'main.buildTag=$(SHORTBUILDTAG)' -X 'main.buildInfo=$(BUILDINFO)'

COVERAGE_PATH ?= .coverage


depend: deps
deps:
	go get ./...
	go mod tidy
	go mod vendor

version:
	@echo $(SHORTBUILDTAG)

build:
	CGO_ENABLED=0 go build -mod=vendor -ldflags "$(LDFLAGS)" -o ./bin/searmage ./cmd/searmage

build_c:
	CGO_ENABLED=1 go build -mod=vendor -ldflags "$(LDFLAGS)" -o ./bin/searmage_cgo ./cmd/searmage

unit-test:
	@go test -race -count=3 ./...

test:
	@go test -v -race -count=2 ./...

bench:
	@go test -benchmem -run=^$ -v -count=2 -bench .  ./...

coverage:
	@go test -covermode=count -coverprofile=$(COVERAGE_PATH)

coverage-html:
	@rm $(COVERAGE_PATH) || true
	@$(MAKE) coverage
	@rm $(COVERAGE_PATH).html || true
	@go tool cover -html=$(COVERAGE_PATH) -o $(COVERAGE_PATH).html

coverage-browser:
	@rm $(COVERAGE_PATH) || true
	@$(MAKE) coverage
	@go tool cover -html=$(COVERAGE_PATH)

update-readme-badge:
	@go tool cover -func=$(COVERAGE_PATH) -o=$(COVERAGE_PATH).badge
	@go run github.com/AlexBeauchemin/gobadge@v0.3.0 -filename=$(COVERAGE_PATH).badge
