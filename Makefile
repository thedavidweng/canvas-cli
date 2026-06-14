.PHONY: build test vet fmt lint check

build:
	go build -o bin/canvas ./cmd/canvas

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w ./cmd ./internal

lint: fmt vet test

check: lint build


changelog:
	git cliff -o CHANGELOG.md

changelog-preview:
	git cliff --latest
