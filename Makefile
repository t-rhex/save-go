VERSION := 0.1.0
BINARY := save
GOFILES := $(wildcard *.go)

# Check if running on macOS or Linux
UNAME := $(shell uname)

# Default to user's home directory for installation
ifeq ($(UNAME), Darwin)
    INSTALL_PATH := $(HOME)/bin
else
    INSTALL_PATH := $(HOME)/.local/bin
endif

.PHONY: all build clean install uninstall user-install update dev

# TODO: Add test target once tests are implemented
# .PHONY: test
# test:
#   go test -v ./...

all: build

build: $(GOFILES)
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY)

clean:
	rm -f $(BINARY)

# System-wide installation (requires sudo)
install: build
	sudo install -d /usr/local/bin
	sudo install -m 755 ./$(BINARY) /usr/local/bin/$(BINARY)
	@echo "Installing shell completion..."
	sudo mkdir -p /etc/bash_completion.d
	sudo ./$(BINARY) --generate-completion bash > /etc/bash_completion.d/$(BINARY)
	sudo mkdir -p /usr/local/share/zsh/site-functions
	sudo ./$(BINARY) --generate-completion zsh > /usr/local/share/zsh/site-functions/_$(BINARY)
	@echo "Installation complete. You may need to restart your shell."

# User-specific installation (no sudo required)
user-install: build
	@mkdir -p $(INSTALL_PATH)
	@install -m 755 ./$(BINARY) $(INSTALL_PATH)/$(BINARY)
	@mkdir -p $(HOME)/.bash_completion.d
	@./$(BINARY) --generate-completion bash > $(HOME)/.bash_completion.d/$(BINARY)
	@mkdir -p $(HOME)/.zsh/completion
	@./$(BINARY) --generate-completion zsh > $(HOME)/.zsh/completion/_$(BINARY)
	@echo "Installation complete."
	@echo "Please make sure $(INSTALL_PATH) is in your PATH."
	@echo "Add this to your .bashrc or .zshrc if it isn't already there:"
	@echo "export PATH=\"$(INSTALL_PATH):\$$PATH\""
	@echo ""
	@echo "For bash completion, add this to your .bashrc:"
	@echo "[[ -f $(HOME)/.bash_completion.d/$(BINARY) ]] && . $(HOME)/.bash_completion.d/$(BINARY)"
	@echo ""
	@echo "For zsh completion, add this to your .zshrc:"
	@echo "fpath=($(HOME)/.zsh/completion \$$fpath)"
	@echo ""
	@echo "You may need to restart your shell or run:"
	@echo "export PATH=\"$(INSTALL_PATH):\$$PATH\""
	@echo "to use the command immediately"

# Update existing installation
update: build
	@echo "Updating $(BINARY) to version $(VERSION)..."
	@$(MAKE) uninstall
	@$(MAKE) user-install
	@echo "Update complete!"

uninstall:
	rm -f $(INSTALL_PATH)/$(BINARY)
	rm -f $(HOME)/.bash_completion.d/$(BINARY)
	rm -f $(HOME)/.zsh/completion/_$(BINARY)

# Development helpers
dev: build
	@./$(BINARY)

.DEFAULT_GOAL := all