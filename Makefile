.PHONY: build test lint clean

BINARY=plugin
PLATFORMS=linux/amd64 linux/arm64 darwin/arm64

build:
	go build -trimpath -ldflags="-s -w" -o $(BINARY) .

test:
	go test ./...

lint:
	golangci-lint run ./...

clean:
	rm -f $(BINARY)

build-all:
	@for platform in $(PLATFORMS); do \
		GOOS=$${platform%%/*} GOARCH=$${platform##*/} CGO_ENABLED=0 \
		go build -trimpath -ldflags="-s -w" -o dist/$(BINARY)-$${platform%%/*}-$${platform##*/} .; \
	done
