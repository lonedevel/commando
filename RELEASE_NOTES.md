# Release notes

## v0.21.0

Better at reading `--help` output, so tools without a manual get fuller forms.

### Improved

- `-a --all`, with no comma between the names, is read as one option with two names, as systemd's and binutils' tools write it. `-c --order=cpu` is read as a flag that stands for that setting.
- wget's layout is read in full: `-T,  --timeout=SECONDS` and the long-only options lined up under it. wget's form goes from 103 option names to 197.
- Arguments written as `<ID,...>` no longer stop the option, and the one above it, from being read (pgrep and pkill: 31 to 54 names).
- `--lint|--enable-checks` gives two names for one option.
- When a manual describes an option twice, new names in the second entry are added to the first (rsync's `--cc` for `--checksum-choice`). A second entry that sets a fixed value becomes its own flag (ls `-p` for `--indicator-style=slash`, sort `-C`).
- `-1 --base, -2 --ours, -3 --theirs` in `git diff` is read as three options.
- Across 770 tools that only have `--help`, commando now knows 777 more option names. Across 334 manuals, it knows 21 more options. None are lost in either.
- Manuals and help text are read again the first time you open them after upgrading.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.21.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.20.0

Hundreds more option names read from manuals, notably for PostgreSQL and Java tools.

### Improved

- Some manuals list each name of an option on its own line, as PostgreSQL's do:

  ```
  -U username
  --username=username
      User name to connect as.
  ```

  Only the last name was read, so short names such as psql's `-c` and pg_dump's `-U`, `-F` and `-f` were unknown. `--explain` called them "not in the manual", and a command line using them couldn't fill in the form. All the names are now read.
- Names joined by "or", as in the Java tools' manuals (`-b addr or --bind-address addr`, `--class-path path, -classpath path, or -cp path`), are now read as one option. jwebserver goes from 1 option to 6, javac from 46 to 54.
- Across the 334 manuals tested, commando now knows 656 more option names and 49 more options than in v0.19.0, and none are lost.
- Manuals are read again the first time you open them after upgrading.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.20.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.19.1

More options read from manuals.

### Fixed

- An option whose argument can repeat, such as `-I dir ...` or `--themes [name ...]`, was skipped. Such options now appear in the form. Among the manuals tested, this adds tar's `--pax-option`, `--dirstat-by-file` in git's diff and log commands, and options of java, javadoc, jdb and jlink.
- Manuals are read again the first time you open them after upgrading, so these options show up.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.19.1
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.19.0

commando gets its own manual and tab completions.

### New: `man commando`

- A manual page covering commando's flags, keys, shell integration, settings and data files, environment variables and exit status.
- Homebrew installs it. The release archives include it as `commando.1`.

### New: tab completions

- `commando --completion zsh`, `bash` or `fish` prints completions, and Homebrew installs them for you.
- Tab completes commando's flags, the shells for `--init`, and theme names after `--themes`. After that comes the command you're running and its own arguments, as if commando weren't in front: `commando git chec<Tab>` completes `checkout`.
- Without Homebrew, add `source <(commando --completion zsh)` to `~/.zshrc` after `compinit`, `source <(commando --completion bash)` to `~/.bashrc`, or `commando --completion fish | source` to `config.fish`.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`. Homebrew now also installs the manual and the tab completions.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.19.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.18.0

Tidy your history from the start screen.

### New: delete from the start screen

- Run `commando` with no command, choose a saved line or preset with ↑↓ (or find it by typing), and press `^D` to delete it. It's removed from whichever command it was saved under, such as `git log` or `tar`.
- Changed your mind? `^Z` puts it back, in its old place.
- The key hint shows `^D delete` whenever a saved line is chosen.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.18.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.17.0

A new look for the README, with a recording of commando in use.

### Documentation

- The README opens with an animated recording: choosing `git log` from git's commands, loading an example from its manual with `^X`, explaining a `find` command with `--explain`, and searching past commands from the start screen.
- A short "At a glance" list now leads, and the full feature list is grouped by topic under "Features in detail".
- commando itself works as in v0.16.0.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.17.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.16.0

Find anything you've run before, from the start screen.

### New: search your history

- Run `commando` with no command, then type. Besides command names, the start screen now searches every command line you've run or saved as a preset, for all commands. A line matches when it, or its preset's name, contains every word you typed: `photos`, `rsync backup`, `git log`.
- Matches are listed under "From your history": presets first, then recent commands, newest first. Press Enter on one to open it in its form, ready to adjust and run.
- With nothing typed, the start screen shows your most recent commands, as before. Esc clears what you've typed before quitting.

### Behind the scenes

- Each release now opens its own Homebrew formula pull request.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.16.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.15.0

See how each color theme looks in your own terminal before choosing one.

### New: preview themes

- `commando --themes` prints a small sample form in each of the 13 themes. The sample uses the same colors and styles as the real form: checkboxes, a dropdown, a radio group, a ⚠ option, a path field and the command they build.
- Name themes to see only those: `commando --themes nord dracula`.
- The theme set in your settings file is marked "← your setting". A misspelled name lists the valid ones.
- To use one, set `theme = "NAME"` in the settings file (`commando --config` creates it).

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.15.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.14.0

Find what you need in long lists of examples and saved commands.

### New: filter the list as you type

- In the list of presets, recent commands and examples (`^L` or `^X`), typing narrows it to the entries that contain every word you type, in the command line, its description or the preset's name. In curl's nearly 300 examples, `retry` leaves the five retry ones.
- The title shows how many entries match. Esc clears the filter first, then closes the list.

### Changed keys in the list

Letters now go to the filter, so the list's single-letter keys moved:

- Delete a preset or recent command: Delete or `^D` (was `d` or `x`).
- Open the manual: `^O` or `F1` (`?` now types into the filter).
- Move: the arrow keys, Home and End (`j`, `k`, `g` and `G` now type into the filter).

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.14.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.13.0

Pick a color scheme to match your terminal.

### New: named themes

- `theme` in the settings file can now name a color scheme: `dracula`, `nord`, `tokyo-night`, `catppuccin-mocha`, `catppuccin-latte`, `gruvbox-dark`, `gruvbox-light`, `solarized-dark` or `solarized-light`.
- Each theme uses its scheme's official colors for every part of the form, including the logo. They're meant for a terminal set to the same scheme: commando colors the text, not the window background.
- `[colors]` still overrides any single color on top of a theme, and `auto`, `dark`, `light` and `contrast` work as before.
- The README shows the same form in each theme side by side.

```toml
# ~/.config/commando/config.toml  (commando --config creates it)
theme = "dracula"
```

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.13.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.12.0

Examples now appear under the options they belong to.

### New: per-option examples

- Some manuals give an example under each option. curl's has one for nearly every option, such as `curl --retry 7 https://example.com`. The help panel now shows the focused option's examples as highlighted commands, right under its name.
- Press `^X` on that option to jump straight to its example in the examples list. Press Enter to load it into the form and adjust it. Examples the form can't hold exactly are still marked ⧉ and copied instead.
- Every option in curl's manual has at least one example.

### Fixes

- The list of presets, recent commands and examples no longer scrolls the selected entry out of view.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.12.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.11.0

commando now has a settings file, for your defaults, colors, and corrections to what it reads from manuals.

### New: settings file

- Run `commando --config` to create a commented settings file at `~/.config/commando/config.toml`. Run it again any time to see where the file is and check it for mistakes. Flags and environment variables still win over it.
- **Defaults:** `long` (prefer `--long` option names), `confirm` (ask before running a ⚠ command), `history`, and `recent` (how many recent commands to keep per command, which used to be fixed at 10).
- **Colors:** `theme = "auto"`, `"dark"`, `"light"` or `"contrast"`, the last for a stronger palette. Under `[colors]` you can override any color by its role: `accent`, `command`, `danger`, `text` and so on.
- **Corrections per command:** under `[commands."docker run"]` (or any other command), `safe` options never get a ⚠, `risky` ones always do, and `values` adds entries to an option's dropdown, for example `"--quoting-style" = ["clocale"]` for `ls`. Corrections apply to the form and to `--explain`.
- If the file has a mistake, commando warns once and carries on with the defaults. `commando --config` lists every problem it finds.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.11.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.10.1

Fixes and polish.

- `ls --sort` offers only the values GNU ls accepts. Values from the shell's completion files that the manual never mentions are now left out when the manual has its own list. fish's file for ls offered `--sort=atime`, `status`, `access` and `use`, which ls rejects.
- Dropdown rows show the short form of a value's meaning ("Symbolic link"). The help panel keeps the full text.
- `docker run --rm` and `docker build --rm` no longer get a ⚠: they only remove what the command itself created.
- GNU `grep --binary-files`: the meaning shown for `binary` is now the manual's description of the default.
- `find -type` is labelled "File is of type" rather than "File is of type c:".

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.10.1
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.10.0

Dropdowns now say what each value means, taken from the manual.

### New: value meanings

- When the manual explains an option's values, the dropdown shows each meaning beside its value. For example, `find -type` lists `b` Block special, `d` Directory, `f` Regular file and `l` Symbolic link.
- The help panel lists every value with its meaning and marks the one you picked.
- `commando --explain` adds the meaning of the value given: `find . -type d` reads "d: Directory".
- Meanings come from lists under the option (`find -type`, `tar --format`, `tar --backup`, curl `--ftp-method`), one paragraph per value (`git log --date=relative …`), "If TYPE is text, …" sentences (`grep --binary-files`), and lists that name the matching flag (`ls --sort`: size → "Like -S: Sort by file size, largest first").

### New dropdowns

- macOS `find -type` and `grep --binary-files`, and `git log --date` (relative, iso, iso-strict, rfc, short, raw, human, unix…), were text fields before.
- Lists of other things stay text fields: unit suffixes (`find -atime 3d`), environment variables (tar `--to-command`) and format variables (curl `-w`).

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.10.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.9.0

Find out what a command line does before you run it.

### New: explain a command line

- `commando --explain 'tar -czvf backup.tgz --exclude=.git src'` prints what each option and argument means, taken from the manual, without opening the form.
- Options show the manual's first sentence and their other names, plus a red ⚠ for risky ones such as `rm -rf` and `rsync --delete`. There's a note when a value isn't one of the listed choices.
- Arguments are named from the usage line (`cp a b` → Source, Dest). Options the manual doesn't list are flagged.
- Pipelines and lists (`|`, `&&`, `||`, `;`, `&`) are explained one command at a time.
- Wrappers such as `sudo`, `xargs`, `env`, `nice` and `time` are explained along with the command they run.
- `NAME=value` settings and redirections (`> file`, `2>&1`, `2>/dev/null`) are explained. Writing to a file with `>` is marked as replacing what it holds.
- `find`'s `!`, parentheses and `-o` are explained.
- Output is plain text when piped.

### Also new

- **Shell shortcut:** `Ctrl-X ?` explains the line you're typing, below your prompt, without changing it. To get it, re-run the `eval "$(commando --init zsh)"` line (or the bash or fish one) in a new shell. The bash shortcuts need bash 4 or later; macOS ships 3.2, so use Homebrew's bash or zsh.
- **In the form:** the preview beside presets, recent commands and examples now explains each one, including ⧉ examples the form can't load.

### Fixes

- With `-p`, and in the `Ctrl-X Ctrl-O` shortcut, the form lost most of its colors because its output was going to a pipe. The colors are back.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.9.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.8.0

Start from the examples in a command's manual, and build `find` commands that run as intended.

### New: examples from the manual

- Press `^X` to list the example command lines from the manual's EXAMPLES section, each with its explanation. `git log`, `find`, `rsync` and the macOS `grep`, `ls` and `sed` pages all have them. Tools whose `--help` has an "Examples:" section (kubectl and others) are covered too.
- Press Enter to load an example into the form, then adjust it and run it. Examples also appear at the end of the `^L` list.
- Examples the form can't hold exactly are marked ⧉, and Enter copies them to the clipboard as written instead. These are ones that use `!` or parentheses in `find`, repeat an option, or put options after arguments.
- Examples that pipe into another command or redirect output are left out.

### Fixed: find expressions

- Forms for `find` used to put every option before the paths (`find -name x .`), which `find` rejects. The form now puts the paths first, then the tests in the order you turn them on, then actions such as `-print`, `-exec` and `-delete`.
- Actions always go after the tests, so a form never builds `find . -delete -name '*.tmp'`, which would delete everything.
- When a command line you start from can't be kept exactly as typed, commando now warns you to check it before running. Before, it was reordered silently.

### Also

- The argument hint leaves out bracketed options: `go build` shows "packages" instead of "output build flags".

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.8.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.7.0

Commands that group other commands now open their own list, so `commando docker` → `container` → `ls` takes you to the right form.

### New: nested command lists

- Picking a command such as `docker container`, `docker image` or `kubectl config` opens a second list of its commands, instead of the tool's own form. The title shows where you are: `docker › container`.
- Esc in a form returns to its list, and Esc in a list goes back up a level, with its filter and selection as you left them.
- `commando docker container` starts at that level, and `commando docker container ls` opens the form directly.
- Commands up to four words deep are recognized, whether they're documented by man pages (`docker-container-ls(1)` on Linux) or by `--help`.

### Improved

- Commands whose `--help` lists no options now try `tool help command`, so `go build`, `go test` and the rest of go's commands get real forms.
- On systems with Docker's man pages, a nested page only counts as a sub-subcommand when its parent page mentions it, so `git remote-ext` and `git commit-tree` stay commands of git itself.

### Fixes

- commando no longer crashes in a terminal window only a few rows tall.
- The argument hint no longer shows the command's own name when its usage line starts with "Usage:" (docker's commands).

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.7.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.6.0

Opening a tool such as git, docker or cargo without naming a command now lists its commands first, so you can find the one you want before filling in its options.

### New: choose a subcommand

- `commando git` lists git's commands, each with a line on what it does, grouped the way git's manual groups them (main porcelain commands, ancillary commands, plumbing…).
- Tools documented by `--help` list the commands from their "Commands:" sections, with aliases and groups: docker's Common and Management Commands, cargo's `build, b`, and `go`.
- Type to filter by name or description, then press Enter to open that command's form. Esc in the form goes back to the list.
- Your presets and recent commands for the tool and all its subcommands are listed first, and the first entry opens the tool's own options.
- `^O` opens the tool's manual from the list.
- Naming the command up front, as in `commando git commit` or `commando docker run`, skips the list as before.

### Fixes

- The argument hint in a subcommand's form no longer repeats the subcommand's name.

### Known limitations

- The list goes one level deep. Docker's management commands (`container`, `image`, `network`…) open docker's own form with the command as an argument, rather than a second list of `ls`, `prune` and so on.
- Commands whose `--help` has no option list, such as `go build`, open the tool's form with the command filled in as an argument.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.6.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.5.0

Each of a command's arguments now gets its own labelled field, read from the usage line in its manual.

### New: a field for each argument

- Instead of one "Arguments" box, commands get a field per argument: `cp` has Source and Dest, `ln` has Target and Link name, `grep` has Patterns and File…, and macOS's `cp` has Source file and Target file.
- Required arguments are marked `*`. If one is empty, Enter warns once ("DEST looks required…"), and pressing Enter again runs the command anyway, since usage lines aren't always strict.
- Fields for files and folders complete with Tab. Fields marked `…` take several values separated by spaces.
- Words you type before opening the form, and saved presets and recent commands, are spread across the fields in order: `commando grep -rn TODO src lib` puts `TODO` in Patterns and `src lib` in File….
- Commands whose usage line is too irregular to read reliably, such as tar, curl and rsync, keep the single Arguments field.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

To upgrade, refresh the tap first: `brew update && brew upgrade commando`.

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.5.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.4.0

commando now warns you about options that can delete or overwrite data, and asks before running a command that uses one.

### New: warnings for risky options

- Options that delete or overwrite data, or that skip a confirmation prompt, are marked with a red ⚠, for example `rm -r` and `-f`, `rsync --delete`, `find -delete`, `tar --remove-files`, `ln -f` and `git push --force`. The help panel says why each one is risky.
- If the command you built uses one, pressing Enter asks "Run it?" first. Press `y` to run it, or any other key to go back to the form.
- Nothing is asked with `-p` or the Ctrl-X Ctrl-O shell shortcut, since they only put the command on your prompt for review. Set `COMMANDO_NO_CONFIRM=1` to never be asked.

### Upgrading with Homebrew

Homebrew only refreshes the commando tap during `brew update`, which `brew upgrade` skips if it ran recently. To get this version:

```sh
brew update && brew upgrade commando
```

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.4.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.3.0

More options are now dropdowns, using the value lists your shell already knows.

### New: values from shell completions

- commando now reads the completion definitions that ship with zsh (always present on macOS) and fish (when installed). Where they list the exact values an option takes, the option becomes a dropdown: for example `tar --format` (gnu, pax, ustar…), `grep --binary-files` (binary, without-match, text) and `rsync -e` (rsh, ssh). With fish installed, `curl -X` also lists the HTTP methods.
- Values found in the man page are still offered after the shell's, and the "custom value…" entry is always there.
- The help panel says when an option's values came from zsh or fish.
- `COMMANDO_NO_COMPLETIONS=1` turns this off. `COMMANDO_ZSH_COMPLETIONS` and `COMMANDO_FISH_COMPLETIONS` (colon-separated folders) add places to look.

### Fixes

- Filtering for an option name now prefers an exact-case match, so typing `-X` goes to `-X`, not `-x`.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando        # or: brew upgrade commando
```

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.3.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`. The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

## v0.2.0

commando now remembers what you run, and lets you save a form you use often as a named preset.

### New: presets and recent commands

- **Recent commands.** Every command you run through commando is remembered, up to the last 10 per command, including ones built with `-p` or the Ctrl-X Ctrl-O shell shortcut.
- **Presets.** Fill in a form, press **Ctrl-T** and give it a name, such as "gzip archive of a folder".
- **Start from them.** Opening a command with no options typed, e.g. `commando tar`, lists its presets and recent commands first. Press Enter to load one into the form, adjust it, and run. `d` deletes an entry, Esc starts with a blank form, and **Ctrl-L** brings the list back at any time.
- **Start screen.** Running `commando` on its own now lists your most recent commands across all tools.
- **Where it's kept.** `~/Library/Application Support/commando/store.json` on macOS and `~/.config/commando/store.json` on Linux, readable only by you. Like your shell history, it holds full command lines, so pass `--no-history` or set `COMMANDO_NO_HISTORY=1` if you'd rather not keep them.

### Also new

- **Ready-to-run downloads.** Each release now includes prebuilt binaries for macOS (Apple silicon and Intel) and Linux (x86-64 and ARM64), with a `checksums.txt` file.
- Releases are now published automatically from `main`.

### Install

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando        # or: brew upgrade commando
```

With Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.2.0
```

Or download the archive for your system from the release page, unpack it, and put `commando` on your `PATH`.
The binaries aren't signed, so if macOS says it can't verify the developer, run `xattr -d com.apple.quarantine commando` once.

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

With Homebrew (macOS or Linux):

```sh
brew tap lonedevel/commando https://github.com/lonedevel/commando
brew install commando
```

Or with Go 1.24 or later:

```sh
go install github.com/lonedevel/commando/cmd/commando@v0.1.0
```

### Known limitations

- Spotting fixed value lists and groups of exclusive options depends on how each manual words them, so some options a person would read as a dropdown stay plain text fields. `commando --dump <cmd>` shows how a page was read.
- The test suite passes on Linux and macOS, but the interactive form has only been tried in a Linux terminal so far, not yet in iTerm2 or Terminal.app. Please open an issue if something looks wrong on your setup.
