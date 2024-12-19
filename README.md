# save - Command History Manager

[![Go Report Card](https://goreportcard.com/badge/github.com/t-rhex/save-go)](https://goreportcard.com/report/github.com/t-rhex/save-go)

**save** is a powerful command-line tool that helps developers track, manage, and reuse their shell commands with additional context and organization features. It's designed to be a more intelligent alternative to basic shell history, offering tagging, searching, and analytics capabilities.

## 🚀 Features

### Core Functionality

- **Command Execution & Storage**: Automatically saves every command with timestamp and exit status
- **Directory Tracking**: Option to save working directory context with commands
- **Tags & Descriptions**: Add metadata to commands for better organization
- **Favorites**: Mark frequently used commands for quick access
- **Interactive Editing**: Edit commands and metadata through an interactive interface
- **Command Chains**: Create and execute sequences of dependent commands
- **Undo Support**: Revert changes to commands and metadata

### Advanced Features

#### Command Chains

- Create complex command sequences with dependencies
- Support for parallel execution
- Conditional execution based on success/failure
- Time-based execution windows
- Environmental condition checks

#### Analytics & Insights

- **Usage Statistics**: Track command success rates and patterns
- **Most Used Commands**: See your most frequently used commands
- **Tag Analytics**: Identify most common command categories
- **Success Rate**: Monitor command reliability
- **Chain Performance**: Track chain execution success rates
- **Time-based Analytics**: Analyze command usage patterns over time

## 📦 Installation

### Homebrew (macOS and Linux) - Coming soon

```bash
brew install t-rhex/tap/save
```

### From Source

```bash
git clone https://github.com/t-rhex/save-go.git
cd save-go
make user-install
```

### Update

```bash
git pull
make update
```

## 💻 Usage

### Basic Command Management

```bash
# Save and execute a command
save 'echo "Hello World"'

# Save with tags and description
save --tag 'docker,prod' --desc 'Start production container' 'docker-compose up'

# Save with current directory context
save --dir 'npm start'
```

### Advanced Features

#### Command Chains

```bash
# Create a deployment chain
save --create-chain 'deploy' 'Deployment process' steps.json deps.json

# Run a chain
save --run-chain 1

# List all chains
save --list-chains
```

#### Interactive Editing

```bash
# Edit command interactively
save --interactive-edit 42

# Add/remove tags
save --add-tags 42 'git,prod'
save --remove-tags 42 'prod'

# Undo last edit
save --undo 42
```

#### Search and Filter

```bash
# Search commands
save --search "git"

# Filter by tag
save --filter-tag "docker"

# Filter by directory
save --filter-dir "/path/to/project"
```

#### Analytics

```bash
# View general statistics
save --stats

# View tag usage
save --list-tags

# Export command history
save --export history.json
```

## 🔧 Configuration

### Shell Integration

Shell completion scripts are automatically installed for bash and zsh.

#### Manual Shell Completion Setup

```bash
# Bash
echo 'source ~/.bash_completion.d/save' >> ~/.bashrc

# Zsh
echo 'fpath=(~/.zsh/completion $fpath)' >> ~/.zshrc
```

## 🤝 Contributing

1. Fork the repository
2. Create your feature branch
3. Commit your changes
4. Push to the branch
5. Create a Pull Request

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Inspired by shell history management tools
- Built with Go's robust standard library
- Community feedback and contributions
