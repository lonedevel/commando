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

## Features

New capabilities, in the order they'd be taken on. The first three matter
most; the three after them are small enough to share a release.

1. **Live values for fields.** Dropdowns filled from the system as well
   as the manual: git branches for `git checkout`, container names for
   `docker stop`, contexts for `kubectl`, hosts from `~/.ssh/config`,
   signal names for `kill`, network interfaces for `ifconfig`. A small
   set of value sources, each tied to a command and an option, next to
   the zsh and fish completions. Each source would need a short timeout
   so the form never waits on a slow command.
2. **Does this line work here?** A line copied from the web often uses
   GNU options on macOS, or BSD ones on Linux (`sed -i ''`,
   `ls --color`, `date -d`). `--explain` already marks options that aren't
   in the manual; it could say which system the option comes from and
   suggest this system's equivalent. The equivalents would start as a
   hand-written table for common tools, checked against the saved GNU and
   BSD pages in the tests.
3. **commando for coding agents.** Offer the parsed manuals, `--explain`
   and the ⚠ checks as JSON (`--dump` already prints a manual) and as an
   MCP server, so an agent can ask what an option does on *this* machine,
   and whether a line is risky, before it runs it.
4. **Your usual options first.** History already records which options
   you use with each command. In big forms such as curl's, show those in
   a section at the top, and rank them first when filtering.
5. **What the exit status means.** After a command runs, explain its
   exit status from the manual's EXIT STATUS section, for example: grep
   returns 1 when nothing matched, not because of an error.
6. **Rewrite a line with long names.** For example,
   `commando --long -p 'tar -czvf x.tgz src'` would print
   `tar --create --gzip --verbose --file=x.tgz src`, which is easier to
   read in a script. Filling the form from a line and `Ctrl-S` already do
   most of the work.
7. **Show output inside commando (a setting).** With `output = "pane"`,
   Enter runs the command in a pane below the form instead of leaving
   commando, so you can adjust the form and run it again. The default
   would still hand the command to your shell. Open questions: commands
   that need the whole terminal (`less`, `vim`, `top`, `ssh`) need a
   pseudo-terminal or a fallback to leaving commando; how much output to
   keep and how to scroll and search it; whether Ctrl-C stops the command
   or closes commando. The ⚠ confirmation would still apply.

## Possible

Ideas with an open question or two; each would want a closer look first.

- **Examples from tldr.** When a manual has no EXAMPLES section, take
  examples from a local tldr cache, marked as coming from tldr. Which
  tldr clients' caches to look for, and where they keep them, varies.
- **Environment variables in the form.** Many manuals have an
  ENVIRONMENT section (`LC_ALL`, `GIT_PAGER`, `CURL_CA_BUNDLE`). These
  could be fields that put `NAME=value` before the command, and
  `--explain` could explain them. Those sections are written less
  consistently than OPTIONS.
- **Try a dry run first.** For a line with a ⚠ option, "Run it?" could
  offer a dry run of the same line first: `rsync -n`, `git clean -n`,
  `make -n`, or `find` without `-delete`. Each risky option would need
  its dry-run equivalent recorded, and most commands don't have one.
- **Pipelines in the form.** `--explain` reads pipelines; the form builds
  one command. A key that adds a `| next` stage with its own form would
  build `find … | xargs … | sort` too. How to show several forms at once
  is the open question.
- **Describe what you want.** Type "files over 100 MB changed this week"
  and a language model fills in `find`'s form, which you check before
  running. It would need a provider setting, off by default, and must
  never run anything without the form.

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
