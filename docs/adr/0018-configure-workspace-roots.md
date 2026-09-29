# Configure workspace roots

The workspace root defaults to `~/workspaces`. `GH_WORKSPACE_ROOT` overrides it with a list of roots separated like `PATH`. When the environment variable is unset, the repeatable git config key `gh-workspace.root` is read. The first root is primary and receives new repositories. Every root is scanned.

Git config keeps the setting persistent without a gh-workspace config file, follows ghq's use of `ghq.root`, and supports several values. The environment variable wins so a single shell or CI job can override it.

**Considered Options**

- Fixed `~/workspaces`: zero-config, but users with an existing `~/src`, `~/code`, or ghq root cannot adopt the tool without moving everything.
- A YAML config file: more room to grow, but a second configuration system for a single setting.
- `gh config set`: gh warns about unknown keys, and the key would not be scoped to the extension.
