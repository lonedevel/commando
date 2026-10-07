package main

import (
	"strings"

	"github.com/lonedevel/commando/internal/config"
)

// flagHelp describes commando's own flags, for the shell completions.
var flagHelp = []struct {
	short, long, desc string
	arg               string // "shell", "theme", "line" or ""
}{
	{"p", "print", "print the command instead of running it", ""},
	{"l", "line", "start from an existing command line", "line"},
	{"", "long", "prefer --long option names", ""},
	{"", "no-cache", "re-parse the manual even if it is cached", ""},
	{"", "no-history", "don't record this command or show history", ""},
	{"", "init", "print shell integration (Ctrl-X Ctrl-O)", "shell"},
	{"", "completion", "print shell completions", "shell"},
	{"", "explain", "describe each part of a command line", ""},
	{"", "themes", "show a sample of each color theme", "theme"},
	{"", "config", "show and check the settings file", ""},
	{"", "dump", "print the parsed options as JSON", ""},
	{"v", "version", "print the version", ""},
	{"h", "help", "show help", ""},
}

// completion returns the completion script for shell. After commando's
// flags, a command name completes, then that command's own arguments, as
// they would without commando in front.
func completion(shell string) (string, bool) {
	themes := strings.Join(config.Themes, " ")
	var b strings.Builder
	switch shell {
	case "zsh":
		b.WriteString("#compdef commando\n# commando zsh completions — add to ~/.zshrc:  source <(commando --completion zsh)\n")
		b.WriteString("_commando() {\n  local curcontext=$curcontext state line ret=1\n  typeset -A opt_args\n  _arguments -s -S \\\n")
		for _, f := range flagHelp {
			spec := "'--" + f.long + "[" + strings.ReplaceAll(f.desc, "'", "'\\''") + "]"
			if f.short != "" {
				excl := "(-" + f.short + " --" + f.long + ")"
				if f.long == "version" || f.long == "help" {
					excl = "(- *)"
				}
				spec = "'" + excl + "'{-" + f.short + ",--" + f.long + "}'[" + strings.ReplaceAll(f.desc, "'", "'\\''") + "]"
			}
			switch f.arg {
			case "shell":
				spec += ":shell:(zsh bash fish)"
			case "line":
				spec += ":command line: "
			}
			b.WriteString("    " + spec + "' \\\n")
		}
		b.WriteString("    '*::command:->command' && ret=0\n")
		b.WriteString("  if [[ $state == command ]]; then\n")
		b.WriteString("    if (( $+opt_args[--themes] )); then\n")
		b.WriteString("      compadd -- " + themes + " && ret=0\n")
		b.WriteString("    else\n      _normal && ret=0\n    fi\n  fi\n  return ret\n}\n")
		b.WriteString("if [[ $funcstack[1] == _commando ]]; then\n  _commando \"$@\"\nelse\n  compdef _commando commando\nfi\n")
	case "bash":
		var flags []string
		for _, f := range flagHelp {
			if f.short != "" {
				flags = append(flags, "-"+f.short)
			}
			flags = append(flags, "--"+f.long)
		}
		b.WriteString("# commando bash completions — add to ~/.bashrc:  source <(commando --completion bash)\n")
		b.WriteString(`_commando() {
  local cur=${COMP_WORDS[COMP_CWORD]} prev=${COMP_WORDS[COMP_CWORD-1]} i themes=
  case $prev in
    --init|--completion) COMPREPLY=($(compgen -W "zsh bash fish" -- "$cur")); return ;;
    -l|--line) return ;;
  esac
  # Find the command after commando's own flags.
  for ((i = 1; i < COMP_CWORD; i++)); do
    case ${COMP_WORDS[i]} in
      --themes) themes=1 ;;
      -l|--line|--init|--completion) ((i++)) ;;
      --) ((i++)); break ;;
      -*) ;;
      *) break ;;
    esac
  done
  if [[ $themes ]]; then
    COMPREPLY=($(compgen -W "` + themes + `" -- "$cur"))
  elif ((i < COMP_CWORD)); then
    # The command's own arguments, as if commando weren't there.
    if declare -F _command_offset >/dev/null; then
      _command_offset $i
    else
      COMPREPLY=($(compgen -f -- "$cur"))
    fi
  elif [[ $cur == -* ]]; then
    COMPREPLY=($(compgen -W "` + strings.Join(flags, " ") + `" -- "$cur"))
  else
    COMPREPLY=($(compgen -c -- "$cur"))
  fi
}
complete -o default -F _commando commando
`)
	case "fish":
		b.WriteString("# commando fish completions — add to config.fish:  commando --completion fish | source\n")
		b.WriteString(`# True until the command after commando's own flags is typed.
function __commando_no_cmd
  set -l skip 0
  for t in (commandline -opc)[2..]
    if test $skip = 1
      set skip 0
      continue
    end
    switch $t
      case -l --line --init --completion
        set skip 1
      case '-*'
      case '*'
        return 1
    end
  end
end
`)
		for _, f := range flagHelp {
			line := "complete -c commando -n __commando_no_cmd"
			if f.short != "" {
				line += " -s " + f.short
			}
			line += " -l " + f.long
			switch f.arg {
			case "shell":
				line += ` -x -a "(printf '%s\\tshell\\n' zsh bash fish)"`
			case "line":
				line += " -x"
			}
			b.WriteString(line + " -d '" + strings.ReplaceAll(f.desc, "'", "\\'") + "'\n")
		}
		b.WriteString("complete -c commando -n '__fish_contains_opt themes' -x -a '" + themes + "'\n")
		b.WriteString("complete -c commando -n 'not __fish_contains_opt themes' -x -a '(__fish_complete_subcommand -l --line --init --completion)'\n")
	default:
		return "", false
	}
	return b.String(), true
}
