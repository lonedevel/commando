package main

// shellInit holds line-editor widgets: press Ctrl-X Ctrl-O while typing a
// command and commando opens with that line pre-filled; on Enter the edited
// command replaces the line so you can review it, run it, and keep it in
// your shell history.
var shellInit = map[string]string{
	"zsh": `# commando zsh integration — add to ~/.zshrc:  eval "$(commando --init zsh)"
_commando_widget() {
  local result
  result=$(commando --print --line "$BUFFER" </dev/tty) || { zle reset-prompt; return }
  BUFFER=$result
  CURSOR=${#BUFFER}
  zle reset-prompt
}
zle -N _commando_widget
bindkey '^X^O' _commando_widget
`,
	"bash": `# commando bash integration — add to ~/.bashrc:  eval "$(commando --init bash)"
_commando_widget() {
  local result
  result=$(commando --print --line "$READLINE_LINE" </dev/tty) || return
  READLINE_LINE=$result
  READLINE_POINT=${#READLINE_LINE}
}
bind -x '"\C-x\C-o": _commando_widget'
`,
	"fish": `# commando fish integration — add to config.fish:  commando --init fish | source
function _commando_widget
  set -l result (commando --print --line (commandline) </dev/tty)
  and commandline -r -- $result
  commandline -f repaint
end
bind \cx\co _commando_widget
`,
}
