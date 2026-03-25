BINARY := cs
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)
INSTALL_DIR := /usr/local/bin

.PHONY: build install uninstall test lint fmt clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/cs/

install: build
	install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)

test:
	go test ./... -count=1 -race

lint:
	go vet ./...

fmt:
	gofmt -s -w .

clean:
	rm -f $(BINARY)
