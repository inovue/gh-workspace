package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/inovue/gh-workspace/internal/domain"
	"github.com/spf13/cobra"
)

const switchOwnerValue = "\x00switch-owner"

func (a *App) cloneCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "clone [repository...|owner] [-- <git clone flags>...]",
		Short: "Clone repositories into the workspace and print their paths",
		Long: `Clone repositories to their workspace location and print each path.

Repositories that are already in the workspace are not cloned again; their
existing path is printed, so "clone" doubles as "get the path, cloning if
needed". The folder name uses the repository's canonical casing on GitHub.

With an owner (a user or organization name) or no argument, a picker lists
that owner's repositories, or your own. Flags after "--" go to git clone.`,
		Example: `  gh workspace clone cli/cli
  gh workspace clone https://github.com/cli/cli/pull/123
  gh workspace clone my-org
  gh workspace clone torvalds/linux -- --depth=1
  code "$(gh workspace clone cli/cli)"`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var gitArgs []string
			if dash := cmd.ArgsLenAtDash(); dash >= 0 {
				args, gitArgs = args[:dash], args[dash:]
			}
			if len(args) == 0 {
				return a.cloneInteractively("")
			}
			if len(args) == 1 && !strings.ContainsAny(args[0], "/:") {
				return a.cloneInteractively(args[0])
			}

			var errs []error
			for _, arg := range args {
				repo, err := domain.ParseReference(arg, a.defaultHost)
				if err == nil {
					err = a.clone(repo, false, gitArgs)
				}
				if err != nil {
					errs = append(errs, err)
				}
			}
			return errors.Join(errs...)
		},
	}
}

func (a *App) cloneInteractively(owner string) error {
	repo, err := a.selectRemote(owner)
	if err != nil {
		return err
	}
	return a.clone(repo, true, nil)
}

// clone prints the path of repo, cloning it first when it is not already in
// the workspace. canonical reports that repo already carries GitHub's casing.
func (a *App) clone(repo domain.Repo, canonical bool, gitArgs []string) error {
	local, err := a.scan()
	if err != nil {
		return err
	}
	if found := find(local, repo); len(found) > 0 {
		fmt.Fprintf(a.stderr, "%s is already cloned\n", repo)
		fmt.Fprintln(a.stdout, found[0].Path)
		return nil
	}
	if !canonical {
		resolved, err := a.github.Resolve(repo)
		if err != nil {
			return err
		}
		if found := find(local, resolved); len(found) > 0 {
			fmt.Fprintf(a.stderr, "%s is already cloned\n", resolved)
			fmt.Fprintln(a.stdout, found[0].Path)
			return nil
		}
		repo = resolved
	}

	destination, err := a.destination(repo)
	if err != nil {
		return err
	}
	if err := a.checkDestination(destination, repo); err != nil {
		return err
	}
	parent := filepath.Dir(destination)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	if err := a.github.Clone(repo, destination, gitArgs); err != nil {
		root, _ := a.primaryRoot()
		removeEmptyParents(parent, root)
		return err
	}
	fmt.Fprintln(a.stdout, destination)
	return nil
}

// selectRemote picks a repository owned by owner, or by the authenticated
// user when owner is empty. A leading entry switches to another owner.
func (a *App) selectRemote(owner string) (domain.Repo, error) {
	if err := a.requireTTY("repository selection"); err != nil {
		return domain.Repo{}, fmt.Errorf("%w; pass owner/repo", err)
	}
	for {
		remote, err := a.github.ListRepositories(a.defaultHost, owner)
		if err != nil {
			return domain.Repo{}, err
		}
		local, err := a.scan()
		if err != nil {
			return domain.Repo{}, err
		}

		current := owner
		if current == "" {
			current = "you"
		}
		options := []SelectionOption{{
			Value: switchOwnerValue,
			Title: fmt.Sprintf("↔ Switch owner (showing %s)", current),
		}}
		for _, item := range remote {
			repo, err := domain.ParseReference(item.NameWithOwner, a.defaultHost)
			if err != nil {
				continue
			}
			var notes []string
			if item.IsFork {
				notes = append(notes, "fork")
			}
			if item.IsArchived {
				notes = append(notes, "archived")
			}
			if item.Description != "" {
				notes = append(notes, truncate(item.Description, 60))
			}
			option := SelectionOption{
				Value:       repo.String(),
				Title:       repo.FullName(),
				Description: dim(strings.Join(notes, " · ")),
			}
			if len(find(local, repo)) > 0 {
				option.Title = dim(repo.FullName() + " (cloned)")
				option.Disabled = true
			}
			options = append(options, option)
		}
		if len(options) == 1 {
			fmt.Fprintf(a.stderr, "%s has no repositories you can access\n", current)
		}

		selected, err := a.selector.Select("Clone a repository", options)
		if err != nil {
			return domain.Repo{}, err
		}
		if selected != switchOwnerValue {
			return domain.ParseReference(selected, a.defaultHost)
		}
		owner, err = a.selectOwner("Switch owner")
		if err != nil {
			return domain.Repo{}, err
		}
	}
}

// selectOwner picks the authenticated user or one of their organizations.
func (a *App) selectOwner(title string) (string, error) {
	login, orgs, err := a.github.Viewer(a.defaultHost)
	if err != nil {
		return "", err
	}
	if len(orgs) == 0 {
		return login, nil
	}
	options := []SelectionOption{{Value: login, Title: login, Description: dim("personal account")}}
	for _, org := range orgs {
		options = append(options, SelectionOption{Value: org, Title: org, Description: dim("organization")})
	}
	return a.selector.Select(title, options)
}

func truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max-1]) + "…"
}
