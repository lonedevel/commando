# Roadmap

Where commando is, and where it might go next. Nothing here is a promise;
it's a list to choose from. Open an issue to suggest something or to say
which of these you'd use.

## Where it is now

commando turns a command's manual, or its `--help`, into a form, and it
covers the whole round trip: the form itself, choosing a subcommand,
filling it in from a line you've typed, explaining a line
(`--explain`), examples from the manual, warnings for risky options,
presets and history, settings and themes, shell integration, its own man
page and completions. [RELEASE_NOTES.md](RELEASE_NOTES.md) has the
details, release by release.

How well it reads manuals is measured, not guessed: `scripts/survey.py`
runs commando over every manual and `--help` on a system and counts the
options and names it finds, and `TestFormats` holds real pages in each
layout it has learned. On the test machine, 334 man pages give about
6,000 options, and nearly 800 `--help`-only tools about 35,000.

## Next

Small, well-understood pieces of work, roughly in order.

- **macOS and BSD manuals, checked on a Mac.** The parser's tests include
  saved BSD pages (ls, find, grep, sed), but the survey has only run on
  Linux. Running it on macOS would show what Apple's manuals lose.
- **How well values are read.** The survey counts options and names.
  Counting dropdowns, value descriptions and radio groups the same way
  would find the next gaps in what the form offers.
- **Options only in the SYNOPSIS.** A few manuals (`git update-ref`,
  `pg_controldata`, `jcmd`, `rsync-ssl`) list their options nowhere but
  the SYNOPSIS line. The form could fall back to reading them there.
- **The long tail of `--help` layouts.** About a tenth of the option
  names in `--help` output are still missed. Much of that is noise (zsh
  lists its shell options), and the rest is spread over one-off layouts
  (bash, valgrind, cmake's `--help-*`); worth taking in batches, a layout
  at a time, measured with the survey.

## Possible

Ideas with an open question or two; each would want a closer look first.

- **commando's own `--init` and `--completion` as dropdowns.** When
  commando's completions are installed, its form could offer zsh, bash
  and fish as values. Small, and mostly a check that completions are read
  well.
- **Save a command as an alias or script.** From the form, write the
  command to a shell alias, function or script file. Which file to write
  to, and how to avoid clobbering one, needs thought.
- **Share presets.** Presets live in one file per user. Importing and
  exporting them (a team's usual `rsync` or `pg_dump` lines) would mean
  agreeing on a format.
- **More shells.** Shell integration and completions cover zsh, bash and
  fish. nushell and PowerShell on Linux and macOS are possible.

## Not planned

- **Windows.** commando reads Unix manuals and runs commands with
  `$SHELL`; Windows tools document themselves differently.
- **Editing commands' config files.** commando builds command lines; a
  tool's own configuration is out of scope.
