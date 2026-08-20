# Makefile for webclone

BIN      := webclone
GUI_BIN  := webclone-gui
PKG      := ./cmd/webclone
GUI_PKG  := ./cmd/webclone-gui
GO       := go
VERSION  ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS  := -X main.version=$(VERSION)

.PHONY: all build run test clean install uninstall lint fmt vet

all: build

build:
        $(GO) build -ldflags "$(LDFLAGS)" -o $(BIN) $(PKG)

gui:
        $(GO) build -ldflags "$(LDFLAGS)" -o $(GUI_BIN) $(GUI_PKG)

run: build
        ./$(BIN) $(ARGS)

test:
        $(GO) test -race ./...

install: build
        $(GO) install $(PKG)

uninstall:
        rm -f $$(go env GOPATH)/bin/$(BIN)

fmt:
        $(GO) fmt ./...

vet:
        $(GO) vet ./...

lint: vet

clean:
        rm -f $(BIN) $(BIN)-* webclone.exe
        rm -rf dist/

# Cross-compile for common platforms
dist:
        mkdir -p dist
        GOOS=linux   GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-linux-amd64   $(PKG)
        GOOS=linux   GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-linux-arm64   $(PKG)
        GOOS=darwin  GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-darwin-amd64  $(PKG)
        GOOS=darwin  GOARCH=arm64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-darwin-arm64  $(PKG)
        GOOS=windows GOARCH=amd64 $(GO) build -ldflags "$(LDFLAGS)" -o dist/$(BIN)-windows-amd64 $(PKG).exe
