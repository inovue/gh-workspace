# Use host/owner/name clone layout

`gh-workspace` will default to cloning repositories under `~/workspaces/{host}/{owner}/{repo}`. This keeps the common GitHub.com case predictable while preserving a clean path for future GitHub Enterprise support without relocating existing clones.

**Considered Options**

- `~/workspaces/{owner}/{repo}`: shorter, but assumes GitHub.com forever.
- `~/workspaces/{host}/{owner}/{repo}`: slightly longer, but makes host identity explicit.
