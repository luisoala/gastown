package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/workspace"
)

func init() {
	townCmd.AddCommand(townListCmd)
}

var townListCmd = &cobra.Command{
	Use:   "list",
	Short: "List Gas Town instances on this machine",
	Long: `Scans the parent directory of the current town root for sibling
directories that contain a mayor/town.json, and displays them with
their session counts.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTownList()
	},
}

// townInfo holds display data for one town.
type townInfo struct {
	Name         string
	Path         string
	SessionCount int
	Current      bool
}

func runTownList() error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	currentName, _ := workspace.GetTownName(townRoot)
	parentDir := filepath.Dir(townRoot)

	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return fmt.Errorf("reading parent directory %s: %w", parentDir, err)
	}

	// Get all tmux sessions once for counting.
	allSessions, _ := listTmuxSessions()

	var towns []townInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		candidatePath := filepath.Join(parentDir, entry.Name())
		townConfigPath := filepath.Join(candidatePath, "mayor", "town.json")
		data, err := os.ReadFile(townConfigPath) //nolint:gosec // G304
		if err != nil {
			continue
		}
		var tc struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(data, &tc); err != nil || tc.Name == "" {
			continue
		}

		// Count sessions matching this town's prefix.
		sessionCount := 0
		prefix := tc.Name + "-"
		for _, s := range allSessions {
			if strings.HasPrefix(s, prefix) {
				sessionCount++
			}
		}

		towns = append(towns, townInfo{
			Name:         tc.Name,
			Path:         candidatePath,
			SessionCount: sessionCount,
			Current:      tc.Name == currentName,
		})
	}

	if len(towns) == 0 {
		fmt.Println("No Gas Town instances found.")
		return nil
	}

	// Find max name width for alignment.
	maxName := 4 // "NAME"
	for _, t := range towns {
		if len(t.Name) > maxName {
			maxName = len(t.Name)
		}
	}

	fmt.Printf("%-*s  %8s  %s\n", maxName, "NAME", "SESSIONS", "PATH")
	for _, t := range towns {
		marker := "  "
		if t.Current {
			marker = "* "
		}
		fmt.Printf("%s%-*s  %8d  %s\n", marker, maxName, t.Name, t.SessionCount, t.Path)
	}

	return nil
}

