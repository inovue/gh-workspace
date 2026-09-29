# gh-workspace

[![CI](https://github.com/inovue/gh-workspace/actions/workflows/ci.yml/badge.svg)](https://github.com/inovue/gh-workspace/actions/workflows/ci.yml)
[![Go Version](https://img.shields.io/github/go-mod/go-version/inovue/gh-workspace)](https://go.dev)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**Every repository in one predictable place, one keystroke away.**

`gh-workspace` is a GitHub CLI extension that clones, finds, creates, and deletes repositories in a single workspace, and jumps you into them. It is a single binary with a built-in fuzzy picker: no `ghq`, no `fzf`, and no shell plugins to install.

```text
~/workspaces/
├── cli/cli
├── inovue/gh-workspace
├── my-org/api
└── ghe.corp.com/          ← only hosts other than github.com get a directory
    └── team/app
```

```console
$ ws clone cli/cli          # clone (or reuse) and cd into it
$ ws api                    # fuzzy jump; picker if several match
$ ws                        # pick from everything
$ code "$(gh workspace path gh-work)"
```

## Why gh-workspace?

- **Only needs `gh`.** It uses your existing `gh` login, SSH/HTTPS preference, and GitHub Enterprise hosts. There is nothing else to configure.
- **Short, honest paths.** `~/workspaces/owner/repo`, with no `github.com/` directory in every path.
- **Knows repositories by their remote, not their folder.** Legacy layouts, renamed or transferred repositories, and repositories you placed by hand are all found.
- **Covers the whole lifecycle.** `clone` from a searchable list of your or your organizations' repositories, `create` new ones on GitHub and locally in one step, and `delete` with checks for unpushed work.
- **Built for scripts.** stdout carries only paths or JSON. Every prompt has a flag. Failures leave stdout empty.

## Install

```bash
gh extension install inovue/gh-workspace
```

Then add the `ws` shell function (recommended):

```bash
# ~/.zshrc or ~/.bashrc
eval "$(gh workspace shell-init)"

# ~/.config/fish/config.fish
gh workspace shell-init fish | source

# PowerShell $PROFILE
Invoke-Expression (& gh workspace shell-init pwsh | Out-String)
```

`ws` changes your directory, completes repository names with Tab, and forwards other subcommands (`ws list`, `ws delete …`) to `gh workspace`. Pick a different name with `--name`.

## Commands

| Command | What it does |
| --- | --- |
| `gh workspace path [repo\|query]` | Print a local repository's path. Queries match fuzzily; no argument opens a picker. |
| `gh workspace list [query]` | List local repositories (`-p` absolute paths, `--json` for scripts). |
| `gh workspace clone [repo…\|owner]` | Clone to the canonical path, or reuse an existing clone, and print the path. An owner or no argument opens a picker of remote repositories. Flags after `--` go to `git clone`. |
| `gh workspace create [[owner/]name]` | Create a repository on GitHub and locally (`--public/--private/--internal`, `-d`, `--template`). |
| `gh workspace delete [repo\|query]` | Delete the local copy. Add `--remote` to also delete the GitHub repository. |
| `gh workspace root [--all]` | Print the workspace root(s). |
| `gh workspace migrate [--apply]` | Move repositories to their canonical paths (old layouts, renamed repositories). |
| `gh workspace shell-init [shell]` | Print the `ws` shell function. |

Repository arguments accept `owner/repo`, `host/owner/repo`, and any GitHub URL, including the one in your browser's address bar:

```bash
gh workspace clone https://github.com/cli/cli/pull/123   # → ~/workspaces/cli/cli
```

### Safe deletion

```console
$ gh workspace delete my-experiment
This will delete:
  local   /home/me/workspaces/me/my-experiment
  (the GitHub repository me/my-experiment is kept; pass --remote to delete it)
  warning: 2 unpushed commit(s)
? Delete? (y/N)
```

- The GitHub repository is deleted only with `--remote`, after you type its name.
- Without a terminal, `--yes` is required, and `--force` is also required when there is unsaved work.

## Configuration

The workspace root defaults to `~/workspaces`. Change it once with git config:

```bash
git config --global gh-workspace.root ~/src
```

Or per shell with an environment variable, which may list several roots. The first root receives new clones, and every root is searched:

```bash
export GH_WORKSPACE_ROOT=~/src:~/work
```

For GitHub Enterprise, use `host/owner/repo` or full URLs, or set `GH_HOST` just as you would for `gh`.

## Coming from ghq?

Point gh-workspace at your ghq root. Because identity comes from each repository's remote, everything is found immediately:

```bash
git config --global gh-workspace.root "$(ghq root)"
gh workspace list
```

When you want the shorter layout, preview and apply the move:

```bash
gh workspace migrate           # prints the plan and changes nothing
gh workspace migrate --apply
```

Alternatively, set the root to `$(ghq root)/github.com` so both tools share the exact same directories.

## Recipes

```bash
# Open a repository in your editor, cloning it first if needed
code "$(gh workspace clone cli/cli)"

# gh alias for the same
gh alias set --shell ow 'code "$(gh workspace clone "$1")"'

# Update every repository
gh workspace list -p | xargs -P8 -I{} git -C {} pull --ff-only --quiet

# Repositories with uncommitted changes
gh workspace list -p | while read -r d; do [ -n "$(git -C "$d" status --porcelain)" ] && echo "$d"; done

# Everything from one organization, as JSON
gh workspace list --json | jq -r '.[] | select(.owner == "my-org") | .path'

# Shallow clone of a huge repository
gh workspace clone torvalds/linux -- --depth=1
```

## How it works

- **Layout.** GitHub.com repositories go to `{root}/{owner}/{repo}`, and other hosts to `{root}/{host}/{owner}/{repo}`. GitHub owners cannot contain `.` while hostnames always do, so the two never collide.
- **Identity.** Each repository is identified by its `origin` remote, compared case-insensitively. Folders use the casing that GitHub reports. Linked git worktrees are recognized.
- **Output contract.** stdout carries only results, and prompts, progress, and errors go to stderr, so `$(gh workspace …)` is always safe.

See [docs/mvp-spec.md](docs/mvp-spec.md) for the full specification and [docs/adr](docs/adr) for design decisions.

## Contributing

Bug reports and pull requests are welcome. See [CONTRIBUTING.md](CONTRIBUTING.md).
