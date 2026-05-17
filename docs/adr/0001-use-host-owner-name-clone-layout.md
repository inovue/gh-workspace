# Use host/owner/name clone layout

`gh-repos` will default to cloning repositories under `~/repos/{host}/{owner}/{repo}`. This keeps the common GitHub.com case predictable while preserving a clean path for future GitHub Enterprise support without relocating existing clones.

**Considered Options**

- `~/repos/{owner}/{repo}`: shorter, but assumes GitHub.com forever.
- `~/repos/{host}/{owner}/{repo}`: slightly longer, but makes host identity explicit.
