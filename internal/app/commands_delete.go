package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/inovue/gh-workspace/internal/domain"
	"github.com/spf13/cobra"
)

type deleteOptions struct {
	remote bool
	yes    bool
	force  bool
}

func (a *App) deleteCommand() *cobra.Command {
	var options deleteOptions
	cmd := &cobra.Command{
		Use:     "delete [repository|query]",
		Aliases: []string{"rm"},
		Short:   "Delete a local repository, and optionally the GitHub repository",
		Long: `Delete a repository from the workspace. The GitHub repository is kept
unless --remote is given; with --remote, a repository that is not cloned is
deleted on GitHub only.

Local deletion checks for uncommitted changes, unpushed commits, stashes, and
linked worktrees. In a terminal you confirm after seeing them; otherwise
--force is required. Remote deletion asks you to type the repository name
unless --yes is given, and needs the delete_repo scope
(gh auth refresh -s delete_repo).`,
		Example: `  gh workspace delete cli/cli
  gh workspace delete my-scratch --remote
  gh workspace delete old-experiment --yes --force`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.runDelete(firstArg(args), options)
		},
	}
	cmd.Flags().BoolVar(&options.remote, "remote", false, "also delete the repository on GitHub")
	cmd.Flags().BoolVarP(&options.yes, "yes", "y", false, "skip confirmation prompts")
	cmd.Flags().BoolVar(&options.force, "force", false, "delete even with uncommitted or unpushed work")
	return cmd
}

func (a *App) runDelete(arg string, options deleteOptions) error {
	var local *LocalRepository
	var remote domain.Repo

	if ref, ok := a.parseReference(arg); ok {
		repos, err := a.scan()
		if err != nil {
			return err
		}
		if found := find(repos, ref); len(found) > 0 {
			local = &found[0]
		} else if !options.remote {
			return fmt.Errorf("%s is not in the workspace; pass --remote to delete it on GitHub", ref)
		}
		remote = ref
	} else {
		repo, err := a.resolveLocal(arg, "Delete which repository?")
		if err != nil {
			return err
		}
		local = &repo
		remote = repo.Repo
	}

	if options.remote {
		if remote.IsZero() {
			return fmt.Errorf("cannot tell which GitHub repository %s belongs to", local.Path)
		}
		if local != nil && local.Worktree {
			return fmt.Errorf("%s is a linked worktree; delete the main working tree to delete the GitHub repository", local.Path)
		}
	}

	var risks []string
	if local != nil {
		risks = localRisks(local.Path, local.Worktree)
	}

	fmt.Fprintln(a.stderr, "This will delete:")
	if local != nil {
		fmt.Fprintf(a.stderr, "  local   %s\n", local.Path)
	}
	if options.remote {
		fmt.Fprintf(a.stderr, "  GitHub  %s\n", remote)
	} else if !remote.IsZero() {
		fmt.Fprintf(a.stderr, "  (the GitHub repository %s is kept; pass --remote to delete it)\n", remote)
	}
	for _, risk := range risks {
		fmt.Fprintf(a.stderr, "  warning: %s\n", risk)
	}

	if len(risks) > 0 && !options.force && (options.yes || !a.isTerminal) {
		return fmt.Errorf("refusing to delete %s with unsaved work; pass --force", local.Path)
	}
	if !options.yes {
		if err := a.requireTTY("delete confirmation"); err != nil {
			return fmt.Errorf("%w; pass --yes", err)
		}
		if options.remote {
			var typed string
			err := a.selector.Input(fmt.Sprintf("Type %s to delete it on GitHub", remote.FullName()), &typed, func(value string) error {
				if !strings.EqualFold(strings.TrimSpace(value), remote.FullName()) {
					return fmt.Errorf("type %s to confirm", remote.FullName())
				}
				return nil
			})
			if err != nil {
				return err
			}
		} else {
			confirmed, err := a.selector.Confirm("Delete?", false)
			if err != nil {
				return err
			}
			if !confirmed {
				return errCancelled
			}
		}
	}

	if options.remote {
		if err := a.github.Delete(remote); err != nil {
			return err
		}
		fmt.Fprintf(a.stderr, "Deleted %s on GitHub\n", remote)
	}
	if local != nil {
		if err := removeWorkingTree(*local); err != nil {
			return err
		}
		fmt.Fprintf(a.stderr, "Deleted %s\n", local.Path)
	}
	return nil
}

// localRisks describes work that would be lost by deleting the working tree.
func localRisks(path string, worktree bool) []string {
	var risks []string
	if out, err := gitOutput(path, "status", "--porcelain"); err != nil {
		risks = append(risks, "could not read git status")
	} else if out != "" {
		risks = append(risks, fmt.Sprintf("%d uncommitted change(s)", len(strings.Split(out, "\n"))))
	}
	if worktree {
		if out, _ := gitOutput(path, "log", "HEAD", "--not", "--remotes", "--oneline"); out != "" {
			risks = append(risks, fmt.Sprintf("%d unpushed commit(s) on this worktree", len(strings.Split(out, "\n"))))
		}
		return risks
	}
	if out, _ := gitOutput(path, "log", "--branches", "--not", "--remotes", "--oneline"); out != "" {
		risks = append(risks, fmt.Sprintf("%d unpushed commit(s)", len(strings.Split(out, "\n"))))
	}
	if out, _ := gitOutput(path, "stash", "list"); out != "" {
		risks = append(risks, fmt.Sprintf("%d stash(es)", len(strings.Split(out, "\n"))))
	}
	if out, _ := gitOutput(path, "worktree", "list", "--porcelain"); strings.Count(out, "worktree ") > 1 {
		risks = append(risks, "linked worktrees will stop working")
	}
	return risks
}

func removeWorkingTree(repo LocalRepository) error {
	var commonDir string
	if repo.Worktree {
		commonDir, _ = gitOutput(repo.Path, "rev-parse", "--path-format=absolute", "--git-common-dir")
	}
	if err := removeAll(repo.Path); err != nil {
		return fmt.Errorf("delete %s: %w", repo.Path, err)
	}
	if commonDir != "" {
		_, _ = gitOutput(filepath.Dir(commonDir), "worktree", "prune")
	}
	removeEmptyParents(filepath.Dir(repo.Path), repo.Root)
	return nil
}

// removeAll removes path, making read-only files such as git objects
// writable first when needed, which Windows requires.
func removeAll(path string) error {
	if err := os.RemoveAll(path); err == nil {
		return nil
	}
	_ = filepath.WalkDir(path, func(p string, _ os.DirEntry, _ error) error {
		_ = os.Chmod(p, 0o700)
		return nil
	})
	return os.RemoveAll(path)
}
