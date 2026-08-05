APP := ais

.PHONY: test build build-windows build-macos-arm64 build-linux-amd64

test:
	go test ./...

build:
	go build -trimpath -ldflags "-s -w" -o $(APP) ./cmd/ais

build-windows:
	GOOS=windows GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o $(APP).exe ./cmd/ais

build-macos-arm64:
	GOOS=darwin GOARCH=arm64 go build -trimpath -ldflags "-s -w" -o $(APP)-darwin-arm64 ./cmd/ais

build-linux-amd64:
	GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w" -o $(APP)-linux-amd64 ./cmd/ais
