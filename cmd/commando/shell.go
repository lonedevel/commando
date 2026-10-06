package main

// shellInit holds line-editor widgets: press Ctrl-X Ctrl-O while typing a
// command and commando opens with that line pre-filled; on Enter the edited
// command replaces the line so you can review it, run it, and keep it in
// your shell history. Ctrl-X ? explains the line instead, below the prompt.
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
_commando_explain() {
  zle -I
  commando --explain --line "$BUFFER"
}
zle -N _commando_explain
bindkey '^X?' _commando_explain
`,
	"bash": `# commando bash integration — add to ~/.bashrc:  eval "$(commando --init bash)"
_commando_widget() {
  local result
  result=$(commando --print --line "$READLINE_LINE" </dev/tty) || return
  READLINE_LINE=$result
  READLINE_POINT=${#READLINE_LINE}
}
bind -x '"\C-x\C-o": _commando_widget'
_commando_explain() {
  local p=''
  ((BASH_VERSINFO[0] > 4 || (BASH_VERSINFO[0] == 4 && BASH_VERSINFO[1] >= 4))) && eval 'p=${PS1@P}'
  printf '%s%s\n' "$p" "$READLINE_LINE"
  commando --explain --line "$READLINE_LINE"
}
bind -x '"\C-x?": _commando_explain'
`,
	"fish": `# commando fish integration — add to config.fish:  commando --init fish | source
function _commando_widget
  set -l result (commando --print --line (commandline) </dev/tty)
  and commandline -r -- $result
  commandline -f repaint
end
bind \cx\co _commando_widget
function _commando_explain
  echo
  commando --explain --line (commandline)
  commandline -f repaint
end
bind \cx'?' _commando_explain
`,
}
