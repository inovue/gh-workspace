# Use lightweight GitHub-native security automation

`gh-workspace` will start with GitHub-native security automation: Dependabot for Go modules and GitHub Actions, plus CodeQL for Go. This gives a solo OSS project useful dependency and code scanning coverage without adding high-noise security workflows or external services.

Artifact attestations may be added to the release workflow once installable releases are working reliably.

**Considered Options**

- Dependabot and CodeQL: low-maintenance defaults that fit a GitHub-hosted Go CLI.
- Artifact attestations from the first release: valuable for binary provenance, but secondary to first making extension releases installable.
- Scorecard or full SLSA hardening immediately: useful for mature projects, but likely to add notification and policy noise before this project has enough release surface to justify it.
- Required signed commits: too much contributor friction for an early solo OSS project.
