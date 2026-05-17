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
	if !reflect.DeepEqual(selector.options, []string{"octo/tool"}) {
		t.Fatalf("options = %#v", selector.options)
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
	selected string
	err      error
	options  []string
}

func (f *fakeSelector) Select(_ string, options []app.SelectionOption) (string, error) {
	f.options = nil
	for _, option := range options {
		f.options = append(f.options, option.Value)
	}
	if f.err != nil {
		return "", f.err
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
}

func (f *fakeGitHub) ListRepositories() ([]app.RemoteRepository, error) {
	return f.repos, nil
}

func (f *fakeGitHub) Clone(nameWithOwner, destination string, stderr io.Writer) error {
	f.cloneCalled = true
	f.cloneRepo = nameWithOwner
	f.cloneDest = destination
	_, _ = io.WriteString(stderr, "cloning output\n")
	return f.cloneErr
}
