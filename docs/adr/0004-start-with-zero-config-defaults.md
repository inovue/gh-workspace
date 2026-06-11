---
status: superseded by ADR-0014
---

# Start with zero-config defaults

`gh-workspace` will start without requiring a config file. The default remote scope is the active GitHub CLI user and the default root is `~/workspaces`.

**Considered Options**

- YAML config with owners, root, editor, and clone method: flexible, but too much setup and migration surface for the MVP.
- Zero-config with no MVP flags: keeps the GitHub CLI extension simple and lets `gh auth` and `gh config` provide the user context.

**Consequences**

Organization repositories are out of scope for the MVP. Persistent config or owner flags may be added later after real use shows which settings deserve to exist.
