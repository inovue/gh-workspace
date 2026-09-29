package domain

import (
	"path/filepath"
	"strings"
)

// RelPath returns where a repository is placed relative to a workspace root:
// {owner}/{name} for the default host and {host}/{owner}/{name} otherwise.
// Owners on the default host never contain dots, so a top-level directory
// containing a dot is always a host directory.
func RelPath(r Repo) string {
	if r.Host == "" || strings.EqualFold(r.Host, DefaultHost) {
		return filepath.Join(r.Owner, r.Name)
	}
	return filepath.Join(strings.ToLower(r.Host), r.Owner, r.Name)
}

// FromRelPath infers an identity from a path relative to a workspace root.
// It understands the current layout and the legacy github.com/{owner}/{name}
// layout. It is only a fallback for repositories without a usable remote.
func FromRelPath(rel string) (Repo, bool) {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	var repo Repo
	switch {
	case len(parts) == 2 && !strings.Contains(parts[0], "."):
		repo = Repo{Host: DefaultHost, Owner: parts[0], Name: parts[1]}
	case len(parts) == 3 && ValidHost(strings.ToLower(parts[0])):
		repo = Repo{Host: strings.ToLower(parts[0]), Owner: parts[1], Name: parts[2]}
	default:
		return Repo{}, false
	}
	if !ValidOwner(repo.Host, repo.Owner) || !ValidName(repo.Name) {
		return Repo{}, false
	}
	return repo, true
}
