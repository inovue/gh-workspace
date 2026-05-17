# Accept GitHub.com repository references

`gh-workspace` will accept explicit repository arguments as either `owner/repo` or supported GitHub.com URL forms, then normalize them to a repository identity before deriving local paths or cloning. This keeps direct commands convenient without accepting ambiguous repo-only shorthands.

**Considered Options**

- `owner/repo` only: simplest, but inconvenient when copying URLs from GitHub.
- `owner/repo` plus GitHub.com URLs: still bounded and useful.
- arbitrary hosts and GitHub Enterprise URLs: consistent with the path layout, but too much remote-clone behavior for the MVP.
