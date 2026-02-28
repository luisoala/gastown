package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/steveyegge/gastown/internal/tmux"
	"github.com/steveyegge/gastown/internal/workspace"
)

func init() {
	townCmd.AddCommand(townSwitchCmd)
}

var townSwitchCmd = &cobra.Command{
	Use:   "switch <town-name>",
	Short: "Switch to another town's mayor session",
	Long: `Switch the current tmux client to the mayor session of another
Gas Town instance. The target town must be running (its mayor session
must exist).

Example:
  gt town switch ih    # Jump to ih's mayor session`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTownSwitch(args[0])
	},
}

func runTownSwitch(targetName string) error {
	townRoot, err := workspace.FindFromCwdOrError()
	if err != nil {
		return fmt.Errorf("not in a Gas Town workspace: %w", err)
	}

	// Validate target town exists by scanning sibling dirs.
	parentDir := filepath.Dir(townRoot)
	entries, err := os.ReadDir(parentDir)
	if err != nil {
		return fmt.Errorf("reading parent directory: %w", err)
	}

	found := false
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		tcPath := filepath.Join(parentDir, entry.Name(), "mayor", "town.json")
		data, err := os.ReadFile(tcPath) //nolint:gosec // G304
		if err != nil {
			continue
		}
		var tc struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(data, &tc); err != nil {
			continue
		}
		if tc.Name == targetName {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("town %q not found (no sibling directory with matching town.json)", targetName)
	}

	// Check if target mayor session is running.
	mayorSession := targetName + "-hq-mayor"
	allSessions, err := listTmuxSessions()
	if err != nil {
		return fmt.Errorf("listing tmux sessions: %w", err)
	}

	running := false
	for _, s := range allSessions {
		if strings.TrimSpace(s) == mayorSession {
			running = true
			break
		}
	}

	if !running {
		return fmt.Errorf("town %q is not running (no %s session found)", targetName, mayorSession)
	}

	// Switch to the target mayor session.
	return tmux.BuildCommand("switch-client", "-t", mayorSession).Run()
}
