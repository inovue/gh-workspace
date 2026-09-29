# Contributing to gh-workspace

Thank you for your interest in contributing to `gh-workspace`! This document guides you through setting up your development environment, understanding the codebase structure, and submitting changes.

---

## Prerequisites

To build and test `gh-workspace`, you will need:

- **Go**: the version in `go.mod` or higher
- **GitHub CLI (`gh`)**: Installed and authenticated (`gh auth status`)
- **Git**: Installed and available on your `PATH`
- **Make**: To run build tasks

---

## Core Concepts

Before diving into the code, read the domain terms in [CONTEXT.md](./CONTEXT.md), the behavior in [docs/mvp-spec.md](./docs/mvp-spec.md), and the decisions in [docs/adr](./docs/adr). The central ideas are:

- **Workspace roots**: the directories that are scanned. The first root receives new repositories.
- **Repository identity**: host, owner, and name, read from the `origin` remote. Paths are only a placement rule.
- **Canonical path**: `{root}/{owner}/{repo}` for GitHub.com, `{root}/{host}/{owner}/{repo}` for other hosts.

---

## Project Structure

```text
├── cmd/gh-workspace/main.go        # entrypoint: TTY detection and version
├── internal/domain/
│   ├── reference.go                # repository identity, reference and remote URL parsing
│   └── layout.go                   # canonical path placement and inference
└── internal/app/
    ├── app.go                      # config, root resolution, command tree, local resolution
    ├── workspace.go                # scanning, remote identity, fuzzy matching
    ├── commands_*.go               # one file per command group
    ├── github.go                   # GitHubCLI interface and its gh implementation
    ├── selector.go                 # huh-based prompts on stderr
    └── *_test.go                   # tests with a fake GitHub CLI and real git
```

Tests replace GitHub with `fakeGitHub` and use real `git` for local behavior. They isolate git from your global configuration.

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

## Trying the Interactive UI

Point the extension at a scratch root so your real workspace is untouched:

```bash
export GH_WORKSPACE_ROOT="$(mktemp -d)"
go run ./cmd/gh-workspace clone cli/go-gh
go run ./cmd/gh-workspace path
```

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
