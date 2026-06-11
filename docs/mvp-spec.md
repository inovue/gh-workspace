# gh-workspace Current Specification

`gh-workspace` is a command-first GitHub CLI extension for managing repositories through a predictable local workspace layout.

## Commands

```bash
gh workspace path [repository]
gh workspace clone [repository|owner]
gh workspace create
gh workspace delete [repository]
```

`gh workspace` without a subcommand shows help. The hidden `monkey` command exercises the selection UI.

## Repository References

Explicit repository references accept:

- `owner/repo`
- `https://github.com/owner/repo`
- `https://github.com/owner/repo.git`
- `git@github.com:owner/repo.git`
- `ssh://git@github.com/owner/repo.git`

References are normalized to lowercase `owner/repo`. Repo-only shorthand, non-GitHub.com URLs, extra path segments, traversal, and `owner/repo.git` shorthand are rejected.

An owner-only argument is accepted by `clone` and opens remote repository selection for that owner.

## Workspace Layout

The workspace root is `~/workspaces`. Repository paths use:

```text
~/workspaces/{host}/{owner}/{repo}
```

GitHub.com commands resolve to:

```text
~/workspaces/github.com/{owner}/{repo}
```

Path identity comes from the layout, not Git remotes. Identity segments are normalized to lowercase.

## Local Discovery

`path` and `delete` discover working-tree repositories exactly three levels below the workspace root:

```text
~/workspaces/*/*/*
```

A repository must contain a `.git` directory or file. Local discovery may include non-GitHub.com hosts. Bare repositories and recursive scanning outside the fixed layout are unsupported.

## `path`

With a repository reference, `path` prints its existing GitHub.com workspace path or fails if it is not cloned.

Without an argument, `path` requires a TTY and opens a searchable selection UI over discovered local repositories. The UI distinguishes repositories owned by the authenticated GitHub user from organization-owned repositories and may display the repository's Git description.

## `clone`

With a repository reference, `clone` returns an existing local repository or runs `gh repo clone` into its GitHub.com workspace path.

With an owner-only argument, `clone` requires a TTY and lists that owner's remote repositories. Without an argument, it starts from the authenticated user's repositories. The selection UI:

- displays repository descriptions
- disables already-cloned repositories
- allows switching between the authenticated user and their organizations

Remote repository lists use `gh repo list --limit 1000`. Cloning protocol and authentication are delegated to GitHub CLI.

## `create`

`create` requires a TTY. It:

1. selects the authenticated user or one of their organizations
2. asks for repository name, public/private visibility, and optional description
3. initializes a local repository on branch `main`
4. creates an empty initial commit
5. creates the GitHub repository
6. adds `origin` using the GitHub CLI protocol preference
7. pushes `main`

If local initialization or GitHub repository creation fails, the newly created local directory is removed. Failures after remote creation may require manual cleanup.

## `delete`

`delete` always requires a TTY because deletion requires confirmation.

With a repository reference, it derives the local GitHub.com workspace path. Without an argument, it selects a discovered local repository.

- If a local repository exists, the user chooses whether to delete the GitHub remote; local deletion is included.
- If no local repository exists, deletion targets the GitHub remote.
- A final confirmation summarizes remote and local effects.
- Remote deletion runs before local deletion so a remote failure leaves the local repository intact.

## Output Contract

On success, every public repository command prints exactly one absolute repository path to stdout:

- `path`, `clone`, and `create` print an existing path.
- `delete` prints the repository's former workspace path, which may no longer exist.

Interactive UI, child-process output, progress, and errors go to stderr. Failure and cancellation leave stdout empty.

## Dependencies

Runtime dependencies:

- GitHub CLI (`gh`)
- Git

Primary Go dependencies:

- Cobra for commands
- Huh, Bubble Tea, and Lip Gloss for interactive selection

## Current Boundaries

- no config file or flags
- no GitHub Enterprise cloning
- no browser, editor, shell, or lazygit launching
- no full-screen dashboard
- no cache
- no recursive local repository scan
