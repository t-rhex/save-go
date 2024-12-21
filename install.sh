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

# Move these to the top, before any functions
BINARY="save"
REPO="t-rhex/save-go"
INSTALL_PATH="$HOME/.local/bin"
INSTALL_TYPE=${INSTALL_TYPE:-"user"}  # Default to user install

# Helper functions
get_latest_version() {
    local api_response
    api_response=$(curl -sS "https://api.github.com/repos/$REPO/releases/latest")
    
    if [ -z "$api_response" ]; then
        warn "Failed to get response from GitHub API"
        return 1
    fi
    
    # Extract tag_name directly using grep and cut
    LATEST_VERSION=$(echo "$api_response" | grep '"tag_name":' | cut -d'"' -f4)
    
    if [ -z "$LATEST_VERSION" ]; then
        warn "Could not extract version from GitHub API response"
        return 1
    fi
    
    # Remove 'v' prefix if present
    echo "${LATEST_VERSION#v}"
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
    
    # Set install path based on installation type
    if [ "$INSTALL_TYPE" = "system" ]; then
        if [ "$(id -u)" -ne 0 ]; then
            error "System-wide installation requires root privileges. Please run with sudo or use INSTALL_TYPE=user"
        fi
        INSTALL_PATH="/usr/local/bin"
    else
        # Set user install path based on platform
        if [[ "$OSTYPE" == "darwin"* ]]; then
            INSTALL_PATH="$HOME/bin"
        else
            INSTALL_PATH="$HOME/.local/bin"
        fi
    fi
    
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
        make install
    else
        make user-install INSTALL_PATH="$INSTALL_PATH"
    fi
    
    # Check installation success and restore on failure
    restore_on_failure
    
    success "Installation complete!"
}

# Setup shell integration
setup_shell() {
    # Detect current shell and set RC file
    case "$SHELL" in
        */zsh)
            SHELL_RC="$HOME/.zshrc"
            ;;
        */bash)
            SHELL_RC="$HOME/.bashrc"
            # For macOS, also check .bash_profile
            if [[ "$OSTYPE" == "darwin"* ]] && [ -f "$HOME/.bash_profile" ]; then
                SHELL_RC="$HOME/.bash_profile"
            fi
            ;;
        *)
            warn "Unsupported shell: $SHELL. Please manually add $INSTALL_PATH to your PATH"
            return
            ;;
    esac

    info "Updating $SHELL_RC..."
    
    # Use the INSTALL_PATH that was set during installation
    if [ "$INSTALL_TYPE" = "system" ]; then
        # System installations don't need PATH modification as /usr/local/bin is usually in PATH
        info "System installation: /usr/local/bin is typically already in PATH"
        return
    fi
    
    # Create bin directory if it doesn't exist
    mkdir -p "$INSTALL_PATH"
    
    # Add PATH to shell RC if not already present
    if ! grep -q "export PATH=\"$INSTALL_PATH:\$PATH\"" "$SHELL_RC"; then
        echo -e "\n# Added by save installer" >> "$SHELL_RC"
        echo "export PATH=\"$INSTALL_PATH:\$PATH\"" >> "$SHELL_RC"
        success "Updated $SHELL_RC with PATH"
    else
        info "PATH already configured in $SHELL_RC"
    fi
    
    # Source the RC file
    info "To use 'save' command immediately, run:"
    echo "    source $SHELL_RC"
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
    if [ "$SHELL_TYPE" = "bash" ]; then
        echo "  2. Or run: source ~/.bashrc"
    elif [ "$SHELL_TYPE" = "zsh" ]; then
        echo "  2. Or run: source ~/.zshrc"
    fi
    
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
