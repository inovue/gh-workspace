package domain_test

import (
	"testing"

	"github.com/inovue/gh-repos-extension/internal/domain"
)

func TestParseGitHubReferenceAcceptsMVPForms(t *testing.T) {
	tests := []string{
		"Inovue3/App",
		"https://github.com/Inovue3/App",
		"https://github.com/Inovue3/App.git",
		"git@github.com:Inovue3/App.git",
		"ssh://git@github.com/Inovue3/App.git",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			ref, err := domain.ParseGitHubReference(input)
			if err != nil {
				t.Fatalf("ParseGitHubReference() error = %v", err)
			}
			if got := ref.String(); got != "inovue3/app" {
				t.Fatalf("ref = %q, want %q", got, "inovue3/app")
			}
		})
	}
}

func TestParseGitHubReferenceRejectsOutOfScopeForms(t *testing.T) {
	tests := []string{
		"app",
		"owner/repo.git",
		"git@github.com:owner/repo",
		"https://gitlab.com/owner/repo",
		"https://github.com/owner/repo/issues",
		"../owner/repo",
		"owner/..",
		"-owner/repo",
		"owner-/repo",
	}

	for _, input := range tests {
		t.Run(input, func(t *testing.T) {
			if _, err := domain.ParseGitHubReference(input); err == nil {
				t.Fatal("ParseGitHubReference() error = nil, want error")
			}
		})
	}
}

func TestValidLocalLayoutAcceptsScanOwnerRules(t *testing.T) {
	if !domain.ValidLocalLayout("github.company.com", "-", "repo") {
		t.Fatal("ValidLocalLayout() = false, want true")
	}
}
