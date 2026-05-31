package app_test

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/inovue/gh-workspace/internal/app"
)

func TestPathExplicitRepositoryPrintsExistingLocalPath(t *testing.T) {
	home := t.TempDir()
	repoPath := filepath.Join(home, "workspaces", "github.com", "inovue3", "app")
	mkdir(t, filepath.Join(repoPath, ".git"))

	var stdout, stderr bytes.Buffer
	exitCode := app.New(app.Config{
		HomeDir: home,
		Stdout:  &stdout,
		Stderr:  &stderr,
	}).Run([]string{"path", "Inovue3/App"})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	if got := stdout.String(); got != repoPath+"\n" {
		t.Fatalf("stdout = %q, want %q", got, repoPath+"\n")
	}
	if got := stderr.String(); got != "" {
		t.Fatalf("stderr = %q, want empty", got)
	}
}

func TestRootCommandShowsHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	exitCode := app.New(app.Config{Stdout: &stdout, Stderr: &stderr}).Run(nil)

	if exitCode != 0 {
		t.Fatalf("exit code = %d, want 0", exitCode)
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	if got := stderr.String(); !strings.Contains(got, "path") || !strings.Contains(got, "clone") {
		t.Fatalf("stderr = %q, want help with commands", got)
	}
}

func TestPathExplicitGitHubURLIsNormalized(t *testing.T) {
	home := t.TempDir()
	repoPath := filepath.Join(home, "workspaces", "github.com", "inovue3", "app")
	mkdir(t, filepath.Join(repoPath, ".git"))

	var stdout bytes.Buffer
	exitCode := app.New(app.Config{HomeDir: home, Stdout: &stdout}).Run([]string{"path", "https://github.com/Inovue3/App.git"})

	if exitCode != 0 {
		t.Fatalf("exit code = %d", exitCode)
	}
	if got := stdout.String(); got != repoPath+"\n" {
		t.Fatalf("stdout = %q, want %q", got, repoPath+"\n")
	}
}

func TestPathExplicitSSHURLIsNormalized(t *testing.T) {
	home := t.TempDir()
	repoPath := filepath.Join(home, "workspaces", "github.com", "inovue3", "app")
	mkdir(t, filepath.Join(repoPath, ".git"))

	var stdout bytes.Buffer
	exitCode := app.New(app.Config{HomeDir: home, Stdout: &stdout}).Run([]string{"path", "ssh://git@github.com/Inovue3/App.git"})

	if exitCode != 0 {
		t.Fatalf("exit code = %d", exitCode)
	}
	if got := stdout.String(); got != repoPath+"\n" {
		t.Fatalf("stdout = %q, want %q", got, repoPath+"\n")
	}
}

func TestPathRejectsInvalidExplicitRepositoryReference(t *testing.T) {
	home := t.TempDir()
	var stdout, stderr bytes.Buffer
	exitCode := app.New(app.Config{HomeDir: home, Stdout: &stdout, Stderr: &stderr}).Run([]string{"path", "owner/repo.git"})

	if exitCode == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	if got := stderr.String(); got == "" {
		t.Fatal("stderr empty, want error")
	}
}

func TestPathWithoutRepositoryRequiresTTY(t *testing.T) {
	home := t.TempDir()
	var stdout, stderr bytes.Buffer
	exitCode := app.New(app.Config{HomeDir: home, Stdout: &stdout, Stderr: &stderr}).Run([]string{"path"})

	if exitCode == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	if got := stderr.String(); got == "" {
		t.Fatal("stderr empty, want error")
	}
}

func TestPathWithoutRepositorySelectsFromLocalScan(t *testing.T) {
	home := t.TempDir()
	githubPath := filepath.Join(home, "workspaces", "github.com", "inovue3", "app")
	companyPath := filepath.Join(home, "workspaces", "github.company.com", "team", "tool")
	mkdir(t, filepath.Join(githubPath, ".git"))
	mkdir(t, filepath.Join(companyPath, ".git"))
	mkdir(t, filepath.Join(home, "workspaces", "badhost", "team", "ignored", ".git"))
	mkdir(t, filepath.Join(home, "workspaces", "github.company.com", "-", "dash-owner", ".git"))

	selector := &fakeSelector{selected: "github.company.com/team/tool"}
	var stdout bytes.Buffer
	exitCode := app.New(app.Config{
		HomeDir:    home,
		Stdout:     &stdout,
		IsTerminal: true,
		Selector:   selector,
	}).Run([]string{"path"})

	if exitCode != 0 {
		t.Fatalf("exit code = %d", exitCode)
	}
	if got := stdout.String(); got != companyPath+"\n" {
		t.Fatalf("stdout = %q, want %q", got, companyPath+"\n")
	}
	wantOptions := []string{"github.com/inovue3/app", "github.company.com/-/dash-owner", "github.company.com/team/tool"}
	if !reflect.DeepEqual(selector.options, wantOptions) {
		t.Fatalf("options = %#v, want %#v", selector.options, wantOptions)
	}
}

func TestPathCancelLeavesStdoutEmpty(t *testing.T) {
	home := t.TempDir()
	mkdir(t, filepath.Join(home, "workspaces", "github.com", "inovue3", "app", ".git"))

	var stdout, stderr bytes.Buffer
	exitCode := app.New(app.Config{
		HomeDir:    home,
		Stdout:     &stdout,
		Stderr:     &stderr,
		IsTerminal: true,
		Selector:   &fakeSelector{err: errors.New("cancelled")},
	}).Run([]string{"path"})

	if exitCode == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	if got := stderr.String(); got == "" {
		t.Fatal("stderr empty, want cancellation error")
	}
}

func TestCloneExplicitRepositoryUsesExistingGitRepository(t *testing.T) {
	home := t.TempDir()
	repoPath := filepath.Join(home, "workspaces", "github.com", "inovue3", "app")
	mkdir(t, filepath.Join(repoPath, ".git"))
	gh := &fakeGitHub{}
	var stdout bytes.Buffer

	exitCode := app.New(app.Config{HomeDir: home, Stdout: &stdout, GitHub: gh}).Run([]string{"clone", "inovue3/app"})

	if exitCode != 0 {
		t.Fatalf("exit code = %d", exitCode)
	}
	if got := stdout.String(); got != repoPath+"\n" {
		t.Fatalf("stdout = %q, want %q", got, repoPath+"\n")
	}
	if gh.cloneCalled {
		t.Fatal("clone called for existing repository")
	}
}

func TestCloneExplicitRepositoryRunsGhRepoClone(t *testing.T) {
	home := t.TempDir()
	repoPath := filepath.Join(home, "workspaces", "github.com", "inovue3", "app")
	gh := &fakeGitHub{}
	var stdout, stderr bytes.Buffer

	exitCode := app.New(app.Config{HomeDir: home, Stdout: &stdout, Stderr: &stderr, GitHub: gh}).Run([]string{"clone", "Inovue3/App"})

	if exitCode != 0 {
		t.Fatalf("exit code = %d, stderr = %q", exitCode, stderr.String())
	}
	if got := stdout.String(); got != repoPath+"\n" {
		t.Fatalf("stdout = %q, want %q", got, repoPath+"\n")
	}
	if gh.cloneRepo != "inovue3/app" || gh.cloneDest != repoPath {
		t.Fatalf("clone = (%q, %q), want (%q, %q)", gh.cloneRepo, gh.cloneDest, "inovue3/app", repoPath)
	}
	if got := stderr.String(); got != "cloning output\n" {
		t.Fatalf("stderr = %q, want forwarded child output", got)
	}
}

func TestCloneFailureLeavesStdoutEmpty(t *testing.T) {
	home := t.TempDir()
	gh := &fakeGitHub{cloneErr: errors.New("clone failed")}
	var stdout, stderr bytes.Buffer

	exitCode := app.New(app.Config{HomeDir: home, Stdout: &stdout, Stderr: &stderr, GitHub: gh}).Run([]string{"clone", "inovue3/app"})

	if exitCode == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	if got := stderr.String(); got == "" {
		t.Fatal("stderr empty, want error")
	}
	if _, err := os.Stat(filepath.Join(home, "workspaces", "github.com", "inovue3")); err != nil {
		t.Fatalf("destination parent not created: %v", err)
	}
}

func TestCloneWithoutRepositoryListsUnclonedRepositoriesForSelection(t *testing.T) {
	home := t.TempDir()
	existingPath := filepath.Join(home, "workspaces", "github.com", "inovue3", "app")
	mkdir(t, filepath.Join(existingPath, ".git"))

	gh := &fakeGitHub{repos: []app.RemoteRepository{
		{NameWithOwner: "inovue3/app"},
		{NameWithOwner: "octo/tool"},
	}}
	selector := &fakeSelector{selected: "octo/tool"}
	var stdout bytes.Buffer

	exitCode := app.New(app.Config{
		HomeDir:    home,
		Stdout:     &stdout,
		GitHub:     gh,
		IsTerminal: true,
		Selector:   selector,
	}).Run([]string{"clone"})

	wantPath := filepath.Join(home, "workspaces", "github.com", "octo", "tool")
	if exitCode != 0 {
		t.Fatalf("exit code = %d", exitCode)
	}
	if got := stdout.String(); got != wantPath+"\n" {
		t.Fatalf("stdout = %q, want %q", got, wantPath+"\n")
	}
	wantOptions := []string{"__SWITCH_ORG__", "octo/tool"}
	if !reflect.DeepEqual(selector.options, wantOptions) {
		t.Fatalf("options = %#v, want %#v", selector.options, wantOptions)
	}
}

func TestCloneWithoutRepositoryRequiresTTY(t *testing.T) {
	home := t.TempDir()
	var stdout, stderr bytes.Buffer
	exitCode := app.New(app.Config{
		HomeDir: home,
		Stdout:  &stdout,
		Stderr:  &stderr,
		GitHub:  &fakeGitHub{},
	}).Run([]string{"clone"})

	if exitCode == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	if got := stderr.String(); got == "" {
		t.Fatal("stderr empty, want error")
	}
}

func TestCloneFailsWhenDestinationExistsButIsNotGitRepository(t *testing.T) {
	home := t.TempDir()
	repoPath := filepath.Join(home, "workspaces", "github.com", "inovue3", "app")
	mkdir(t, repoPath)

	var stdout, stderr bytes.Buffer
	exitCode := app.New(app.Config{HomeDir: home, Stdout: &stdout, Stderr: &stderr, GitHub: &fakeGitHub{}}).Run([]string{"clone", "inovue3/app"})

	if exitCode == 0 {
		t.Fatal("exit code = 0, want non-zero")
	}
	if got := stdout.String(); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	if got := stderr.String(); got == "" {
		t.Fatal("stderr empty, want error")
	}
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

type fakeSelector struct {
	selected     string
	selectedList []string
	selectCount  int
	err          error
	options      []string
	optionsList  [][]string
}

func (f *fakeSelector) Select(_ string, options []app.SelectionOption) (string, error) {
	var opts []string
	for _, option := range options {
		opts = append(opts, option.Value)
	}
	f.optionsList = append(f.optionsList, opts)
	f.options = opts
	if f.err != nil {
		return "", f.err
	}
	if len(f.selectedList) > 0 {
		if f.selectCount < len(f.selectedList) {
			res := f.selectedList[f.selectCount]
			f.selectCount++
			return res, nil
		}
	}
	if f.selected == "" {
		return "", errors.New("cancelled")
	}
	return f.selected, nil
}

type fakeGitHub struct {
	repos       []app.RemoteRepository
	cloneErr    error
	cloneCalled bool
	cloneRepo   string
	cloneDest   string
	listOwner   string
	username    string
	orgs        []string
	orgsErr     error
}

func (f *fakeGitHub) ListRepositories(owner string) ([]app.RemoteRepository, error) {
	f.listOwner = owner
	return f.repos, nil
}

func (f *fakeGitHub) ListOrganizations() (string, []string, error) {
	username := f.username
	if username == "" {
		username = "Personal"
	}
	return username, f.orgs, f.orgsErr
}

func (f *fakeGitHub) Clone(nameWithOwner, destination string, stderr io.Writer) error {
	f.cloneCalled = true
	f.cloneRepo = nameWithOwner
	f.cloneDest = destination
	_, _ = io.WriteString(stderr, "cloning output\n")
	return f.cloneErr
}

func TestCloneWithOwnerArgumentListsOwnerRepositories(t *testing.T) {
	home := t.TempDir()
	gh := &fakeGitHub{repos: []app.RemoteRepository{
		{NameWithOwner: "my-org/tool"},
	}}
	selector := &fakeSelector{selected: "my-org/tool"}
	var stdout bytes.Buffer

	exitCode := app.New(app.Config{
		HomeDir:    home,
		Stdout:     &stdout,
		GitHub:     gh,
		IsTerminal: true,
		Selector:   selector,
	}).Run([]string{"clone", "my-org"})

	wantPath := filepath.Join(home, "workspaces", "github.com", "my-org", "tool")
	if exitCode != 0 {
		t.Fatalf("exit code = %d", exitCode)
	}
	if got := stdout.String(); got != wantPath+"\n" {
		t.Fatalf("stdout = %q, want %q", got, wantPath+"\n")
	}
	if gh.listOwner != "my-org" {
		t.Fatalf("listOwner = %q, want %q", gh.listOwner, "my-org")
	}
	wantOptions := []string{"__SWITCH_ORG__", "my-org/tool"}
	if !reflect.DeepEqual(selector.options, wantOptions) {
		t.Fatalf("options = %#v, want %#v", selector.options, wantOptions)
	}
}

func TestCloneSelectsOrganizationInteractively(t *testing.T) {
	home := t.TempDir()
	gh := &fakeGitHub{
		orgs: []string{"org-a", "org-b"},
		repos: []app.RemoteRepository{
			{NameWithOwner: "org-b/tool"},
		},
	}
	selector := &fakeSelector{
		selectedList: []string{"__SWITCH_ORG__", "org-b", "org-b/tool"},
	}
	var stdout bytes.Buffer

	exitCode := app.New(app.Config{
		HomeDir:    home,
		Stdout:     &stdout,
		GitHub:     gh,
		IsTerminal: true,
		Selector:   selector,
	}).Run([]string{"clone"})

	wantPath := filepath.Join(home, "workspaces", "github.com", "org-b", "tool")
	if exitCode != 0 {
		t.Fatalf("exit code = %d", exitCode)
	}
	if got := stdout.String(); got != wantPath+"\n" {
		t.Fatalf("stdout = %q, want %q", got, wantPath+"\n")
	}
	if gh.listOwner != "org-b" {
		t.Fatalf("listOwner = %q, want %q", gh.listOwner, "org-b")
	}
	if len(selector.optionsList) != 3 {
		t.Fatalf("optionsList len = %d, want 3", len(selector.optionsList))
	}
	wantFirstList := []string{"__SWITCH_ORG__", "org-b/tool"}
	if !reflect.DeepEqual(selector.optionsList[0], wantFirstList) {
		t.Fatalf("optionsList[0] = %#v, want %#v", selector.optionsList[0], wantFirstList)
	}
	wantSecondList := []string{"", "org-a", "org-b"}
	if !reflect.DeepEqual(selector.optionsList[1], wantSecondList) {
		t.Fatalf("optionsList[1] = %#v, want %#v", selector.optionsList[1], wantSecondList)
	}
	wantThirdList := []string{"__SWITCH_ORG__", "org-b/tool"}
	if !reflect.DeepEqual(selector.optionsList[2], wantThirdList) {
		t.Fatalf("optionsList[2] = %#v, want %#v", selector.optionsList[2], wantThirdList)
	}
}
