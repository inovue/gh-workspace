package app

import (
	"bufio"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/inovue/gh-workspace/internal/domain"
)

// maxScanDepth covers {host}/{owner}/{repo} below a root.
const maxScanDepth = 3

// LocalRepository is a working tree found under a workspace root.
type LocalRepository struct {
	// Repo is the identity read from the origin remote, or inferred from the
	// path when no remote identifies it. It is zero when neither works.
	Repo     domain.Repo
	Path     string
	Root     string
	Rel      string
	Remote   string
	Worktree bool
}

// Display returns the name users see and type: the path relative to its root.
func (r LocalRepository) Display() string {
	return filepath.ToSlash(r.Rel)
}

// scan finds working trees under every root. Repositories are identified by
// their origin remote so their paths may follow any layout.
func scan(roots []string) []LocalRepository {
	var repos []LocalRepository
	for _, root := range roots {
		walk(root, root, 0, &repos)
	}
	sort.SliceStable(repos, func(i, j int) bool {
		return strings.ToLower(repos[i].Display()) < strings.ToLower(repos[j].Display())
	})
	return repos
}

func walk(root, dir string, depth int, repos *[]LocalRepository) {
	if depth > 0 {
		if gitDir, ok := gitDirOf(dir); ok {
			*repos = append(*repos, describe(root, dir, gitDir))
			return
		}
	}
	if depth == maxScanDepth {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		name := entry.Name()
		if strings.HasPrefix(name, ".") {
			continue
		}
		path := filepath.Join(dir, name)
		if !entry.IsDir() {
			if entry.Type()&os.ModeSymlink == 0 {
				continue
			}
			if info, err := os.Stat(path); err != nil || !info.IsDir() {
				continue
			}
		}
		walk(root, path, depth+1, repos)
	}
}

func describe(root, dir, gitDir string) LocalRepository {
	rel, _ := filepath.Rel(root, dir)
	repo := LocalRepository{Path: dir, Root: root, Rel: rel}
	commonDir := gitDir
	if data, err := os.ReadFile(filepath.Join(gitDir, "commondir")); err == nil {
		repo.Worktree = true
		commonDir = strings.TrimSpace(string(data))
		if !filepath.IsAbs(commonDir) {
			commonDir = filepath.Join(gitDir, commonDir)
		}
	}
	repo.Remote = originURL(filepath.Join(commonDir, "config"))
	if identity, ok := domain.ParseRemoteURL(repo.Remote); ok {
		repo.Repo = identity
	} else if identity, ok := domain.FromRelPath(rel); ok {
		repo.Repo = identity
	}
	return repo
}

// gitDirOf returns the git directory of a working tree rooted at dir,
// following the gitdir file used by linked worktrees and submodules.
func gitDirOf(dir string) (string, bool) {
	dotGit := filepath.Join(dir, ".git")
	info, err := os.Stat(dotGit)
	if err != nil {
		return "", false
	}
	if info.IsDir() {
		return dotGit, true
	}
	data, err := os.ReadFile(dotGit)
	if err != nil {
		return "", false
	}
	line := strings.TrimSpace(string(data))
	if !strings.HasPrefix(line, "gitdir:") {
		return "", false
	}
	gitDir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
	if !filepath.IsAbs(gitDir) {
		gitDir = filepath.Join(dir, gitDir)
	}
	return gitDir, true
}

// originURL reads the origin URL from a git config file, falling back to the
// first remote with a URL. Parsing the file directly keeps scans fast.
func originURL(configPath string) string {
	file, err := os.Open(configPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	var section, first, origin string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			section = strings.ToLower(strings.Join(strings.Fields(strings.Trim(line, "[]")), " "))
			continue
		}
		if !strings.HasPrefix(section, "remote ") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || !strings.EqualFold(strings.TrimSpace(key), "url") {
			continue
		}
		value = strings.Trim(strings.TrimSpace(value), `"`)
		if section == `remote "origin"` && origin == "" {
			origin = value
		}
		if first == "" {
			first = value
		}
	}
	if origin != "" {
		return origin
	}
	return first
}

// find returns local repositories with the given identity, preferring main
// working trees over linked worktrees and earlier roots over later ones.
func find(repos []LocalRepository, target domain.Repo) []LocalRepository {
	var found []LocalRepository
	for _, repo := range repos {
		if !repo.Repo.IsZero() && repo.Repo.Equal(target) {
			found = append(found, repo)
		}
	}
	sort.SliceStable(found, func(i, j int) bool {
		return !found[i].Worktree && found[j].Worktree
	})
	return found
}

// match returns the best matches for a free-form query. Exact repository
// names beat name prefixes, which beat substrings of the displayed path or
// identity.
func match(repos []LocalRepository, query string) []LocalRepository {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return repos
	}
	best := 0
	var matches []LocalRepository
	for _, repo := range repos {
		score := matchScore(repo, q)
		if score == 0 || score < best {
			continue
		}
		if score > best {
			best = score
			matches = nil
		}
		matches = append(matches, repo)
	}
	return matches
}

func matchScore(repo LocalRepository, q string) int {
	display := strings.ToLower(repo.Display())
	identity := strings.ToLower(repo.Repo.String())
	name := strings.ToLower(filepath.Base(repo.Path))
	switch {
	case display == q || (!repo.Repo.IsZero() && identity == q):
		return 5
	case name == q || strings.ToLower(repo.Repo.Name) == q:
		return 4
	case strings.HasPrefix(name, q):
		return 3
	case strings.Contains(display, q) || strings.Contains(identity, q):
		return 2
	case isSubsequence(q, display):
		return 1
	}
	return 0
}

func isSubsequence(q, s string) bool {
	want := []rune(q)
	i := 0
	for _, r := range s {
		if i < len(want) && want[i] == r {
			i++
		}
	}
	return i == len(want)
}

// removeEmptyParents removes empty directories from dir up to, but not
// including, root.
func removeEmptyParents(dir, root string) {
	for {
		rel, err := filepath.Rel(root, dir)
		if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

func exists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}
