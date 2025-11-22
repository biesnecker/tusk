.PHONY: all build release install clean test

BINARY_NAME=tusk
INSTALL_PATH=$(HOME)/.local/bin

all: build

build:
	go build -o $(BINARY_NAME) .

release:
	go build -ldflags="-s -w" -o $(BINARY_NAME) .

install: release
	mkdir -p $(INSTALL_PATH)
	cp $(BINARY_NAME) $(INSTALL_PATH)/$(BINARY_NAME)
	@echo "Installed to $(INSTALL_PATH)/$(BINARY_NAME)"

test:
	go test -v ./...

clean:
	rm -f $(BINARY_NAME)
