#!/usr/bin/env bash

# Uncomment for debugging
# set -x  # Print each command before executing
set -e    # Exit on error

# Color codes
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Print with color functions
info() { echo -e "${BLUE}INFO:${NC} $1"; }
success() { echo -e "${GREEN}SUCCESS:${NC} $1"; }
warn() { echo -e "${YELLOW}WARNING:${NC} $1"; }
error() { echo -e "${RED}ERROR:${NC} $1"; exit 1; }

# Helper functions
get_latest_version() {
    curl --silent "https://api.github.com/repos/$REPO/releases/latest" | 
    grep '"tag_name":' | sed -E 's/.*"v([^"]+)".*/\1/' || echo ""
}

version_gt() {
    test "$(printf '%s\n' "$@" | sort -V | head -n 1)" != "$1"
}

# Version to install (can be overridden by environment variable)
VERSION=${VERSION:-$(get_latest_version)}
if [ -z "$VERSION" ]; then
    VERSION="0.1.0"  # Fallback version
    warn "Could not fetch latest version, defaulting to $VERSION"
fi
BINARY="save"
REPO="t-rhex/save-go"
INSTALL_PATH="$HOME/.local/bin"
INSTALL_TYPE=${INSTALL_TYPE:-"user"}  # Default to user install

# Check if command exists
check_command() {
    if ! command -v "$1" >/dev/null 2>&1; then
        error "$1 is required but not installed. Please install $1 first."
    fi
}

# Check prerequisites
check_prerequisites() {
    info "Checking prerequisites..."
    
    # Check required commands
    check_command "go"
    check_command "git"
    check_command "make"
    
    # Check Go version
    GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
    if ! [[ "$GO_VERSION" =~ ^1\.([2-9][1-9]|[3-9][0-9]) ]]; then
        error "Go version 1.21 or higher is required. Current version: $GO_VERSION"
    fi
    
    success "All prerequisites met!"
}

# Create required directories
setup_directories() {
    info "Setting up directories..."
    if [ "$INSTALL_TYPE" = "user" ]; then
        mkdir -p "$INSTALL_PATH"
        mkdir -p "$HOME/.bash_completion.d"
        mkdir -p "$HOME/.zsh/completion"
    fi
    # System directories are created by make install
}

# Check if save is already installed
check_existing_installation() {
    if [ -f "$INSTALL_PATH/$BINARY" ]; then
        CURRENT_VERSION=$("$INSTALL_PATH/$BINARY" --version 2>/dev/null || echo "unknown")
        
        if [ "$CURRENT_VERSION" = "unknown" ]; then
            warn "Could not determine current version"
            return
        fi

        if [ "$CURRENT_VERSION" = "$VERSION" ]; then
            warn "Version $VERSION is already installed"
            read -p "Do you want to reinstall? [y/N] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                exit 0
            fi
        elif version_gt "$VERSION" "$CURRENT_VERSION"; then
            info "Upgrading from version $CURRENT_VERSION to $VERSION"
            
            # Backup existing configuration
            if [ -f "$HOME/.save_history.json" ]; then
                info "Backing up command history..."
                cp "$HOME/.save_history.json" "$HOME/.save_history.json.backup-$CURRENT_VERSION"
            fi
            
            # Clean old completion files
            info "Cleaning old completion files..."
            if [ "$INSTALL_TYPE" = "user" ]; then
                rm -f "$HOME/.bash_completion.d/$BINARY"
                rm -f "$HOME/.zsh/completion/_$BINARY"
            else
                sudo rm -f "/etc/bash_completion.d/$BINARY"
                sudo rm -f "/usr/local/share/zsh/site-functions/_$BINARY"
            fi
        else
            warn "Attempting to install older version $VERSION (current: $CURRENT_VERSION)"
            read -p "Do you want to downgrade? [y/N] " -n 1 -r
            echo
            if [[ ! $REPLY =~ ^[Yy]$ ]]; then
                exit 0
            fi
        fi
    fi
}

# Add a backup function
backup_config() {
    local timestamp=$(date +%Y%m%d_%H%M%S)
    if [ -f "$HOME/.save_history.json" ]; then
        info "Backing up command history..."
        cp "$HOME/.save_history.json" "$HOME/.save_history.json.backup-$timestamp"
    fi
}

# Add a restore function
restore_on_failure() {
    if [ $? -ne 0 ]; then
        error "Installation failed!"
        if [ -f "$HOME/.save_history.json.backup-$timestamp" ]; then
            warn "Restoring previous configuration..."
            mv "$HOME/.save_history.json.backup-$timestamp" "$HOME/.save_history.json"
        fi
        exit 1
    fi
}

# Install or update save
install_save() {
    info "Installing save version $VERSION..."
    
    # Backup before installation
    local timestamp=$(date +%Y%m%d_%H%M%S)
    backup_config
    
    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    trap 'rm -rf "$TMP_DIR"' EXIT
    
    # Clone repository
    info "Cloning repository..."
    git clone --quiet https://github.com/$REPO.git "$TMP_DIR"
    cd "$TMP_DIR"
    
    # Checkout specific version if not main
    if [ "$VERSION" != "main" ]; then
        git checkout --quiet "v$VERSION"
    fi
    
    # Build and install
    info "Building and installing..."
    if [ "$INSTALL_TYPE" = "system" ]; then
        if [ "$(id -u)" -ne 0 ]; then
            error "System-wide installation requires root privileges. Please run with sudo or use INSTALL_TYPE=user"
        fi
        make install
        INSTALL_PATH="/usr/local/bin"  # Update install path for system install
    else
        make user-install
    fi
    
    # Check installation success and restore on failure
    restore_on_failure
    
    success "Installation complete!"
}

# Setup shell integration
setup_shell() {
    info "Setting up shell integration..."
    
    # Detect shell
    SHELL_TYPE=$(basename "$SHELL")
    
    case "$SHELL_TYPE" in
        "bash")
            if ! grep -q "PATH=\"$INSTALL_PATH:\$PATH\"" "$HOME/.bashrc"; then
                echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.bashrc"
            fi
            if ! grep -q "bash_completion.d/save" "$HOME/.bashrc"; then
                echo '[[ -f ~/.bash_completion.d/save ]] && . ~/.bash_completion.d/save' >> "$HOME/.bashrc"
            fi
            success "Added Bash configuration"
            ;;
        "zsh")
            if ! grep -q "PATH=\"$INSTALL_PATH:\$PATH\"" "$HOME/.zshrc"; then
                echo 'export PATH="$HOME/.local/bin:$PATH"' >> "$HOME/.zshrc"
            fi
            if ! grep -q "fpath=(~/.zsh/completion" "$HOME/.zshrc"; then
                echo 'fpath=(~/.zsh/completion $fpath)' >> "$HOME/.zshrc"
            fi
            success "Added Zsh configuration"
            ;;
        *)
            warn "Unknown shell type: $SHELL_TYPE"
            warn "Please manually add $INSTALL_PATH to your PATH"
            ;;
    esac
}

# Main installation process
main() {
    echo -e "${BLUE}=== Save Command Manager Installer ===${NC}"
    
    check_prerequisites
    setup_directories
    check_existing_installation
    install_save
    setup_shell
    
    echo
    success "Save Command Manager has been installed successfully!"
    echo
    info "To start using save, either:"
    echo "  1. Restart your terminal"
    echo "  2. Or run: source ~/.${SHELL_TYPE}rc"
    echo
    info "Get started with: save --help"
}

# Run main installation
if [ "${1:-}" = "--test" ]; then
    # Test specific functions
    check_prerequisites
    setup_directories
else
    main
fi
