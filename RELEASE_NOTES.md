# Release notes

## v0.1.0

First release of **commando**, a terminal app that turns any Unix command's man page into a form. Put it in front of a command, choose options with checkboxes, dropdowns and radio buttons, read what each option does, then press Enter to run the command you built.

```sh
commando ls
commando grep -rn TODO .     # opens with -r and -n already checked
commando git commit
```

### Highlights

- **Reads the real documentation.** Options come from the command's man page, in both the GNU and the BSD/macOS formats. Tools without a man page fall back to their `--help` output (cargo, rustc and other modern CLIs).
- **The right control for each option:**
  - checkboxes for on/off flags; repeatable ones such as `-v` keep a count
  - dropdowns for fixed values, e.g. `--color` → always / auto / never, with a "custom value…" entry
  - radio buttons for options that override each other, e.g. `ls -1 / -C / -x / -l`
  - number fields that step with `+`/`-`, and file fields with Tab completion
- **Help as you go.** A side panel explains the focused option in full. `Ctrl-O` opens the whole manual at that option, with search.
- **Conflicts handled.** When the manual says two options are mutually exclusive, turning one on turns the other off.
- **Subcommands.** `git commit` uses the git-commit man page, and `cargo build` uses `cargo build --help`.
- **Fast.** The window opens immediately and the manual loads in the background. Results are cached, so reopening even curl's very long manual takes about 20 ms.
- **Looks good in iTerm2 and Terminal.app**, with truecolor gradients, adaptive light and dark colors, and mouse support.

### Running the result

- **Enter** runs the command in your `$SHELL`.
- **`-p` / `--print`** writes the command to stdout instead: `cmd=$(commando -p find)`.
- **Shell integration:** `eval "$(commando --init zsh)"` (also `bash` and `fish`) binds **Ctrl-X Ctrl-O**. It opens the form on the line you're typing and puts the result back on your prompt, so you can review it and it goes into your history.

### Install

Requires Go 1.24 or later.

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.1.0
```

### Known limitations

- Spotting fixed value lists and groups of exclusive options depends on how each manual words them, so some options a person would read as a dropdown stay plain text fields. `commando --dump <cmd>` shows how a page was read.
- The test suite passes on Linux and macOS, but the interactive form has only been tried in a Linux terminal so far, not yet in iTerm2 or Terminal.app. Please open an issue if something looks wrong on your setup.
