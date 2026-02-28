package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
)

var refsAddName string

func init() {
	rootCmd.AddCommand(refsCmd)
	refsCmd.AddCommand(refsListCmd)
	refsCmd.AddCommand(refsAddCmd)
	refsCmd.AddCommand(refsSyncCmd)

	refsAddCmd.Flags().StringVar(&refsAddName, "name", "", "Alias name for the cloned repo (defaults to repo basename)")
}

var refsCmd = &cobra.Command{
	Use:   "refs",
	Short: "Manage the cross-repo reference pool",
	Long: `Manage shallow git clones in $GT_REF_POOL for cross-repo context.

Polecats can read these repos for reference without needing full clones
in their worktrees. The pool is a directory of shallow (--depth 1)
clones that can be synced on demand.

Set GT_REF_POOL in daemon.json to enable:
  {"env": {"GT_REF_POOL": "/data/refs"}}`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRefsList()
	},
}

var refsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List repos in the reference pool",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRefsList()
	},
}

var refsAddCmd = &cobra.Command{
	Use:   "add <url>",
	Short: "Add a repo to the reference pool",
	Long: `Clone a repository into the reference pool as a shallow clone (--depth 1).
Use --name to specify a custom alias; otherwise the repo basename is used.

Example:
  gt refs add https://github.com/org/repo
  gt refs add git@github.com:org/repo.git --name myrepo`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRefsAdd(args[0], refsAddName)
	},
}

var refsSyncCmd = &cobra.Command{
	Use:   "sync [alias]",
	Short: "Update reference pool repos",
	Long: `Pull latest changes for repos in the reference pool.
If an alias is given, only that repo is synced; otherwise all repos are synced.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		alias := ""
		if len(args) > 0 {
			alias = args[0]
		}
		return runRefsSync(alias)
	},
}

func getRefPool() (string, error) {
	pool := os.Getenv("GT_REF_POOL")
	if pool == "" {
		return "", fmt.Errorf("GT_REF_POOL not set — configure it in daemon.json:\n  {\"env\": {\"GT_REF_POOL\": \"/data/refs\"}}")
	}
	return pool, nil
}

func runRefsList() error {
	pool, err := getRefPool()
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(pool)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Printf("Reference pool at %s is empty (directory does not exist).\n", pool)
			return nil
		}
		return fmt.Errorf("reading %s: %w", pool, err)
	}

	var repos []struct {
		Name   string
		Remote string
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		repoPath := filepath.Join(pool, entry.Name())
		// Check if it's a git repo.
		if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
			continue
		}
		// Get remote URL.
		out, err := exec.Command("git", "-C", repoPath, "remote", "get-url", "origin").Output()
		remote := "(unknown)"
		if err == nil {
			remote = strings.TrimSpace(string(out))
		}
		repos = append(repos, struct {
			Name   string
			Remote string
		}{entry.Name(), remote})
	}

	if len(repos) == 0 {
		fmt.Printf("No repos in reference pool at %s\n", pool)
		fmt.Println("Add one with: gt refs add <url>")
		return nil
	}

	// Find max name width.
	maxName := 4
	for _, r := range repos {
		if len(r.Name) > maxName {
			maxName = len(r.Name)
		}
	}

	fmt.Printf("Reference pool: %s\n\n", pool)
	fmt.Printf("%-*s  %s\n", maxName, "NAME", "REMOTE")
	for _, r := range repos {
		fmt.Printf("%-*s  %s\n", maxName, r.Name, r.Remote)
	}

	return nil
}

func runRefsAdd(url, name string) error {
	pool, err := getRefPool()
	if err != nil {
		return err
	}

	// Derive name from URL if not specified.
	if name == "" {
		base := filepath.Base(url)
		name = strings.TrimSuffix(base, ".git")
		if name == "" || name == "." {
			return fmt.Errorf("cannot derive repo name from URL %q — use --name", url)
		}
	}

	targetDir := filepath.Join(pool, name)
	if _, err := os.Stat(targetDir); err == nil {
		return fmt.Errorf("directory %s already exists — remove it or use a different --name", targetDir)
	}

	// Ensure pool directory exists.
	if err := os.MkdirAll(pool, 0o755); err != nil {
		return fmt.Errorf("creating pool directory: %w", err)
	}

	fmt.Printf("Cloning %s → %s ...\n", url, targetDir)
	cmd := exec.Command("git", "clone", "--depth", "1", url, targetDir)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git clone failed: %w", err)
	}

	fmt.Printf("Added %s to reference pool.\n", name)
	return nil
}

func runRefsSync(alias string) error {
	pool, err := getRefPool()
	if err != nil {
		return err
	}

	if alias != "" {
		return syncOneRepo(pool, alias)
	}

	entries, err := os.ReadDir(pool)
	if err != nil {
		return fmt.Errorf("reading %s: %w", pool, err)
	}

	synced := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		repoPath := filepath.Join(pool, entry.Name())
		if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
			continue
		}
		if err := syncOneRepo(pool, entry.Name()); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to sync %s: %v\n", entry.Name(), err)
		} else {
			synced++
		}
	}

	if synced == 0 {
		fmt.Println("No repos to sync.")
	} else {
		fmt.Printf("Synced %d repo(s).\n", synced)
	}
	return nil
}

func syncOneRepo(pool, name string) error {
	repoPath := filepath.Join(pool, name)
	if _, err := os.Stat(filepath.Join(repoPath, ".git")); err != nil {
		return fmt.Errorf("%s is not a git repository", repoPath)
	}

	fmt.Printf("Syncing %s ...\n", name)
	cmd := exec.Command("git", "-C", repoPath, "pull", "--depth", "1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}
