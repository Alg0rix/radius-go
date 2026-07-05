.PHONY: build test vulncheck

build:
	go build ./cmd/api

test:
	go test ./...

vulncheck:
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...
