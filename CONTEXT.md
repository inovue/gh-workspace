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
The predictable local directory tree in which repositories are organized by host, owner, and name.
_Avoid_: ghq root

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
An explicit argument that identifies a GitHub.com repository by owner and name.
_Avoid_: repo-only shorthand

**Repository Path**:
The absolute workspace location associated with a repository.
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
