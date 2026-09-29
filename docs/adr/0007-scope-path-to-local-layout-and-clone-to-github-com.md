---
status: superseded by ADR-0016
---

# Scope path to local layout and clone to GitHub.com

`path` will operate on local repositories found under the `~/workspaces/{host}/{owner}/{repo}` layout, including non-GitHub.com hosts. `clone` will operate only on GitHub.com repositories in the MVP.

This keeps local path resolution host-agnostic while avoiding the authentication and host-selection complexity of cloning GitHub Enterprise repositories.
