package app

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/spf13/cobra"
)

var functionNamePattern = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_-]*$`)

// passthroughCommands run without changing directory.
const passthroughCommands = "list ls delete rm root migrate shell-init help -h --help --version"

func (a *App) shellInitCommand() *cobra.Command {
	var name string
	cmd := &cobra.Command{
		Use:       "shell-init [bash|zsh|fish|pwsh]",
		Short:     "Print a shell function that cd's into repositories",
		ValidArgs: []string{"bash", "zsh", "fish", "pwsh"},
		Long: `Print a shell function (named "ws" by default) with tab completion.

  ws                 pick a repository and cd into it
  ws <query>         cd into the best match (picker when ambiguous)
  ws clone <repo>    clone if needed, then cd into it
  ws create <name>   create, then cd into it
  ws <other>         run gh workspace <other>

After "ws delete" removes the current directory, the function returns to the
workspace root. Without an argument, the shell is detected from $SHELL.`,
		Example: `  # ~/.bashrc or ~/.zshrc
  eval "$(gh workspace shell-init)"

  # ~/.config/fish/config.fish
  gh workspace shell-init fish | source

  # PowerShell $PROFILE
  Invoke-Expression (& gh workspace shell-init pwsh | Out-String)`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(_ *cobra.Command, args []string) error {
			if !functionNamePattern.MatchString(name) {
				return fmt.Errorf("invalid function name: %s", name)
			}
			shell := firstArg(args)
			if shell == "" {
				shell = strings.TrimSuffix(filepath.Base(a.config.Getenv("SHELL")), ".exe")
			}
			script, ok := shellScripts[shell]
			if !ok {
				return fmt.Errorf("unsupported shell %q; choose bash, zsh, fish, or pwsh", shell)
			}
			script = strings.ReplaceAll(script, "__NAME__", name)
			script = strings.ReplaceAll(script, "__PASSTHROUGH__", passthroughFor(shell))
			fmt.Fprint(a.stdout, script)
			return nil
		},
	}
	cmd.Flags().StringVar(&name, "name", "ws", "name of the shell function")
	return cmd
}

func passthroughFor(shell string) string {
	words := strings.Fields(passthroughCommands)
	switch shell {
	case "bash", "zsh":
		return strings.Join(words, "|")
	case "pwsh":
		return "'" + strings.Join(words, "','") + "'"
	default:
		return strings.Join(words, " ")
	}
}

const posixFunction = `__NAME__() {
  local __ws_out __ws_status
  case "$1" in
    __PASSTHROUGH__)
      command gh workspace "$@"
      __ws_status=$?
      [ -d "$PWD" ] || cd "$(command gh workspace root)"
      return $__ws_status ;;
    clone|create|path) __ws_out="$(command gh workspace "$@")" || return ;;
    *) __ws_out="$(command gh workspace path "$@")" || return ;;
  esac
  [ -n "$__ws_out" ] && cd "${__ws_out##*$'\n'}"
}
`

var shellScripts = map[string]string{
	"bash": posixFunction + `
___NAME___complete() {
  local cur="${COMP_WORDS[COMP_CWORD]}" words
  if [ "$COMP_CWORD" -eq 1 ]; then
    words="clone create path list delete root migrate $(command gh workspace list 2>/dev/null)"
  elif [ "$COMP_CWORD" -eq 2 ] && [[ "${COMP_WORDS[1]}" =~ ^(path|delete|rm)$ ]]; then
    words="$(command gh workspace list 2>/dev/null)"
  fi
  COMPREPLY=($(compgen -W "$words" -- "$cur"))
}
complete -F ___NAME___complete __NAME__
`,
	"zsh": posixFunction + `
___NAME___complete() {
  local -a items
  if (( CURRENT == 2 )); then
    items=(clone create path list delete root migrate ${(f)"$(command gh workspace list 2>/dev/null)"})
  elif (( CURRENT == 3 )) && [[ $words[2] == (path|delete|rm) ]]; then
    items=(${(f)"$(command gh workspace list 2>/dev/null)"})
  fi
  compadd -a items
}
(( $+functions[compdef] )) && compdef ___NAME___complete __NAME__
`,
	"fish": `function __NAME__ --description 'cd into a gh workspace repository'
  switch "$argv[1]"
    case __PASSTHROUGH__
      command gh workspace $argv
      set -l ws_status $status
      test -d "$PWD"; or cd (command gh workspace root)
      return $ws_status
    case clone create path
      set -l ws_out (command gh workspace $argv); or return
      test -n "$ws_out[-1]"; and cd $ws_out[-1]
    case '*'
      set -l ws_out (command gh workspace path $argv); or return
      test -n "$ws_out[-1]"; and cd $ws_out[-1]
  end
end
complete -c __NAME__ -f -n __fish_use_subcommand -a 'clone create path list delete root migrate'
complete -c __NAME__ -f -n __fish_use_subcommand -a '(command gh workspace list 2>/dev/null)'
complete -c __NAME__ -f -n '__fish_seen_subcommand_from path delete rm' -a '(command gh workspace list 2>/dev/null)'
`,
	"pwsh": `function __NAME__ {
  $sub = if ($args.Count -gt 0) { [string]$args[0] } else { '' }
  if ($sub -in @(__PASSTHROUGH__)) {
    gh workspace @args
    if (-not (Test-Path -LiteralPath $PWD.Path)) { Set-Location (gh workspace root) }
    return
  }
  if ($sub -in @('clone', 'create', 'path')) { $out = gh workspace @args } else { $out = gh workspace path @args }
  if ($LASTEXITCODE -eq 0 -and $out) { Set-Location -LiteralPath (@($out)[-1]) }
}
Register-ArgumentCompleter -CommandName __NAME__ -ScriptBlock {
  param($wordToComplete)
  @('clone', 'create', 'path', 'list', 'delete', 'root', 'migrate') + @(gh workspace list 2>$null) |
    Where-Object { $_ -like "$wordToComplete*" } |
    ForEach-Object { [System.Management.Automation.CompletionResult]::new($_) }
}
`,
}
