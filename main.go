// Copyright (c) 2024 Andrew Adhikari
// This file is licensed under the MIT License.
// See LICENSE in the project root for license information.

package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
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

// Advanced chain types
type ChainDependency struct {
    ChainID     int    `json:"chain_id"`
    DependsOn   []int  `json:"depends_on"` // IDs of chains this chain depends on
    WaitPolicy  string `json:"wait_policy"` // "all" or "any"
}

type CommandCondition struct {
    Type      string `json:"type"`      // "exit_code", "output_contains", "env_var"
    Value     string `json:"value"`     // The value to check against
    Operation string `json:"operation"` // "equals", "not_equals", "contains", "greater_than", etc.
}

type ChainStep struct {
    CommandID   int               `json:"command_id"`
    Conditions  []CommandCondition `json:"conditions,omitempty"`
    ParallelWith []int            `json:"parallel_with,omitempty"` // Command IDs to run in parallel
    OnSuccess   []int            `json:"on_success,omitempty"`    // Command IDs to run if successful
    OnFailure   []int            `json:"on_failure,omitempty"`    // Command IDs to run if failed
}

type CommandChain struct {
    ID          int               `json:"id"`
    Name        string           `json:"name"`
    Description string           `json:"description,omitempty"`
    Steps       []ChainStep      `json:"steps"`
    Dependencies []ChainDependency `json:"dependencies,omitempty"`
    CreatedAt   time.Time        `json:"created_at"`
    LastRun     time.Time        `json:"last_run,omitempty"`
    SuccessRate float64          `json:"success_rate"`
    RunCount    int              `json:"run_count"`
}

type CommandStore struct {
    filepath    string
    commands    []Command
    chains      []CommandChain
    lastID      int
    lastChainID int
    stats       Statistics
    editHistory []EditHistory
}

type EditHistory struct {
    CommandID   int                    `json:"command_id"`
    Timestamp   time.Time              `json:"timestamp"`
    PrevState   Command                `json:"previous_state"`
    EditType    string                 `json:"edit_type"`
}

type ExecutionContext struct {
    LastExitCode int
    LastOutput   string
    ExecError    error
}

// Add method to validate commands
func validateCommand(cmd string) error {
    // Test if command is empty
    if strings.TrimSpace(cmd) == "" {
        return fmt.Errorf("command cannot be empty")
    }

    // Test if command can be parsed by shell
    testCmd := exec.Command("sh", "-n", "-c", cmd)
    if err := testCmd.Run(); err != nil {
        return fmt.Errorf("invalid shell syntax: %v", err)
    }

    return nil
}


func NewCommandStore() (*CommandStore, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("failed to get home directory: %w", err)
	}

	// Use ConfigPath if set during build, otherwise use default
	configPath := ConfigPath
	if configPath == "" {
		configPath = filepath.Join(homeDir, ".save_history.json")
	} else {
		// For development builds, always append the history filename
		configPath = filepath.Join(configPath, "history.json")
	}

	return &CommandStore{
		filepath: configPath,
		commands: []Command{},
	}, nil
}

func (cs *CommandStore) save() error {
    // Create a structure to hold both commands and chains
    type SaveData struct {
        Commands []Command      `json:"commands"`
        Chains   []CommandChain `json:"chains"`
    }
    
    data := SaveData{
        Commands: cs.commands,
        Chains:   cs.chains,
    }
    
    jsonData, err := json.MarshalIndent(data, "", "    ")
    if err != nil {
        return err
    }
    return os.WriteFile(cs.filepath, jsonData, 0644)
}

// Add method for tag manipulation
func (cs *CommandStore) ManipulateTags(id int, addTags, removeTags []string) error {
    for i := range cs.commands {
        if cs.commands[i].ID == id {
            // Create a map of existing tags for efficient lookup
            tagMap := make(map[string]bool)
            for _, tag := range cs.commands[i].Tags {
                tagMap[tag] = true
            }

            // Add new tags
            for _, tag := range addTags {
                tagMap[tag] = true
            }

            // Remove specified tags
            for _, tag := range removeTags {
                delete(tagMap, tag)
            }

            // Convert back to slice
            newTags := make([]string, 0, len(tagMap))
            for tag := range tagMap {
                newTags = append(newTags, tag)
            }
            sort.Strings(newTags)

            // Save previous state for undo
            prevState := cs.commands[i]
            cs.editHistory = append(cs.editHistory, EditHistory{
                CommandID:  id,
                Timestamp: time.Now(),
                PrevState: prevState,
                EditType:  "tag_manipulation",
            })

            cs.commands[i].Tags = newTags
            return cs.save()
        }
    }
    return fmt.Errorf("command with ID %d not found", id)
}

// Add method for interactive editing
func (cs *CommandStore) InteractiveEdit(id int) error {
    var cmd *Command
    for i := range cs.commands {
        if cs.commands[i].ID == id {
            cmd = &cs.commands[i]
            break
        }
    }
    if cmd == nil {
        return fmt.Errorf("command with ID %d not found", id)
    }

    // Store original state for undo
    prevState := *cmd

    // Create bufio reader for user input
    reader := bufio.NewReader(os.Stdin)

    fmt.Printf("\nInteractive Command Editor\n")
    fmt.Printf("Current command: %s\n", cmd.Raw)
    fmt.Print("Enter new command (or press Enter to keep current): ")
    if input, err := reader.ReadString('\n'); err == nil {
        input = strings.TrimSpace(input)
        if input != "" {
            if err := validateCommand(input); err != nil {
                return fmt.Errorf("invalid command: %v", err)
            }
            cmd.Raw = input
        }
    }

    fmt.Printf("Current description: %s\n", cmd.Description)
    fmt.Print("Enter new description (or press Enter to keep current): ")
    if input, err := reader.ReadString('\n'); err == nil {
        input = strings.TrimSpace(input)
        if input != "" {
            cmd.Description = input
        }
    }

    fmt.Printf("Current tags: %s\n", strings.Join(cmd.Tags, ", "))
    fmt.Print("Enter tags to add (comma-separated, or press Enter to skip): ")
    if input, err := reader.ReadString('\n'); err == nil {
        input = strings.TrimSpace(input)
        if input != "" {
            addTags := strings.Split(input, ",")
            for i := range addTags {
                addTags[i] = strings.TrimSpace(addTags[i])
            }
            if err := cs.ManipulateTags(id, addTags, nil); err != nil {
                return err
            }
        }
    }

    fmt.Print("Enter tags to remove (comma-separated, or press Enter to skip): ")
    if input, err := reader.ReadString('\n'); err == nil {
        input = strings.TrimSpace(input)
        if input != "" {
            removeTags := strings.Split(input, ",")
            for i := range removeTags {
                removeTags[i] = strings.TrimSpace(removeTags[i])
            }
            if err := cs.ManipulateTags(id, nil, removeTags); err != nil {
                return err
            }
        }
    }

    // Save edit history
    cs.editHistory = append(cs.editHistory, EditHistory{
        CommandID:  id,
        Timestamp: time.Now(),
        PrevState: prevState,
        EditType:  "interactive_edit",
    })

    return cs.save()
}

// Add method to undo last edit
func (cs *CommandStore) UndoLastEdit(id int) error {
    // Find the last edit for this command
    var lastEdit *EditHistory
    var lastEditIndex int
    for i := len(cs.editHistory) - 1; i >= 0; i-- {
        if cs.editHistory[i].CommandID == id {
            lastEdit = &cs.editHistory[i]
            lastEditIndex = i
            break
        }
    }

    if lastEdit == nil {
        return fmt.Errorf("no edit history found for command %d", id)
    }

    // Find and update the command
    for i := range cs.commands {
        if cs.commands[i].ID == id {
            cs.commands[i] = lastEdit.PrevState
            // Remove this edit from history
            cs.editHistory = append(cs.editHistory[:lastEditIndex], cs.editHistory[lastEditIndex+1:]...)
            return cs.save()
        }
    }

    return fmt.Errorf("command with ID %d not found", id)
}


func (cs *CommandStore) load() error {
    // Create directory if it doesn't exist
    dir := filepath.Dir(cs.filepath)
    if err := os.MkdirAll(dir, 0755); err != nil {
        return fmt.Errorf("failed to create config directory: %w", err)
    }

    data, err := os.ReadFile(cs.filepath)
    if err != nil {
        if os.IsNotExist(err) {
            // If file doesn't exist, create it with empty data
            return cs.save()
        }
        return err
    }

    // Create a structure to hold both commands and chains
    type SaveData struct {
        Commands []Command      `json:"commands"`
        Chains   []CommandChain `json:"chains"`
    }

    var saveData SaveData
    if err := json.Unmarshal(data, &saveData); err != nil {
        // Try loading legacy format (just commands)
        if err := json.Unmarshal(data, &cs.commands); err != nil {
            return err
        }
    } else {
        cs.commands = saveData.Commands
        cs.chains = saveData.Chains
    }

    // Update lastID and lastChainID
    for _, cmd := range cs.commands {
        if cmd.ID > cs.lastID {
            cs.lastID = cmd.ID
        }
    }
    for _, chain := range cs.chains {
        if chain.ID > cs.lastChainID {
            cs.lastChainID = chain.ID
        }
    }

    cs.updateStats()
    return nil
}

func (cs *CommandStore) RemoveCommands(ids []int) error {
    // Create a map for quick lookup of IDs to remove
    toRemove := make(map[int]bool)
    for _, id := range ids {
        toRemove[id] = true
    }
    
    // Create a new slice with commands that should be kept
    newCommands := make([]Command, 0, len(cs.commands))
    for _, cmd := range cs.commands {
        if !toRemove[cmd.ID] {
            newCommands = append(newCommands, cmd)
        }
    }
    
    // If no commands were removed, return an error
    if len(newCommands) == len(cs.commands) {
        return fmt.Errorf("no commands found with the specified IDs")
    }
    
    cs.commands = newCommands
    cs.updateStats()
    return cs.save()
}

func (cs *CommandStore) RemoveCommand(id int) error {
    return cs.RemoveCommands([]int{id})
}

func (cs *CommandStore) updateCommandStats(id int, exitCode int) error {
    for i := range cs.commands {
        if cs.commands[i].ID == id {
            cs.commands[i].RunCount++
            if exitCode == 0 {
                cs.commands[i].SuccessCount++
            }
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

func (cs *CommandStore) Execute(cmdString string, saveDir bool, tags []string, description string, existingID int) error {
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

    if existingID > 0 {
        // Update existing command stats
        return cs.updateCommandStats(existingID, exitCode)
    }

    // Create new command
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

// Add methods for advanced chain execution
func (cs *CommandStore) ExecuteChainWithDependencies(chainID int) error {
    var chain *CommandChain
    for i := range cs.chains {
        if cs.chains[i].ID == chainID {
            chain = &cs.chains[i]
            break
        }
    }
    if chain == nil {
        return fmt.Errorf("chain with ID %d not found", chainID)
    }

    // Check and execute dependencies first
    for _, dep := range chain.Dependencies {
        if dep.WaitPolicy == "all" {
            for _, depChainID := range dep.DependsOn {
                if err := cs.ExecuteChainWithDependencies(depChainID); err != nil {
                    return fmt.Errorf("dependency chain %d failed: %v", depChainID, err)
                }
            }
        } else if dep.WaitPolicy == "any" {
            depSuccess := false
            var lastErr error
            for _, depChainID := range dep.DependsOn {
                if err := cs.ExecuteChainWithDependencies(depChainID); err == nil {
                    depSuccess = true
                    break
                } else {
                    lastErr = err
                }
            }
            if !depSuccess {
                return fmt.Errorf("all dependency chains failed, last error: %v", lastErr)
            }
        }
    }

    return cs.executeChainSteps(chain)
}

func (cs *CommandStore) executeChainSteps(chain *CommandChain) error {
    // Create a wait group for parallel execution
    var wg sync.WaitGroup
    results := make(map[int]error)
    var resultsMutex sync.Mutex

    // Helper function to execute a single command
    executeCmd := func(cmdID int) error {
        var cmd *Command
        for i := range cs.commands {
            if cs.commands[i].ID == cmdID {
                cmd = &cs.commands[i]
                break
            }
        }
        if cmd == nil {
            return fmt.Errorf("command with ID %d not found", cmdID)
        }

        execCmd := exec.Command("sh", "-c", cmd.Raw)
        // Either use the output
        output, err := execCmd.CombinedOutput()
        if err != nil {
            return fmt.Errorf("command failed with output: %s: %v", output, err)
        }
        return nil
    }

    // Execute steps
    for _, step := range chain.Steps {
        // Check conditions before executing
        execContext := &ExecutionContext{}
		if !cs.evaluateConditions(step.Conditions, execContext) {
            continue
        }

        // Handle parallel execution
        if len(step.ParallelWith) > 0 {
            // Execute main command and parallel commands concurrently
            wg.Add(1 + len(step.ParallelWith))
            
            // Execute main command
            go func(cmdID int) {
                defer wg.Done()
                if err := executeCmd(cmdID); err != nil {
                    resultsMutex.Lock()
                    results[cmdID] = err
                    resultsMutex.Unlock()
                }
            }(step.CommandID)

            // Execute parallel commands
            for _, parallelCmdID := range step.ParallelWith {
                go func(cmdID int) {
                    defer wg.Done()
                    if err := executeCmd(cmdID); err != nil {
                        resultsMutex.Lock()
                        results[cmdID] = err
                        resultsMutex.Unlock()
                    }
                }(parallelCmdID)
            }

            wg.Wait()

            // Check results
            if err, ok := results[step.CommandID]; ok {
                // Main command failed, execute OnFailure commands
                for _, failureCmdID := range step.OnFailure {
                    if err := executeCmd(failureCmdID); err != nil {
                        return fmt.Errorf("failure handler command %d failed: %v", failureCmdID, err)
                    }
                }
                return fmt.Errorf("main command %d failed: %v", step.CommandID, err)
            }

            // Execute OnSuccess commands
            for _, successCmdID := range step.OnSuccess {
                if err := executeCmd(successCmdID); err != nil {
                    return fmt.Errorf("success handler command %d failed: %v", successCmdID, err)
                }
            }
        } else {
            // Sequential execution
            if err := executeCmd(step.CommandID); err != nil {
                // Execute OnFailure commands
                for _, failureCmdID := range step.OnFailure {
                    if err := executeCmd(failureCmdID); err != nil {
                        return fmt.Errorf("failure handler command %d failed: %v", failureCmdID, err)
                    }
                }
                return err
            }

            // Execute OnSuccess commands
            for _, successCmdID := range step.OnSuccess {
                if err := executeCmd(successCmdID); err != nil {
                    return fmt.Errorf("success handler command %d failed: %v", successCmdID, err)
                }
            }
        }
    }

    return nil
}

func (cs *CommandStore) evaluateConditions(conditions []CommandCondition, context *ExecutionContext) bool {
    if len(conditions) == 0 {
        return true
    }

    for _, cond := range conditions {
        satisfied := false

        switch cond.Type {
        case "exit_code":
            exitCode, err := strconv.Atoi(cond.Value)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Warning: invalid exit code value '%s', condition will fail\n", cond.Value)
                return false
            }

            switch cond.Operation {
            case "equals":
                satisfied = context.LastExitCode == exitCode
            case "not_equals":
                satisfied = context.LastExitCode != exitCode
            case "less_than":
                satisfied = context.LastExitCode < exitCode
            case "greater_than":
                satisfied = context.LastExitCode > exitCode
            case "less_equals":
                satisfied = context.LastExitCode <= exitCode
            case "greater_equals":
                satisfied = context.LastExitCode >= exitCode
            default:
                fmt.Fprintf(os.Stderr, "Warning: unknown operation '%s' for exit_code condition\n", cond.Operation)
                return false
            }

        case "output_contains":
            switch cond.Operation {
            case "contains":
                satisfied = strings.Contains(context.LastOutput, cond.Value)
            case "not_contains":
                satisfied = !strings.Contains(context.LastOutput, cond.Value)
            case "starts_with":
                satisfied = strings.HasPrefix(context.LastOutput, cond.Value)
            case "ends_with":
                satisfied = strings.HasSuffix(context.LastOutput, cond.Value)
            case "matches":
                matched, err := regexp.MatchString(cond.Value, context.LastOutput)
                if err != nil {
                    fmt.Fprintf(os.Stderr, "Warning: invalid regex pattern '%s': %v\n", cond.Value, err)
                    return false
                }
                satisfied = matched
            default:
                fmt.Fprintf(os.Stderr, "Warning: unknown operation '%s' for output_contains condition\n", cond.Operation)
                return false
            }

        case "env_var":
            envValue := os.Getenv(cond.Value)
            
            switch cond.Operation {
            case "exists":
                satisfied = envValue != ""
            case "not_exists":
                satisfied = envValue == ""
            case "equals":
                parts := strings.SplitN(cond.Value, "=", 2)
                if len(parts) != 2 {
                    fmt.Fprintf(os.Stderr, "Warning: invalid env_var condition format, expected KEY=VALUE\n")
                    return false
                }
                satisfied = os.Getenv(parts[0]) == parts[1]
            case "contains":
                parts := strings.SplitN(cond.Value, "=", 2)
                if len(parts) != 2 {
                    fmt.Fprintf(os.Stderr, "Warning: invalid env_var condition format, expected KEY=VALUE\n")
                    return false
                }
                satisfied = strings.Contains(os.Getenv(parts[0]), parts[1])
            default:
                fmt.Fprintf(os.Stderr, "Warning: unknown operation '%s' for env_var condition\n", cond.Operation)
                return false
            }

        case "time_window":
            // Format: "HH:MM-HH:MM"
            timeRange := strings.Split(cond.Value, "-")
            if len(timeRange) != 2 {
                fmt.Fprintf(os.Stderr, "Warning: invalid time window format, expected HH:MM-HH:MM\n")
                return false
            }

            now := time.Now()
            start, err := time.Parse("15:04", timeRange[0])
            if err != nil {
                fmt.Fprintf(os.Stderr, "Warning: invalid start time format: %v\n", err)
                return false
            }

            end, err := time.Parse("15:04", timeRange[1])
            if err != nil {
                fmt.Fprintf(os.Stderr, "Warning: invalid end time format: %v\n", err)
                return false
            }

            // Adjust times to today
            start = time.Date(now.Year(), now.Month(), now.Day(), start.Hour(), start.Minute(), 0, 0, now.Location())
            end = time.Date(now.Year(), now.Month(), now.Day(), end.Hour(), end.Minute(), 0, 0, now.Location())

            switch cond.Operation {
            case "within":
                satisfied = now.After(start) && now.Before(end)
            case "outside":
                satisfied = now.Before(start) || now.After(end)
            default:
                fmt.Fprintf(os.Stderr, "Warning: unknown operation '%s' for time_window condition\n", cond.Operation)
                return false
            }

        case "file_exists":
            switch cond.Operation {
            case "exists":
                _, err := os.Stat(cond.Value)
                satisfied = err == nil
            case "not_exists":
                _, err := os.Stat(cond.Value)
                satisfied = os.IsNotExist(err)
            default:
                fmt.Fprintf(os.Stderr, "Warning: unknown operation '%s' for file_exists condition\n", cond.Operation)
                return false
            }

        default:
            fmt.Fprintf(os.Stderr, "Warning: unknown condition type '%s'\n", cond.Type)
            return false
        }

        if !satisfied {
            return false
        }
    }

    return true
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
    opts="--dir --list --search --filter-dir --filter-tag --export --import --rerun --tag --desc --favorite --stats --remove --interactive-edit --add-tags --remove-tags --undo --create-chain --create-chain-with-deps --run-chain --list-chains --help --config-path"

    case "${prev}" in
        --rerun|--favorite|--remove|--interactive-edit|--undo)
            # Complete with command IDs
            COMPREPLY=( $(save --list | grep "^#" | cut -d" " -f1 | cut -c2- | grep "^${cur}") )
            return 0
            ;;
        --tag|--add-tags|--remove-tags|--filter-tag)
            # Complete with existing tags
            COMPREPLY=( $(save --list-tags | grep "^${cur}") )
            return 0
            ;;
        --filter-dir)
            # Directory completion
            COMPREPLY=( $(compgen -d -- "${cur}") )
            return 0
            ;;
        --run-chain)
            # Complete with chain IDs
            COMPREPLY=( $(save --list-chains | grep "^#" | cut -d" " -f1 | cut -c2- | grep "^${cur}") )
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
        '--filter-tag[Filter by tag]'
        '--export[Export history]'
        '--import[Import commands]'
        '--rerun[Re-run command]'
        '--tag[Add tags]'
        '--desc[Add description]'
        '--favorite[Mark as favorite]'
        '--stats[Show statistics]'
        '--remove[Remove command(s)]'
        '--interactive-edit[Edit command interactively]'
        '--add-tags[Add tags to command]'
        '--remove-tags[Remove tags from command]'
        '--undo[Undo last edit]'
        '--create-chain[Create new command chain]'
        '--create-chain-with-deps[Create chain with dependencies]'
        '--run-chain[Run a command chain]'
        '--list-chains[List all chains]'
        '--help[Show help]'
        '--config-path[Show config file location]'
    )

    _arguments -C \
        "${opts[@]}" \
        "*::arg:->args"

    case $state in
        args)
            case $words[1] in
                --rerun|--favorite|--remove|--interactive-edit|--undo)
                    _values "command IDs" $(save --list | grep "^#" | cut -d" " -f1 | cut -c2-)
                    ;;
                --tag|--add-tags|--remove-tags|--filter-tag)
                    _values "tags" $(save --list-tags)
                    ;;
                --filter-dir)
                    _path_files -/
                    ;;
                --run-chain)
                    _values "chain IDs" $(save --list-chains | grep "^#" | cut -d" " -f1 | cut -c2-)
                    ;;
            esac
            ;;
    esac
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

var Version string // This will be set during build
var ConfigPath string // This will be set during build

// Add this function at the top level to maintain a list of valid command flags
var validCommandFlags = map[string]bool{
    "--favorite": true,
    "--remove": true,
    "--list": true,
    "--search": true,
    "--filter-dir": true,
    "--filter-tag": true,
    "--stats": true,
    "--rerun": true,
    "--interactive-edit": true,
    "--add-tags": true,
    "--remove-tags": true,
    "--undo": true,
    "--import": true,
    "--export": true,
    "--create-chain": true,
    "--run-chain": true,
    "--list-chains": true,
    "--help": true,
    "--config-path": true,
    "--version": true,
    "--install-completion": true,
    "--verify": true,
    "--backup": true,
}

// Add these new types for backup management
type BackupMetadata struct {
    Version     string    `json:"version"`
    CreatedAt   time.Time `json:"created_at"`
    CommandCount int      `json:"command_count"`
    ChainCount  int      `json:"chain_count"`
}

type BackupData struct {
    Metadata BackupMetadata  `json:"metadata"`
    Commands []Command       `json:"commands"`
    Chains   []CommandChain  `json:"chains"`
}

// Add these methods to CommandStore
func (cs *CommandStore) createBackup(backupPath string) error {
    backup := BackupData{
        Metadata: BackupMetadata{
            Version:      Version,
            CreatedAt:    time.Now(),
            CommandCount: len(cs.commands),
            ChainCount:   len(cs.chains),
        },
        Commands: cs.commands,
        Chains:   cs.chains,
    }

    data, err := json.MarshalIndent(backup, "", "    ")
    if err != nil {
        return fmt.Errorf("failed to marshal backup data: %w", err)
    }

    // Create backup directory if it doesn't exist
    backupDir := filepath.Join(filepath.Dir(cs.filepath), "backups")
    if err := os.MkdirAll(backupDir, 0755); err != nil {
        return fmt.Errorf("failed to create backup directory: %w", err)
    }

    // Use timestamp in backup filename if not provided
    if backupPath == "" {
        timestamp := time.Now().Format("20060102-150405")
        backupPath = filepath.Join(backupDir, fmt.Sprintf("save-history-%s.json", timestamp))
    }

    if err := os.WriteFile(backupPath, data, 0644); err != nil {
        return fmt.Errorf("failed to write backup file: %w", err)
    }

    return nil
}

func (cs *CommandStore) restoreFromBackup(backupPath string) error {
    data, err := os.ReadFile(backupPath)
    if err != nil {
        return fmt.Errorf("failed to read backup file: %w", err)
    }

    var backup BackupData
    if err := json.Unmarshal(data, &backup); err != nil {
        return fmt.Errorf("failed to parse backup data: %w", err)
    }

    // Create a backup of current data before restoring
    if err := cs.createBackup(""); err != nil {
        return fmt.Errorf("failed to create safety backup: %w", err)
    }

    // Restore data
    cs.commands = backup.Commands
    cs.chains = backup.Chains

    // Update IDs
    for _, cmd := range cs.commands {
        if cmd.ID > cs.lastID {
            cs.lastID = cmd.ID
        }
    }
    for _, chain := range cs.chains {
        if chain.ID > cs.lastChainID {
            cs.lastChainID = chain.ID
        }
    }

    cs.updateStats()
    return cs.save()
}

func (cs *CommandStore) verifyIntegrity() error {
    // Check for duplicate command IDs
    idMap := make(map[int]bool)
    for _, cmd := range cs.commands {
        if idMap[cmd.ID] {
            return fmt.Errorf("duplicate command ID found: %d", cmd.ID)
        }
        idMap[cmd.ID] = true
    }

    // Check chain IDs
    chainMap := make(map[int]bool)
    for _, chain := range cs.chains {
        if chainMap[chain.ID] {
            return fmt.Errorf("duplicate chain ID found: %d", chain.ID)
        }
        chainMap[chain.ID] = true
    }

    // Verify chain dependencies
    for _, chain := range cs.chains {
        for _, dep := range chain.Dependencies {
            if !chainMap[dep.ChainID] {
                return fmt.Errorf("chain %d depends on non-existent chain %d", chain.ID, dep.ChainID)
            }
        }
    }

    // Verify command references in chains
    for _, chain := range cs.chains {
        for _, step := range chain.Steps {
            if !idMap[step.CommandID] {
                return fmt.Errorf("chain %d references non-existent command %d", chain.ID, step.CommandID)
            }
            for _, parallelCmd := range step.ParallelWith {
                if !idMap[parallelCmd] {
                    return fmt.Errorf("chain %d references non-existent parallel command %d", chain.ID, parallelCmd)
                }
            }
        }
    }

    // Verify timestamps
    for _, cmd := range cs.commands {
        if cmd.Timestamp.IsZero() {
            return fmt.Errorf("command %d has invalid timestamp", cmd.ID)
        }
    }

    // Verify run counts
    for _, cmd := range cs.commands {
        if cmd.SuccessCount > cmd.RunCount {
            return fmt.Errorf("command %d has more successes than runs", cmd.ID)
        }
    }

    return nil
}

func (cs *CommandStore) repairIntegrity() error {
    // Remove commands with duplicate IDs (keep the first occurrence)
    idMap := make(map[int]bool)
    newCommands := make([]Command, 0, len(cs.commands))
    for _, cmd := range cs.commands {
        if !idMap[cmd.ID] {
            idMap[cmd.ID] = true
            newCommands = append(newCommands, cmd)
        }
    }
    cs.commands = newCommands

    // Remove chains with duplicate IDs
    chainMap := make(map[int]bool)
    newChains := make([]CommandChain, 0, len(cs.chains))
    for _, chain := range cs.chains {
        if !chainMap[chain.ID] {
            chainMap[chain.ID] = true
            newChains = append(newChains, chain)
        }
    }
    cs.chains = newChains

    // Remove invalid chain dependencies
    for i := range cs.chains {
        validDeps := make([]ChainDependency, 0)
        for _, dep := range cs.chains[i].Dependencies {
            if chainMap[dep.ChainID] {
                validDeps = append(validDeps, dep)
            }
        }
        cs.chains[i].Dependencies = validDeps
    }

    // Fix timestamps
    now := time.Now()
    for i := range cs.commands {
        if cs.commands[i].Timestamp.IsZero() {
            cs.commands[i].Timestamp = now
        }
    }

    // Fix run counts
    for i := range cs.commands {
        if cs.commands[i].SuccessCount > cs.commands[i].RunCount {
            cs.commands[i].SuccessCount = cs.commands[i].RunCount
        }
    }

    cs.updateStats()
    return cs.save()
}

func (cs *CommandStore) listFavorites() {
    hasFavorites := false
    fmt.Println("\n🌟 Favorite Commands:")
    fmt.Println("-------------------")
    
    for _, cmd := range cs.commands {
        if cmd.IsFavorite {
            hasFavorites = true
            cs.printCommandDetails(cmd)
            fmt.Println()
        }
    }
    
    if !hasFavorites {
        fmt.Println("No favorite commands found.")
        fmt.Println("\nTip: Mark a command as favorite using:")
        fmt.Println("  save --favorite <command_id>")
    }
}

func (cs *CommandStore) printCommandDetails(cmd Command) {
    description := cmd.Description
    if description == "" {
        description = "No description"
    }
    
    workDir := cmd.Dir
    if workDir == "" {
        workDir = "Current directory"
    }
    
    fmt.Printf("#%d: %s\n", cmd.ID, cmd.Raw)
    fmt.Printf("   📝 %s\n", description)
    fmt.Printf("   📂 %s\n", workDir)
    if len(cmd.Tags) > 0 {
        fmt.Printf("   🏷️  %s\n", strings.Join(cmd.Tags, ", "))
    }
    fmt.Printf("   ✨ Success rate: %.1f%% (%d runs)\n", 
        calculateSuccessRate(cmd.RunCount, cmd.SuccessCount),
        cmd.RunCount)
}

func calculateSuccessRate(total, success int) float64 {
    if total == 0 {
        return 0.0
    }
    return (float64(success) / float64(total)) * 100
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
	
	// Add these cases to main() switch statement
	case "--interactive-edit":
		if len(os.Args) < 3 {
			fmt.Println("Error: --interactive-edit requires a command ID")
			os.Exit(1)
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid command ID\n")
			os.Exit(1)
		}
		if err := store.InteractiveEdit(id); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully updated command #%d\n", id)

	case "--add-tags":
		if len(os.Args) < 4 {
			fmt.Println("Error: --add-tags requires a command ID and tags")
			os.Exit(1)
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid command ID\n")
			os.Exit(1)
		}
		tags := strings.Split(os.Args[3], ",")
		if err := store.ManipulateTags(id, tags, nil); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully added tags to command #%d\n", id)

	case "--remove-tags":
		if len(os.Args) < 4 {
			fmt.Println("Error: --remove-tags requires a command ID and tags")
			os.Exit(1)
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid command ID\n")
			os.Exit(1)
		}
		tags := strings.Split(os.Args[3], ",")
		if err := store.ManipulateTags(id, nil, tags); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully removed tags from command #%d\n", id)

	case "--undo":
		if len(os.Args) < 3 {
			fmt.Println("Error: --undo requires a command ID")
			os.Exit(1)
		}
		id, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid command ID\n")
			os.Exit(1)
		}
		if err := store.UndoLastEdit(id); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Successfully undid last edit for command #%d\n", id)

	case "--create-chain-with-deps":
		if len(os.Args) < 6 {
			fmt.Println("Error: --create-chain-with-deps requires name, description, steps file, and dependencies file")
			fmt.Println("Usage: save --create-chain-with-deps <name> <description> <steps.json> <dependencies.json>")
			os.Exit(1)
		}
		
		// Read and parse steps and dependencies from JSON files
		stepsData, err := os.ReadFile(os.Args[4])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading steps file: %v\n", err)
			os.Exit(1)
		}
		
		depsData, err := os.ReadFile(os.Args[5])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error reading dependencies file: %v\n", err)
			os.Exit(1)
		}
		
		var steps []ChainStep
		var deps []ChainDependency
		
		if err := json.Unmarshal(stepsData, &steps); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing steps: %v\n", err)
			os.Exit(1)
		}
		
		if err := json.Unmarshal(depsData, &deps); err != nil {
			fmt.Fprintf(os.Stderr, "Error parsing dependencies: %v\n", err)
			os.Exit(1)
		}
		
		chain := CommandChain{
			Name:         os.Args[2],
			Description:  os.Args[3],
			Steps:        steps,
			Dependencies: deps,
			CreatedAt:    time.Now(),
		}
		
		store.lastChainID++
		chain.ID = store.lastChainID
		store.chains = append(store.chains, chain)
		
		if err := store.save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error saving chain: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Successfully created chain #%d: %s\n", chain.ID, chain.Name)

	case "--help":
		printUsage()
		os.Exit(0)
	
	case "--remove":
		if len(os.Args) < 3 {
			fmt.Println("Error: --remove requires at least one command ID")
			os.Exit(1)
		}
		
		// Split the comma-separated IDs
		idStrs := strings.Split(os.Args[2], ",")
		ids := make([]int, 0, len(idStrs))
		
		// Convert each ID string to int
		for _, idStr := range idStrs {
			id, err := strconv.Atoi(strings.TrimSpace(idStr))
			if err != nil {
				fmt.Fprintf(os.Stderr, "Error: invalid command ID '%s'\n", idStr)
				os.Exit(1)
			}
			ids = append(ids, id)
		}
		
		// Remove the commands
		if err := store.RemoveCommands(ids); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Removed %d command(s)\n", len(ids))
	
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

	case "--filter-tag":
		if len(os.Args) < 3 {
			fmt.Println("Error: --filter-tag requires a tag name")
			os.Exit(1)
		}
		filterTag := strings.ToLower(os.Args[2])
		for _, cmd := range store.commands {
			// Check if any of the command's tags match the filter
			for _, tag := range cmd.Tags {
				if strings.ToLower(tag) == filterTag {
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
					break
				}
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
	
	case "--list-tags":
		// Create a map to count tag occurrences
		tagCount := make(map[string]int)
		for _, cmd := range store.commands {
			for _, tag := range cmd.Tags {
				tagCount[tag]++
			}
		}
	
		// Convert to slice for sorting
		type tagInfo struct {
			name  string
			count int
		}
		var tags []tagInfo
		for tag, count := range tagCount {
			tags = append(tags, tagInfo{tag, count})
		}
	
		// Sort tags by usage count (descending) and then alphabetically
		sort.Slice(tags, func(i, j int) bool {
			if tags[i].count != tags[j].count {
				return tags[i].count > tags[j].count
			}
			return tags[i].name < tags[j].name
		})
	
		// Print tags and their usage count
		fmt.Println("Available tags (with usage count):")
		for _, t := range tags {
			fmt.Printf("  %s (%d)\n", t.name, t.count)
		}
	
	case "--rerun":
		if len(os.Args) < 3 {
			fmt.Println("Error: --rerun requires a command ID")
			fmt.Println("Usage: save --rerun <id>")
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
		
		// Rerun the command with the existing ID
		if err := store.Execute(cmdToRerun.Raw, cmdToRerun.Dir != "", cmdToRerun.Tags, cmdToRerun.Description, id); err != nil {
			fmt.Fprintf(os.Stderr, "Error re-running command: %v\n", err)
			os.Exit(1)
		}
	
	case "--config-path":
		store, err := NewCommandStore()
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Config file location: %s\n", store.filepath)
	
	case "--list-chains":
		if len(store.chains) == 0 {
			fmt.Println("No command chains found")
			return
		}
		fmt.Println("\nAvailable Command Chains:")
		for _, chain := range store.chains {
			fmt.Printf("#%d %s\n", chain.ID, chain.Name)
			if chain.Description != "" {
				fmt.Printf("    Description: %s\n", chain.Description)
			}
			fmt.Printf("    Steps: %d, Run Count: %d, Success Rate: %.2f%%\n", 
				len(chain.Steps), chain.RunCount, chain.SuccessRate)
			fmt.Println()
		}

	case "--create-chain":
		if len(os.Args) < 4 {
			fmt.Println("Error: --create-chain requires name and description")
			fmt.Println("Usage: save --create-chain <name> <description>")
			os.Exit(1)
		}
		
		chain := CommandChain{
			Name:        os.Args[2],
			Description: os.Args[3],
			CreatedAt:   time.Now(),
			Steps:       []ChainStep{},
		}
		
		store.lastChainID++
		chain.ID = store.lastChainID
		store.chains = append(store.chains, chain)
		
		if err := store.save(); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating chain: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Created chain #%d: %s\n", chain.ID, chain.Name)

	case "--run-chain":
		if len(os.Args) < 3 {
			fmt.Println("Error: --run-chain requires a chain ID")
			fmt.Println("Usage: save --run-chain <chain-id>")
			os.Exit(1)
		}
		
		chainID, err := strconv.Atoi(os.Args[2])
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error: invalid chain ID\n")
			os.Exit(1)
		}
		
		// Check if --continue-on-error flag is present
		continueOnError := false
		if len(os.Args) > 3 && os.Args[3] == "--continue-on-error" {
			continueOnError = true
		}
		
		if err := store.ExecuteChainWithDependencies(chainID); err != nil {
			if !continueOnError {
				fmt.Fprintf(os.Stderr, "Error executing chain: %v\n", err)
				os.Exit(1)
			}
			fmt.Fprintf(os.Stderr, "Warning: chain execution had errors: %v\n", err)
		}

	case "--version":
		fmt.Printf("save version %s\n", Version)
		os.Exit(0)

	case "--install-completion":
		shell := os.Getenv("SHELL")
		if shell == "" {
			fmt.Println("Error: Could not detect shell. Please specify: save --generate-completion <bash|zsh>")
			os.Exit(1)
		}
		
		shellType := filepath.Base(shell) // Gets "bash" or "zsh" from path
		script := generateShellCompletion(shellType)
		if script == "" {
			fmt.Printf("Error: Unsupported shell: %s\n", shellType)
			os.Exit(1)
		}
		
		// Install to appropriate location based on shell
		var completionPath string
		switch shellType {
		case "bash":
			completionPath = filepath.Join(os.Getenv("HOME"), ".bash_completion.d", "save")
		case "zsh":
			completionPath = filepath.Join(os.Getenv("HOME"), ".zsh/completions", "_save")
		}
		
		// Create directory if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(completionPath), 0755); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating completion directory: %v\n", err)
			os.Exit(1)
		}
		
		if err := os.WriteFile(completionPath, []byte(script), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing completion script: %v\n", err)
			os.Exit(1)
		}
		
		fmt.Printf("Installed completion script to %s\n", completionPath)
		fmt.Printf("Add 'source %s' to your shell's rc file\n", completionPath)

	case "--verify":
		if err := store.verifyIntegrity(); err != nil {
			fmt.Fprintf(os.Stderr, "Data integrity issues found: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Data integrity verified successfully")

	case "--backup":
		backupPath := store.filepath + ".backup-" + time.Now().Format("20060102-150405")
		if err := store.createBackup(backupPath); err != nil {
			fmt.Fprintf(os.Stderr, "Error creating backup: %v\n", err)
			os.Exit(1)
		}
		fmt.Printf("Backup created at: %s\n", backupPath)

	case "--repair":
		if err := store.repairIntegrity(); err != nil {
			fmt.Fprintf(os.Stderr, "Error repairing data: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Data repair completed successfully")
		// Run verification after repair
		if err := store.verifyIntegrity(); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: some issues remain after repair: %v\n", err)
			os.Exit(1)
		}

	case "--restore":
		if len(os.Args) < 3 {
			fmt.Println("Error: --restore requires a backup file path")
			os.Exit(1)
		}
		if err := store.restoreFromBackup(os.Args[2]); err != nil {
			fmt.Fprintf(os.Stderr, "Error restoring from backup: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("Successfully restored from backup")

	case "--list-backups":
		backupDir := filepath.Join(filepath.Dir(store.filepath), "backups")
		files, err := os.ReadDir(backupDir)
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No backups found")
				return
			}
			fmt.Fprintf(os.Stderr, "Error reading backup directory: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("Available backups:")
		for _, file := range files {
			if !file.IsDir() && strings.HasPrefix(file.Name(), "save-history-") {
				path := filepath.Join(backupDir, file.Name())
				info, err := file.Info()
				if err != nil {
					continue
				}
				fmt.Printf("%s (%s, %d bytes)\n", path, info.ModTime().Format("2006-01-02 15:04:05"), info.Size())
			}
		}

	case "--list-favorites", "-lf":
		store.listFavorites()

	default:
		var tags []string
		var description string
		var saveDir bool
		cmdArgs := os.Args[1:]

		// Check if the command is just a flag without required arguments
		if len(cmdArgs) == 1 && validCommandFlags[cmdArgs[0]] {
			fmt.Fprintf(os.Stderr, "Error: %s requires additional arguments\n", cmdArgs[0])
			os.Exit(1)
		}

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

		// Check if the remaining command is just a flag
		if len(cmdArgs) > 0 && validCommandFlags[cmdArgs[0]] {
			fmt.Fprintf(os.Stderr, "Error: %s is a command flag and cannot be saved as a command\n", cmdArgs[0])
			os.Exit(1)
		}

		cmdString := strings.Join(cmdArgs, " ")
		if err := store.Execute(cmdString, saveDir, tags, description, 0); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
	}
}

func printUsage() {
    // ANSI color codes for better readability
    const (
        reset  = "\033[0m"
        bold   = "\033[1m"
        blue   = "\033[34m"
        green  = "\033[32m"
        yellow = "\033[33m"
    )

    // Title and Description
    fmt.Printf("\n%s%sSave Command Manager%s\n", bold, blue, reset)
    fmt.Print("A tool to save, manage, and run shell commands with advanced features\n")

    // Basic Usage
    fmt.Printf("%sUSAGE:%s\n", bold, reset)
    fmt.Printf("  save [flags] <command>     Save and execute a command\n")
    fmt.Printf("  save <subcommand> [args]   Run a specific subcommand\n\n")

    // Flags Section
    fmt.Printf("%sBASIC FLAGS:%s\n", bold, reset)
    fmt.Printf("  %-30s Add a description to the command\n", "--desc <description>")
    fmt.Printf("  %-30s Save with current directory\n", "--dir")
    fmt.Printf("  %-30s Add comma-separated tags\n", "--tag <tags>")
    fmt.Printf("  %-30s Add a favorite command\n", "--favorite <id>")

    // Basic Commands Section
    fmt.Printf("\n%sBASIC COMMANDS:%s\n", bold, reset)
    fmt.Printf("  %-30s List last n commands (default: 10)\n", "--list [n]")
    fmt.Printf("  %-30s Search commands\n", "--search <query>")
    fmt.Printf("  %-30s Show command statistics\n", "--stats")
    fmt.Printf("  %-30s Re-run command by ID\n", "--rerun <id>")
    fmt.Printf("  %-30s Mark command as favorite\n", "--favorite <id>")
    fmt.Printf("  %-30s Remove command(s) by ID(s)\n", "--remove <id1,id2,...>")
    fmt.Printf("  %-30s Filter commands by directory\n", "--filter-dir <path>")
    fmt.Printf("  %-30s Show config file location\n", "--config-path")

    // Tag Management
    fmt.Printf("\n%sTAG MANAGEMENT:%s\n", bold, reset)
    fmt.Printf("  %-30s List all available tags\n", "--list-tags")
    fmt.Printf("  %-30s Filter commands by tag\n", "--filter-tag <tag>")
    fmt.Printf("  %-30s Add tags to existing command\n", "--add-tags <id> <tags>")
    fmt.Printf("  %-30s Remove tags from command\n", "--remove-tags <id> <tags>")

    // Command Editing
    fmt.Printf("\n%sCOMMAND EDITING:%s\n", bold, reset)
    fmt.Printf("  %-30s Edit command interactively\n", "--interactive-edit <id>")
    fmt.Printf("  %-30s Edit specific command fields\n", "--edit <id> [flags]")
    fmt.Printf("  %-30s Undo last edit for command\n", "--undo <id>")

    // Chain Management
    fmt.Printf("\n%sCHAIN MANAGEMENT:%s\n", bold, reset)
    fmt.Printf("  %-30s Create a new command chain\n", "--create-chain <name> <desc>")
    fmt.Printf("  %-30s Create chain with dependencies\n", "--create-chain-with-deps <name> <desc> <steps.json> <deps.json>")
    fmt.Printf("  %-30s List all command chains\n", "--list-chains")
    fmt.Printf("  %-30s Run a command chain\n", "--run-chain <chain-id>")
    fmt.Printf("  %-30s Run chain ignoring errors\n", "--run-chain <chain-id> --continue-on-error")

    // Import/Export
    fmt.Printf("\n%sIMPORT/EXPORT:%s\n", bold, reset)
    fmt.Printf("  %-30s Export command history\n", "--export <filename>")
    fmt.Printf("  %-30s Import commands from file\n", "--import <filename>")

    // Examples Section
    fmt.Printf("\n%sEXAMPLES:%s\n", yellow, reset)
    
    fmt.Printf("\n%s  Basic Command Usage:%s\n", yellow, reset)
    fmt.Printf("    save 'echo Hello World'                   # Save and run simple command\n")
    fmt.Printf("    save --desc 'Greeting' 'echo Hello'       # Save with description\n")
    fmt.Printf("    save --tag cli,test 'npm test'            # Save with tags\n")
    fmt.Printf("    save --rerun 42                           # Rerun command #42\n")
    fmt.Printf("    save --favorite 42                        # Mark command #42 as favorite\n")
    fmt.Printf("    save --remove 42                          # Remove command #42\n")
    fmt.Printf("    save --config-path                        # Show config file location\n")
    
    fmt.Printf("\n%s  Command Editing:%s\n", yellow, reset)
    fmt.Printf("    save --interactive-edit 1                 # Edit command interactively\n")
    fmt.Printf("    save --add-tags 1 'git,prod'              # Add tags to command\n")
    fmt.Printf("    save --edit 1 --desc 'New description'    # Update description\n")
    fmt.Printf("    save --undo 1                             # Undo last edit\n")

    fmt.Printf("\n%s  Chain Management:%s\n", yellow, reset)
    fmt.Printf("    save --create-chain 'deploy' 'Deployment process' steps.json    # Create chain\n")
    fmt.Printf("    save --run-chain 1                        # Run chain #1\n")
    fmt.Printf("    save --list-chains                        # List all chains\n")

    fmt.Printf("\n%s  Filtering and Organization:%s\n", yellow, reset)
    fmt.Printf("    save --search 'git'                       # Search for git commands\n")
    fmt.Printf("    save --filter-tag docker                  # Show docker commands\n")
    fmt.Printf("    save --filter-dir ~/projects              # Show commands from directory\n")
    fmt.Printf("    save --list-tags                          # Show all tags\n")
    fmt.Printf("    save --favorite 42                        # Mark command as favorite\n")

    fmt.Printf("\n%s  Backup and Stats:%s\n", yellow, reset)
    fmt.Printf("    save --export backup.json                 # Export commands\n")
    fmt.Printf("    save --import backup.json                 # Import commands\n")
    fmt.Printf("    save --stats                              # Show statistics\n\n")

    fmt.Printf("%sFor more information and documentation, visit: https://github.com/t-rhex/save-go%s\n\n", blue, reset)
}