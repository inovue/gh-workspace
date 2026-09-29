# Workspace Launcher

A GitHub CLI extension for managing repositories through predictable local workspace paths.

## Language

**Workspace Launcher**:
A tool for selecting, locating, cloning, creating, or deleting repositories through a local workspace.
_Avoid_: Repository Manager, Project Manager

**gh-workspace**:
The product and GitHub CLI extension name for this workspace launcher.
_Avoid_: generic repository-manager naming

**Workspace**:
The predictable local directory tree, under one or more workspace roots, in which repositories are organized by owner and name, with a host directory only for hosts other than GitHub.com.
_Avoid_: ghq root

**Workspace Root**:
A configured directory scanned for local repositories. The first root is primary and receives new repositories.
_Avoid_: base dir

**Repository Identity**:
The host, owner, and name of a repository, read from its `origin` remote or, when absent, inferred from its path. Compared case-insensitively.
_Avoid_: checkout name

**Canonical Path**:
The path the layout assigns to a repository identity: `{root}/{owner}/{repo}` for GitHub.com, `{root}/{host}/{owner}/{repo}` otherwise.
_Avoid_: ghq path

**Local Repository**:
A working-tree repository present in the workspace.
_Avoid_: cloned state

**Remote Repository**:
A GitHub repository accessible through the authenticated GitHub CLI user.
_Avoid_: cloud repository

**Repository Owner**:
The GitHub user or organization that owns a remote repository.
_Avoid_: account

**Repository Reference**:
An explicit argument that identifies a repository: `owner/repo`, `host/owner/repo`, or a URL. It must match exactly.
_Avoid_: repo-only shorthand

**Query**:
A free-form argument matched fuzzily against local repositories.
_Avoid_: search term

**Shell Function**:
The function printed by `shell-init` that changes the caller's directory to a resolved repository path.
_Avoid_: alias

**Repository Path**:
The absolute location of a local repository. It is usually, but not necessarily, the canonical path.
_Avoid_: checkout ID

**Selection UI**:
A small interactive repository or owner picker used when a command needs user choice.
_Avoid_: dashboard TUI

**Direct Command**:
A repository-focused subcommand that performs one workspace action.
_Avoid_: full-screen launcher

**GitHub CLI Extension Release**:
The installable release unit consumed by GitHub CLI.
_Avoid_: source-only install
