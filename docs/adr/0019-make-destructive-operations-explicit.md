# Make destructive operations explicit and scriptable

`delete` deletes the local working tree only. Deleting the GitHub repository requires `--remote`, including when the repository is not cloned. Previously, a reference that was not cloned locally was treated as a request to delete the GitHub repository, so a typo in a local cleanup could target a remote repository.

Before deleting a working tree, `delete` reports uncommitted changes, unpushed commits, stashes, and linked worktrees. In a terminal, the user confirms after seeing them. Remote deletion requires typing the repository name, as GitHub's web UI does. `--yes` skips prompts for scripts, but unsaved local work still needs `--force`. Remote deletion runs before local deletion, so a failure leaves the local copy intact. Missing `delete_repo` scope produces the `gh auth refresh` command that fixes it.

`create` and `clone` accept everything they prompt for as arguments or flags, so they run without a terminal. `create` follows the user's git default branch, and hands remote creation and push to `gh repo create --source --push` instead of assembling remote URLs itself.

`delete` prints nothing on stdout, because printing a path that no longer exists served no purpose. Other commands keep the rule that stdout carries only the resulting paths.
