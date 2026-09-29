package app_test

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/inovue/gh-workspace/internal/app"
)

func TestRootCommandShowsHelpAndVersion(t *testing.T) {
	h := newHarness(t)
	h.mustRun()
	if h.stdout.Len() != 0 || !strings.Contains(h.stderr.String(), "shell-init") {
		t.Fatalf("help: stdout=%q stderr=%q", h.stdout.String(), h.stderr.String())
	}
	h.mustRun("--version")
	if !strings.Contains(h.stderr.String(), "v9.9.9") {
		t.Fatalf("version output = %q", h.stderr.String())
	}
}

func TestPathFindsRepositoryByIdentityIgnoringCase(t *testing.T) {
	h := newHarness(t)
	repo := fakeRepo(t, h.path("Microsoft/TypeScript"), "https://github.com/Microsoft/TypeScript.git")

	for _, query := range []string{"microsoft/typescript", "https://github.com/microsoft/TypeScript/pull/1", "git@github.com:Microsoft/TypeScript.git"} {
		if got := h.mustRun("path", query); got != repo+"\n" {
			t.Fatalf("path %s = %q, want %q", query, got, repo)
		}
	}
}

func TestPathFindsLegacyAndRenamedRepositoriesThroughTheirRemote(t *testing.T) {
	h := newHarness(t)
	legacy := fakeRepo(t, h.path("github.com/cli/cli"), "git@github.com:cli/cli.git")
	renamed := fakeRepo(t, h.path("me/old-name"), "https://github.com/me/new-name.git")
	loose := fakeRepo(t, h.path("scratch"), "https://github.com/me/scratch.git")

	tests := map[string]string{"cli/cli": legacy, "me/new-name": renamed, "me/scratch": loose}
	for query, want := range tests {
		if got := h.mustRun("path", query); got != want+"\n" {
			t.Fatalf("path %s = %q, want %q", query, got, want)
		}
	}
}

func TestPathInfersIdentityFromLayoutWithoutRemote(t *testing.T) {
	h := newHarness(t)
	repo := fakeRepo(t, h.path("me/notes"), "")
	if got := h.mustRun("path", "me/notes"); got != repo+"\n" {
		t.Fatalf("path = %q", got)
	}
}

func TestPathExplicitReferenceDoesNotFallBackToFuzzyMatching(t *testing.T) {
	h := newHarness(t)
	fakeRepo(t, h.path("cli/cli-extra"), "https://github.com/cli/cli-extra.git")
	stderr := h.mustFail("path", "cli/cli")
	if !strings.Contains(stderr, "gh workspace clone cli/cli") {
		t.Fatalf("stderr = %q, want clone hint", stderr)
	}
}

func TestPathQueryMatchesFuzzily(t *testing.T) {
	h := newHarness(t)
	cli := fakeRepo(t, h.path("cli/cli"), "https://github.com/cli/cli.git")
	fakeRepo(t, h.path("cli/go-gh"), "https://github.com/cli/go-gh.git")
	workspace := fakeRepo(t, h.path("inovue/gh-workspace"), "https://github.com/inovue/gh-workspace.git")

	tests := map[string]string{"cli": cli, "gh-work": workspace, "workspace": workspace, "INOVUE": workspace}
	for query, want := range tests {
		if got := h.mustRun("path", query); got != want+"\n" {
			t.Fatalf("path %s = %q, want %q", query, got, want)
		}
	}
}

func TestPathAmbiguousQueryFailsWithoutTTYAndPromptsWithTTY(t *testing.T) {
	h := newHarness(t)
	fakeRepo(t, h.path("a/api"), "https://github.com/a/api.git")
	b := fakeRepo(t, h.path("b/api"), "https://github.com/b/api.git")

	stderr := h.mustFail("path", "api")
	if !strings.Contains(stderr, "a/api") || !strings.Contains(stderr, "b/api") {
		t.Fatalf("stderr = %q, want candidates", stderr)
	}

	h.tty = true
	h.selector.selections = []string{"1"}
	if got := h.mustRun("path", "api"); got != b+"\n" {
		t.Fatalf("path = %q, want %q", got, b)
	}
	if got := len(h.selector.options[0]); got != 2 {
		t.Fatalf("picker showed %d options, want 2", got)
	}
}

func TestPathWithoutArgumentNeedsTTY(t *testing.T) {
	h := newHarness(t)
	fakeRepo(t, h.path("cli/cli"), "https://github.com/cli/cli.git")
	if stderr := h.mustFail("path"); !strings.Contains(stderr, "interactive terminal") {
		t.Fatalf("stderr = %q", stderr)
	}
	h.tty = true
	h.selector.selections = []string{"0"}
	if got := h.mustRun("path"); got != h.path("cli/cli")+"\n" {
		t.Fatalf("path = %q", got)
	}
}

func TestPathCancelledSelectionLeavesStdoutEmpty(t *testing.T) {
	h := newHarness(t)
	h.tty = true
	fakeRepo(t, h.path("cli/cli"), "https://github.com/cli/cli.git")
	h.mustFail("path")
}

func TestPathOnEmptyWorkspaceSuggestsClone(t *testing.T) {
	h := newHarness(t)
	h.tty = true
	if stderr := h.mustFail("path"); !strings.Contains(stderr, "gh workspace clone") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestListPrintsRelativePathsAbsolutePathsAndJSON(t *testing.T) {
	h := newHarness(t)
	fakeRepo(t, h.path("cli/cli"), "https://github.com/cli/cli.git")
	fakeRepo(t, h.path("ghe.corp.com/team/app"), "https://ghe.corp.com/team/app.git")
	fakeRepo(t, h.path("node_modules_like/.hidden/repo"), "")
	mkdir(t, h.path("empty/dir"))

	if got := lines(h.mustRun("list")); !reflect.DeepEqual(got, []string{"cli/cli", "ghe.corp.com/team/app"}) {
		t.Fatalf("list = %q", got)
	}
	if got := lines(h.mustRun("list", "--full-path", "app")); !reflect.DeepEqual(got, []string{h.path("ghe.corp.com/team/app")}) {
		t.Fatalf("list -p app = %q", got)
	}

	var entries []map[string]any
	if err := json.Unmarshal([]byte(h.mustRun("list", "--json")), &entries); err != nil {
		t.Fatal(err)
	}
	if len(entries) != 2 || entries[1]["host"] != "ghe.corp.com" || entries[1]["owner"] != "team" || entries[0]["remote"] != "https://github.com/cli/cli.git" {
		t.Fatalf("json = %v", entries)
	}
}

func TestListOnMissingRootPrintsNothing(t *testing.T) {
	h := newHarness(t)
	if got := h.mustRun("list"); got != "" {
		t.Fatalf("list = %q", got)
	}
}

func TestRootResolution(t *testing.T) {
	isolateGit(t)
	home := t.TempDir()
	run := func(env map[string]string, args ...string) string {
		var stdout, stderr strings.Builder
		code := app.New(app.Config{
			HomeDir: home,
			Getenv:  func(key string) string { return env[key] },
			Stdout:  &stdout,
			Stderr:  &stderr,
		}).Run(append([]string{"root"}, args...))
		if code != 0 {
			t.Fatalf("root: %s", stderr.String())
		}
		return stdout.String()
	}

	if got := run(nil); got != filepath.Join(home, "workspaces")+"\n" {
		t.Fatalf("default root = %q", got)
	}
	other := t.TempDir()
	env := map[string]string{app.RootEnv: "~/src" + string(os.PathListSeparator) + other}
	if got := run(env); got != filepath.Join(home, "src")+"\n" {
		t.Fatalf("env root = %q", got)
	}
	if got := lines(run(env, "--all")); !reflect.DeepEqual(got, []string{filepath.Join(home, "src"), other}) {
		t.Fatalf("env roots = %q", got)
	}
}

func TestListScansEveryRoot(t *testing.T) {
	isolateGit(t)
	first, second := t.TempDir(), t.TempDir()
	fakeRepo(t, filepath.Join(first, "cli", "cli"), "https://github.com/cli/cli.git")
	fakeRepo(t, filepath.Join(second, "github.com", "me", "dots"), "https://github.com/me/dots.git")
	var stdout, stderr strings.Builder
	code := app.New(app.Config{
		Roots:  []string{first, second},
		Getenv: func(string) string { return "" },
		Stdout: &stdout,
		Stderr: &stderr,
	}).Run([]string{"list"})
	if code != 0 {
		t.Fatal(stderr.String())
	}
	if got := lines(stdout.String()); !reflect.DeepEqual(got, []string{"cli/cli", "github.com/me/dots"}) {
		t.Fatalf("list = %q", got)
	}
}

func TestCloneUsesCanonicalCasingAndLayout(t *testing.T) {
	h := newHarness(t)
	h.github.canonical = map[string]string{"microsoft/typescript": "microsoft/TypeScript"}

	got := h.mustRun("clone", "MICROSOFT/typescript", "--", "--depth=1")
	want := h.path("microsoft/TypeScript")
	if got != want+"\n" {
		t.Fatalf("clone = %q, want %q", got, want)
	}
	if !reflect.DeepEqual(h.github.cloned, []string{"microsoft/TypeScript"}) || !reflect.DeepEqual(h.github.cloneArgs, []string{"--depth=1"}) {
		t.Fatalf("cloned %v with %v", h.github.cloned, h.github.cloneArgs)
	}
}

func TestClonePlacesOtherHostsUnderHostDirectory(t *testing.T) {
	h := newHarness(t)
	got := h.mustRun("clone", "https://ghe.corp.com/team/app")
	if got != h.path("ghe.corp.com/team/app")+"\n" {
		t.Fatalf("clone = %q", got)
	}
}

func TestCloneUsesGHHostForShorthand(t *testing.T) {
	h := newHarness(t)
	h.env["GH_HOST"] = "ghe.corp.com"
	if got := h.mustRun("clone", "team/app"); got != h.path("ghe.corp.com/team/app")+"\n" {
		t.Fatalf("clone = %q", got)
	}
}

func TestCloneReusesExistingCloneAnywhereInTheWorkspace(t *testing.T) {
	h := newHarness(t)
	legacy := fakeRepo(t, h.path("github.com/cli/cli"), "https://github.com/cli/cli.git")
	renamed := fakeRepo(t, h.path("me/tool"), "https://github.com/me/tool-v2.git")
	h.github.canonical = map[string]string{"me/tool": "me/tool-v2"}

	if got := h.mustRun("clone", "cli/cli"); got != legacy+"\n" {
		t.Fatalf("clone = %q", got)
	}
	if got := h.mustRun("clone", "me/tool"); got != renamed+"\n" {
		t.Fatalf("clone renamed = %q", got)
	}
	if len(h.github.cloned) != 0 {
		t.Fatalf("cloned %v, want nothing", h.github.cloned)
	}
}

func TestCloneMultipleRepositoriesPrintsEachPath(t *testing.T) {
	h := newHarness(t)
	got := lines(h.mustRun("clone", "cli/cli", "cli/go-gh"))
	if !reflect.DeepEqual(got, []string{h.path("cli/cli"), h.path("cli/go-gh")}) {
		t.Fatalf("clone = %q", got)
	}
}

func TestCloneFailureLeavesNoEmptyDirectories(t *testing.T) {
	h := newHarness(t)
	h.github.cloneErr = errors.New("boom")
	h.mustFail("clone", "cli/cli")
	assertExists(t, h.path("cli"), false)
}

func TestCloneRefusesToPlaceRepositoryInsideAnother(t *testing.T) {
	h := newHarness(t)
	fakeRepo(t, h.path("me"), "https://github.com/someone/me.git")
	if stderr := h.mustFail("clone", "me/app"); !strings.Contains(stderr, "inside the repository") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestCloneRejectsNonRepositoryAtDestination(t *testing.T) {
	h := newHarness(t)
	writeFile(t, h.path("cli/cli/README"), "not a repo")
	if stderr := h.mustFail("clone", "cli/cli"); !strings.Contains(stderr, "already exists") {
		t.Fatalf("stderr = %q", stderr)
	}
}

func TestClonePickerMarksClonedRepositoriesAndSwitchesOwner(t *testing.T) {
	h := newHarness(t)
	h.tty = true
	h.github.orgs = []string{"my-org"}
	h.github.repos = map[string][]app.RemoteRepository{
		"":       {{NameWithOwner: "me/dots"}, {NameWithOwner: "me/site", IsFork: true}},
		"my-org": {{NameWithOwner: "my-org/tool", Description: "internal tool"}},
	}
	fakeRepo(t, h.path("me/dots"), "https://github.com/me/dots.git")
	h.selector.selections = []string{"\x00switch-owner", "my-org", "my-org/tool"}

	if got := h.mustRun("clone"); got != h.path("my-org/tool")+"\n" {
		t.Fatalf("clone = %q", got)
	}
	first := h.selector.options[0]
	if len(first) != 3 || !first[1].Disabled || first[2].Disabled {
		t.Fatalf("first picker = %+v", first)
	}
	if !reflect.DeepEqual(h.github.listed, []string{"", "my-org"}) {
		t.Fatalf("listed owners = %v", h.github.listed)
	}
}

func TestCloneOwnerArgumentOpensThatOwner(t *testing.T) {
	h := newHarness(t)
	h.tty = true
	h.github.repos = map[string][]app.RemoteRepository{"cli": {{NameWithOwner: "cli/cli"}}}
	h.selector.selections = []string{"cli/cli"}
	h.mustRun("clone", "cli")
	if !reflect.DeepEqual(h.github.listed, []string{"cli"}) {
		t.Fatalf("listed = %v", h.github.listed)
	}
}

func TestCloneWithoutArgumentNeedsTTY(t *testing.T) {
	h := newHarness(t)
	h.mustFail("clone")
}

func TestCreateWithNameRunsNonInteractively(t *testing.T) {
	h := newHarness(t)
	got := h.mustRun("create", "tool", "--public", "-d", "A tool")
	want := h.path("me/tool")
	if got != want+"\n" {
		t.Fatalf("create = %q, want %q", got, want)
	}
	if subject := git(t, want, "log", "--format=%s"); subject != "Initial commit" {
		t.Fatalf("commit = %q", subject)
	}
	opts := h.github.createOpts[0]
	if h.github.created[0].FullName() != "me/tool" || opts.Visibility != "public" || opts.Description != "A tool" || opts.SourceDir != want {
		t.Fatalf("created %v with %+v", h.github.created, opts)
	}
}

func TestCreateDefaultsToPrivateWithoutTTY(t *testing.T) {
	h := newHarness(t)
	h.mustRun("create", "my-org/service")
	if got := h.github.createOpts[0].Visibility; got != "private" {
		t.Fatalf("visibility = %q", got)
	}
	if h.github.created[0].Owner != "my-org" {
		t.Fatalf("owner = %q", h.github.created[0].Owner)
	}
}

func TestCreateRollsBackWhenGitHubFails(t *testing.T) {
	h := newHarness(t)
	h.github.createErr = errors.New("name taken")
	h.mustFail("create", "tool")
	assertExists(t, h.path("me"), false)
}

func TestCreateFromTemplateClonesTheNewRepository(t *testing.T) {
	h := newHarness(t)
	got := h.mustRun("create", "svc", "--template", "me/template")
	if got != h.path("me/svc")+"\n" {
		t.Fatalf("create = %q", got)
	}
	if h.github.createOpts[0].Template != "me/template" || h.github.createOpts[0].SourceDir != "" {
		t.Fatalf("options = %+v", h.github.createOpts[0])
	}
	if !reflect.DeepEqual(h.github.cloned, []string{"me/svc"}) {
		t.Fatalf("cloned = %v", h.github.cloned)
	}
}

func TestCreateRejectsExistingRepositoryAndConflictingFlags(t *testing.T) {
	h := newHarness(t)
	fakeRepo(t, h.path("github.com/me/tool"), "https://github.com/me/tool.git")
	if stderr := h.mustFail("create", "tool"); !strings.Contains(stderr, "already in the workspace") {
		t.Fatalf("stderr = %q", stderr)
	}
	h.mustFail("create", "other", "--public", "--private")
	h.mustFail("create", "bad name")
}

func TestCreateInteractivelyAsksForEverything(t *testing.T) {
	h := newHarness(t)
	h.tty = true
	h.github.orgs = []string{"my-org"}
	h.selector.selections = []string{"my-org", "internal"}
	h.selector.inputs = []string{"svc", "Service"}
	if got := h.mustRun("create"); got != h.path("my-org/svc")+"\n" {
		t.Fatalf("create = %q", got)
	}
	opts := h.github.createOpts[0]
	if opts.Visibility != "internal" || opts.Description != "Service" {
		t.Fatalf("options = %+v", opts)
	}
}

func TestDeleteLocalKeepsGitHubRepository(t *testing.T) {
	h := newHarness(t)
	repo := realRepo(t, h.path("me/tool"), "https://github.com/me/tool.git")

	h.mustFail("delete", "me/tool")
	assertExists(t, repo, true)

	if got := h.mustRun("delete", "me/tool", "--yes"); got != "" {
		t.Fatalf("stdout = %q, want empty", got)
	}
	assertExists(t, repo, false)
	assertExists(t, h.path("me"), false)
	if len(h.github.deleted) != 0 {
		t.Fatalf("deleted on GitHub: %v", h.github.deleted)
	}
}

func TestDeleteRefusesUnsavedWorkWithoutForce(t *testing.T) {
	h := newHarness(t)
	repo := realRepo(t, h.path("me/tool"), "https://github.com/me/tool.git")
	writeFile(t, filepath.Join(repo, "draft.txt"), "wip")
	git(t, repo, "commit", "--quiet", "--allow-empty", "-m", "local only")

	stderr := h.mustFail("delete", "me/tool", "--yes")
	if !strings.Contains(stderr, "uncommitted") || !strings.Contains(stderr, "unpushed") {
		t.Fatalf("stderr = %q", stderr)
	}
	assertExists(t, repo, true)

	h.mustRun("delete", "me/tool", "--yes", "--force")
	assertExists(t, repo, false)
}

func TestDeleteInTerminalConfirmsDespiteWarnings(t *testing.T) {
	h := newHarness(t)
	h.tty = true
	repo := realRepo(t, h.path("me/tool"), "https://github.com/me/tool.git")
	writeFile(t, filepath.Join(repo, "draft.txt"), "wip")

	h.selector.confirms = []bool{false}
	h.mustFail("delete", "tool")
	assertExists(t, repo, true)

	h.selector.confirms = []bool{true}
	h.mustRun("delete", "tool")
	assertExists(t, repo, false)
}

func TestDeleteNotClonedRequiresRemoteFlag(t *testing.T) {
	h := newHarness(t)
	if stderr := h.mustFail("delete", "me/typo"); !strings.Contains(stderr, "--remote") {
		t.Fatalf("stderr = %q", stderr)
	}
	if len(h.github.deleted) != 0 {
		t.Fatalf("deleted on GitHub: %v", h.github.deleted)
	}
	h.mustRun("delete", "me/typo", "--remote", "--yes")
	if len(h.github.deleted) != 1 || h.github.deleted[0].FullName() != "me/typo" {
		t.Fatalf("deleted = %v", h.github.deleted)
	}
}

func TestDeleteRemoteRequiresTypingTheName(t *testing.T) {
	h := newHarness(t)
	h.tty = true
	repo := realRepo(t, h.path("me/tool"), "https://github.com/me/tool.git")

	h.selector.inputs = []string{"me/wrong"}
	h.mustFail("delete", "me/tool", "--remote")
	assertExists(t, repo, true)
	if len(h.github.deleted) != 0 {
		t.Fatalf("deleted = %v", h.github.deleted)
	}

	h.selector.inputs = []string{"me/tool"}
	h.mustRun("delete", "me/tool", "--remote")
	assertExists(t, repo, false)
	if len(h.github.deleted) != 1 {
		t.Fatalf("deleted = %v", h.github.deleted)
	}
}

func TestDeleteKeepsLocalWhenRemoteDeletionFails(t *testing.T) {
	h := newHarness(t)
	repo := realRepo(t, h.path("me/tool"), "https://github.com/me/tool.git")
	h.github.deleteErr = errors.New("missing scope")
	h.mustFail("delete", "me/tool", "--remote", "--yes")
	assertExists(t, repo, true)
}

func TestDeleteLinkedWorktreeKeepsMainRepository(t *testing.T) {
	h := newHarness(t)
	main := realRepo(t, h.path("me/tool"), "https://github.com/me/tool.git")
	worktree := h.path("me/tool@feature")
	git(t, main, "worktree", "add", "--quiet", "-b", "feature", worktree)
	git(t, worktree, "push", "--quiet", "backup", "feature")

	if got := h.mustRun("path", "me/tool"); got != main+"\n" {
		t.Fatalf("path prefers %q, want main working tree", got)
	}
	if got := lines(h.mustRun("list")); !reflect.DeepEqual(got, []string{"me/tool", "me/tool@feature"}) {
		t.Fatalf("list = %q", got)
	}
	h.mustFail("delete", "tool@feature", "--remote", "--yes")

	h.mustRun("delete", "tool@feature", "--yes")
	assertExists(t, worktree, false)
	assertExists(t, main, true)
	if out := git(t, main, "worktree", "list"); strings.Contains(out, "feature") {
		t.Fatalf("worktree not pruned: %s", out)
	}
}

func TestMigrateMovesLegacyAndRenamedRepositoriesOnlyWithApply(t *testing.T) {
	h := newHarness(t)
	legacy := realRepo(t, h.path("github.com/cli/cli"), "https://github.com/cli/cli.git")
	renamed := fakeRepo(t, h.path("me/old"), "https://github.com/me/new.git")
	fakeRepo(t, h.path("github.com/x/taken"), "https://github.com/x/taken.git")
	fakeRepo(t, h.path("x/taken"), "https://github.com/x/other.git")
	worktree := h.path("github.com/cli/cli-wt")
	git(t, legacy, "worktree", "add", "--quiet", "-b", "wt", worktree)

	h.mustRun("migrate")
	assertExists(t, legacy, true)
	if !strings.Contains(h.stderr.String(), "--apply") {
		t.Fatalf("stderr = %q", h.stderr.String())
	}

	h.mustRun("migrate", "--apply")
	assertExists(t, legacy, false)
	assertExists(t, renamed, false)
	assertExists(t, h.path("cli/cli/.git"), true)
	assertExists(t, h.path("me/new/.git"), true)
	assertExists(t, h.path("github.com/x/taken"), true)
	if !strings.Contains(h.stderr.String(), "skip") {
		t.Fatalf("stderr = %q, want skipped conflict", h.stderr.String())
	}
	if top := git(t, worktree, "rev-parse", "--show-toplevel"); !samePath(t, top, worktree) {
		t.Fatalf("worktree broken after migrate: %s", top)
	}
}

func TestShellInitScriptsAreValid(t *testing.T) {
	h := newHarness(t)
	for _, shell := range []string{"bash", "zsh", "fish", "pwsh"} {
		script := h.mustRun("shell-init", shell, "--name", "gw")
		if !strings.Contains(script, "gw") || strings.Contains(script, "__NAME__") || strings.Contains(script, "__PASSTHROUGH__") {
			t.Fatalf("%s script = %s", shell, script)
		}
		check := map[string][]string{"bash": {"bash", "-n"}, "zsh": {"zsh", "-n"}, "fish": {"fish", "--no-execute"}}[shell]
		if check == nil {
			continue
		}
		if _, err := exec.LookPath(check[0]); err != nil || runtime.GOOS == "windows" {
			continue
		}
		file := filepath.Join(t.TempDir(), "init")
		writeFile(t, file, script)
		if out, err := exec.Command(check[0], append(check[1:], file)...).CombinedOutput(); err != nil {
			t.Fatalf("%s syntax: %v\n%s", shell, err, out)
		}
	}

	h.env["SHELL"] = "/usr/bin/zsh"
	if script := h.mustRun("shell-init"); !strings.Contains(script, "compdef") {
		t.Fatalf("detected script = %s", script)
	}
	h.mustFail("shell-init", "tcsh")
	h.mustFail("shell-init", "bash", "--name", "bad;name")
}

func TestShellFunctionChangesDirectory(t *testing.T) {
	if _, err := exec.LookPath("bash"); err != nil || runtime.GOOS == "windows" {
		t.Skip("needs a POSIX shell")
	}
	h := newHarness(t)
	repo := fakeRepo(t, h.path("cli/cli"), "https://github.com/cli/cli.git")
	script := h.mustRun("shell-init", "bash")

	bin := t.TempDir()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	// A fake gh that forwards "gh workspace ..." to this test binary.
	writeFile(t, filepath.Join(bin, "gh"), "#!/bin/sh\nshift\nexec \""+exe+"\" -test.run=TestHelperProcess -- \"$@\"\n")
	if err := os.Chmod(filepath.Join(bin, "gh"), 0o755); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command("bash", "-c", script+"\nws cli && pwd")
	cmd.Env = append(os.Environ(), "PATH="+bin+string(os.PathListSeparator)+os.Getenv("PATH"), "HELPER_ROOT="+h.root)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("ws: %v\n%s", err, out)
	}
	if got := strings.TrimSpace(string(out)); got != repo {
		t.Fatalf("pwd = %q, want %q", got, repo)
	}
}

// TestHelperProcess runs the app as the fake gh extension for shell tests.
func TestHelperProcess(t *testing.T) {
	root := os.Getenv("HELPER_ROOT")
	if root == "" {
		return
	}
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}
	os.Exit(app.New(app.Config{Roots: []string{root}, GitHub: &fakeGitHub{}}).Run(args))
}
