# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

commando is a Go TUI (Bubble Tea + Lip Gloss) that parses a command's man page, or its `--help` output, into a form, then builds and runs the command line. README.md describes user-facing behavior in detail.

## Commands

```sh
make build                 # ./bin/commando
make test                  # go test ./...
make lint                  # gofmt -l must print nothing, then go vet (CI runs this)
go test ./internal/manpage -run TestFormats   # a single test
bin/commando --dump <cmd>  # print the parsed Spec as JSON: the first step when a page parses wrongly
bin/commando --no-cache <cmd>                 # bypass the parsed-manual cache
```

CI (`.github/workflows/ci.yml`) runs lint/test/build on Ubuntu and macOS. On macOS it also checks that `--dump` yields option values sourced from zsh's completions, and it audits `Formula/commando.rb` with Homebrew.

## Architecture

Data flows in one direction: **manual text → `manpage.Spec` → form state → command line**.

- `internal/manpage`: produces a `Spec` (see `spec.go`) for a command.
  - `fetch.go` (`Load`): finds the page with `man -w`, renders it at `RenderWidth` (100 columns), and falls back to running the binary with `--help`/`-h` (or `<sub> --help`, `help <sub>`) when there's no manual. Subcommands map to pages joined by `-` (`git commit` → `git-commit`). Results are cached as JSON, keyed on the page's (or binary's) path, size, mtime and `parserVersion`. **Bump `parserVersion` whenever parse output changes**, or users keep stale cached specs.
  - `parse.go`: layout-agnostic parsing. It finds option tags by indentation relative to their descriptions, learns each page's layout, and infers labels, sections, conflicts and radio groups from the prose. `values.go`/`choices.go` infer enumerated values, `completions.go` adds values from `internal/complete` (zsh/fish completion definitions), `args.go` reads positionals from the synopsis, and `examples.go`, `subcommands.go` and `danger.go` cover the rest of the Spec.
- `internal/cmdline`: converts between form state (`[]Value`, indexed like `Spec.Options`) and command lines: `Build` → tokens → `Render`, prefill from a typed line (`prefill.go`), and `--explain` (`explain.go`). Ordering rules for expression commands like `find` (paths, then tests in the order they were turned on, then actions) live here, using `Value.Seq`.
- `internal/shellword`: shell-style tokenizing used by cmdline and the UI.
- `internal/ui`: one Bubble Tea `Model` (`model.go`) with modes `modePick` (start screen/history search), `modeLoading`, `modeForm`, `modeManual` and `modeSub` (subcommand list, `subs.go`). Loading runs in the background and arrives as a `loadedMsg`. `view.go` renders, `library.go` covers presets/recent/examples, and `themes.go`/`styles.go` hold the color roles and named themes.
- `internal/store`: history and presets (`store.json`). `internal/config`: the TOML settings file, including per-command `safe`/`risky`/`values` corrections.
- `cmd/commando`: flag handling, `--explain`, `--themes`, `--config`, `--init <shell>` integration (`shell.go`) and `--completion` scripts (`completion.go`). Holds `var version`, which releases read.

## Changing the parser

The parser must gain option names without losing any. Before and after a change:

```sh
scripts/survey.py run old-commando old.json      # binary built before the change
scripts/survey.py run bin/commando new.json
scripts/survey.py compare old.json new.json
```

This only reads manuals (`COMMANDO_NO_HELP=1`). Don't use `--help-tools` outside a container: it runs every program on `PATH`.

When a fix handles a new page layout, save the page in `internal/manpage/testdata/` and add a case to `TestFormats` (`formats_test.go`), with expected option and name counts plus sample name sets. Keep BSD/macOS (`mandoc`) and GNU (`groff`) layouts working; testdata has both.

## Docs to keep in sync

User-visible flags, keys or settings appear in README.md, `docs/commando.1` (the shipped man page) and `cmd/commando/completion.go`. Update all three together.

## Releasing

Set `version` in `cmd/commando/main.go` and add a `## vX.Y.Z` section at the top of `RELEASE_NOTES.md`. Merging to `main` makes the Release workflow tag the version, publish it, and open a `formula-vX.Y.Z` PR that updates `Formula/commando.rb`. Don't edit the formula's URLs or checksums by hand. `scripts/update-formula.sh` is the manual fallback.

## Environment variables useful in development

`COMMANDO_CACHE_DIR`, `COMMANDO_DATA_DIR`, `COMMANDO_CONFIG`, `COMMANDO_NO_HISTORY`, `COMMANDO_NO_COMPLETIONS`, `COMMANDO_ZSH_COMPLETIONS`/`COMMANDO_FISH_COMPLETIONS`, `COMMANDO_NO_CONFIRM`, `COMMANDO_NO_HELP`.
