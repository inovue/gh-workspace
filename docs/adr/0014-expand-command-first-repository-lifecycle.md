# Expand the command-first repository lifecycle

`gh-workspace` keeps its command-first interface and expands the supported repository lifecycle to `path`, `clone`, `create`, and `delete`. Interactive selection remains subordinate to a direct command: it chooses local repositories, remote repositories, or repository owners only when the command needs that choice.

Owner selection includes the authenticated GitHub user and their organizations. This replaces the earlier active-user-only MVP scope while retaining zero-config defaults and script-friendly path output.

**Considered Options**

- Keep only `path` and `clone`: smaller surface, but leaves common workspace creation and cleanup outside the tool.
- Build a full-screen repository dashboard: broader interaction model, but weakens command composition and adds unnecessary UI state.
- Expand direct commands with focused prompts: covers the repository lifecycle while preserving predictable command behavior.
