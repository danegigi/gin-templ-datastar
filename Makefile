# Makefile for go-tut
# Usage: make [target]

BINARY=bin/go-tut
TEMPL_CMD=$(shell which templ || echo $(HOME)/go/bin/templ)
TAILWIND_CMD=./node_modules/.bin/tailwindcss

.PHONY: all build generate css run dev clean test tidy

## all: generate + build (default)
all: generate css build

## generate: run templ code generation
generate:
	@echo "→ generating templ..."
	$(TEMPL_CMD) generate ./...

## css: build Tailwind CSS
css:
	@echo "→ building Tailwind CSS..."
	$(TAILWIND_CMD) -i static/css/input.css -o static/css/app.css --minify

## build: compile the Go binary
build:
	@echo "→ building binary..."
	mkdir -p bin
	go build -o $(BINARY) ./cmd/server

## run: build and start the server
run: all
	$(BINARY)

## dev: watch mode – templ + tailwind + go run (requires air or just runs directly)
dev: generate css
	@echo "→ starting dev server on :8080..."
	go run ./cmd/server

## clean: remove build artifacts
clean:
	rm -rf bin/
	rm -f static/css/app.css
	find . -name '*_templ.go' -delete

## test: run Go tests
test:
	go test ./...

## tidy: tidy Go modules
tidy:
	go mod tidy

## help: print this help message
help:
	@grep -E '^## ' Makefile | sed 's/## /  /'
