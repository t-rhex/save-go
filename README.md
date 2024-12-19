# save - Command History Manager

[![Go Report Card](https://goreportcard.com/badge/github.com/t-rhex/save-go)](https://goreportcard.com/report/github.com/t-rhex/save-go)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Version](https://img.shields.io/badge/version-0.1.0-blue.svg)](https://github.com/t-rhex/save-go/releases)

**save** is a powerful command-line tool that helps developers track, manage, and reuse their shell commands with additional context and organization features. It's designed to be a more intelligent alternative to basic shell history, offering tagging, searching, and analytics capabilities.

## 🚀 Quick Install

### Prerequisites
- Go 1.21 or higher
- Git
- Make (for Unix/Linux/macOS)

### One-Line Installation

#### Linux/macOS
```bash
curl -sSL https://raw.githubusercontent.com/t-rhex/save-go/main/install.sh | bash
```

#### Windows (PowerShell)
```powershell
# Run as administrator
Invoke-Expression (New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/t-rhex/save-go/main/install.ps1')
```

### Manual Installation

1. Clone and Build:
```bash
git clone https://github.com/t-rhex/save-go.git
cd save-go
```

2. Choose Installation Method:

#### Unix/Linux/macOS:
**User Installation** (Recommended, no sudo required):
```bash
make user-install
```

**System-wide Installation** (Requires sudo):
```bash
make install
```

#### Windows:
```powershell
# Build
go build -o save.exe

# Copy to a directory in your PATH (e.g., %USERPROFILE%\AppData\Local\save)
$installPath = "$env:USERPROFILE\AppData\Local\save"
New-Item -ItemType Directory -Force -Path $installPath
Copy-Item save.exe -Destination "$installPath\save.exe"

# Add to PATH if not already there
$userPath = [Environment]::GetEnvironmentVariable("Path", "User")
if ($userPath -notlike "*$installPath*") {
    [Environment]::SetEnvironmentVariable("Path", "$userPath;$installPath", "User")
}
```

3. Add to Shell (if using user installation):

#### Unix/Linux/macOS:
```bash
# For Bash
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.bashrc
echo '[[ -f ~/.bash_completion.d/save ]] && . ~/.bash_completion.d/save' >> ~/.bashrc

# For Zsh
echo 'export PATH="$HOME/.local/bin:$PATH"' >> ~/.zshrc
echo 'fpath=(~/.zsh/completion $fpath)' >> ~/.zshrc
```

#### Windows (PowerShell):
```powershell
# Add to PowerShell profile
if (!(Test-Path $PROFILE)) { New-Item -Type File -Force $PROFILE }
Add-Content $PROFILE "`$env:Path = [Environment]::GetEnvironmentVariable('Path', 'User')"

# Setup completion
New-Item -ItemType Directory -Force -Path "$env:USERPROFILE\Documents\WindowsPowerShell\Completions"
save --generate-completion powershell > "$env:USERPROFILE\Documents\WindowsPowerShell\Completions\save.ps1"
```

4. Reload Shell:

#### Unix/Linux/macOS:
```bash
source ~/.bashrc  # or source ~/.zshrc for Zsh
```

#### Windows:
```powershell
# Reload PATH
$env:Path = [Environment]::GetEnvironmentVariable("Path", "User")
```

## 🚀 Quick Start

```bash
# Save and run a command
save 'echo "Hello World"'

# Save with tags and description
save --tag docker,prod --desc 'Start services' 'docker-compose up'

# List recent commands
save --list

# Search commands
save --search "docker"
```

## 📚 Features

### Core Features
- ✨ Command execution and storage with metadata
- 📁 Directory context tracking
- 🏷️ Tagging system
- 📝 Command descriptions
- ⭐ Favorite commands
- 🔄 Command rerunning
- 📊 Usage statistics

### Advanced Features
- 🔗 Command chains with dependencies
- ⚡ Parallel execution support
- 🎯 Conditional execution
- ⏰ Time-based execution
- 🔄 Undo support
- 📈 Analytics and insights

## 💻 Usage Examples

### Basic Command Management
```bash
# Save with current directory
save --dir 'npm start'

# Add tags to existing command
save --add-tags 42 git,prod

# Mark as favorite
save --favorite 42

# Edit command interactively
save --interactive-edit 42
```

### Command Chains
```bash
# Create deployment chain
save --create-chain 'deploy' 'Deploy to prod' steps.json deps.json

# Run chain
save --run-chain 1
```

### Search and Analytics
```bash
# Search by tag
save --filter-tag docker

# View statistics
save --stats

# Export history
save --export history.json
```

## ⚙️ Configuration

### Default Paths
- Config: `~/.config/save/config.json`
- History: `~/.save_history.json`
- Completions:
  - Bash: `~/.bash_completion.d/save`
  - Zsh: `~/.zsh/completion/_save`

### Environment Variables
```bash
SAVE_CONFIG_PATH   # Custom config file location
SAVE_HISTORY_PATH  # Custom history file location
SAVE_NO_COLOR      # Disable color output
```

## 🔄 Updates

### Unix/Linux/macOS:
```bash
# Update to latest version
cd save-go
git pull
make update
```

### Windows:
```powershell
# Update to latest version
cd save-go
git pull
go build -o save.exe
Copy-Item save.exe -Destination "$env:USERPROFILE\AppData\Local\save\save.exe" -Force
```

## 🐛 Troubleshooting

### Common Issues

1. Command not found
```bash
# Add to PATH
export PATH="$HOME/.local/bin:$PATH"
```

2. No shell completion
```bash
# Reload shell completion
source ~/.bashrc  # or source ~/.zshrc
```

3. Permission denied
```bash
# Fix permissions
chmod +x $HOME/.local/bin/save
```

## 🤝 Contributing

1. Fork the repository
2. Create feature branch (`git checkout -b feature/amazing-feature`)
3. Commit changes (`git commit -m 'Add amazing feature'`)
4. Push branch (`git push origin feature/amazing-feature`)
5. Open Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by shell history management tools
- Built with Go's robust standard library
- Community feedback and contributions

## 📬 Support

- 🐛 [Report Bug](https://github.com/t-rhex/save-go/issues)
- 💡 [Request Feature](https://github.com/t-rhex/save-go/issues)
- 📧 [Email Support](mailto:support@example.com)
