# gh-repos-extension

GitHub CLI extension name: `gh-repos`.

Small GitHub CLI extension for resolving local repository paths and cloning GitHub.com repositories into `~/repos/{host}/{owner}/{repo}`.

## Commands

```bash
gh repos path [repository]
gh repos clone [repository]
```

Explicit repositories accept `owner/repo` and supported GitHub.com URL forms. Outputs are designed for shell composition: success writes exactly one absolute path to stdout; UI, progress, and errors go to stderr.

## Build

```bash
make build
```

The binary is written to `bin/gh-repos`.

## Test

```bash
make test
```
