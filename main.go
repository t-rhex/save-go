// Copyright (c) 2024 Andrew Adhikari
// This file is licensed under the MIT License.
// See LICENSE in the project root for license information.

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"strconv"
)

type Command struct {
	Raw         string    `json:"command"`
	Timestamp   time.Time `json:"timestamp"`
	Dir         string    `json:"working_dir,omitempty"`
	ExitCode    int      `json:"exit_code"`
	ID          int      `json:"id"`
	Tags        []string  `json:"tags,omitempty"`
	Description string    `json:"description,omitempty"`
	IsFavorite  bool     `json:"is_favorite"`
	RunCount    int      `json:"run_count"`
	SuccessCount int     `json:"success_count"`
}

type Statistics struct {
	TotalRuns      int     `json:"total_runs"`
	SuccessCount   int     `json:"success_count"`
	SuccessRate    float64 `json:"success_rate"`
	FavoriteCount  int     `json:"favorite_count"`
	MostUsedTags   []string `json:"most_used_tags"`
	CommonCommands []string `json:"common_commands"`
}

type CommandStore struct {
	filepath string
	commands []Command
	lastID   int
	stats    Statistics
}

func NewCommandStore() (*CommandStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	return &CommandStore{
		filepath: filepath.Join(homeDir, ".save_history.json"),
		commands: []Command{},
	}, nil
}

func (cs *CommandStore) save() error {
    data, err := json.MarshalIndent(cs.commands, "", "    ")
    if err != nil {
        return err
    }
    return os.WriteFile(cs.filepath, data, 0644)
}

func (cs *CommandStore) load() error {
	data, err := os.ReadFile(cs.filepath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	if err := json.Unmarshal(data, &cs.commands); err != nil {
		return err
	}

	for _, cmd := range cs.commands {
		if cmd.ID > cs.lastID {
			cs.lastID = cmd.ID
		}
	}

	cs.updateStats()
	return nil
}

func (cs *CommandStore) RemoveCommand(id int) error {
    for i, cmd := range cs.commands {
        if cmd.ID == id {
            // Remove the command by slicing
            cs.commands = append(cs.commands[:i], cs.commands[i+1:]...)
            cs.updateStats()
            return cs.save()
        }
    }
    return fmt.Errorf("command with ID %d not found", id)
}

func (cs *CommandStore) updateStats() {
	stats := Statistics{}
	tagCount := make(map[string]int)
	cmdCount := make(map[string]int)

	for _, cmd := range cs.commands {
		stats.TotalRuns += cmd.RunCount
		stats.SuccessCount += cmd.SuccessCount
		if cmd.IsFavorite {
			stats.FavoriteCount++
		}

		for _, tag := range cmd.Tags {
			tagCount[tag]++
		}
		cmdCount[cmd.Raw]++
	}

	// Calculate success rate
	if stats.TotalRuns > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.TotalRuns) * 100
	}

	// Get most used tags
	stats.MostUsedTags = getTopKeys(tagCount, 5)
	stats.CommonCommands = getTopKeys(cmdCount, 5)

	cs.stats = stats
}

func getTopKeys(m map[string]int, n int) []string {
	// Convert map to slice of pairs
	type pair struct {
		key   string
		value int
	}
	var pairs []pair
	for k, v := range m {
		pairs = append(pairs, pair{k, v})
	}

	// Sort by value (descending)
	for i := 0; i < len(pairs)-1; i++ {
		for j := i + 1; j < len(pairs); j++ {
			if pairs[i].value < pairs[j].value {
				pairs[i], pairs[j] = pairs[j], pairs[i]
			}
		}
	}

	// Get top N
	result := make([]string, 0, n)
	for i := 0; i < n && i < len(pairs); i++ {
		result = append(result, pairs[i].key)
	}
	return result
}

func (cs *CommandStore) Execute(cmdString string, saveDir bool, tags []string, description string) error {
	var dir string
	if saveDir {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			dir = "unknown"
		}
	}

	cmd := exec.Command("sh", "-c", cmdString)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		}
	}

	cs.lastID++
	command := Command{
		Raw:         cmdString,
		Timestamp:   time.Now(),
		Dir:         dir,
		ExitCode:    exitCode,
		ID:          cs.lastID,
		Tags:        tags,
		Description: description,
		RunCount:    1,
		SuccessCount: func() int {
			if exitCode == 0 {
				return 1
			}
			return 0
		}(),
	}

	cs.commands = append(cs.commands, command)
	cs.updateStats()
	return cs.save()
}

func (cs *CommandStore) ImportCommands(filename string) error {
    // Read the import file
    data, err := os.ReadFile(filename)
    if err != nil {
        return fmt.Errorf("failed to read import file: %w", err)
    }

    // Parse the imported commands
    var importedCommands []Command
    if err := json.Unmarshal(data, &importedCommands); err != nil {
        return fmt.Errorf("failed to parse import file: %w", err)
    }

    // Add imported commands to existing commands
    for _, cmd := range importedCommands {
        // Update ID to avoid conflicts
        cs.lastID++
        cmd.ID = cs.lastID
        cs.commands = append(cs.commands, cmd)
    }

    cs.updateStats()
    return cs.save()
}

func (cs *CommandStore) SetFavorite(id int, favorite bool) error {
	for i := range cs.commands {
		if cs.commands[i].ID == id {
			cs.commands[i].IsFavorite = favorite
			cs.updateStats()
			return cs.save()
		}
	}
	return fmt.Errorf("command with ID %d not found", id)
}

func (cs *CommandStore) AddTags(id int, tags []string) error {
	for i := range cs.commands {
		if cs.commands[i].ID == id {
			// Add new tags without duplicates
			tagMap := make(map[string]bool)
			for _, tag := range cs.commands[i].Tags {
				tagMap[tag] = true
			}
			for _, tag := range tags {
				if !tagMap[tag] {
					cs.commands[i].Tags = append(cs.commands[i].Tags, tag)
				}
			}
			cs.updateStats()
			return cs.save()
		}
	}
	return fmt.Errorf("command with ID %d not found", id)
}

func (cs *CommandStore) SetDescription(id int, description string) error {
	for i := range cs.commands {
		if cs.commands[i].ID == id {
			cs.commands[i].Description = description
			return cs.save()
		}
	}
	return fmt.Errorf("command with ID %d not found", id)
}

func (cs *CommandStore) GetStats() Statistics {
	return cs.stats
}

// Generate shell completion scripts
func generateShellCompletion(shell string) string {
	switch shell {
	case "bash":
		return `
_save_completion() {
    local cur prev opts
    COMPREPLY=()
    cur="${COMP_WORDS[COMP_CWORD]}"
    prev="${COMP_WORDS[COMP_CWORD-1]}"
    opts="--dir --list --search --filter-dir --export --rerun --tag --desc --favorite --stats"

    case "${prev}" in
        --rerun)
            # Complete with command IDs
            COMPREPLY=( $(save --list | grep "^#" | cut -d" " -f1 | cut -c2- | grep "^${cur}") )
            return 0
            ;;
        --tag)
            # Complete with existing tags
            COMPREPLY=( $(save --list-tags | grep "^${cur}") )
            return 0
            ;;
        *)
            COMPREPLY=( $(compgen -W "${opts}" -- ${cur}) )
            return 0
            ;;
    esac
}

complete -F _save_completion save`

	case "zsh":
		return `
#compdef save

_save() {
    local -a opts
    opts=(
        '--dir[Save with directory]'
        '--list[List commands]'
        '--search[Search commands]'
        '--filter-dir[Filter by directory]'
        '--export[Export history]'
        '--rerun[Re-run command]'
        '--tag[Add tags]'
        '--desc[Add description]'
        '--favorite[Mark as favorite]'
        '--stats[Show statistics]'
    )

    _arguments -C \
        "${opts[@]}"
}

_save`

	default:
		return ""
	}
}

func containsTag(tags []string, query string) bool {
    query = strings.ToLower(query)
    for _, tag := range tags {
        if strings.Contains(strings.ToLower(tag), query) {
            return true
        }
    }
    return false
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	store, err := NewCommandStore()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize: %v\n", err)
		os.Exit(1)
	}

	if err := store.load(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to load history: %v\n", err)
		os.Exit(1)
	}

	switch os.Args[1] {
	case "--generate-completion":
		if len(os.Args) != 3 {
			fmt.Println("Usage: save --generate-completion <shell>")
			os.Exit(1)
		}
		fmt.Println(generateShellCompletion(os.Args[2]))

	case "--stats":
		stats := store.GetStats()
		fmt.Printf("Command Statistics:\n")
		fmt.Printf("Total Runs: %d\n", stats.TotalRuns)
		fmt.Printf("Success Rate: %.2f%%\n", stats.SuccessRate)
		fmt.Printf("Favorite Commands: %d\n", stats.FavoriteCount)
		fmt.Printf("\nMost Used Tags:\n")
		for _, tag := range stats.MostUsedTags {
			fmt.Printf("  - %s\n", tag)
		}
		fmt.Printf("\nMost Common Commands:\n")
		for _, cmd := range stats.CommonCommands {
			fmt.Printf("  - %s\n", cmd)
		}

	case "--favorite":
		if len(os.Args) < 3 {
			fmt.Println("Error: --favorite requires a command ID")
			os.Exit(1)
		}
		id, _ := strconv.Atoi(os.Args[2])
		if err := store.SetFavorite(id, true); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Marked command #%d as favorite\n", id)

	case "--help":
		printUsage()
		os.Exit(0)
	
	case "--remove":
		if len(os.Args) < 3 {
			fmt.Println("Error: --remove requires a command ID")
			os.Exit(1)
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid command ID\n")
			os.Exit(1)
		}
		if err := store.RemoveCommand(id); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed command #%d\n", id)
	
	case "--list":
		// Default to showing last 10 commands if n is not specified
		n := 10
		if len(os.Args) > 2 {
			if val, err := strconv.Atoi(os.Args[2]); err == nil {
				n = val
			}
		}
		// Show last n commands in reverse order (newest first)
		start := len(store.commands) - n
		if start < 0 {
			start = 0
		}
		for i := len(store.commands) - 1; i >= start; i-- {
			cmd := store.commands[i]
			fmt.Printf("#%d [%s] %s\n", cmd.ID, cmd.Timestamp.Format("2006-01-02 15:04:05"), cmd.Raw)
			if cmd.Description != "" {
				fmt.Printf("    Description: %s\n", cmd.Description)
			}
			if len(cmd.Tags) > 0 {
				fmt.Printf("    Tags: %s\n", strings.Join(cmd.Tags, ", "))
			}
			if cmd.Dir != "" {
				fmt.Printf("    Directory: %s\n", cmd.Dir)
			}
			fmt.Println()
		}
	
	case "--search":
		if len(os.Args) < 3 {
			fmt.Println("Error: --search requires a query")
			os.Exit(1)
		}
		query := strings.ToLower(os.Args[2])
		for _, cmd := range store.commands {
			if strings.Contains(strings.ToLower(cmd.Raw), query) ||
			   strings.Contains(strings.ToLower(cmd.Description), query) ||
			   containsTag(cmd.Tags, query) {
				fmt.Printf("#%d [%s] %s\n", cmd.ID, cmd.Timestamp.Format("2006-01-02 15:04:05"), cmd.Raw)
			}
		}
	
	case "--filter-dir":
		if len(os.Args) < 3 {
			fmt.Println("Error: --filter-dir requires a directory path")
			os.Exit(1)
		}
		filterDir := os.Args[2]
		for _, cmd := range store.commands {
			if cmd.Dir == filterDir {
				fmt.Printf("#%d [%s] %s\n", cmd.ID, cmd.Timestamp.Format("2006-01-02 15:04:05"), cmd.Raw)
			}
		}

	case "--import":
		if len(os.Args) < 3 {
			fmt.Println("Error: --import requires a filename")
			os.Exit(1)
		}
		importFile := os.Args[2]
		if err := store.ImportCommands(importFile); err != nil {
			fmt.Fprintf(os.Stderr, "Error importing commands: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully imported commands from %s\n", importFile)
	
	case "--export":
		if len(os.Args) < 3 {
			fmt.Println("Error: --export requires a filename")
			os.Exit(1)
		}
		exportFile := os.Args[2]
		data, err := json.MarshalIndent(store.commands, "", "    ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error exporting commands: %v\n", err)
			os.Exit(1)
		}
		if err := os.WriteFile(exportFile, data, 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing export file: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Exported %d commands to %s\n", len(store.commands), exportFile)
	
	case "--rerun":
    if len(os.Args) < 3 {
        fmt.Println("Error: --rerun requires a command ID")
        os.Exit(1)
    }
    id, err := strconv.Atoi(os.Args[2])
    if err != nil {
        fmt.Fprintf(os.Stderr, "Error: invalid command ID\n")
        os.Exit(1)
    }
    // Find the command to rerun
    var cmdToRerun *Command
    for i := range store.commands {
        if store.commands[i].ID == id {
            cmdToRerun = &store.commands[i]
            break
        }
    }
    if cmdToRerun == nil {
        fmt.Fprintf(os.Stderr, "Error: command with ID %d not found\n", id)
        os.Exit(1)
    }
    // Rerun the command
    if err := store.Execute(cmdToRerun.Raw, cmdToRerun.Dir != "", cmdToRerun.Tags, cmdToRerun.Description); err != nil {
        fmt.Fprintf(os.Stderr, "Error re-running command: %v\n", err)
        os.Exit(1)
    }
	
	default:
		var tags []string
		var description string
		var saveDir bool
		cmdArgs := os.Args[1:]

		// Parse flags
		for i := 0; i < len(cmdArgs); i++ {
			switch cmdArgs[i] {
			case "--tag":
				if i+1 < len(cmdArgs) {
					tags = strings.Split(cmdArgs[i+1], ",")
					cmdArgs = append(cmdArgs[:i], cmdArgs[i+2:]...)
					i--
				}
			case "--desc":
				if i+1 < len(cmdArgs) {
					description = cmdArgs[i+1]
					cmdArgs = append(cmdArgs[:i], cmdArgs[i+2:]...)
					i--
				}
			case "--dir":
				saveDir = true
				cmdArgs = append(cmdArgs[:i], cmdArgs[i+1:]...)
				i--
			}
		}

		cmdString := strings.Join(cmdArgs, " ")
		if err := store.Execute(cmdString, saveDir, tags, description); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func printUsage() {
	fmt.Println("Usage:")
	fmt.Println("  save [flags] <command>           - Execute and save a command")
	fmt.Println("\nFlags:")
	fmt.Println("  --dir                           - Save with current directory")
	fmt.Println("  --tag <tags>                    - Add comma-separated tags")
	fmt.Println("  --desc <description>            - Add description")
	fmt.Println("\nCommands:")
	fmt.Println("  --list [n]                      - List last n commands")
	fmt.Println("  --remove <id>                   - Remove command by ID")
	fmt.Println("  --search <query>                - Search commands")
	fmt.Println("  --filter-dir <dir>              - Filter by directory")
	fmt.Println("  --import <filename>             - Import commands from file")
	fmt.Println("  --export <filename>             - Export history")
	fmt.Println("  --rerun <id>                    - Re-run command by ID")
	fmt.Println("  --favorite <id>                 - Mark command as favorite")
	fmt.Println("  --stats                         - Show command statistics")
	fmt.Println("  --generate-completion <shell>   - Generate shell completion")
	fmt.Println("\nExamples:")
	fmt.Println("  save --dir --tag git,prod 'git push'")
	fmt.Println("  save --desc 'Deploy to prod' './deploy.sh'")
	fmt.Println("  save --favorite 42")
	fmt.Println("  save --stats")
	fmt.Println("  save --import backup.json")
}