package domain_test

import (
	"path/filepath"
	"testing"

	"github.com/inovue/gh-workspace/internal/domain"
)

func TestParseReferenceAcceptsSupportedForms(t *testing.T) {
	tests := map[string]domain.Repo{
		"Inovue3/App":                               {Host: "github.com", Owner: "Inovue3", Name: "App"},
		"inovue3/app.git":                           {Host: "github.com", Owner: "inovue3", Name: "app"},
		"https://github.com/Inovue3/App":            {Host: "github.com", Owner: "Inovue3", Name: "App"},
		"https://github.com/Inovue3/App.git":        {Host: "github.com", Owner: "Inovue3", Name: "App"},
		"https://github.com/cli/cli/pull/123":       {Host: "github.com", Owner: "cli", Name: "cli"},
		"https://GitHub.com/cli/cli/":               {Host: "github.com", Owner: "cli", Name: "cli"},
		"git@github.com:Inovue3/App.git":            {Host: "github.com", Owner: "Inovue3", Name: "App"},
		"git@github.com:Inovue3/App":                {Host: "github.com", Owner: "Inovue3", Name: "App"},
		"ssh://git@github.com/Inovue3/App.git":      {Host: "github.com", Owner: "Inovue3", Name: "App"},
		"ssh://git@ghe.corp.com:2222/team/app.git":  {Host: "ghe.corp.com", Owner: "team", Name: "app"},
		"ghe.corp.com/team/app":                     {Host: "ghe.corp.com", Owner: "team", Name: "app"},
		"https://gitlab.com/some.group/project.git": {Host: "gitlab.com", Owner: "some.group", Name: "project"},
		"github.com/emu_user/repo":                  {Host: "github.com", Owner: "emu_user", Name: "repo"},
	}

	for input, want := range tests {
		t.Run(input, func(t *testing.T) {
			got, err := domain.ParseReference(input, "")
			if err != nil {
				t.Fatalf("ParseReference() error = %v", err)
			}
			if got != want {
				t.Fatalf("ParseReference() = %+v, want %+v", got, want)
			}
		})
	}
}

func TestParseReferenceUsesDefaultHost(t *testing.T) {
	got, err := domain.ParseReference("team/app", "ghe.corp.com")
	if err != nil {
		t.Fatal(err)
	}
	if got.Host != "ghe.corp.com" {
		t.Fatalf("host = %q, want ghe.corp.com", got.Host)
	}
}

func TestParseReferenceRejectsInvalidForms(t *testing.T) {
	tests := []string{
		"",
		"app",
		"a/b/c/d",
		"notahost/owner/repo",
		"ssh://git@github.com/owner/repo/extra.git",
		"../owner/repo",
		"owner/..",
		"-owner/repo",
		"own.er/repo",
		"owner/re po",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if got, err := domain.ParseReference(input, ""); err == nil {
				t.Fatalf("ParseReference() = %+v, want error", got)
			}
		})
	}
}

func TestParseRemoteURLRequiresURLForm(t *testing.T) {
	if _, ok := domain.ParseRemoteURL("owner/repo"); ok {
		t.Fatal("ParseRemoteURL(owner/repo) ok = true, want false")
	}
	if _, ok := domain.ParseRemoteURL("/srv/git/repo.git"); ok {
		t.Fatal("ParseRemoteURL(local path) ok = true, want false")
	}
	got, ok := domain.ParseRemoteURL("git@github.com:cli/cli.git")
	if !ok || got.String() != "cli/cli" {
		t.Fatalf("ParseRemoteURL() = %+v, %v", got, ok)
	}
}

func TestRepoEqualIgnoresCase(t *testing.T) {
	a := domain.Repo{Host: "github.com", Owner: "Microsoft", Name: "TypeScript"}
	b := domain.Repo{Host: "GitHub.com", Owner: "microsoft", Name: "typescript"}
	if !a.Equal(b) {
		t.Fatal("Equal() = false, want true")
	}
}

func TestRepoStringOmitsDefaultHost(t *testing.T) {
	if got := (domain.Repo{Host: "github.com", Owner: "cli", Name: "cli"}).String(); got != "cli/cli" {
		t.Fatalf("String() = %q", got)
	}
	if got := (domain.Repo{Host: "ghe.corp.com", Owner: "team", Name: "app"}).String(); got != "ghe.corp.com/team/app" {
		t.Fatalf("String() = %q", got)
	}
}

func TestRelPathOmitsDefaultHost(t *testing.T) {
	tests := map[domain.Repo]string{
		{Host: "github.com", Owner: "Microsoft", Name: "TypeScript"}: filepath.Join("Microsoft", "TypeScript"),
		{Host: "ghe.corp.com", Owner: "team", Name: "app"}:           filepath.Join("ghe.corp.com", "team", "app"),
	}
	for repo, want := range tests {
		if got := domain.RelPath(repo); got != want {
			t.Fatalf("RelPath(%+v) = %q, want %q", repo, got, want)
		}
	}
}

func TestFromRelPathUnderstandsCurrentAndLegacyLayouts(t *testing.T) {
	tests := map[string]domain.Repo{
		"cli/cli":               {Host: "github.com", Owner: "cli", Name: "cli"},
		"github.com/cli/cli":    {Host: "github.com", Owner: "cli", Name: "cli"},
		"ghe.corp.com/team/app": {Host: "ghe.corp.com", Owner: "team", Name: "app"},
	}
	for rel, want := range tests {
		got, ok := domain.FromRelPath(rel)
		if !ok || got != want {
			t.Fatalf("FromRelPath(%q) = %+v, %v; want %+v", rel, got, ok, want)
		}
	}
	for _, rel := range []string{"project", "a/b/c/d", "owner/repo/extra", "some.dir/repo"} {
		if got, ok := domain.FromRelPath(rel); ok {
			t.Fatalf("FromRelPath(%q) = %+v, want not ok", rel, got)
		}
	}
}

func TestParseRemoteURLDoesNotTruncateNestedGroups(t *testing.T) {
	if got, ok := domain.ParseRemoteURL("https://gitlab.com/group/sub/project.git"); ok {
		t.Fatalf("ParseRemoteURL() = %+v, want not ok", got)
	}
}
