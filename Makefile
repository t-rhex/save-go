VERSION := $(shell git describe --tags --always --dirty)
BINARY := save
GOFILES := $(wildcard *.go)
DEV_BINARY := $(BINARY)-dev
DEV_VERSION := $(VERSION)-dev
DEV_CONFIG_PATH := $(HOME)/.config/save-dev

# Check if running on macOS or Linux
UNAME := $(shell uname)

# Default to user's home directory for installation
ifeq ($(UNAME), Darwin)
    INSTALL_PATH := $(HOME)/bin
else
    INSTALL_PATH := $(HOME)/.local/bin
endif

LDFLAGS := -X main.Version=$(VERSION)
DEV_LDFLAGS := -X main.Version=$(DEV_VERSION) -X main.ConfigPath=$(DEV_CONFIG_PATH)

.PHONY: all build build-dev clean install uninstall user-install update dev test help

# Add help target as default
.DEFAULT_GOAL := help

# Help target
help:
	@echo "Available targets:"
	@echo "  help           - Show this help message"
	@echo "  all            - Build the application (default)"
	@echo "  build          - Build the application"
	@echo "  build-dev      - Build the application with debug information"
	@echo "  clean          - Remove built binaries"
	@echo "  install        - Install system-wide (requires sudo)"
	@echo "  user-install   - Install for current user only"
	@echo "  uninstall      - Remove the application"
	@echo "  update         - Update to latest version"
	@echo "  dev            - Build and run development version"
	@echo "  test           - Run tests"
	@echo
	@echo "Version: $(VERSION)"
	@echo "Install path: $(INSTALL_PATH)"

all: build

build: $(GOFILES)
	go build -ldflags "$(LDFLAGS)" -o $(BINARY)

build-dev: $(GOFILES)
	@mkdir -p $(DEV_CONFIG_PATH)
	@echo "Building development version with config path: $(DEV_CONFIG_PATH)"
	go build -ldflags "$(DEV_LDFLAGS)" -gcflags="all=-N -l" -o $(DEV_BINARY)

clean:
	rm -f $(BINARY)
	rm -f $(DEV_BINARY)
	@echo "Note: Development config directory $(DEV_CONFIG_PATH) is preserved"

test:
	go test -v ./...

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
dev: build-dev
	@./$(DEV_BINARY)