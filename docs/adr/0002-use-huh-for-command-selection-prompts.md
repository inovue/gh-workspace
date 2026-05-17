# Use Huh for command selection prompts

`gh-workspace` will use Go with Charmbracelet Huh for the small interactive selection prompts used by commands that need a repository choice. This avoids external tools such as `fzf`, keeps selection behavior inside the binary, and is simpler than maintaining a custom Bubble Tea model for a command-first extension.

Huh supports configuring form output with `WithOutput`, so selection prompts can be kept on stderr while command results remain machine-readable on stdout.

**Considered Options**

- Bubble Tea directly: powerful and active, but too much UI surface for a command-first extension that only needs repository selection.
- external `fzf`: lightweight, but would add a runtime dependency and make behavior depend on user shell tooling.
- `promptui`: small and suitable, but much less active than Huh.
- TypeScript with Ink: attractive for React-style development, but less direct for single-binary GitHub CLI extension distribution.
- TypeScript with OpenTUI or other newer frameworks: promising, but less mature for a small personal tool that should remain easy to maintain.
