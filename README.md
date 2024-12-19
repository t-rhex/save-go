# save - Command History Manager

**save** is a powerful command-line tool that helps developers track, manage, and reuse their shell commands with additional context and organization features. It's designed to be a more intelligent alternative to basic shell history, offering tagging, searching, and analytics capabilities.

## 🚀 Features

### Core Functionality

- **Command Execution & Storage**: Automatically saves every command with timestamp and exit status
- **Directory Tracking**: Option to save working directory context with commands
- **Tags & Descriptions**: Add metadata to commands for better organization
- **Favorites**: Mark frequently used commands for quick access

### Command Management

- **Listing**: View command history with context
- **Search**: Find commands by content, tags, or description
- **Directory Filtering**: Filter commands by working directory
- **Command Removal**: Delete specific commands from history
- **Export**: Backup or share your command history

### 📊 Analytics

- **Usage Statistics**: Track command success rates and patterns
- **Most Used Commands**: See your most frequently used commands
- **Tag Analytics**: Identify most common command categories
- **Success Rate**: Monitor command reliability

### 🐚 Shell Integration

- **Shell Completion**: Supports both Bash and Zsh completion
- **Directory Awareness**: Maintains working directory context

## Installation

### From Source

Clone the repository:

```bash
git clone https://github.com/t-rhex/save-go.git
cd save
```

Install:

```bash
sudo make install
```

This will install the binary and shell completions.

### Uninstall

```bash
sudo make uninstall
```

## Usage

### Command Examples

```bash
# Save a command with tags
save --tag git,prod 'git push'

# Save a command with description
save --desc 'Deploy to production' './deploy.sh'

# Save a command with current directory
save --dir 'npm start'

# List recent commands
save --list

# Search commands
save --search "git"

# Show statistics
save --stats

# Re-run a command by ID
save --rerun 42

# Mark a command as favorite
save --favorite 42
```

### Command Management

```bash
# List last 10 commands
save --list

# List specific number of commands
save --list 20

# Search commands
save --search "git"

# Filter by directory
save --filter-dir "/path/to/project"

# Remove a command
save --remove 42

# Export command history
save --export backup.json
```

### Metadata Management

```bash
# Mark command as favorite
save --favorite 42

# Add tags to existing command
save --tag deploy,production 42

# Add/update description
save --desc "Important deployment script" 42
```

### Statistics

```bash
# View command statistics
save --stats
```

### Shell Completion

```bash
# Generate shell completion
save --generate-completion bash > ~/.bash_completion.d/save
save --generate-completion zsh > ~/.zsh_completion.d/save
```

### Shell Completion

Shell completion scripts are automatically installed for bash and zsh during installation. You may need to restart your shell for completions to take effect.

## 🤔 Why Use save?

### Context Preservation

- Maintains working directory context
- Supports tags and descriptions
- Helps remember command purpose and category

### Intelligent Organization

- Tag-based categorization
- Searchable descriptions
- Directory-aware command tracking

### Analytics and Insights

- Track command success rates
- Identify frequently used commands
- Analyze command patterns

### Improved Productivity

- Quick command reuse
- Easy access to command history
- Smart searching and filtering

### Team Collaboration

- Shareable command history
- Documented command context
- Exportable configurations
