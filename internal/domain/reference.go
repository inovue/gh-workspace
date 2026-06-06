package domain

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

type RepositoryRef struct {
	Owner string
	Name  string
}

func (r RepositoryRef) String() string {
	return r.Owner + "/" + r.Name
}

func ParseGitHubReference(input string) (RepositoryRef, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return RepositoryRef{}, fmt.Errorf("repository reference is required")
	}

	if strings.HasPrefix(raw, "git@") {
		return parseSCPStyle(raw, input)
	}
	if strings.Contains(raw, "://") {
		return parseURL(raw, input)
	}
	if strings.HasSuffix(strings.ToLower(raw), ".git") {
		return RepositoryRef{}, fmt.Errorf("invalid repository reference: %s", input)
	}
	return parseOwnerRepo(raw, input)
}

func parseSCPStyle(raw, original string) (RepositoryRef, error) {
	const prefix = "git@github.com:"
	if !strings.HasPrefix(strings.ToLower(raw), prefix) {
		return RepositoryRef{}, fmt.Errorf("unsupported repository host: %s", original)
	}
	if !strings.HasSuffix(strings.ToLower(raw), ".git") {
		return RepositoryRef{}, fmt.Errorf("invalid repository reference: %s", original)
	}
	return parseOwnerRepo(strings.TrimSuffix(raw[len(prefix):], ".git"), original)
}

func parseURL(raw, original string) (RepositoryRef, error) {
	u, err := url.Parse(raw)
	if err != nil {
		return RepositoryRef{}, fmt.Errorf("invalid repository reference: %s", original)
	}
	if strings.ToLower(u.Host) != "github.com" {
		return RepositoryRef{}, fmt.Errorf("unsupported repository host: %s", original)
	}
	path := strings.TrimPrefix(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")
	return parseOwnerRepo(path, original)
}

func parseOwnerRepo(raw, original string) (RepositoryRef, error) {
	parts := strings.Split(raw, "/")
	if len(parts) != 2 {
		return RepositoryRef{}, fmt.Errorf("invalid repository reference: %s", original)
	}
	owner := strings.ToLower(parts[0])
	name := strings.ToLower(parts[1])
	if !ValidOwner(owner) || !ValidRepoName(name) {
		return RepositoryRef{}, fmt.Errorf("invalid repository reference: %s", original)
	}
	return RepositoryRef{Owner: owner, Name: name}, nil
}

func ValidOwner(s string) bool {
	if s == "" || s[0] == '-' || s[len(s)-1] == '-' {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}

func ValidRepoName(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func ValidLocalLayout(host, owner, repo string) bool {
	host = strings.ToLower(host)
	owner = strings.ToLower(owner)
	repo = strings.ToLower(repo)
	if !strings.Contains(host, ".") || strings.Contains(host, "/") || host == "." || host == ".." {
		return false
	}
	return ValidLocalOwner(owner) && ValidRepoName(repo)
}

func DecodeJSON(data []byte, target any) error {
	return json.Unmarshal(data, target)
}

func ValidLocalOwner(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			continue
		}
		return false
	}
	return true
}
