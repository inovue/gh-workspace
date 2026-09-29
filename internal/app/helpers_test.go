package app_test

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/inovue/gh-workspace/internal/app"
	"github.com/inovue/gh-workspace/internal/domain"
)

type harness struct {
	t        *testing.T
	root     string
	stdout   bytes.Buffer
	stderr   bytes.Buffer
	github   *fakeGitHub
	selector *fakeSelector
	tty      bool
	env      map[string]string
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	isolateGit(t)
	return &harness{
		t:        t,
		root:     filepath.Join(t.TempDir(), "workspaces"),
		github:   &fakeGitHub{login: "me"},
		selector: &fakeSelector{},
		env:      map[string]string{},
	}
}

func (h *harness) run(args ...string) int {
	h.t.Helper()
	h.stdout.Reset()
	h.stderr.Reset()
	return app.New(app.Config{
		HomeDir:    filepath.Dir(h.root),
		Roots:      []string{h.root},
		Getenv:     func(key string) string { return h.env[key] },
		Stdout:     &h.stdout,
		Stderr:     &h.stderr,
		IsTerminal: h.tty,
		Selector:   h.selector,
		GitHub:     h.github,
		Version:    "v9.9.9",
	}).Run(args)
}

func (h *harness) mustRun(args ...string) string {
	h.t.Helper()
	if code := h.run(args...); code != 0 {
		h.t.Fatalf("%v: exit %d\nstderr: %s", args, code, h.stderr.String())
	}
	return h.stdout.String()
}

func (h *harness) mustFail(args ...string) string {
	h.t.Helper()
	if code := h.run(args...); code == 0 {
		h.t.Fatalf("%v: exit 0, want failure\nstdout: %s", args, h.stdout.String())
	}
	if h.stdout.Len() != 0 {
		h.t.Fatalf("%v: stdout = %q, want empty on failure", args, h.stdout.String())
	}
	return h.stderr.String()
}

func (h *harness) path(rel string) string {
	return filepath.Join(h.root, filepath.FromSlash(rel))
}

// isolateGit keeps tests independent of the developer's git configuration.
func isolateGit(t *testing.T) {
	t.Helper()
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_AUTHOR_NAME", "Test")
	t.Setenv("GIT_AUTHOR_EMAIL", "test@example.com")
	t.Setenv("GIT_COMMITTER_NAME", "Test")
	t.Setenv("GIT_COMMITTER_EMAIL", "test@example.com")
}

// fakeRepo creates a minimal working tree with an optional origin remote.
func fakeRepo(t *testing.T, path, remote string) string {
	t.Helper()
	mkdir(t, filepath.Join(path, ".git"))
	config := "[core]\n\tbare = false\n"
	if remote != "" {
		config += fmt.Sprintf("[remote \"origin\"]\n\turl = %s\n\tfetch = +refs/heads/*:refs/remotes/origin/*\n", remote)
	}
	writeFile(t, filepath.Join(path, ".git", "config"), config)
	return path
}

// realRepo creates a git repository with one commit that is fully pushed to
// a local bare origin, so it has no unsaved work.
func realRepo(t *testing.T, path, remote string) string {
	t.Helper()
	mkdir(t, path)
	git(t, path, "init", "--quiet")
	git(t, path, "commit", "--quiet", "--allow-empty", "-m", "init")
	bare := filepath.Join(t.TempDir(), "origin.git")
	git(t, path, "clone", "--quiet", "--bare", path, bare)
	git(t, path, "remote", "add", "backup", bare)
	git(t, path, "fetch", "--quiet", "backup")
	if remote != "" {
		git(t, path, "remote", "add", "origin", remote)
	}
	return path
}

func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

func mkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	mkdir(t, filepath.Dir(path))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertExists(t *testing.T, path string, want bool) {
	t.Helper()
	_, err := os.Stat(path)
	if got := err == nil; got != want {
		t.Fatalf("exists(%s) = %v, want %v", path, got, want)
	}
}

func lines(s string) []string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}

type fakeSelector struct {
	selections []string
	inputs     []string
	confirms   []bool
	titles     []string
	options    [][]app.SelectionOption
}

var errNoAnswer = errors.New("cancelled")

func (f *fakeSelector) Select(title string, options []app.SelectionOption) (string, error) {
	f.titles = append(f.titles, title)
	f.options = append(f.options, options)
	if len(f.selections) == 0 {
		return "", errNoAnswer
	}
	answer := f.selections[0]
	f.selections = f.selections[1:]
	return answer, nil
}

func (f *fakeSelector) Input(title string, value *string, validate func(string) error) error {
	f.titles = append(f.titles, title)
	if len(f.inputs) == 0 {
		return errNoAnswer
	}
	*value = f.inputs[0]
	f.inputs = f.inputs[1:]
	if validate != nil {
		return validate(*value)
	}
	return nil
}

func (f *fakeSelector) Confirm(title string, _ bool) (bool, error) {
	f.titles = append(f.titles, title)
	if len(f.confirms) == 0 {
		return false, errNoAnswer
	}
	answer := f.confirms[0]
	f.confirms = f.confirms[1:]
	return answer, nil
}

func (f *fakeSelector) optionValues(i int) []string {
	var values []string
	for _, option := range f.options[i] {
		values = append(values, option.Value)
	}
	return values
}

type fakeGitHub struct {
	login     string
	orgs      []string
	repos     map[string][]app.RemoteRepository
	canonical map[string]string
	listed    []string

	cloned    []string
	cloneArgs []string
	cloneErr  error

	created    []domain.Repo
	createOpts []app.CreateOptions
	createErr  error

	deleted   []domain.Repo
	deleteErr error
}

func (f *fakeGitHub) Viewer(string) (string, []string, error) {
	return f.login, f.orgs, nil
}

func (f *fakeGitHub) ListRepositories(_ string, owner string) ([]app.RemoteRepository, error) {
	f.listed = append(f.listed, owner)
	return f.repos[owner], nil
}

func (f *fakeGitHub) Resolve(repo domain.Repo) (domain.Repo, error) {
	if name, ok := f.canonical[strings.ToLower(repo.FullName())]; ok {
		owner, repoName, _ := strings.Cut(name, "/")
		return domain.Repo{Host: repo.Host, Owner: owner, Name: repoName}, nil
	}
	return repo, nil
}

func (f *fakeGitHub) Clone(repo domain.Repo, destination string, gitArgs []string) error {
	f.cloned = append(f.cloned, repo.String())
	f.cloneArgs = gitArgs
	if f.cloneErr != nil {
		return f.cloneErr
	}
	fakeRepoNoT(destination, "https://"+repo.Host+"/"+repo.FullName()+".git")
	return nil
}

func (f *fakeGitHub) Create(repo domain.Repo, options app.CreateOptions) error {
	f.created = append(f.created, repo)
	f.createOpts = append(f.createOpts, options)
	return f.createErr
}

func (f *fakeGitHub) Delete(repo domain.Repo) error {
	f.deleted = append(f.deleted, repo)
	return f.deleteErr
}

func fakeRepoNoT(path, remote string) {
	_ = os.MkdirAll(filepath.Join(path, ".git"), 0o755)
	_ = os.WriteFile(filepath.Join(path, ".git", "config"), []byte("[remote \"origin\"]\n\turl = "+remote+"\n"), 0o644)
}

func samePath(t *testing.T, a, b string) bool {
	t.Helper()
	resolve := func(p string) string {
		p = filepath.Clean(filepath.FromSlash(p))
		if real, err := filepath.EvalSymlinks(p); err == nil {
			return real
		}
		return p
	}
	return strings.EqualFold(resolve(a), resolve(b))
}
