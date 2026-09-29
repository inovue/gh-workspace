package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/inovue/gh-workspace/internal/domain"
	"github.com/spf13/cobra"
)

func (a *App) createCommand() *cobra.Command {
	var public, private, internal bool
	var description, template string
	cmd := &cobra.Command{
		Use:   "create [[owner/]name]",
		Short: "Create a repository on GitHub and in the workspace",
		Long: `Create a repository locally and on GitHub, then print its path.

Without a template, a git repository with an empty initial commit is created
in the workspace and pushed to the new GitHub repository, using your git
default branch name. With --template, the repository is generated on GitHub
from the template and then cloned.

A bare name is created under your personal account. Missing values are asked
for interactively; without a terminal, visibility defaults to private.`,
		Example: `  gh workspace create
  gh workspace create my-tool --public -d "A small tool"
  gh workspace create my-org/service --private --template my-org/service-template
  cd "$(gh workspace create scratch)"`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			visibility, err := visibilityFlag(public, private, internal)
			if err != nil {
				return err
			}
			return a.runCreate(firstArg(args), CreateOptions{
				Visibility:  visibility,
				Description: description,
				Template:    template,
			})
		},
	}
	cmd.Flags().BoolVar(&public, "public", false, "make the repository public")
	cmd.Flags().BoolVar(&private, "private", false, "make the repository private (default)")
	cmd.Flags().BoolVar(&internal, "internal", false, "make the repository internal")
	cmd.Flags().StringVarP(&description, "description", "d", "", "repository description")
	cmd.Flags().StringVarP(&template, "template", "t", "", "create from a template repository (owner/repo)")
	return cmd
}

func visibilityFlag(public, private, internal bool) (string, error) {
	var chosen []string
	if public {
		chosen = append(chosen, "public")
	}
	if private {
		chosen = append(chosen, "private")
	}
	if internal {
		chosen = append(chosen, "internal")
	}
	if len(chosen) > 1 {
		return "", fmt.Errorf("choose only one of --public, --private, --internal")
	}
	if len(chosen) == 0 {
		return "", nil
	}
	return chosen[0], nil
}

func (a *App) runCreate(arg string, options CreateOptions) error {
	interactive := arg == ""
	if interactive {
		if err := a.requireTTY("create without a name"); err != nil {
			return fmt.Errorf("%w; pass [owner/]name", err)
		}
	}

	local, err := a.scan()
	if err != nil {
		return err
	}
	var repo domain.Repo
	switch {
	case strings.Contains(arg, "/"):
		repo, err = domain.ParseReference(arg, a.defaultHost)
		if err != nil {
			return err
		}
	case arg != "":
		if !domain.ValidName(arg) {
			return fmt.Errorf("invalid repository name: %s", arg)
		}
		login, _, err := a.github.Viewer(a.defaultHost)
		if err != nil {
			return err
		}
		repo = domain.Repo{Host: a.defaultHost, Owner: login, Name: arg}
	default:
		owner, err := a.selectOwner("Owner")
		if err != nil {
			return err
		}
		repo = domain.Repo{Host: a.defaultHost, Owner: owner}
		var name string
		err = a.selector.Input("Repository name", &name, func(value string) error {
			value = strings.TrimSpace(value)
			if !domain.ValidName(value) {
				return fmt.Errorf("use letters, digits, '.', '-', or '_'")
			}
			return a.checkCreatable(local, domain.Repo{Host: repo.Host, Owner: repo.Owner, Name: value})
		})
		if err != nil {
			return err
		}
		repo.Name = strings.TrimSpace(name)
	}
	if err := a.checkCreatable(local, repo); err != nil {
		return err
	}

	if options.Visibility == "" {
		options.Visibility = "private"
		if a.isTerminal {
			options.Visibility, err = a.selector.Select("Visibility", []SelectionOption{
				{Value: "private", Title: "Private"},
				{Value: "public", Title: "Public"},
				{Value: "internal", Title: "Internal", Description: dim("organizations on GitHub Enterprise")},
			})
			if err != nil {
				return err
			}
		}
	}
	if interactive && options.Description == "" {
		if err := a.selector.Input("Description (optional)", &options.Description, nil); err != nil {
			return err
		}
		options.Description = strings.TrimSpace(options.Description)
	}

	if options.Template != "" {
		if err := a.github.Create(repo, options); err != nil {
			return err
		}
		return a.clone(repo, true, nil)
	}

	destination, err := a.destination(repo)
	if err != nil {
		return err
	}
	root, _ := a.primaryRoot()
	if err := os.MkdirAll(destination, 0o755); err != nil {
		return err
	}
	rollback := func() {
		_ = removeAll(destination)
		removeEmptyParents(filepath.Dir(destination), root)
	}
	if err := a.git(destination, "init"); err != nil {
		rollback()
		return err
	}
	if err := a.git(destination, "commit", "--allow-empty", "--message", "Initial commit"); err != nil {
		rollback()
		return fmt.Errorf("%w (is git user.name/user.email configured?)", err)
	}
	options.SourceDir = destination
	if err := a.github.Create(repo, options); err != nil {
		rollback()
		return err
	}
	fmt.Fprintln(a.stdout, destination)
	return nil
}

func (a *App) checkCreatable(local []LocalRepository, repo domain.Repo) error {
	if found := find(local, repo); len(found) > 0 {
		return fmt.Errorf("%s is already in the workspace: %s", repo, found[0].Path)
	}
	destination, err := a.destination(repo)
	if err != nil {
		return err
	}
	return a.checkDestination(destination, repo)
}

// checkDestination rejects placing repo at a path that is taken or that lies
// inside another working tree.
func (a *App) checkDestination(destination string, repo domain.Repo) error {
	if exists(destination) {
		return fmt.Errorf("%s already exists and does not contain %s", destination, repo)
	}
	root, err := a.primaryRoot()
	if err != nil {
		return err
	}
	if inside := enclosingWorkingTree(destination, root); inside != "" {
		return fmt.Errorf("cannot place %s at %s: it is inside the repository %s", repo, destination, inside)
	}
	return nil
}

// git runs git in dir, sending its output to stderr.
func (a *App) git(dir string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Stdout = a.stderr
	cmd.Stderr = a.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s: %w", args[0], err)
	}
	return nil
}

// gitOutput runs git in dir and returns its trimmed stdout.
func gitOutput(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return strings.TrimSpace(string(out)), err
}
