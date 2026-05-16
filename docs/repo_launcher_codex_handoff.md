# Repository Launcher / Manager 開発引き継ぎ資料

## 0. 背景

ユーザーが求めているのは、`gh + ghq + fzf` をそのまま組み合わせることではない。

この表現はあくまで比喩であり、真の要件は次の通り。

> GitHub 上のリモートリポジトリと、ローカルに clone 済みのリポジトリを、選択式 UI で横断的に管理できる repository launcher / repository manager がほしい。

既存の `gh-dash` は PR / Issue / Notification 管理には強いが、repo 管理ツールではない。`gh-q` 系 extension は一部要件に近いが、リモート/ローカル統合状態管理、clone 済み判定、editor 起動、lazygit 起動、terminal 起動、未 clone repo の clone までを一貫して扱うものではない。

そのため、Codex には小さな CLI/TUI ツールとして実装を依頼する。

---

## 1. 作りたいもの

仮称：`gh-repos` または `repo-launcher`

GitHub CLI extension として動作する repository launcher。

```bash
gh repos
```

または standalone CLI としても動かせるようにする。

```bash
repo
repo search
repo sync
repo open
```

ただし最初の MVP では GitHub CLI extension を優先する。

---

## 2. コアコンセプト

GitHub remote repositories と local repositories を統合して 1 つの選択 UI に表示する。

例：

```text
Repository Launcher

> inovue3/chatbase-app        cloned   ~/ghq/github.com/inovue3/chatbase-app
  inovue3/agent-server        remote   not cloned
  usememos/memos              cloned   ~/ghq/github.com/usememos/memos
  cloudflare/workers-sdk      remote   not cloned
```

選択後に操作できる。

```text
Enter  open shell
z      open with Zed
l      open with lazygit
c      clone
b      open browser
i      show repository info
r      refresh remote list
q      quit
```

---

## 3. 解決したい課題

### 3.1 現状の不満

- GitHub remote repo 一覧は `gh repo list` で見られるが、ローカル clone 済み状態と統合されない。
- ローカル repo は `ghq list` や `find` で探せるが、GitHub 側の metadata と統合されない。
- `fzf` の shell function でも実現できるが、設定が散らばる。
- `gh-dash` は非常に強いが、PR / Issue / Notification が中心で、repo launcher ではない。
- `gh-q` 系は近いが、clone/open/shell/editor/lazygit/browser/status を統合する思想ではない。

### 3.2 目標

- remote と local を同じ UI で扱う。
- clone 済みかどうかを一目で分かるようにする。
- 未 clone repo は即 clone できる。
- clone 済み repo は editor / shell / lazygit で開ける。
- GitHub CLI と連携し、認証は `gh auth` に任せる。
- 個人開発でも保守しやすい小さな実装にする。

---

## 4. 想定ユーザー

- GitHub 上に複数 repo を持つ開発者。
- ローカルに多数の repo を clone している開発者。
- CLI/TUI 中心で作業するユーザー。
- Zed / VS Code / lazygit / Codex CLI / Claude Code などを repo 単位で起動したいユーザー。

---

## 5. 非目標

MVP では以下をやらない。

- PR / Issue / Notification の高度な管理。
- GitHub Projects 管理。
- 複雑な Git 操作 UI。
- GitHub 以外の forge 対応。
- GUI アプリ化。
- daemon / background sync。
- organization 横断の権限管理 UI。
- 完全な `ghq` 互換。

これらは将来拡張。

---

## 6. 技術方針

### 6.1 推奨実装言語

第一候補：Go

理由：

- GitHub CLI extension と相性がよい。
- 単一バイナリ配布しやすい。
- TUI ライブラリが強い。
- 起動が速い。
- Linux 環境で扱いやすい。

TUI ライブラリ候補：

- Bubble Tea
- Bubbles
- Lip Gloss

代替候補：Rust + ratatui

ただし MVP では Go が妥当。

### 6.2 GitHub 連携

認証は自前実装しない。

`gh` コマンドに委譲する。

使用候補：

```bash
gh repo list OWNER --limit 1000 --json nameWithOwner,description,isPrivate,isArchived,isFork,updatedAt,url,sshUrl,defaultBranchRef
```

または、より柔軟にするなら：

```bash
gh api --paginate ...
```

MVP では `gh repo list` ベースでよい。

### 6.3 ローカル repo 検出

MVP では以下のルールでローカル repo を検出する。

優先順：

1. 設定ファイルの `rootDirs`
2. `GHQ_ROOT` 環境変数
3. `ghq root` が使えるならそれ
4. fallback: `~/ghq`, `~/src`, `~/dev`, `~/workspace`

ただし無制限に deep scan しない。

`.git` ディレクトリを持つ directory を repo として扱う。

remote origin を見て GitHub repo と対応付ける。

```bash
git -C <path> remote get-url origin
```

対応する URL 例：

```text
git@github.com:owner/repo.git
https://github.com/owner/repo.git
https://github.com/owner/repo
```

正規化して `owner/repo` に変換する。

---

## 7. データモデル

### 7.1 Repository

```go
type Repository struct {
    NameWithOwner string // owner/repo
    Owner         string
    Name          string
    Description   string
    URL           string
    SSHURL        string
    IsPrivate     bool
    IsArchived    bool
    IsFork        bool
    UpdatedAt     time.Time

    LocalPath     string
    LocalStatus   LocalStatus
    OriginURL     string
}
```

### 7.2 LocalStatus

```go
type LocalStatus string

const (
    LocalStatusCloned    LocalStatus = "cloned"
    LocalStatusRemoteOnly LocalStatus = "remote"
    LocalStatusLocalOnly  LocalStatus = "local-only"
    LocalStatusUnknown    LocalStatus = "unknown"
)
```

### 7.3 Config

```yaml
rootDirs:
  - ~/ghq
  - ~/dev

owners:
  - inovue3

editor: zed
terminal: $SHELL
cloneMethod: ssh
cloneRoot: ~/ghq/github.com

commands:
  openEditor: "zed {{.LocalPath}}"
  openGit: "cd {{.LocalPath}} && lazygit"
  openCodex: "cd {{.LocalPath}} && codex"
```

設定ファイル候補：

```text
~/.config/gh-repos/config.yaml
```

または gh extension 名に寄せて：

```text
~/.config/gh-repo-launcher/config.yaml
```

---

## 8. UI 仕様

### 8.1 一覧表示

表示項目：

```text
status  visibility  nameWithOwner           updatedAt      description/path
```

例：

```text
● private  inovue3/chatbase-app      2026-05-15  AI chatbase app
○ public   usememos/memos            2026-05-10  An open-source note-taking app
◆ local    local/playground          -           ~/dev/playground
```

status 表示：

```text
● cloned
○ remote only
◆ local only
! archived
```

色は任意。ただし Nerd Font 前提にしすぎない。

### 8.2 検索

インクリメンタル検索。

検索対象：

- owner/repo
- description
- local path
- visibility
- status

### 8.3 キーバインド

MVP：

```text
/      search
enter  default action
z      open editor
s      open shell
l      open lazygit
c      clone
b      open browser
i      show info
r      refresh
?      help
q      quit
```

デフォルトアクション：

- cloned: shell or editor を開く。設定で切替。
- remote only: clone。
- local only: shell。

---

## 9. CLI 仕様

### 9.1 起動

```bash
gh repos
```

### 9.2 owner 指定

```bash
gh repos --owner inovue3
```

複数 owner：

```bash
gh repos --owner inovue3 --owner usememos
```

### 9.3 local only

```bash
gh repos --local
```

### 9.4 remote only

```bash
gh repos --remote
```

### 9.5 cloned only

```bash
gh repos --cloned
```

### 9.6 直接 action

```bash
gh repos open

gh repos open --editor zed

gh repos clone

gh repos shell

gh repos browser
```

ただし MVP では `gh repos` の TUI だけでもよい。

---

## 10. clone 仕様

### 10.1 clone path

`cloneRoot` がある場合：

```text
~/ghq/github.com/{owner}/{repo}
```

例：

```text
~/ghq/github.com/inovue3/chatbase-app
```

### 10.2 clone command

SSH：

```bash
git clone git@github.com:owner/repo.git ~/ghq/github.com/owner/repo
```

HTTPS：

```bash
git clone https://github.com/owner/repo.git ~/ghq/github.com/owner/repo
```

`cloneMethod` で切替。

### 10.3 既存 directory がある場合

- `.git` があれば cloned とみなす。
- `.git` がなければ error 表示。
- overwrite はしない。

---

## 11. キャッシュ仕様

GitHub API / `gh repo list` は毎回実行すると遅い可能性がある。

MVP では簡易 cache を導入する。

```text
~/.cache/gh-repos/repos.json
```

内容：

```json
{
  "fetchedAt": "2026-05-16T00:00:00Z",
  "owners": ["inovue3"],
  "repositories": []
}
```

TTL：

- default 10 minutes
- `--refresh` で強制更新
- TUI 内 `r` で強制更新

ローカル scan は軽ければ毎回実行してよい。

---

## 12. エラーハンドリング

### 12.1 gh がない

```text
GitHub CLI `gh` is required. Please install and run `gh auth login`.
```

### 12.2 gh 未認証

```text
GitHub CLI is not authenticated. Run: gh auth login
```

### 12.3 owner 未指定

初回起動時：

- `gh api user` で login を取得。
- それを default owner にする。

取得できない場合は config 作成案内。

### 12.4 clone 失敗

stderr を短く表示。

詳細ログは `--debug` で出す。

---

## 13. 実装ステップ

### Step 1: project skeleton

- Go module 作成。
- Cobra で CLI 作成。
- `gh repos` extension として動く repository 名にする。
- repo 名は `gh-repos` が自然。

### Step 2: config loader

- YAML config 読み込み。
- default 値定義。
- `~` 展開。
- env var 展開。

### Step 3: remote repo fetcher

- `gh repo list` を exec する。
- JSON parse。
- owner 複数対応。
- cache 導入。

### Step 4: local repo scanner

- rootDirs 配下を scan。
- `.git` 検出。
- `origin` URL 取得。
- GitHub URL を `owner/repo` に normalize。

### Step 5: merger

remote repo と local repo を `owner/repo` で merge。

結果：

- remote + local => cloned
- remote only => remote
- local only => local-only

### Step 6: TUI

Bubble Tea で一覧、検索、選択、help を実装。

MVP では simple list でよい。

### Step 7: actions

- open shell
- open editor
- open lazygit
- open browser
- clone
- refresh

### Step 8: release

- GitHub Actions で GoReleaser。
- Linux x86_64 を最低対応。
- gh extension install 可能にする。

---

## 14. MVP 受け入れ条件

以下を満たせば MVP 完了。

1. `gh repos` で TUI が起動する。
2. GitHub remote repo 一覧が表示される。
3. local clone 済み repo が表示される。
4. remote と local が同一 repo として merge される。
5. cloned / remote / local-only が判別できる。
6. 検索できる。
7. cloned repo を Zed で開ける。
8. cloned repo で shell を開ける。
9. cloned repo で lazygit を開ける。
10. remote-only repo を clone できる。
11. browser で GitHub repo を開ける。
12. config で rootDirs / owner / editor / cloneRoot を変更できる。

---

## 15. 将来拡張

### 15.1 GitHub organization 対応

```yaml
owners:
  - inovue3
  - my-org
```

### 15.2 Starred repositories

`gh api user/starred` から取得。

### 15.3 Recently used repositories

起動履歴を記録して sort に使う。

### 15.4 Worktree 対応

repo ごとに worktree 一覧表示。

### 15.5 PR / Issue 連携

`gh-dash` と競合しない範囲で、選択 repo の PR / Issue count だけ表示。

### 15.6 Codex / Claude Code 起動

```yaml
commands:
  openCodex: "cd {{.LocalPath}} && codex"
  openClaude: "cd {{.LocalPath}} && claude"
```

TUI keybind：

```text
x  open codex
C  open claude
```

---

## 16. 既存ツールとの差分

### gh-dash

強い領域：

- PR
- Issue
- Notification
- Review workflow

今回作るもの：

- Repository selection
- Remote/local repository state
- Clone/open/shell/editor/lazygit

結論：競合ではなく補完関係。

### gh-q 系

強い領域：

- GitHub repo を選択して clone
- ghq 風の repo 管理

今回作るもの：

- clone 済み状態と remote repo を統合表示
- editor / shell / lazygit / browser / clone まで統一
- config / cache / local-only 対応

### shell function + fzf

強い領域：

- 軽量
- 自由

今回作るもの：

- 設定の一元化
- 状態表示
- GitHub metadata 統合
- gh extension として配布可能

---

## 17. 最初に作ってほしいファイル構成案

```text
gh-repos/
  cmd/
    gh-repos/
      main.go
  internal/
    config/
      config.go
    github/
      fetcher.go
    local/
      scanner.go
    repo/
      model.go
      merge.go
    tui/
      app.go
      list.go
      actions.go
    shell/
      exec.go
  go.mod
  go.sum
  README.md
  .goreleaser.yaml
  .github/
    workflows/
      release.yml
```

---

## 18. README に書くべき概要

```md
# gh-repos

A GitHub CLI extension for browsing, cloning, and opening GitHub repositories across remote and local state.

It combines:

- GitHub remote repository discovery
- Local clone detection
- Fuzzy/TUI repository selection
- Editor / shell / lazygit / browser actions
- Clone missing repositories

```bash
gh extension install <owner>/gh-repos
gh repos
```
```

---

## 19. Codex への作業指示テンプレート

以下を Codex に渡す。

```text
この資料に沿って、GitHub CLI extension `gh-repos` を Go で実装してください。

最優先は MVP です。

MVP 要件：
- `gh repos` で TUI 起動
- `gh repo list` から remote repo 取得
- rootDirs から local repo scan
- remote/local を owner/repo で merge
- cloned / remote / local-only を一覧表示
- fuzzy search
- cloned repo を editor / shell / lazygit で開く
- remote-only repo を clone
- browser で repo URL を開く
- config file: `~/.config/gh-repos/config.yaml`

実装は Go + Bubble Tea + Cobra を優先してください。
小さく、保守しやすく、過剰抽象化しないでください。
まずは Linux x86_64 で動けばよいです。
```

---

## 20. 注意点

- `ghq` は必須にしない。あくまで cloneRoot の default として ghq 風 layout を採用するだけ。
- `fzf` も必須にしない。TUI 内検索で代替する。
- `gh` は必須。認証と GitHub API は `gh` に委譲する。
- `gh-dash` の再実装をしない。
- repo launcher に集中する。
- 最初から多機能にしすぎない。
- local scan は重くなりすぎないよう rootDirs を限定する。
- private repo 情報をログに出しすぎない。

