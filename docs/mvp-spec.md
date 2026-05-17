# gh-workspace MVP Spec

`gh-workspace` is a small GitHub CLI extension for resolving local repository paths and cloning GitHub.com repositories into a predictable local layout.

## Commands

```bash
gh workspace path [repository]
gh workspace clone [repository]
```

`gh workspace` without a subcommand shows help.

## Repository References

Explicit repository arguments may be:

- `owner/repo`
- `https://github.com/owner/repo`
- `https://github.com/owner/repo.git`
- `git@github.com:owner/repo.git`
- `ssh://git@github.com/owner/repo.git`

All accepted forms are normalized to `owner/repo` for GitHub CLI operations and to `~/workspaces/github.com/{owner}/{repo}` for local paths.
Host, owner, and repository path segments are lowercased during normalization.

MVP rejects:

- repo-only shorthand such as `repo`
- shorthand with `.git` suffix such as `owner/repo.git`
- non-GitHub.com URLs
- paths with extra segments
- path traversal such as `../owner/repo`

Owner validation:

- ASCII letters, digits, and `-`
- must start and end with an ASCII letter or digit

Repository name validation:

- ASCII letters, digits, `.`, `_`, and `-`
- must not be `.` or `..`

## Local Layout

Default root:

```text
~/workspaces
```

Implementation derives this from the current user's home directory. If the home directory cannot be determined, commands fail with stderr output, empty stdout, and a non-zero exit code.

Repository path:

```text
~/workspaces/{host}/{owner}/{repo}
```

All path identity segments are lowercase.

GitHub.com clone example:

```text
~/workspaces/github.com/inovue3/app
```

## Local Scan

`path` scans local repositories in the host/owner/name layout under `~/workspaces`.

Scan candidates are exactly:

```text
~/workspaces/*/*/*
```

A local repository is a directory with either:

- `.git` directory
- `.git` file

Bare repositories are out of scope. `path` is intended to return working-tree directories for `cd` and editor composition.

Scan validation:

- host must contain at least one `.`
- owner must be non-empty and contain only ASCII letters, digits, or `-`
- repo must be non-empty and contain only ASCII letters, digits, `.`, `_`, or `-`
- `.` and `..` path segments are rejected

`path` does not recursively search for `.git` directories below the fixed host/owner/name layout.

`path` may include non-GitHub.com hosts discovered locally, such as:

```text
~/workspaces/github.company.com/team/app
```

`clone` only targets GitHub.com repositories in the MVP.

The MVP does not inspect `origin` remotes. Repository identity for local paths comes from the `~/workspaces/{host}/{owner}/{repo}` layout.

## path

```bash
gh workspace path owner/repo
```

Behavior:

- cloned: print absolute local path to stdout and exit `0`
- not cloned: print error to stderr and exit non-zero
- invalid or unknown repository reference: print error to stderr and exit non-zero
- no side effects

```bash
gh workspace path
```

Behavior:

- open selection UI with local scanned repositories only
- selection options show `owner/repo` with `host` and absolute path as supporting text
- print selected absolute local path to stdout
- cancel leaves stdout empty and exits non-zero
- if no TTY is available, print an error to stderr and exit non-zero

## clone

```bash
gh workspace clone owner/repo
```

Behavior:

- if destination is already a git repository: print absolute path to stdout and exit `0`
- if destination does not exist: run `gh repo clone owner/repo <destination>`
- before cloning, create only the destination parent directory
- if clone succeeds: print absolute path to stdout and exit `0`
- if destination exists but is not a git repository: print error to stderr and exit non-zero
- if clone fails: print error to stderr and exit non-zero
- do not delete partially created directories after clone failure

`clone` may attempt `gh repo clone` for an explicit `owner/repo` even if it is not present in the active-user repository list.

When running `gh repo clone`, forward both child stdout and child stderr to parent stderr. Only `gh-workspace` prints the final absolute path to stdout after a successful clone.
Clone protocol selection is delegated entirely to GitHub CLI configuration.

```bash
gh workspace clone
```

Behavior:

- fetch active GitHub CLI user's GitHub.com repositories with `gh repo list --limit 1000 --json nameWithOwner`
- exclude already cloned repositories
- open selection UI
- selection options show `owner/repo` with clone destination as supporting text
- repository descriptions are not fetched or shown in the MVP
- clone selected repository
- print cloned absolute path to stdout
- cancel leaves stdout empty and exits non-zero
- if no TTY is available, print an error to stderr and exit non-zero

Remote fetch captures child stdout for JSON parsing and forwards child stderr to parent stderr. The MVP does not paginate beyond 1000 repositories.

## Output Contract

stdout:

- path only
- exactly one absolute path on success
- paths are lexical absolute paths; symlinks are not resolved
- empty on failure or cancellation
- intended for shell composition

stderr:

- selection UI
- progress
- errors

Selection prompts must explicitly write to stderr. With Huh, use form output configuration rather than relying on stdout defaults.
Commands without a repository argument require a TTY because they must open the selection UI. Commands with an explicit repository argument must not require a TTY.

Examples:

```bash
cd "$(gh workspace path)"
zed "$(gh workspace path owner/repo)"
code "$(gh workspace clone owner/repo)"
cd "$(gh workspace clone)"
```

## Dependencies

Runtime:

- `gh`
- `git`

Go modules:

- Cobra for CLI commands
- Huh for selection prompts

No external picker such as `fzf` is required.

## Out of Scope

- config file
- flags
- organization owner selection
- GitHub Enterprise cloning
- browser opening
- editor launching
- shell launching
- lazygit integration
- list command
- cache
