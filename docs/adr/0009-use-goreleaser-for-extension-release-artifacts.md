# Use GoReleaser for extension release artifacts

`gh-workspace` will use GoReleaser to build and publish GitHub CLI Extension Release artifacts. This avoids hand-maintaining cross-platform build matrices, archive names, checksums, and GitHub Release upload logic in workflow YAML while leaving room for future signing, attestations, and package-manager distribution.

The initial release targets are `darwin/arm64`, `darwin/amd64`, `linux/amd64`, `linux/arm64`, and `windows/amd64`. Less common architectures can be added after real user demand appears.

**Considered Options**

- Plain GitHub Actions with `go build` and `gh release`: fewer moving parts at first, but release asset naming, checksums, and platform matrices become custom release infrastructure.
- GoReleaser: adds one tool and one configuration file, but concentrates release behavior in a conventional Go CLI release tool.
- Root executable committed to the repository: would satisfy script-extension installation, but commits generated binaries and makes source history noisier.
