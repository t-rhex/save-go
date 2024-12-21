VERSION := 0.1.0
BINARY := save
GOFILES := $(wildcard *.go)
DEV_BINARY := $(BINARY)-dev
DEV_VERSION := 0.1.0
DEV_CONFIG_PATH := $(HOME)/.config/save-dev

# Check if running on macOS or Linux
UNAME := $(shell uname)

# Default installation path (can be overridden)
ifeq ($(UNAME), Darwin)
    INSTALL_PATH ?= $(HOME)/bin
else
    INSTALL_PATH ?= $(HOME)/.local/bin
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
	@echo "  clean-all      - Remove binaries and all config files"
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
	go build -ldflags "-X main.Version=$(DEV_VERSION) -X main.ConfigPath=$(DEV_CONFIG_PATH)" -gcflags="all=-N -l" -o $(DEV_BINARY)

clean:
	rm -f $(BINARY)
	rm -f $(DEV_BINARY)
	@echo "Note: Config files are preserved at:"
	@echo "  - Production: $(HOME)/.save_history.json"
	@echo "  - Development: $(DEV_CONFIG_PATH)/history.json"

clean-all: clean
	@echo "Removing all config files..."
	rm -f $(HOME)/.save_history.json
	rm -rf $(DEV_CONFIG_PATH)
	@# Remove shell configurations
	@SHELL_TYPE=$$(basename $$SHELL); \
	if [ "$$SHELL_TYPE" = "bash" ]; then \
		sed -i.bak '/# Added by save installer/,+1d' $(HOME)/.bashrc; \
		rm -f $(HOME)/.bash_completion.d/$(BINARY); \
		echo "Cleaned up bash configuration"; \
	elif [ "$$SHELL_TYPE" = "zsh" ]; then \
		sed -i.bak '/# Added by save installer/,+1d' $(HOME)/.zshrc; \
		rm -f $(HOME)/.zsh/completion/_$(BINARY); \
		echo "Cleaned up zsh configuration"; \
	fi
	@# Remove backup files
	rm -f $(HOME)/.bashrc.bak $(HOME)/.zshrc.bak
	@echo "All config files removed"

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
	@# Setup shell completion and PATH
	@SHELL_TYPE=$$(basename $$SHELL); \
	if [ "$$SHELL_TYPE" = "bash" ]; then \
		if ! grep -q "$(HOME)/.bash_completion.d/$(BINARY)" "$(HOME)/.bashrc"; then \
			echo "[ -f $(HOME)/.bash_completion.d/$(BINARY) ] && . $(HOME)/.bash_completion.d/$(BINARY)" >> "$(HOME)/.bashrc"; \
			echo "Added bash completion to .bashrc"; \
		fi; \
		if ! grep -q "export PATH=\"$(INSTALL_PATH):\$$PATH\"" "$(HOME)/.bashrc"; then \
			echo "export PATH=\"$(INSTALL_PATH):\$$PATH\"" >> "$(HOME)/.bashrc"; \
			echo "Added $(INSTALL_PATH) to PATH in .bashrc"; \
		fi; \
	elif [ "$$SHELL_TYPE" = "zsh" ]; then \
		if ! grep -q "fpath=($(HOME)/.zsh/completion" "$(HOME)/.zshrc"; then \
			echo "fpath=($(HOME)/.zsh/completion \$$fpath)" >> "$(HOME)/.zshrc"; \
			echo "autoload -U compinit && compinit" >> "$(HOME)/.zshrc"; \
			echo "Added zsh completion to .zshrc"; \
		fi; \
		if ! grep -q "export PATH=\"$(INSTALL_PATH):\$$PATH\"" "$(HOME)/.zshrc"; then \
			echo "export PATH=\"$(INSTALL_PATH):\$$PATH\"" >> "$(HOME)/.zshrc"; \
			echo "Added $(INSTALL_PATH) to PATH in .zshrc"; \
		fi; \
	fi
	@echo "Installation complete."
	@echo "To use save in current session, run:"
	@echo "  export PATH=\"$(INSTALL_PATH):\$$PATH\""
	@echo "Or start a new terminal session."

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