// Command commando presents a colorful form for a Unix command's options,
// built from its manual page, and then runs (or prints) the result.
//
//	commando ls            pick options for ls, then run it
//	commando -p tar        print the command instead of running it
//	commando               ask which command first
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/term"
	"github.com/muesli/termenv"

	"github.com/lonedevel/commando/internal/cmdline"
	"github.com/lonedevel/commando/internal/manpage"
	"github.com/lonedevel/commando/internal/store"
	"github.com/lonedevel/commando/internal/ui"
)

var version = "0.10.0"

const usage = `commando — a friendly front-end for Unix command options

Usage:
  commando [flags] [command [existing args...]]

Opens a form with every option documented in the command's manual page:
checkboxes for flags, dropdowns for enumerated values, radio buttons for
mutually exclusive choices and text fields for arguments. Press Enter to
run the assembled command.

Flags:
  -p, --print        print the command to stdout instead of running it
  -l, --line LINE    start from an existing command line (for shell widgets)
      --long         prefer --long option names over short ones
      --no-cache     re-parse the manual even if it is cached
      --no-history   don't record this command or show presets and history
      --init SHELL   print shell integration for zsh, bash or fish
      --explain      describe each option and argument of a command line,
                     from its manual, without opening the form
      --dump         print the parsed options as JSON and exit
  -v, --version      print the version
  -h, --help         show this help

Examples:
  commando ls
  commando grep -rn TODO .        # pre-fills -r and -n
  commando git                   # choose a git command, then its options
  commando git commit
  commando --explain 'tar -czvf backup.tgz --exclude=.git src'
  eval "$(commando --init zsh)"  # then press Ctrl-X Ctrl-O on any command line
`

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(argv []string) int {
	var (
		printOnly, long, noCache, dump, noHistory, explain bool
		line, initShell                                    string
	)
	i := 0
	for ; i < len(argv); i++ {
		a := argv[i]
		if !strings.HasPrefix(a, "-") || a == "-" {
			break
		}
		if a == "--" {
			i++
			break
		}
		next := func() string {
			if i+1 < len(argv) {
				i++
				return argv[i]
			}
			fmt.Fprintf(os.Stderr, "commando: %s needs a value\n", a)
			os.Exit(2)
			return ""
		}
		switch a {
		case "-p", "--print":
			printOnly = true
		case "--long":
			long = true
		case "--no-cache":
			noCache = true
		case "--dump":
			dump = true
		case "--explain":
			explain = true
		case "--no-history":
			noHistory = true
		case "-l", "--line":
			line = next()
		case "--init":
			initShell = next()
		case "-v", "--version":
			fmt.Println("commando", version)
			return 0
		case "-h", "--help":
			fmt.Print(usage)
			return 0
		default:
			fmt.Fprintf(os.Stderr, "commando: unknown flag %s\n\n%s", a, usage)
			return 2
		}
	}
	if initShell != "" {
		s, ok := shellInit[initShell]
		if !ok {
			fmt.Fprintf(os.Stderr, "commando: unsupported shell %q (zsh, bash, fish)\n", initShell)
			return 2
		}
		fmt.Print(s)
		return 0
	}
	if rest := argv[i:]; explain && len(rest) == 1 {
		// One argument is the whole line: commando --explain 'find . | xargs rm'
		line = strings.TrimSpace(line + " " + rest[0])
	} else if len(rest) > 0 {
		quoted := make([]string, len(rest))
		for k, r := range rest {
			quoted[k] = cmdline.Quote(r)
		}
		// Keep globs and ~ as the user typed them only when passed via --line;
		// argv words are already expanded by the shell, so quote them.
		line = strings.TrimSpace(line + " " + strings.Join(quoted, " "))
	}

	if dump {
		return dumpSpec(line, !noCache)
	}
	if explain {
		return explainLine(line, !noCache)
	}

	tty, err := os.OpenFile("/dev/tty", os.O_RDWR, 0)
	if err != nil {
		fmt.Fprintln(os.Stderr, "commando: needs an interactive terminal:", err)
		return 1
	}
	defer tty.Close()

	// Detect colors from the terminal, not stdout (which may be a pipe).
	ui.UseRenderer(lipgloss.NewRenderer(tty, termenv.WithColorCache(true)))
	out := termenv.NewOutput(tty)

	var st *store.Store
	if !noHistory && os.Getenv("COMMANDO_NO_HISTORY") == "" {
		st = store.Load()
	}
	// Printing only puts the command on the prompt for review, so there is
	// nothing to confirm.
	confirm := !printOnly && os.Getenv("COMMANDO_NO_CONFIRM") == ""
	model := ui.New(ui.Config{Line: line, UseCache: !noCache, PreferLong: long, Output: out, Store: st, Confirm: confirm})
	p := tea.NewProgram(model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
		tea.WithInput(tty),
		tea.WithOutput(tty),
	)
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "commando:", err)
		return 1
	}
	res := model.Result()
	if !res.Accepted || strings.TrimSpace(res.Command) == "" {
		return 130
	}
	if st != nil {
		st.AddRecent(res.Key, res.Command, time.Now())
		if err := st.Save(); err != nil {
			fmt.Fprintln(os.Stderr, "commando: could not save history:", err)
		}
	}
	if printOnly {
		fmt.Println(res.Command)
		return 0
	}

	prompt := lipgloss.NewStyle().Foreground(lipgloss.Color("#F472B6")).Bold(true).Render("❯ ")
	fmt.Fprintln(tty, prompt+res.Command)
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	err = syscall.Exec(shell, []string{shell, "-c", res.Command}, os.Environ())
	fmt.Fprintln(os.Stderr, "commando:", err)
	return 1
}

func explainLine(line string, useCache bool) int {
	if strings.TrimSpace(line) == "" {
		fmt.Fprintln(os.Stderr, "commando: --explain needs a command line")
		return 2
	}
	// Color only when stdout is a terminal.
	ui.UseRenderer(lipgloss.NewRenderer(os.Stdout))
	width := 100
	if w, _, err := term.GetSize(os.Stdout.Fd()); err == nil && w > 0 {
		width = w
	} else if c, err := strconv.Atoi(os.Getenv("COLUMNS")); err == nil && c > 0 {
		width = c
	}
	fmt.Print(ui.Explain(context.Background(), line, width, useCache))
	return 0
}

func dumpSpec(line string, useCache bool) int {
	var vals []string
	for _, w := range cmdline.Split(line) {
		vals = append(vals, w.Value)
	}
	if len(vals) == 0 {
		fmt.Fprintln(os.Stderr, "commando: --dump needs a command")
		return 2
	}
	spec, _, err := manpage.LoadLine(context.Background(), vals, useCache)
	if err != nil {
		fmt.Fprintln(os.Stderr, "commando:", err)
		return 1
	}
	spec.Manual = ""
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	_ = enc.Encode(spec)
	return 0
}
