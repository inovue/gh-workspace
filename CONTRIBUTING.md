# Contributing to gh-workspace

Thank you for your interest in contributing to `gh-workspace`! This document guides you through setting up your development environment, understanding the codebase structure, and submitting changes.

---

## Prerequisites

To build and test `gh-workspace`, you will need:

- **Go**: 1.23.0 or higher
- **GitHub CLI (`gh`)**: Installed and authenticated (`gh auth status`)
- **Git**: Installed and available on your `PATH`
- **Make**: To run build tasks

---

## Core Concepts

Before diving into the code, please align with the domain terms defined in the [CONTEXT.md](file:///home/inovue/workspaces/github.com/inovue/gh-workspace/CONTEXT.md):

- **Workspace Launcher**: A tool (this extension) for selecting a repository, resolving its local path, or cloning it.
- **Managed Root**: The directory (`~/workspaces`) where `gh-workspace` clones repositories using a stable `host/owner/name` layout.
- **Repository Reference**: An argument that identifies a GitHub repository (e.g., `owner/repo`, HTTPS/SSH URLs).

---

## Project Structure

The project follows a standard Go layout:

```text
├── cmd/
│   └── gh-workspace/
│       └── main.go         # App entrypoint (initializes config & parses TTY status)
├── internal/
│   ├── app/
│   │   ├── app.go          # Business logic, Cobra command routing, & TUI (huh/bubbletea)
│   │   ├── path_test.go    # Unit/Integration tests with mocks
│   │   └── skipping_select_test.go
│   └── domain/
│       ├── reference.go    # GitHub Repository Reference parsing and validation logic
│       └── reference_test.go
├── docs/                   # Specifications, ADRs, and other documentations
└── Makefile                # Make tasks for building and testing
```

### Key Commands & Implementation

The core logic resides in `internal/app/app.go` and is structured around these main commands:

- **`path`**: Scans the local workspace directory (`~/workspaces`) for existing Git repositories and prints the matching absolute path. If no repository is specified, it provides an interactive selection using `huh`.
- **`clone`**: Clones a remote repository into the predictable workspace layout. If no argument is provided, it fetches available repositories from GitHub and offers a search/select UI.
- **`create`**: Guides the user through creating a new repository. It prompts for owner (personal account or organization), name, visibility, and description. It then initializes a local git repo (via `git init`), commits, creates the GitHub repository (via `gh repo create`), and pushes the initial commit.
- **`delete`**: Safely deletes a repository. It prompts the user to confirm whether to delete the local directory, the remote GitHub repository, or both.

---

## Local Development

### 1. Build the Binary
To compile the application to `bin/gh-workspace`:
```bash
make build
```

### 2. Run Tests
To run unit and integration tests:
```bash
make test
```

### 3. Test as a Local GitHub CLI Extension
To test the extension live inside the `gh` tool:

1. **Ensure the checkout directory is named `gh-workspace`**. GitHub CLI looks for this specific directory name to register the extension.
2. Build the extension binary in the root directory:
   ```bash
   make extension
   ```
3. Install the current directory as a local extension:
   ```bash
   gh extension install .
   ```
4. Now you can run it:
   ```bash
   gh workspace --help
   ```

To uninstall the local version later:
```bash
gh extension remove workspace
```

---

## Automated UI Testing (Monkey Test)

The project includes an interactive TUI built with [huh](https://github.com/charmbracelet/huh). To verify the rendering and navigation flow without manual keypresses, you can run a **Monkey Test**:

```bash
go run ./cmd/gh-workspace monkey
```

This hidden command triggers a simulated interaction loop inside the TUI selection list.

---

## Release Process

Releases are automated using [GoReleaser](https://goreleaser.com/) via GitHub Actions.

To publish a new release:
1. Create a SemVer tag:
   ```bash
   git tag v0.1.0
   ```
2. Push the tag to GitHub:
   ```bash
   git push origin v0.1.0
   ```

The release workflow will automatically compile the binaries for macOS, Linux, and Windows and upload them to the GitHub Release.
