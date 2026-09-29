# Accept host-qualified references and fuzzy queries

Explicit repository references accept `owner/repo`, `host/owner/repo`, HTTPS and SSH clone URLs, and browser URLs with extra path segments such as `/pull/1`. `GH_HOST` sets the host for `owner/repo`, matching GitHub CLI. Remote operations pass the host to `gh`, so GitHub Enterprise works wherever `gh` is authenticated.

Commands that act on local repositories also accept free-form queries. An explicit reference must match a local identity exactly and never falls back to fuzzy matching, so scripts cannot resolve the wrong repository. Other queries rank exact names, then name prefixes, then substrings, then subsequences. When several repositories share the best rank, a picker opens in a terminal and the command fails with the candidates otherwise.

**Considered Options**

- GitHub.com references only: simpler, but the layout already reserves host directories, and `gh` already handles host authentication.
- Fuzzy matching everywhere: convenient, but `path cli/cli` could silently return `cli/cli-extra`.
