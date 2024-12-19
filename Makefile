VERSION := 0.1.0
BINARY := save
GOFILES := $(wildcard *.go)
PREFIX := /usr/local
INSTALL_PATH := $(PREFIX)/bin

.PHONY: all build clean install uninstall

all: build

build: $(GOFILES)
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY)

clean:
	rm -f $(BINARY)

install: build
	install -d $(INSTALL_PATH)
	install -m 755 $(BINARY) $(INSTALL_PATH)
	@echo "Installing shell completion..."
	@mkdir -p /etc/bash_completion.d
	@$(BINARY) --generate-completion bash > /etc/bash_completion.d/save
	@mkdir -p /usr/local/share/zsh/site-functions
	@$(BINARY) --generate-completion zsh > /usr/local/share/zsh/site-functions/_save
	@echo "Installation complete. You may need to restart your shell."

uninstall:
	rm -f $(INSTALL_PATH)/$(BINARY)
	rm -f /etc/bash_completion.d/save
	rm -f /usr/local/share/zsh/site-functions/_save