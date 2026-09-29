# Ship shell integration and run CI on every platform

`gh workspace shell-init` prints a shell function (default `ws`) for bash, zsh, fish, and PowerShell. The function changes directory to the path printed by `path`, `clone`, or `create`, and completes repository names. A child process cannot change its parent's directory, so every "jump to repo" tool ships such a function. Asking users to write their own, as the README used to, was the largest adoption obstacle for daily use.

`list` prints workspace names, absolute paths, or JSON, making the workspace scriptable (for example, bulk `git pull`).

CI runs tests on Linux, macOS, and Windows. This supersedes ADR-0010's single-OS CI, because path handling, root expansion, and PowerShell integration now behave differently per OS.

**Considered Options**

- Document a hand-written `cd` function: no code to maintain, but every user repeats the work and completion is missing.
- Make `gh workspace` spawn a subshell in the repository: works without shell config, but nests shells and loses shell state.
