# Use command-first CLI with optional selection

`gh-repos` will expose repository actions as direct subcommands and open an interactive selection prompt only when the repository argument is omitted. The MVP commands are `path` and `clone`. This keeps the extension scriptable and predictable while preserving the convenience of choosing from discovered repositories.

**Considered Options**

- Default full-screen launcher: convenient, but too much UI for a simple GitHub CLI extension.
- Broader command set with `list`, `open`, `shell`, `git`, and `browser`: convenient, but most behavior is already composable with `path`, `gh`, the shell, or the user's editor.
- Direct commands only: simple, but requires remembering exact repository names.
- Command-first with optional selection: keeps direct execution fast and uses UI only where it removes friction.

**Consequences**

Commands that require changing the caller's shell directory are not included. `path` prints the repository path instead, so users can compose it with their shell: `cd "$(gh repos path)"`.
Browser opening is out of scope. `path` and `clone` print absolute local paths to stdout so they compose with `cd`, editors, and other local workspace tools.
When `clone` receives an explicit `owner/repo`, it may attempt `gh repo clone` even if that repository is not present in the active-user repository list. This keeps direct clone useful for accessible organization or collaborator repositories without adding owner flags to the MVP.
