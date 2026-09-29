---
status: superseded by ADR-0020
---

# Use minimal CI for pushes and PRs

`gh-workspace` will run a minimal CI workflow on pushes and pull requests: `go test ./...`, `go vet ./...`, and a normal build of `./cmd/gh-workspace`. This gives solo OSS development a useful safety net without adding lint configuration, coverage gates, or OS matrices before the command surface needs them.

The release workflow will also run tests before publishing extension artifacts.

**Considered Options**

- Test, vet, and build: catches common regressions while staying fast and low-maintenance.
- Add golangci-lint immediately: useful later, but adds configuration and rule debates before the project needs another policy layer.
- Add coverage gates: easy to game and too noisy for a small CLI at this stage.
- Run an OS matrix for every change: useful once OS-specific behavior appears, but unnecessary cost for the current implementation.
