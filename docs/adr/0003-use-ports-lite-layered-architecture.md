# Use ports-lite layered architecture

`gh-repos` will use a small layered architecture with `cmd/gh-repos`, `internal/domain`, `internal/app`, `internal/adapters`, and `internal/ui`. This keeps the domain and use cases testable without introducing a full clean-architecture package tree that would be too heavy for a personal GitHub CLI extension.

**Considered Options**

- Many narrow packages such as `github`, `local`, `cache`, `actions`, and `shell`: clearer at larger scale, but too much navigation and interface overhead for the MVP.
- A flat CLI/UI implementation: fastest initially, but would couple command prompts to clone, scan, and GitHub command behavior.

**Consequences**

Interfaces are introduced only at the `app` boundary when a dependency must be faked in tests. Adapter packages may be split later if a file becomes large or a dependency develops its own lifecycle.
