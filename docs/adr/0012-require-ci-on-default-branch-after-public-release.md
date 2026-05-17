# Require CI on the default branch after public release

After `gh-workspace` becomes a public OSS project with installable releases, the default branch will require the minimal CI workflow before merge. Required reviews will not be enabled initially because this is a solo-maintained project, and maintainer bypass will remain available for emergency fixes.

**Considered Options**

- No branch protection: lowest friction, but weakens trust once external users install release artifacts.
- Require CI only: keeps the default branch protected against obvious regressions without pretending solo review adds value.
- Require reviews: useful for teams, but mostly ceremony for a solo maintainer until regular contributors exist.
- Disable maintainer bypass: stricter, but makes emergency recovery harder than this project's risk profile justifies.
