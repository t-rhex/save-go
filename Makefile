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

.PHONY: all build clean install uninstall user-install

all: build

build: $(GOFILES)
	go build -ldflags "-X main.Version=$(VERSION)" -o $(BINARY)

clean:
	rm -f $(BINARY)

# System-wide installation (requires sudo)
install: build
	sudo install -d /usr/local/bin
	sudo install -m 755 $(BINARY) /usr/local/bin
	@echo "Installing shell completion..."
	sudo mkdir -p /etc/bash_completion.d
	sudo $(BINARY) --generate-completion bash > /etc/bash_completion.d/save
	sudo mkdir -p /usr/local/share/zsh/site-functions
	sudo $(BINARY) --generate-completion zsh > /usr/local/share/zsh/site-functions/_save
	@echo "Installation complete. You may need to restart your shell."

# User-specific installation (no sudo required)
user-install: build
	@mkdir -p $(INSTALL_PATH)
	@install -m 755 $(BINARY) $(INSTALL_PATH)
	@mkdir -p $(HOME)/.bash_completion.d
	@$(BINARY) --generate-completion bash > $(HOME)/.bash_completion.d/save
	@mkdir -p $(HOME)/.zsh/completion
	@$(BINARY) --generate-completion zsh > $(HOME)/.zsh/completion/_save
	@echo "Installation complete."
	@echo "Please make sure $(INSTALL_PATH) is in your PATH."
	@echo "Add this to your .bashrc or .zshrc if it isn't already there:"
	@echo "export PATH=\"$(INSTALL_PATH):\$$PATH\""
	@echo ""
	@echo "For bash completion, add this to your .bashrc:"
	@echo "[[ -f $(HOME)/.bash_completion.d/save ]] && . $(HOME)/.bash_completion.d/save"
	@echo ""
	@echo "For zsh completion, add this to your .zshrc:"
	@echo "fpath=($(HOME)/.zsh/completion \$$fpath)"

uninstall:
	rm -f $(INSTALL_PATH)/$(BINARY)
	rm -f $(HOME)/.bash_completion.d/save
	rm -f $(HOME)/.zsh/completion/_save