# Identify repositories by remote and drop the default host directory

`gh-workspace` places GitHub.com repositories at `{root}/{owner}/{repo}` and repositories from other hosts at `{root}/{host}/{owner}/{repo}`. A local repository's identity comes from its `origin` remote, not from its path; the path layout is only the placement rule for new clones and creations. When no remote identifies a repository, its identity is inferred from the layout, including the legacy `{root}/github.com/{owner}/{repo}` layout.

GitHub owner names cannot contain `.`, and hostnames always do, so a top-level directory is unambiguously a host or an owner. The `github.com` directory carried no information for a GitHub CLI extension that almost only clones from GitHub.com, and made every path one level deeper.

Folders keep the repository's canonical casing on GitHub. Identity comparisons are case-insensitive. The earlier lowercase normalization made paths diverge from repository names and from folders created by other tools.

**Considered Options**

- Keep `{root}/{host}/{owner}/{repo}` (ghq layout): shares a root with ghq, but always adds a constant directory. ghq users get the same effect by setting the root to `~/ghq/github.com`.
- Flat `{root}/{repo}`: shortest, but forks and same-named repositories across organizations (`docs`, `infra`, `.github`) collide.
- Owner/repo with host only for non-default hosts, identity by path: simple, but repositories renamed or transferred on GitHub, or placed by hand, become invisible.
- Owner/repo with host only for non-default hosts, identity by remote: chosen.

**Consequences**

- `path`, `clone`, `create`, and `delete` find repositories anywhere under the configured roots, including legacy and renamed ones.
- `migrate` moves repositories to their canonical paths, and does nothing without `--apply`.
- Linked worktrees are recognized and share their main working tree's identity. Commands that resolve an identity prefer the main working tree.
- Scanning reads `.git/config` directly, up to three directory levels below each root, and skips hidden directories.
