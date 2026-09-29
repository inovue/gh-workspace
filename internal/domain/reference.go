package domain

import (
	"fmt"
	"net/url"
	"strings"
)

// DefaultHost is the host whose repositories are placed directly under the
// workspace root without a host directory.
const DefaultHost = "github.com"

// Repo identifies a repository by host, owner, and name. Case is preserved as
// given; comparisons are case-insensitive because hosting services treat
// identities that way.
type Repo struct {
	Host  string
	Owner string
	Name  string
}

// IsZero reports whether the identity is unknown.
func (r Repo) IsZero() bool {
	return r.Owner == "" || r.Name == ""
}

// FullName returns owner/name.
func (r Repo) FullName() string {
	return r.Owner + "/" + r.Name
}

// String returns owner/name for the default host and host/owner/name otherwise.
func (r Repo) String() string {
	if r.Host == "" || strings.EqualFold(r.Host, DefaultHost) {
		return r.FullName()
	}
	return r.Host + "/" + r.FullName()
}

// Key returns a normalized key for case-insensitive identity comparison.
func (r Repo) Key() string {
	host := r.Host
	if host == "" {
		host = DefaultHost
	}
	return strings.ToLower(host + "/" + r.FullName())
}

// Equal reports whether two identities refer to the same repository.
func (r Repo) Equal(other Repo) bool {
	return r.Key() == other.Key()
}

// ParseReference parses an explicit repository reference. Accepted forms are
// owner/repo, host/owner/repo, HTTPS and SSH clone URLs, and browser URLs such
// as https://github.com/owner/repo/pull/1. defaultHost is used when the
// reference does not name a host.
func ParseReference(input, defaultHost string) (Repo, error) {
	return parse(input, defaultHost, true)
}

func parse(input, defaultHost string, allowBrowserPaths bool) (Repo, error) {
	raw := strings.TrimSpace(input)
	if raw == "" {
		return Repo{}, fmt.Errorf("repository reference is required")
	}
	if defaultHost == "" {
		defaultHost = DefaultHost
	}

	var host, path string
	browserURL := false
	switch {
	case strings.Contains(raw, "://"):
		u, err := url.Parse(raw)
		if err != nil || u.Hostname() == "" {
			return Repo{}, fmt.Errorf("invalid repository reference: %s", input)
		}
		host = u.Hostname()
		path = u.Path
		browserURL = allowBrowserPaths && (u.Scheme == "http" || u.Scheme == "https")
	case isSCPLike(raw):
		at := strings.Index(raw, "@")
		colon := strings.Index(raw, ":")
		host = raw[at+1 : colon]
		path = raw[colon+1:]
	default:
		parts := strings.Split(raw, "/")
		switch {
		case len(parts) == 2:
			host = defaultHost
			path = raw
		case len(parts) == 3 && strings.Contains(parts[0], "."):
			host = parts[0]
			path = parts[1] + "/" + parts[2]
		default:
			return Repo{}, fmt.Errorf("invalid repository reference: %s (expected owner/repo)", input)
		}
	}

	segments := strings.Split(strings.Trim(path, "/"), "/")
	if browserURL && len(segments) > 2 {
		segments = segments[:2]
	}
	if len(segments) != 2 {
		return Repo{}, fmt.Errorf("invalid repository reference: %s", input)
	}
	repo := Repo{
		Host:  strings.ToLower(host),
		Owner: segments[0],
		Name:  strings.TrimSuffix(segments[1], ".git"),
	}
	if !ValidHost(repo.Host) || !ValidOwner(repo.Host, repo.Owner) || !ValidName(repo.Name) {
		return Repo{}, fmt.Errorf("invalid repository reference: %s", input)
	}
	return repo, nil
}

// ParseRemoteURL parses a git remote URL into an identity. It accepts any
// host so repositories cloned from other services are still identified.
func ParseRemoteURL(remote string) (Repo, bool) {
	raw := strings.TrimSpace(remote)
	if !strings.Contains(raw, "://") && !isSCPLike(raw) {
		return Repo{}, false
	}
	repo, err := parse(raw, "", false)
	if err != nil {
		return Repo{}, false
	}
	return repo, true
}

func isSCPLike(raw string) bool {
	at := strings.Index(raw, "@")
	colon := strings.Index(raw, ":")
	slash := strings.Index(raw, "/")
	return at > 0 && colon > at && (slash == -1 || slash > colon)
}

// ValidHost reports whether s looks like a hostname. Hosts always contain a
// dot, which keeps them distinguishable from owners in the workspace layout.
func ValidHost(s string) bool {
	if !strings.Contains(s, ".") || strings.HasPrefix(s, ".") || strings.HasSuffix(s, ".") {
		return false
	}
	for _, r := range s {
		if isAlnum(r) || r == '.' || r == '-' {
			continue
		}
		return false
	}
	return true
}

// ValidOwner validates an owner segment. Owners on the default host never
// contain dots, which the layout relies on to tell owners and hosts apart.
func ValidOwner(host, s string) bool {
	if s == "" || s == "." || s == ".." || strings.HasPrefix(s, "-") {
		return false
	}
	isDefault := host == "" || strings.EqualFold(host, DefaultHost)
	for _, r := range s {
		if isAlnum(r) || r == '-' || r == '_' || (r == '.' && !isDefault) {
			continue
		}
		return false
	}
	return true
}

// ValidName validates a repository name.
func ValidName(s string) bool {
	if s == "" || s == "." || s == ".." {
		return false
	}
	for _, r := range s {
		if isAlnum(r) || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func isAlnum(r rune) bool {
	return (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
}
