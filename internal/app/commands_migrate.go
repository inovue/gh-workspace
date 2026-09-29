package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/inovue/gh-workspace/internal/domain"
	"github.com/spf13/cobra"
)

type move struct {
	repo LocalRepository
	to   string
}

func (a *App) migrateCommand() *cobra.Command {
	var apply bool
	cmd := &cobra.Command{
		Use:   "migrate",
		Short: "Move repositories to their canonical workspace paths",
		Long: `Move repositories whose folder does not match their identity to the path
the current layout gives them, within the same root. This covers the legacy
{root}/github.com/{owner}/{repo} layout and repositories renamed or
transferred on GitHub.

Nothing is moved without --apply. Linked worktrees stay where they are and
are repaired after their main working tree moves.`,
		Example: `  gh workspace migrate           # show the plan
  gh workspace migrate --apply   # move the repositories`,
		Args: cobra.NoArgs,
		RunE: func(_ *cobra.Command, _ []string) error {
			return a.runMigrate(apply)
		},
	}
	cmd.Flags().BoolVar(&apply, "apply", false, "move the repositories instead of only showing the plan")
	return cmd
}

func (a *App) runMigrate(apply bool) error {
	repos, err := a.scan()
	if err != nil {
		return err
	}
	var moves []move
	targets := map[string]bool{}
	for _, repo := range repos {
		if repo.Repo.IsZero() || repo.Worktree {
			continue
		}
		to := filepath.Join(repo.Root, domain.RelPath(repo.Repo))
		if strings.EqualFold(to, repo.Path) {
			continue
		}
		if strings.HasPrefix(to, repo.Path+string(filepath.Separator)) {
			fmt.Fprintf(a.stderr, "skip  %s: its canonical path %s is inside it\n", repo.Path, to)
			continue
		}
		if inside := enclosingWorkingTree(to, repo.Root); inside != "" {
			fmt.Fprintf(a.stderr, "skip  %s: %s would be inside the repository %s\n", repo.Path, to, inside)
			continue
		}
		if exists(to) || targets[strings.ToLower(to)] {
			fmt.Fprintf(a.stderr, "skip  %s: %s already exists\n", repo.Path, to)
			continue
		}
		targets[strings.ToLower(to)] = true
		moves = append(moves, move{repo: repo, to: to})
	}
	if len(moves) == 0 {
		fmt.Fprintln(a.stderr, "Every repository is already at its canonical path.")
		return nil
	}

	for _, m := range moves {
		rel, _ := filepath.Rel(m.repo.Root, m.to)
		fmt.Fprintf(a.stderr, "move  %s -> %s\n", m.repo.Display(), filepath.ToSlash(rel))
	}
	if !apply {
		fmt.Fprintf(a.stderr, "\n%d repositories would move. Run with --apply to move them.\n", len(moves))
		return nil
	}

	for _, m := range moves {
		if err := os.MkdirAll(filepath.Dir(m.to), 0o755); err != nil {
			return err
		}
		if err := os.Rename(m.repo.Path, m.to); err != nil {
			return fmt.Errorf("move %s: %w", m.repo.Path, err)
		}
		_, _ = gitOutput(m.to, "worktree", "repair")
		removeEmptyParents(filepath.Dir(m.repo.Path), m.repo.Root)
	}
	fmt.Fprintf(a.stderr, "\nMoved %d repositories.\n", len(moves))
	return nil
}

// enclosingWorkingTree returns a working tree between root and path, if any.
func enclosingWorkingTree(path, root string) string {
	for dir := filepath.Dir(path); ; dir = filepath.Dir(dir) {
		rel, err := filepath.Rel(root, dir)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			return ""
		}
		if _, ok := gitDirOf(dir); ok {
			return dir
		}
	}
}
