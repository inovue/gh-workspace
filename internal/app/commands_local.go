package app

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
)

func (a *App) pathCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "path [repository|query]",
		Short: "Print the path of a local repository",
		Long: `Print the absolute path of a repository in the workspace.

An explicit reference (owner/repo, host/owner/repo, or a URL) must match a
cloned repository exactly. Any other query matches repository names fuzzily;
when several match, an interactive picker opens, or the command fails when
not attached to a terminal. Without an argument, the picker lists everything.`,
		Example: `  cd "$(gh workspace path cli/cli)"
  code "$(gh workspace path workspace)"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			repo, err := a.resolveLocal(firstArg(args), "Select a repository")
			if err != nil {
				return err
			}
			fmt.Fprintln(a.stdout, repo.Path)
			return nil
		},
	}
}

type listEntry struct {
	Name     string `json:"name"`
	Path     string `json:"path"`
	Root     string `json:"root"`
	Host     string `json:"host,omitempty"`
	Owner    string `json:"owner,omitempty"`
	Repo     string `json:"repo,omitempty"`
	Remote   string `json:"remote,omitempty"`
	Worktree bool   `json:"worktree"`
}

func (a *App) listCommand() *cobra.Command {
	var fullPath, asJSON bool
	cmd := &cobra.Command{
		Use:     "list [query]",
		Aliases: []string{"ls"},
		Short:   "List local repositories",
		Long: `List repositories in the workspace, one per line, as paths relative to their
root. The optional query filters fuzzily, like "path".`,
		Example: `  gh workspace list
  gh workspace list --full-path | xargs -I{} git -C {} pull --ff-only
  gh workspace list --json | jq -r '.[] | select(.owner == "cli") | .path'`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			repos, err := a.scan()
			if err != nil {
				return err
			}
			repos = match(repos, firstArg(args))
			if asJSON {
				entries := make([]listEntry, 0, len(repos))
				for _, repo := range repos {
					entries = append(entries, listEntry{
						Name:     repo.Display(),
						Path:     repo.Path,
						Root:     repo.Root,
						Host:     repo.Repo.Host,
						Owner:    repo.Repo.Owner,
						Repo:     repo.Repo.Name,
						Remote:   repo.Remote,
						Worktree: repo.Worktree,
					})
				}
				encoder := json.NewEncoder(a.stdout)
				encoder.SetIndent("", "  ")
				return encoder.Encode(entries)
			}
			for _, repo := range repos {
				if fullPath {
					fmt.Fprintln(a.stdout, repo.Path)
				} else {
					fmt.Fprintln(a.stdout, repo.Display())
				}
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&fullPath, "full-path", "p", false, "print absolute paths")
	cmd.Flags().BoolVar(&asJSON, "json", false, "print JSON with identity, path, and remote")
	return cmd
}

func (a *App) rootCommand() *cobra.Command {
	var all bool
	cmd := &cobra.Command{
		Use:   "root",
		Short: "Print the workspace root",
		Long: `Print the primary workspace root, where new repositories are placed.

Roots come from ` + RootEnv + ` (separated like PATH), then from the git config
key ` + RootGitConfigKey + ` (may be repeated), then default to ~/workspaces.`,
		Example: `  git config --global ` + RootGitConfigKey + ` ~/src
  export ` + RootEnv + `=~/src:~/work`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			roots, err := a.Roots()
			if err != nil {
				return err
			}
			if !all {
				roots = roots[:1]
			}
			for _, root := range roots {
				fmt.Fprintln(a.stdout, root)
			}
			return nil
		},
	}
	cmd.Flags().BoolVar(&all, "all", false, "print every configured root")
	return cmd
}

func firstArg(args []string) string {
	if len(args) == 0 {
		return ""
	}
	return args[0]
}
