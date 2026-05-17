# gh-workspace

GitHub CLI extension name: `gh-workspace`.

Small GitHub CLI extension for resolving local repository paths and cloning GitHub.com repositories into `~/workspaces/{host}/{owner}/{repo}`.

## Commands

```bash
gh workspace path [repository]
gh workspace clone [repository]
```

Explicit repositories accept `owner/repo` and supported GitHub.com URL forms. Outputs are designed for shell composition: success writes exactly one absolute path to stdout; UI, progress, and errors go to stderr.

## Build

```bash
make build
```

The binary is written to `bin/gh-workspace`.

## Local extension install

The local checkout directory must be named `gh-workspace`; GitHub CLI uses the repository name to locate the root executable.

```bash
make extension
gh extension install .
gh workspace path [repository]
```

## Test

```bash
make test
```
