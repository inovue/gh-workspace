package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/inovue/gh-workspace/internal/domain"
)

// GitHubCLI is the subset of GitHub operations gh-workspace needs. The
// default implementation shells out to gh so authentication, hosts, and git
// protocol preferences come from the user's gh configuration.
type GitHubCLI interface {
	Viewer(host string) (login string, orgs []string, err error)
	ListRepositories(host, owner string) ([]RemoteRepository, error)
	Resolve(repo domain.Repo) (domain.Repo, error)
	Clone(repo domain.Repo, destination string, gitArgs []string) error
	Create(repo domain.Repo, options CreateOptions) error
	Delete(repo domain.Repo) error
}

type RemoteRepository struct {
	NameWithOwner string `json:"nameWithOwner"`
	Description   string `json:"description"`
	IsArchived    bool   `json:"isArchived"`
	IsFork        bool   `json:"isFork"`
}

type CreateOptions struct {
	Visibility  string
	Description string
	Template    string
	// SourceDir is a local repository to push as the initial content. It is
	// empty when the repository is created from a template.
	SourceDir string
}

type ghCLI struct {
	stderr io.Writer
}

func (g ghCLI) command(host string, args ...string) *exec.Cmd {
	cmd := exec.Command("gh", args...)
	if host != "" && !strings.EqualFold(host, domain.DefaultHost) {
		cmd.Env = append(os.Environ(), "GH_HOST="+host)
	}
	return cmd
}

func (g ghCLI) Viewer(host string) (string, []string, error) {
	cmd := g.command(host, "api", "graphql", "-f", "query=query { viewer { login organizations(first: 100) { nodes { login } } } }")
	cmd.Stderr = g.stderr
	out, err := cmd.Output()
	if err != nil {
		return "", nil, fmt.Errorf("gh api graphql: %w", err)
	}
	var res struct {
		Data struct {
			Viewer struct {
				Login         string `json:"login"`
				Organizations struct {
					Nodes []struct {
						Login string `json:"login"`
					} `json:"nodes"`
				} `json:"organizations"`
			} `json:"viewer"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &res); err != nil {
		return "", nil, err
	}
	var orgs []string
	for _, org := range res.Data.Viewer.Organizations.Nodes {
		orgs = append(orgs, org.Login)
	}
	return res.Data.Viewer.Login, orgs, nil
}

func (g ghCLI) ListRepositories(host, owner string) ([]RemoteRepository, error) {
	args := []string{"repo", "list"}
	if owner != "" {
		args = append(args, owner)
	}
	args = append(args, "--limit", "1000", "--json", "nameWithOwner,description,isArchived,isFork")
	cmd := g.command(host, args...)
	cmd.Stderr = g.stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh repo list: %w", err)
	}
	var repos []RemoteRepository
	if err := json.Unmarshal(out, &repos); err != nil {
		return nil, err
	}
	return repos, nil
}

func (g ghCLI) Resolve(repo domain.Repo) (domain.Repo, error) {
	cmd := g.command(repo.Host, "repo", "view", repo.String(), "--json", "nameWithOwner", "--jq", ".nameWithOwner")
	cmd.Stderr = g.stderr
	out, err := cmd.Output()
	if err != nil {
		return domain.Repo{}, fmt.Errorf("cannot access %s", repo)
	}
	owner, name, ok := strings.Cut(strings.TrimSpace(string(out)), "/")
	if !ok {
		return repo, nil
	}
	return domain.Repo{Host: repo.Host, Owner: owner, Name: name}, nil
}

func (g ghCLI) Clone(repo domain.Repo, destination string, gitArgs []string) error {
	args := []string{"repo", "clone", repo.String(), destination}
	if len(gitArgs) > 0 {
		args = append(append(args, "--"), gitArgs...)
	}
	cmd := g.command(repo.Host, args...)
	cmd.Stdout = g.stderr
	cmd.Stderr = g.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh repo clone %s: %w", repo, err)
	}
	return nil
}

func (g ghCLI) Create(repo domain.Repo, options CreateOptions) error {
	args := []string{"repo", "create", repo.String(), "--" + options.Visibility}
	if options.Description != "" {
		args = append(args, "--description", options.Description)
	}
	if options.Template != "" {
		args = append(args, "--template", options.Template)
	}
	if options.SourceDir != "" {
		args = append(args, "--source", options.SourceDir, "--remote", "origin", "--push")
	}
	cmd := g.command(repo.Host, args...)
	cmd.Stdout = g.stderr
	cmd.Stderr = g.stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh repo create %s: %w", repo, err)
	}
	return nil
}

func (g ghCLI) Delete(repo domain.Repo) error {
	var captured bytes.Buffer
	cmd := g.command(repo.Host, "repo", "delete", repo.String(), "--yes")
	cmd.Stdout = g.stderr
	cmd.Stderr = io.MultiWriter(g.stderr, &captured)
	if err := cmd.Run(); err != nil {
		if strings.Contains(captured.String(), "delete_repo") {
			host := repo.Host
			if host == "" {
				host = domain.DefaultHost
			}
			return fmt.Errorf("gh repo delete %s: missing the delete_repo scope; run: gh auth refresh -h %s -s delete_repo", repo, host)
		}
		return fmt.Errorf("gh repo delete %s: %w", repo, err)
	}
	return nil
}
