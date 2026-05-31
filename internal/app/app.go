package app

import (
	"fmt"
	"io"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
	"github.com/inovue/gh-workspace/internal/domain"
	"github.com/muesli/termenv"
	"github.com/spf13/cobra"
)

type Config struct {
	HomeDir    string
	Stdout     io.Writer
	Stderr     io.Writer
	IsTerminal bool
	Selector   Selector
	GitHub     GitHubCLI
}

type GitHubCLI interface {
	ListRepositories(owner string) ([]RemoteRepository, error)
	ListOrganizations() (string, []string, error)
	Clone(nameWithOwner, destination string, stderr io.Writer) error
	CurrentUsername() (string, error)
}

type Selector interface {
	Select(title string, options []SelectionOption) (string, error)
}

type RemoteRepository struct {
	NameWithOwner string `json:"nameWithOwner"`
	Description   string `json:"description"`
}

type SelectionOption struct {
	Value       string
	Title       string
	Description string
	Disabled    bool
}

type App struct {
	homeDir    string
	homeErr    error
	stdout     io.Writer
	stderr     io.Writer
	isTerminal bool
	selector   Selector
	github     GitHubCLI
}

func New(config Config) *App {
	stdout := config.Stdout
	if stdout == nil {
		stdout = os.Stdout
	}
	stderr := config.Stderr
	if stderr == nil {
		stderr = os.Stderr
	}
	home := config.HomeDir
	var homeErr error
	if home == "" {
		home, homeErr = homeDir()
	}
	selector := config.Selector
	if selector == nil {
		selector = huhSelector{output: stderr}
	}
	github := config.GitHub
	if github == nil {
		github = ghCLI{stderr: stderr}
	}
	if config.IsTerminal {
		profile := termenv.NewOutput(stderr).ColorProfile()
		lipgloss.SetColorProfile(profile)
	}
	return &App{
		homeDir:    home,
		homeErr:    homeErr,
		stdout:     stdout,
		stderr:     stderr,
		isTerminal: config.IsTerminal,
		selector:   selector,
		github:     github,
	}
}

func (a *App) Run(args []string) int {
	cmd := a.command()
	cmd.SetArgs(args)
	cmd.SetOut(a.stderr)
	cmd.SetErr(a.stderr)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintln(a.stderr, err)
		return 1
	}
	return 0
}

func (a *App) command() *cobra.Command {
	root := &cobra.Command{
		Use:           "workspace",
		Short:         "Resolve and clone repositories in a local workspace",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	pathCmd := &cobra.Command{
		Use:   "path [repository]",
		Short: "Print a local repository path",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.runPath(args)
		},
	}
	cloneCmd := &cobra.Command{
		Use:   "clone [repository|owner]",
		Short: "Clone a repository and print its path",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			return a.runClone(args)
		},
	}
	monkeyCmd := &cobra.Command{
		Use:    "monkey",
		Short:  "Run an animated monkey test of the selection UI",
		Hidden: true,
		RunE: func(_ *cobra.Command, _ []string) error {
			return a.runMonkey()
		},
	}
	root.AddCommand(pathCmd, cloneCmd, monkeyCmd)
	return root
}

func (a *App) runMonkey() error {
	if !a.isTerminal {
		return fmt.Errorf("monkey test requires a TTY")
	}

	var options []SelectionOption
	for i := 1; i <= 15; i++ {
		isCloned := i%3 == 0
		title := fmt.Sprintf("monkey-org/repo-%02d", i)
		desc := filepath.Join(a.repoRoot(), "github.com", "monkey-org", fmt.Sprintf("repo-%02d", i))
		if isCloned {
			title = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title + " (already cloned)")
			desc = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(desc)
		}
		options = append(options, SelectionOption{
			Value:       fmt.Sprintf("monkey-org/repo-%02d", i),
			Title:       title,
			Description: desc,
			Disabled:    isCloned,
		})
	}

	selector := huhSelector{output: a.stderr, isMonkey: true}
	selected, err := selector.Select("Monkey Test Select", options)
	if err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, "Selected:", selected)
	return nil
}

func (a *App) runPath(args []string) error {
	if err := a.checkHome(); err != nil {
		return err
	}
	if len(args) == 0 {
		repos, err := a.scanLocalRepositories()
		if err != nil {
			return err
		}
		selected, err := a.selectLocalRepository("Select repository", repos)
		if err != nil {
			return err
		}
		fmt.Fprintln(a.stdout, selected.Path)
		return nil
	}

	ref, err := domain.ParseGitHubReference(args[0])
	if err != nil {
		return err
	}

	path := a.destinationFor(ref)
	if !isGitWorkingTree(path) {
		return fmt.Errorf("repository not cloned: %s", ref.String())
	}

	fmt.Fprintln(a.stdout, path)
	return nil
}

func (a *App) runClone(args []string) error {
	if err := a.checkHome(); err != nil {
		return err
	}
	if len(args) == 0 {
		ref, err := a.selectRemoteRepository("")
		if err != nil {
			return err
		}
		return a.clone(ref)
	}

	input := args[0]
	if !strings.Contains(input, "/") && !strings.Contains(input, ":") {
		ref, err := a.selectRemoteRepository(input)
		if err != nil {
			return err
		}
		return a.clone(ref)
	}

	ref, err := domain.ParseGitHubReference(input)
	if err != nil {
		return err
	}
	return a.clone(ref)
}

func (a *App) clone(ref domain.RepositoryRef) error {
	destination := a.destinationFor(ref)
	if exists(destination) {
		if !isGitWorkingTree(destination) {
			return fmt.Errorf("destination exists but is not a git repository: %s", destination)
		}
		fmt.Fprintln(a.stdout, destination)
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	if err := a.github.Clone(ref.String(), destination, a.stderr); err != nil {
		return err
	}
	fmt.Fprintln(a.stdout, destination)
	return nil
}

func (a *App) selectRemoteRepository(org string) (domain.RepositoryRef, error) {
	if !a.isTerminal {
		return domain.RepositoryRef{}, fmt.Errorf("clone selection requires a TTY")
	}
	remoteRepos, err := a.github.ListRepositories(org)
	if err != nil {
		return domain.RepositoryRef{}, err
	}
	localRepos, err := a.scanLocalRepositories()
	if err != nil {
		return domain.RepositoryRef{}, err
	}
	cloned := map[string]bool{}
	for _, repo := range localRepos {
		if repo.Host == "github.com" {
			cloned[repo.Owner+"/"+repo.Name] = true
		}
	}
	var options []SelectionOption
	currentOwner := org
	if currentOwner == "" {
		currentOwner = "Personal"
		if len(remoteRepos) > 0 {
			parts := strings.Split(remoteRepos[0].NameWithOwner, "/")
			if len(parts) > 0 {
				currentOwner = parts[0]
			}
		}
	}
	options = append(options, SelectionOption{
		Value:       "__SWITCH_ORG__",
		Title:       fmt.Sprintf("🔄 Switch Owner... [%s]", currentOwner),
		Description: "",
	})
	for _, remote := range remoteRepos {
		ref, err := domain.ParseGitHubReference(remote.NameWithOwner)
		if err != nil {
			continue
		}
		isCloned := cloned[ref.String()]
		desc := remote.Description
		if desc != "" {
			desc = truncateString(desc, 40)
		}
		title := ref.Name
		if isCloned {
			title = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(title + " (already cloned)")
			if desc != "" {
				desc = lipgloss.NewStyle().Foreground(lipgloss.Color("244")).Render(desc)
			}
		}
		options = append(options, SelectionOption{
			Value:       ref.String(),
			Title:       title,
			Description: desc,
			Disabled:    isCloned,
		})
	}
	if len(options) == 0 {
		return domain.RepositoryRef{}, fmt.Errorf("no remote repositories available to clone")
	}
	selected, err := a.selector.Select("Clone repository", options)
	if err != nil {
		return domain.RepositoryRef{}, err
	}
	if selected == "__SWITCH_ORG__" {
		selectedOrg, err := a.selectOrganization()
		if err != nil {
			return domain.RepositoryRef{}, err
		}
		return a.selectRemoteRepository(selectedOrg)
	}
	return domain.ParseGitHubReference(selected)
}

func (a *App) selectOrganization() (string, error) {
	if !a.isTerminal {
		return "", fmt.Errorf("organization selection requires a TTY")
	}
	username, orgs, err := a.github.ListOrganizations()
	if err != nil {
		return "", err
	}
	var options []SelectionOption
	options = append(options, SelectionOption{
		Value: "",
		Title: fmt.Sprintf("👤 %s", username),
	})
	for _, o := range orgs {
		options = append(options, SelectionOption{
			Value: o,
			Title: fmt.Sprintf("🏢 %s", o),
		})
	}
	return a.selector.Select("Select Owner", options)
}

func (a *App) selectLocalRepository(title string, repos []LocalRepository) (LocalRepository, error) {
	if !a.isTerminal {
		return LocalRepository{}, fmt.Errorf("path selection requires a TTY")
	}
	username, _ := a.github.CurrentUsername()
	var options []SelectionOption
	byValue := map[string]LocalRepository{}
	for _, repo := range repos {
		value := repo.Host + "/" + repo.Owner + "/" + repo.Name
		byValue[value] = repo

		emoji := "🏢"
		if username != "" && strings.EqualFold(repo.Owner, username) {
			emoji = "👤"
		}

		desc := readGitDescription(repo.Path)

		options = append(options, SelectionOption{
			Value:       value,
			Title:       fmt.Sprintf("%s %s/%s", emoji, repo.Owner, repo.Name),
			Description: desc,
		})
	}
	selected, err := a.selector.Select(title, options)
	if err != nil {
		return LocalRepository{}, err
	}
	repo, ok := byValue[selected]
	if !ok {
		return LocalRepository{}, fmt.Errorf("selected repository not found: %s", selected)
	}
	return repo, nil
}

type LocalRepository struct {
	Host  string
	Owner string
	Name  string
	Path  string
}

func (a *App) scanLocalRepositories() ([]LocalRepository, error) {
	matches, err := filepath.Glob(filepath.Join(a.repoRoot(), "*", "*", "*"))
	if err != nil {
		return nil, err
	}
	var repos []LocalRepository
	for _, path := range matches {
		if !isGitWorkingTree(path) {
			continue
		}
		name := filepath.Base(path)
		owner := filepath.Base(filepath.Dir(path))
		host := filepath.Base(filepath.Dir(filepath.Dir(path)))
		if !domain.ValidLocalLayout(host, owner, name) {
			continue
		}
		repos = append(repos, LocalRepository{
			Host:  strings.ToLower(host),
			Owner: strings.ToLower(owner),
			Name:  strings.ToLower(name),
			Path:  path,
		})
	}
	sort.Slice(repos, func(i, j int) bool {
		aValue := repos[i].Host + "/" + repos[i].Owner + "/" + repos[i].Name
		bValue := repos[j].Host + "/" + repos[j].Owner + "/" + repos[j].Name
		return aValue < bValue
	})
	return repos, nil
}

func (a *App) destinationFor(ref domain.RepositoryRef) string {
	return filepath.Join(a.repoRoot(), "github.com", ref.Owner, ref.Name)
}

func (a *App) checkHome() error {
	return a.homeErr
}

func (a *App) repoRoot() string {
	return filepath.Join(a.homeDir, "workspaces")
}

func isGitWorkingTree(path string) bool {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func homeDir() (string, error) {
	dir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(dir) == "" {
		return "", fmt.Errorf("home directory could not be determined")
	}
	return dir, nil
}

type huhSelector struct {
	output   io.Writer
	isMonkey bool
}

func (s huhSelector) Select(title string, options []SelectionOption) (string, error) {
	huhOptions := make([]huh.Option[string], 0, len(options))
	for _, option := range options {
		title := option.Title
		if option.Description != "" {
			title += "  " + option.Description
		}
		huhOptions = append(huhOptions, huh.NewOption(title, option.Value))
	}
	var selected string
	for _, option := range options {
		if !option.Disabled {
			selected = option.Value
			break
		}
	}

	disabledValues := make(map[string]bool)
	for _, option := range options {
		if option.Disabled {
			disabledValues[option.Value] = true
		}
	}

	selectField := huh.NewSelect[string]().
		Title(title).
		Options(huhOptions...).
		Filtering(true).
		Value(&selected).
		Validate(func(val string) error {
			if disabledValues[val] {
				return fmt.Errorf("repository is already cloned")
			}
			return nil
		})

	if len(options) > 10 {
		selectField.Height(10)
	}

	wrappedSelect := &skippingSelect{
		Select:   selectField,
		disabled: disabledValues,
		options:  options,
		isMonkey: s.isMonkey,
		rng:      rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	err := huh.NewForm(huh.NewGroup(wrappedSelect)).
		WithOutput(s.output).
		WithProgramOptions(
			tea.WithMouseCellMotion(),
			tea.WithFilter(func(m tea.Model, msg tea.Msg) tea.Msg {
				if mouseMsg, ok := msg.(tea.MouseMsg); ok {
					if mouseMsg.Button == tea.MouseButtonWheelUp {
						return tea.KeyMsg{Type: tea.KeyUp}
					} else if mouseMsg.Button == tea.MouseButtonWheelDown {
						return tea.KeyMsg{Type: tea.KeyDown}
					}
				}
				return msg
			}),
		).
		Run()
	return selected, err
}

type skippingSelect struct {
	*huh.Select[string]
	disabled    map[string]bool
	options     []SelectionOption
	isMonkey    bool
	monkeySteps int
	rng         *rand.Rand
}

type monkeyMsg struct{}

func monkeyTick() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return monkeyMsg{}
	})
}

func (s *skippingSelect) Init() tea.Cmd {
	if s.isMonkey {
		return monkeyTick()
	}
	return s.Select.Init()
}

func (s *skippingSelect) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if _, ok := msg.(monkeyMsg); ok {
		randomKey := s.generateRandomKey()
		m, cmd := s.Update(randomKey)
		if s.monkeySteps > 50 {
			return m, cmd
		}
		return m, tea.Batch(cmd, monkeyTick())
	}

	hasEnabled := false
	for _, opt := range s.options {
		if !s.disabled[opt.Value] {
			hasEnabled = true
			break
		}
	}

	newSelectModel, cmd := s.Select.Update(msg)
	s.Select = newSelectModel.(*huh.Select[string])

	isNavigation := false
	if keyMsg, ok := msg.(tea.KeyMsg); ok {
		switch keyMsg.Type {
		case tea.KeyUp, tea.KeyDown, tea.KeyLeft, tea.KeyRight, tea.KeyCtrlN, tea.KeyCtrlP:
			isNavigation = true
		case tea.KeyRunes:
			runes := string(keyMsg.Runes)
			if (runes == "j" || runes == "k" || runes == "h" || runes == "l") && !s.GetFiltering() {
				isNavigation = true
			}
		}
	}

	if hasEnabled && isNavigation {
		for i := 0; i < len(s.options); i++ {
			hovered, ok := s.Hovered()
			if !ok || !s.disabled[hovered] {
				break
			}
			newSelectModel, _ = s.Select.Update(msg)
			s.Select = newSelectModel.(*huh.Select[string])
		}
	}

	return s, cmd
}

func (s *skippingSelect) generateRandomKey() tea.Msg {
	s.monkeySteps++
	if s.monkeySteps > 50 {
		return tea.KeyMsg{Type: tea.KeyEnter}
	}

	r := s.rng.Float64()
	if r < 0.05 {
		runes := []rune("abcdefghijklmnopqrstuvwxyz")
		char := runes[s.rng.Intn(len(runes))]
		return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
	} else if r < 0.10 {
		return tea.KeyMsg{Type: tea.KeyBackspace}
	} else if r < 0.70 {
		return tea.KeyMsg{Type: tea.KeyDown}
	} else if r < 0.90 {
		return tea.KeyMsg{Type: tea.KeyUp}
	} else {
		if s.rng.Float64() < 0.5 {
			return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}}
		}
		return tea.KeyMsg{Type: tea.KeyEsc}
	}
}

func (s *skippingSelect) WithTheme(theme *huh.Theme) huh.Field {
	s.Select = s.Select.WithTheme(theme).(*huh.Select[string])
	return s
}

func (s *skippingSelect) WithKeyMap(keymap *huh.KeyMap) huh.Field {
	s.Select = s.Select.WithKeyMap(keymap).(*huh.Select[string])
	return s
}

func (s *skippingSelect) WithWidth(width int) huh.Field {
	s.Select = s.Select.WithWidth(width).(*huh.Select[string])
	return s
}

func (s *skippingSelect) WithHeight(height int) huh.Field {
	s.Select = s.Select.WithHeight(height).(*huh.Select[string])
	return s
}

func (s *skippingSelect) WithPosition(position huh.FieldPosition) huh.Field {
	s.Select = s.Select.WithPosition(position).(*huh.Select[string])
	return s
}

func (s *skippingSelect) WithAccessible(accessible bool) huh.Field {
	s.Select = s.Select.WithAccessible(accessible).(*huh.Select[string])
	return s
}

type ghCLI struct {
	stderr io.Writer
}

func (g ghCLI) ListRepositories(owner string) ([]RemoteRepository, error) {
	var args []string
	args = append(args, "repo", "list")
	if owner != "" {
		args = append(args, owner)
	}
	args = append(args, "--limit", "1000", "--json", "nameWithOwner,description")
	cmd := exec.Command("gh", args...)
	cmd.Stderr = g.stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	var repos []RemoteRepository
	if err := domain.DecodeJSON(out, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

func (g ghCLI) ListOrganizations() (string, []string, error) {
	cmd := exec.Command("gh", "api", "graphql", "-f", "query=query { viewer { login organizations(first: 100) { nodes { login } } } }")
	cmd.Stderr = g.stderr
	out, err := cmd.Output()
	if err != nil {
		return "", nil, err
	}
	var res struct {
		Data struct {
			Viewer struct {
				Login         string `json:"login"`
				Organizations struct {
					Nodes []struct {
						Login string `json:"login"`
					} `json:"nodes"`
				} `json:"organizations"`
			} `json:"viewer"`
		} `json:"data"`
	}
	if err := domain.DecodeJSON(out, &res); err != nil {
		return "", nil, err
	}
	username := res.Data.Viewer.Login
	var names []string
	for _, o := range res.Data.Viewer.Organizations.Nodes {
		names = append(names, o.Login)
	}
	return username, names, nil
}

func (g ghCLI) Clone(nameWithOwner, destination string, stderr io.Writer) error {
	cmd := exec.Command("gh", "repo", "clone", nameWithOwner, destination)
	cmd.Stdout = stderr
	cmd.Stderr = stderr
	return cmd.Run()
}

func (g ghCLI) CurrentUsername() (string, error) {
	cmd := exec.Command("gh", "config", "get", "-h", "github.com", "user")
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func truncateString(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func readGitDescription(repoPath string) string {
	gitPath := filepath.Join(repoPath, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return ""
	}

	var descPath string
	if info.IsDir() {
		descPath = filepath.Join(gitPath, "description")
	} else {
		content, err := os.ReadFile(gitPath)
		if err == nil {
			line := strings.TrimSpace(string(content))
			if strings.HasPrefix(line, "gitdir: ") {
				realGitDir := strings.TrimPrefix(line, "gitdir: ")
				if !filepath.IsAbs(realGitDir) {
					realGitDir = filepath.Join(repoPath, realGitDir)
				}
				descPath = filepath.Join(realGitDir, "description")
			}
		}
	}
	if descPath == "" {
		return ""
	}
	data, err := os.ReadFile(descPath)
	if err != nil {
		return ""
	}
	s := strings.TrimSpace(string(data))
	if s != "" && !strings.HasPrefix(s, "Unnamed repository") {
		return s
	}
	return ""
}
