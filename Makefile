BINARY := cs
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -s -w -X main.version=$(VERSION)
INSTALL_DIR := /usr/local/bin
GOMOBILE := $(HOME)/go/bin/gomobile

.PHONY: build install uninstall test test-cscore test-mobile fuzz-cscore mobile-ios mobile-clean lint fmt clean

build:
	go build -trimpath -ldflags "$(LDFLAGS)" -o $(BINARY) ./cmd/cs/

install: build
	install -m 755 $(BINARY) $(INSTALL_DIR)/$(BINARY)

uninstall:
	rm -f $(INSTALL_DIR)/$(BINARY)

test:
	go test ./... -count=1 -race

test-cscore:
	go test ./pkg/cscore/... -count=1 -race -v

fuzz-cscore:
	go test ./pkg/cscore/ -fuzz=FuzzCalcEval -fuzztime=30s
	go test ./pkg/cscore/ -fuzz=FuzzSubnetCalc -fuzztime=30s
	go test ./pkg/cscore/ -fuzz=FuzzSearchJSON -fuzztime=30s
	go test ./pkg/cscore/ -fuzz=FuzzRenderMarkdownToHTML -fuzztime=30s
	go test ./pkg/cscore/ -fuzz=FuzzGetSheetJSON -fuzztime=30s

test-mobile:
	go test ./mobile/... -count=1 -race -v

mobile-ios:
	PATH="$(HOME)/go/bin:$(PATH)" $(GOMOBILE) bind -target=ios -ldflags="-s -w" -o mobile/Cscore.xcframework ./mobile/

mobile-clean:
	rm -rf mobile/Cscore.xcframework

lint:
	go vet ./...

fmt:
	gofmt -s -w .

clean:
	rm -f $(BINARY)
	rm -rf mobile/Cscore.xcframework
