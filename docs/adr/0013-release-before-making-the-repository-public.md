# Release before making the repository public

`gh-workspace` will publish and verify its first installable GitHub CLI Extension Release before the repository is made public. The first public release version will be `v0.1.0`, giving users an installable initial version without implying a stable `v1.0.0` contract.

This avoids exposing a public README whose primary installation command fails, while still leaving room for CLI behavior to evolve during the `v0.x` series.
