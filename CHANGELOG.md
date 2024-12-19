# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Development build support with separate config path
- Improved Makefile with help target and better documentation
- Windows support with PowerShell installation script

## [0.1.0] - 2024-12-19

### Added
- Core Command Management
  - Basic command execution and storage
  - Command history with unique IDs
  - Exit code tracking
  - Command validation
  - Working directory tracking
  - Command rerunning capability

- Metadata and Organization
  - Tag support with add/remove functionality
  - Command descriptions
  - Favorite command marking
  - Run count and success rate tracking
  - Interactive command editing
  - Undo support for command edits

- Advanced Features
  - Command chains with dependencies
  - Parallel execution support
  - Conditional command execution
  - Chain success/failure handling
  - Command statistics and analytics

- Search and Filter
  - Text-based search across commands
  - Tag-based filtering
  - Directory-based filtering
  - Most used commands tracking
  - Tag usage statistics

- Shell Integration
  - Shell completion for bash and zsh
  - ANSI color support in CLI
  - Comprehensive help documentation
  - User-friendly command-line interface

- Data Management
  - JSON-based storage
  - Import/export capabilities
  - Edit history tracking
  - Automatic stats updates

### Fixed
- Shell command execution issues
- Makefile sudo requirements
- Workflow configuration

### Changed
- Improved documentation
- Enhanced installation process
- Added single installation script

