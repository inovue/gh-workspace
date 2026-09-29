# gh-workspace Current Specification

`gh-workspace` is a command-first GitHub CLI extension that keeps repositories in one predictable workspace and resolves them to local paths.

## Commands

```bash
gh workspace path   [repository|query]
gh workspace list   [query] [--full-path] [--json]
gh workspace clone  [repository...|owner] [-- <git clone flags>]
gh workspace create [[owner/]name] [--public|--private|--internal] [--description text] [--template owner/repo]
gh workspace delete [repository|query] [--remote] [--yes] [--force]
gh workspace root   [--all]
gh workspace migrate [--apply]
gh workspace shell-init [bash|zsh|fish|pwsh] [--name ws]
```

`gh workspace` without a subcommand shows help; `--version` prints the version.

## Workspace Roots

Roots come from the first source that is set:

1. `GH_WORKSPACE_ROOT`, separated like `PATH`
2. git config `gh-workspace.root`, which may be repeated
3. `~/workspaces`

A leading `~` is expanded. The first root is primary: new repositories go there. Every root is scanned.

## Layout and Identity

New repositories are placed at their canonical path:

```text
{root}/{owner}/{repo}          GitHub.com
{root}/{host}/{owner}/{repo}   any other host
```

Owners on GitHub.com never contain `.`, so a top-level directory containing `.` is a host directory.

Local repositories are found by scanning up to three directory levels below each root. Hidden directories are skipped, and scanning does not descend into repositories. A repository's identity comes from its `origin` remote (or its first remote). Without a parsable remote, the identity is inferred from the current or legacy `{root}/github.com/{owner}/{repo}` layout. Repositories without an identity are still listed and can be matched by path.

Identities compare case-insensitively. Folders use the canonical casing that GitHub reports.

Linked worktrees are recognized. They share their main working tree's identity, and commands that resolve an identity prefer the main working tree.

## References and Queries

Repository references:

- `owner/repo` (host from `GH_HOST`, default `github.com`)
- `host/owner/repo`
- `https://host/owner/repo[.git]`, including browser URLs such as `/owner/repo/pull/1`
- `git@host:owner/repo[.git]`, `ssh://git@host[:port]/owner/repo[.git]`

An argument containing `/` or `:` that parses as a reference must match a local identity, or a local path relative to its root, exactly. Other arguments are queries, ranked as follows:

1. exact relative path or identity
2. exact repository name
3. repository name prefix
4. substring of the relative path or identity
5. subsequence of the relative path

The best-ranked matches win. When several remain, a picker opens in a terminal; otherwise the command fails and lists the candidates.

## `path`

Prints the path of one local repository. Without an argument, it opens a picker of every local repository and needs a terminal.

## `list`

Prints local repositories sorted by relative path: relative paths by default, absolute paths with `--full-path`, or JSON with `--json` (`name`, `path`, `root`, `host`, `owner`, `repo`, `remote`, `worktree`). An optional query filters the list with the ranking above.

## `clone`

For each reference, prints the path of the local repository. If the repository is already present anywhere in the workspace, its existing path is printed without cloning. Otherwise, `gh repo view` resolves the canonical name, which also handles renames and transfers. The repository is then cloned with `gh repo clone` to its canonical path in the primary root, and the new path is printed. Arguments after `--` go to `git clone`.

With a single owner name or no argument, a picker lists the owner's (or the authenticated user's) repositories. Cloned repositories are shown but cannot be selected. A leading entry switches to the user or one of their organizations.

A failed clone removes the directories it created. Cloning into a path that exists, or that lies inside another working tree, is refused.

## `create`

Creates a repository on GitHub and in the workspace, then prints its path.

- `name` creates under the authenticated user; `owner/name` under that owner.
- Without an argument, the command asks for owner, name, visibility, and description, and needs a terminal.
- Visibility comes from the flags. Otherwise the command asks in a terminal and defaults to private without one.
- Without `--template`: `git init` (respecting `init.defaultBranch`), then an empty `Initial commit`, then `gh repo create --source <path> --remote origin --push`. Any failure removes the local directory.
- With `--template`: `gh repo create --template`, then `clone`.

## `delete`

Deletes a local working tree, and the GitHub repository with `--remote`. Prints nothing on stdout.

- A reference that is not cloned fails unless `--remote` is given, in which case only the GitHub repository is deleted.
- The command reports uncommitted changes, unpushed commits, stashes, and linked worktrees. With `--yes`, or without a terminal, deletion with any of them requires `--force`.
- Without `--yes`, a terminal is required. Local-only deletion asks for confirmation. Remote deletion asks the user to type `owner/repo`.
- Remote deletion runs first, so a failure leaves the local copy intact. A missing `delete_repo` scope produces the `gh auth refresh` command that fixes it.
- A linked worktree is removed and pruned from its main repository. `--remote` is refused for a linked worktree.
- Empty owner and host directories left behind are removed.

## `root`

Prints the primary root, or every root with `--all`.

## `migrate`

Plans moves of main working trees whose path differs from their canonical path within the same root. This covers legacy layouts and renamed repositories. Nothing moves without `--apply`. Moves are skipped when the target exists, when the target lies inside another working tree, or when the target lies inside the repository itself. Case-only differences are ignored. After a move, `git worktree repair` reconnects linked worktrees.

## `shell-init`

Prints a function (default `ws`) with completion for bash, zsh, fish, or PowerShell. The shell is detected from `$SHELL` when not given.

- `ws` or `ws <query>`: `cd` to `gh workspace path`
- `ws clone|create|path …`: `cd` to the last printed path
- `ws <other subcommand> …`: run it; if the current directory disappeared, `cd` to the root

## Output Contract

- stdout carries only results: paths from `path`, `clone`, `create`, and `root`; lines or JSON from `list`; the script from `shell-init`.
- Prompts, child-process output, progress, warnings, and errors go to stderr.
- Failure and cancellation exit non-zero and leave stdout empty.
- Prompts are used only when stdin and stderr are terminals.

## Dependencies

Runtime: GitHub CLI (`gh`) and Git. Go libraries: Cobra; Huh, Bubble Tea, and Lip Gloss.

## Boundaries

- no dashboard or full-screen UI
- no editor or browser launching (compose with `path` instead)
- no metadata cache; scans read `.git/config` directly
- no worktree creation yet
