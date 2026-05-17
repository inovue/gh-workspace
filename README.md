# gh-workspace

Keep every GitHub repository in one predictable place.

`gh-workspace` is a small [GitHub CLI](https://cli.github.com/) extension that resolves and clones repositories under:

```text
~/workspaces/github.com/{owner}/{repo}
```

It gives scripts, shells, editors, and humans the same answer every time: the absolute local path for a repository.

## Why

Most local checkouts drift into `~/src`, `~/code`, `~/Downloads`, project folders, and one-off temp directories. That makes automation brittle:

```bash
cd ~/workspaces/github.com/cli/cli
```

`gh-workspace` makes that layout the default and gives you composable commands for finding or creating the right checkout.

## Installation

Install with GitHub CLI:

```bash
gh extension install inovue/gh-workspace
```

Requirements:

- `gh` installed and authenticated
- `git` available on your `PATH`

Install a specific release:

```bash
gh extension install inovue/gh-workspace --pin v0.1.0
```

Check the extension:

```bash
gh workspace --help
```

## Usage

Print the path for a repository you already have locally:

```bash
gh workspace path cli/cli
```

Output:

```text
/home/you/workspaces/github.com/cli/cli
```

Clone a repository into the workspace layout:

```bash
gh workspace clone cli/cli
```

If the repository is already cloned, `clone` simply prints the existing path.

## Shell Workflows

Jump to a checkout:

```bash
cd "$(gh workspace path cli/cli)"
```

Clone and open in your editor:

```bash
code "$(gh workspace clone cli/cli)"
```

Use GitHub URLs directly:

```bash
gh workspace clone https://github.com/cli/cli.git
gh workspace path git@github.com:cli/cli.git
```

Pick from an interactive list:

```bash
gh workspace path
gh workspace clone
```

Without an argument:

- `path` lists local repositories under `~/workspaces/*/*/*`
- `clone` lists GitHub repositories available to your `gh` account and hides repositories already cloned

## Commands

```bash
gh workspace path [repository]
gh workspace clone [repository]
```

Repository references can be:

```text
owner/repo
https://github.com/owner/repo
https://github.com/owner/repo.git
ssh://git@github.com/owner/repo.git
git@github.com:owner/repo.git
```

Only GitHub.com repository references are supported for cloning. Local repository discovery can still show any valid workspace path shaped like:

```text
~/workspaces/{host}/{owner}/{repo}
```

## Script Friendly

Successful commands write exactly one absolute path to stdout.

Progress, prompts, help, and errors go to stderr, so command substitution stays clean:

```bash
repo_path="$(gh workspace clone cli/cli)"
make -C "$repo_path" test
```

## Local Development

Build:

```bash
make build
```

Run tests:

```bash
make test
```

Install this checkout as a local extension:

```bash
make extension
gh extension install .
```

The checkout directory must be named `gh-workspace`; GitHub CLI uses the directory name to find the extension executable.

## Release

Releases are published from SemVer tags:

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow builds GitHub CLI extension binaries for macOS, Linux, and Windows using GoReleaser. If `gh extension install inovue/gh-workspace` reports that no usable release artifact was found, the release artifacts are missing or misnamed.
