APP := ais
# 版本取自最近的标签;非标签提交会带上 -<n>-g<sha>,工作树脏则带 -dirty。
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null | sed 's/^v//' || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)

# 与 .github/workflows/build.yml 里的版本保持一致,避免本地过了 CI 却挂。
GOLANGCI_VERSION := v2.13.2
GOLANGCI := $(shell go env GOPATH)/bin/golangci-lint

.PHONY: test lint fmt check build build-windows build-macos-arm64 build-linux-amd64

test:
	go test ./...

# 首次运行或本地版本与 CI 不一致时自动安装对应版本。
lint:
	@if ! "$(GOLANGCI)" --version 2>/dev/null | grep -q "$(patsubst v%,%,$(GOLANGCI_VERSION))"; then \
		echo "installing golangci-lint $(GOLANGCI_VERSION)"; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION); \
	fi
	"$(GOLANGCI)" run ./...

# 按 .golangci.yml 的 formatters 自动改写(目前是 gofmt)。
fmt:
	@if ! "$(GOLANGCI)" --version 2>/dev/null | grep -q "$(patsubst v%,%,$(GOLANGCI_VERSION))"; then \
		echo "installing golangci-lint $(GOLANGCI_VERSION)"; \
		go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_VERSION); \
	fi
	"$(GOLANGCI)" fmt ./...

# 提交前跑这个:与 CI 的 test + lint 两个 job 等价。
check: test lint

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP) ./cmd/ais

build-windows:
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP).exe ./cmd/ais

build-macos-arm64:
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP)-darwin-arm64 ./cmd/ais

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "$(LDFLAGS)" -o $(APP)-linux-amd64 ./cmd/ais
