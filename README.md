# gh-workspace

[![GitHub CLI Extension](https://img.shields.io/badge/gh--extension-installed-blue.svg)](https://cli.github.com/)
[![Go Version](https://img.shields.io/github/go-mod/go-version/inovue/gh-workspace)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

**gh-workspace** is a GitHub CLI (`gh`) extension designed to keep all your cloned Git repositories organized in one predictable, standardized directory layout:

```text
~/workspaces/github.com/{owner}/{repository}
```

By enforcing a consistent directory structure, it eliminates "repository drift" (where projects end up scattered across `~/src`, `~/Downloads`, or `~/Desktop`) and enables clean automation for terminal workflows, editor setups, and scripting.

---

## 🌟 Why gh-workspace?

- **Zero Friction Directory Switching**: Navigate straight to any repository without searching or tab-completing complex directory trees.
- **Predictable Shell Scripts**: Write scripts that reference local paths knowing they will resolve identically across different machines.
- **Rich Interactive TUI**: Built with [huh](https://github.com/charmbracelet/huh) for beautiful filtering, owner-switching, and single-keypress interactions.
- **Unified Local & Remote Workflows**: Easily create or delete repositories both locally and on GitHub in a single command.

---

## 🚀 Quick Start

Follow these steps to set up and try your first command in 2 minutes.

### 1. Prerequisites
Make sure you have the following installed:
- **GitHub CLI (`gh`)** (version 2.0 or higher) and authenticated (`gh auth login`)
- **Git** available in your system path

### 2. Installation
Install the extension via the GitHub CLI:
```bash
gh extension install inovue/gh-workspace
```

Verify the installation:
```bash
gh workspace --help
```

### 3. Basic Operations

* **Find the path of a repository:**
  If you have a repository cloned locally under the workspaces folder, print its path:
  ```bash
  gh workspace path cli/cli
  # Output: /home/username/workspaces/github.com/cli/cli
  ```

* **Clone a repository into the workspace structure:**
  ```bash
  gh workspace clone cli/cli
  # Clones to ~/workspaces/github.com/cli/cli and outputs the path.
  # If it is already cloned, it just prints the path instantly.
  ```

---

## 🛠️ Feature Walkthrough & Commands

`gh-workspace` provides four main commands. 

### 1. `gh workspace path [repository]`
Resolves the absolute path for a local repository.

- **With Argument:** `gh workspace path owner/repo` (supports full URLs, SSH shortcuts).
  - *Example:* `gh workspace path git@github.com:cli/cli.git` → `/home/username/workspaces/github.com/cli/cli`
- **Without Argument (Interactive TUI):** Opens a list of all repositories found inside your `~/workspaces` directory. Search and select to output its path.

### 2. `gh workspace clone [repository|owner]`
Ensures a remote repository is cloned into your workspace layout.

- **With Repository Argument:** Clones the repository to its standardized location and outputs the absolute path. If it already exists, it outputs the path immediately without re-cloning.
- **With Owner Argument:** Opens the TUI listing remote repositories owned by the specified user or organization. Already-cloned items are dimmed and disabled.
- **Without Argument:** Opens the TUI listing remote repositories for your personal account.
  - *Tip:* Select `🔄 Switch Owner...` at the top of the TUI list to browse repositories belonging to your other GitHub Organizations.

### 3. `gh workspace create`
Creates a brand new repository locally and on GitHub in one workflow.

Run `gh workspace create` to launch an interactive setup:
1. **Select Owner**: Choose your personal account or one of your GitHub organizations.
2. **Repository Name**: Enter the name (validated for GitHub rules).
3. **Visibility**: Select `Public` or `Private`.
4. **Description**: Optionally input a brief description.

Once confirmed, the command automatically:
- Creates the local directory under `~/workspaces/github.com/{owner}/{name}`.
- Runs `git init`, checks out a `main` branch, and creates an empty initial commit.
- Creates the remote repository on GitHub with the chosen visibility.
- Sets up the `origin` remote and pushes the `main` branch.
- Outputs the new repository path.

### 4. `gh workspace delete [repository]`
Safely cleans up a repository both locally and remotely.

- **Usage:** `gh workspace delete [repository]` (omitting the argument opens a TUI of your local workspace repos).
- **Workflow:**
  1. Prompts you to select whether you want to delete the **local directory**, the **remote repository on GitHub**, or **both**.
  2. Displays a summary of the actions to be taken.
  3. Asks for a final confirmation.
  4. Deletes the selected targets and prints completion messages.

---

## 💻 Integrations & Workflows

`gh-workspace` is designed to be composed with other command-line tools.

### GitHub CLI Aliases

You can set up custom shortcuts directly within the GitHub CLI using `gh alias set --shell`.

#### 1. Open in VS Code (`gh ow <repo>`)
Clones the repository (if not already local) and opens it in Visual Studio Code immediately:
```bash
gh alias set --shell ow 'code "$(gh workspace clone "$1")"'
```
*Usage:*
```bash
gh ow cli/cli
```

#### 2. Run Tests Remotely (`gh wtest <repo>`)
Executes `make test` inside the repository without manually changing directories:
```bash
gh alias set --shell wtest 'make -C "$(gh workspace path "$1")" test'
```
*Usage:*
```bash
gh wtest cli/cli
```

#### 3. List Repository Contents (`gh wlist <repo>`)
Lists all files in the target repository layout directory:
```bash
gh alias set --shell wlist 'ls -la "$(gh workspace path "$1")"'
```
*Usage:*
```bash
gh wlist cli/cli
```

> [!NOTE]
> **Shell Navigation Limitation**
> GitHub CLI aliases run in an isolated subshell. This means you cannot use a `gh` alias to change the working directory of your current terminal session (e.g., trying to run `cd` inside a `gh` alias will not affect your active shell).
>
> If you want a quick-navigation command to change your directory, you must define a **shell function** inside your shell configuration (like `~/.bashrc` or `~/.zshrc`):
>
> ```bash
> # Usage: cdw [owner/repo]
> cdw() {
>   local target
>   target=$(gh workspace clone "$@") && cd "$target"
> }
> ```

### Script-Friendly Design
`gh-workspace` adheres strictly to UNIX philosophy:
- **`stdout`** is reserved exclusively for the resolved absolute path of the repository.
- **`stderr`** is used for interactive prompts, progress indicators, status logs, and error messages.

This guarantees that command substitution (e.g. `path="$(gh workspace clone cli/cli)"`) stays clean and doesn't capture interactive UI text or warnings.

---

## ⚙️ Design & Specifications

- **Normalized Paths**: Hostnames, owners, and repository names are always parsed and saved in lowercase to prevent case-sensitivity issues on different OS filesystems.
- **Git Protocol**: The extension respects your GitHub CLI protocol settings (SSH vs HTTPS) when resolving remote URLs for cloning.
- **Standard Layout**: The extension works inside `~/workspaces` and supports arbitrary hosts under it (e.g., `~/workspaces/github.com/` or `~/workspaces/gitlab.com/`). However, cloning is currently optimized for GitHub.

---

## 🤝 Contributing

We welcome contributions to fix bugs, add features, or improve documentation!
Please refer to [CONTRIBUTING.md](file:///home/inovue/workspaces/github.com/inovue/gh-workspace/CONTRIBUTING.md) for information on setting up the local codebase, running tests, and understanding project architecture.
