# Workspace Launcher

A GitHub CLI extension that helps developers resolve local repository paths and clone GitHub.com repositories into a predictable local workspace layout.

## Language

**Workspace Launcher**:
A tool for selecting a repository, resolving its local path, or cloning it into a local workspace.
_Avoid_: Repository Manager

**gh-workspace**:
The product and GitHub CLI extension name for this workspace launcher.
_Avoid_: generic repository-manager naming

**Workspace Switcher**:
The use case of composing a resolved repository path with `cd`, an editor, or another local workspace command.
_Avoid_: Project Manager

**Managed Root**:
The product-owned directory where **gh-workspace** clones repositories using a stable host/owner/name layout.
_Avoid_: ghq root

**Scan Root**:
A directory that **gh-workspace** searches for existing cloned repositories without taking ownership of their location.
_Avoid_: deep scan

**Active User Scope**:
The default remote repository scope from the currently authenticated GitHub CLI user.
_Avoid_: required owner config

**Direct Command**:
A subcommand such as `path` or `clone` that performs one repository-focused action.
_Avoid_: full manager command

**Selection UI**:
A small interactive repository picker used only when a command needs the user to choose a repository.
_Avoid_: dashboard TUI

**Repository Reference**:
An explicit repository argument that identifies a GitHub.com repository as `owner/repo` or a supported GitHub.com URL.
_Avoid_: repo-only shorthand

## Relationships

- A **Workspace Launcher** supports **Workspace Switcher** workflows.
- A **Workspace Switcher** operates on local repository paths.
- **gh-workspace** is a **Workspace Launcher**.
- A **Managed Root** is also a **Scan Root** by default.
- The **Active User Scope** controls which remote repositories **gh-workspace** fetches in the MVP.
- A **Direct Command** reuses the same repository discovery and action model.
- A **Direct Command** may open the **Selection UI** when no repository argument is provided.
- `path` selection only includes repositories with a local path.
- `clone` selection only includes active-user remote repositories that are not already cloned.
- A **Repository Reference** is normalized before deriving a local path or invoking clone behavior.
- `path` operates on local repositories under the host/owner/name layout and may include non-GitHub.com hosts.
- `clone` operates on GitHub.com repositories only.
- Local repository identity comes from the host/owner/name layout, not from Git remotes.
- Host, owner, and repository path identity segments are normalized to lowercase.

## Example dialogue

> **Dev:** "Should this tool open editors and browsers too?"
> **Domain expert:** "No. The **Workspace Launcher** returns local paths and clones repositories, so shell commands and editors can compose with it."

## Flagged ambiguities

- "repository manager" sounded broader than the intended scope. Resolved: the canonical term is **Workspace Launcher**; **Workspace Switcher** describes path composition into local tools.
- Product naming resolved to **gh-workspace** because the primary concept is the local workspace and the GitHub CLI command `gh workspace` is distinct from the built-in `gh repo` command.
- The clone layout is inspired by ghq, but **gh-workspace** must not default to a `ghq` directory name.
- The default **Managed Root** uses a host/owner/name layout under `~/workspaces` so GitHub.com and future GitHub Enterprise hosts can coexist.
- Remote repository discovery uses the **Active User Scope** in the MVP; explicit owner or organization selection is deferred until the command surface needs it.
