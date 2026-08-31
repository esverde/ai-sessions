APP := ais
# 版本取自最近的标签;非标签提交会带上 -<n>-g<sha>,工作树脏则带 -dirty。
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

.PHONY: test build build-windows build-macos-arm64 build-linux-amd64

test:
	go test ./...

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP) ./cmd/ais

build-windows:
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP).exe ./cmd/ais

build-macos-arm64:
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP)-darwin-arm64 ./cmd/ais

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP)-linux-amd64 ./cmd/ais
