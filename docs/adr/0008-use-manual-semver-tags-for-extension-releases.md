# Use manual SemVer tags for extension releases

`gh-workspace` will publish GitHub CLI Extension Releases from manually pushed SemVer tags such as `v0.1.0`. This keeps release intent explicit for solo OSS maintenance while avoiding release automation that requires stricter commit conventions, bot PRs, or generated version decisions before the project needs them.

**Considered Options**

- `workflow_dispatch`: easy to click, but separates release intent from a durable versioned Git reference.
- release-please or semantic-release: useful later, but adds process and convention overhead before contributor volume justifies it.
- release on every default-branch push: fast, but too risky for installable CLI artifacts.
