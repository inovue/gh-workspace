package app

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/inovue/gh-workspace/internal/domain"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

// RootEnv lists workspace roots, separated like PATH. The first root receives
// new clones.
const RootEnv = "GH_WORKSPACE_ROOT"

// RootGitConfigKey is the git config key read when RootEnv is unset. It may be
// repeated to configure several roots.
const RootGitConfigKey = "gh-workspace.root"

type Config struct {
	HomeDir    string
	Roots      []string
	Getenv     func(string) string
	Stdout     io.Writer
	Stderr     io.Writer
	IsTerminal bool
	Selector   Selector
	GitHub     GitHubCLI
	Version    string
}

type App struct {
	config      Config
	stdout      io.Writer
	stderr      io.Writer
	isTerminal  bool
	selector    Selector
	github      GitHubCLI
	defaultHost string
	roots       []string
	rootsErr    error
}

func New(config Config) *App {
	if config.Stdout == nil {
		config.Stdout = os.Stdout
	}
	if config.Stderr == nil {
		config.Stderr = os.Stderr
	}
	if config.Getenv == nil {
		config.Getenv = os.Getenv
	}
	if config.Selector == nil {
		config.Selector = huhSelector{output: config.Stderr}
	}
	if config.GitHub == nil {
		config.GitHub = ghCLI{stderr: config.Stderr}
	}
	if config.IsTerminal {
		lipgloss.SetColorProfile(termenv.NewOutput(config.Stderr).ColorProfile())
	}
	defaultHost := strings.ToLower(strings.TrimSpace(config.Getenv("GH_HOST")))
	if defaultHost == "" {
		defaultHost = domain.DefaultHost
	}
	return &App{
		config:      config,
		stdout:      config.Stdout,
		stderr:      config.Stderr,
		isTerminal:  config.IsTerminal,
		selector:    config.Selector,
		github:      config.GitHub,
		defaultHost: defaultHost,
	}
}

func (a *App) Run(args []string) int {
	cmd := a.command()
	cmd.SetArgs(args)
	cmd.SetOut(a.stderr)
	cmd.SetErr(a.stderr)
	if err := cmd.Execute(); err != nil {
		if !errors.Is(err, errCancelled) {
			fmt.Fprintln(a.stderr, "error:", err)
		}
		return 1
	}
	return 0
}

var errCancelled = errors.New("cancelled")

func (a *App) command() *cobra.Command {
	version := a.config.Version
	if version == "" {
		version = "dev"
	}
	root := &cobra.Command{
		Use:   "workspace",
		Short: "Keep every repository in one predictable place",
		Long: `gh workspace clones, finds, creates, and deletes repositories under a
workspace root (default ~/workspaces). GitHub.com repositories live at
{root}/{owner}/{repo}; other hosts at {root}/{host}/{owner}/{repo}.

Commands that resolve a repository print its absolute path on stdout, and
everything else goes to stderr, so they compose with cd, editors, and scripts.
Run "gh workspace shell-init --help" to set up a cd shortcut.`,
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.SetVersionTemplate("gh-workspace {{.Version}}\n")
	root.CompletionOptions.HiddenDefaultCmd = true
	root.AddCommand(
		a.pathCommand(),
		a.listCommand(),
		a.cloneCommand(),
		a.createCommand(),
		a.deleteCommand(),
		a.rootCommand(),
		a.migrateCommand(),
		a.shellInitCommand(),
	)
	return root
}

// Roots returns the configured workspace roots. The first root is primary.
func (a *App) Roots() ([]string, error) {
	if a.roots != nil || a.rootsErr != nil {
		return a.roots, a.rootsErr
	}
	a.roots, a.rootsErr = a.resolveRoots()
	return a.roots, a.rootsErr
}

func (a *App) resolveRoots() ([]string, error) {
	var raw []string
	switch {
	case len(a.config.Roots) > 0:
		raw = a.config.Roots
	case strings.TrimSpace(a.config.Getenv(RootEnv)) != "":
		raw = filepath.SplitList(a.config.Getenv(RootEnv))
	default:
		raw = gitConfigAll(RootGitConfigKey)
	}

	home := a.config.HomeDir
	if home == "" {
		if dir, err := os.UserHomeDir(); err == nil {
			home = dir
		}
	}
	var roots []string
	seen := map[string]bool{}
	for _, root := range raw {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if root == "~" || strings.HasPrefix(root, "~/") || strings.HasPrefix(root, `~\`) {
			if home == "" {
				return nil, fmt.Errorf("cannot expand %s: home directory is unknown", root)
			}
			root = filepath.Join(home, root[1:])
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			return nil, err
		}
		if !seen[abs] {
			seen[abs] = true
			roots = append(roots, abs)
		}
	}
	if len(roots) > 0 {
		return roots, nil
	}
	if home == "" {
		return nil, fmt.Errorf("home directory could not be determined; set %s", RootEnv)
	}
	return []string{filepath.Join(home, "workspaces")}, nil
}

func gitConfigAll(key string) []string {
	out, err := exec.Command("git", "config", "--get-all", key).Output()
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSpace(string(out)), "\n")
}

func (a *App) primaryRoot() (string, error) {
	roots, err := a.Roots()
	if err != nil {
		return "", err
	}
	return roots[0], nil
}

// destination returns where a new repository with this identity is placed.
func (a *App) destination(repo domain.Repo) (string, error) {
	root, err := a.primaryRoot()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, domain.RelPath(repo)), nil
}

func (a *App) scan() ([]LocalRepository, error) {
	roots, err := a.Roots()
	if err != nil {
		return nil, err
	}
	return scan(roots), nil
}

// parseReference parses an explicit reference, or reports false when the
// input is a free-form query.
func (a *App) parseReference(input string) (domain.Repo, bool) {
	if !strings.ContainsAny(input, "/:") {
		return domain.Repo{}, false
	}
	repo, err := domain.ParseReference(input, a.defaultHost)
	return repo, err == nil
}

// resolveLocal finds one local repository. Explicit references must match
// exactly; other queries match fuzzily and prompt when ambiguous.
func (a *App) resolveLocal(query, title string) (LocalRepository, error) {
	repos, err := a.scan()
	if err != nil {
		return LocalRepository{}, err
	}
	if strings.TrimSpace(query) == "" {
		if len(repos) == 0 {
			return LocalRepository{}, a.emptyWorkspaceError()
		}
		return a.selectLocal(title, repos)
	}

	if ref, ok := a.parseReference(query); ok {
		if found := find(repos, ref); len(found) > 0 {
			return found[0], nil
		}
		for _, repo := range repos {
			if strings.EqualFold(repo.Display(), query) {
				return repo, nil
			}
		}
		return LocalRepository{}, fmt.Errorf("%s is not in the workspace; run: gh workspace clone %s", ref, ref)
	}

	matches := match(repos, query)
	switch {
	case len(matches) == 1:
		return matches[0], nil
	case len(matches) == 0:
		return LocalRepository{}, fmt.Errorf("no repository in the workspace matches %q", query)
	case a.isTerminal:
		return a.selectLocal(title, matches)
	default:
		var names []string
		for _, repo := range matches {
			names = append(names, "  "+repo.Display())
		}
		return LocalRepository{}, fmt.Errorf("%q matches several repositories:\n%s", query, strings.Join(names, "\n"))
	}
}

func (a *App) emptyWorkspaceError() error {
	roots, _ := a.Roots()
	return fmt.Errorf("no repositories found under %s; run: gh workspace clone", strings.Join(roots, ", "))
}

func (a *App) requireTTY(what string) error {
	if a.isTerminal {
		return nil
	}
	return fmt.Errorf("%s needs an interactive terminal", what)
}

func (a *App) selectLocal(title string, repos []LocalRepository) (LocalRepository, error) {
	if err := a.requireTTY("repository selection"); err != nil {
		return LocalRepository{}, fmt.Errorf("%w; pass a repository argument", err)
	}
	options := make([]SelectionOption, 0, len(repos))
	for i, repo := range repos {
		var notes []string
		if !repo.Repo.IsZero() && !strings.EqualFold(repo.Repo.String(), repo.Display()) {
			notes = append(notes, "→ "+repo.Repo.String())
		}
		if repo.Worktree {
			notes = append(notes, "worktree")
		}
		if len(a.mustRoots()) > 1 {
			notes = append(notes, repo.Root)
		}
		options = append(options, SelectionOption{
			Value:       fmt.Sprint(i),
			Title:       repo.Display(),
			Description: dim(strings.Join(notes, " · ")),
		})
	}
	selected, err := a.selector.Select(title, options)
	if err != nil {
		return LocalRepository{}, err
	}
	for i, repo := range repos {
		if fmt.Sprint(i) == selected {
			return repo, nil
		}
	}
	return LocalRepository{}, fmt.Errorf("selected repository not found: %s", selected)
}

func (a *App) mustRoots() []string {
	roots, _ := a.Roots()
	return roots
}

func dim(s string) string {
	if s == "" {
		return ""
	}
	return lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(s)
}
