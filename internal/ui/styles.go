package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Palette. Bright accents read well on dark iTerm/Terminal profiles; the
// adaptive neutrals keep light profiles legible. ApplyTheme can replace it.
var (
	cViolet, cPink, cCyan, cGreen, cYellow, cOrange, cBlue, cRed lipgloss.AdaptiveColor
	cText, cDim, cFaint                                          lipgloss.AdaptiveColor
	cOption                                                      lipgloss.AdaptiveColor // option names
	cSelBg                                                       = lipgloss.AdaptiveColor{Light: "#EDE9FE", Dark: "#2E1F4F"}
	cField                                                       = lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#24243A"}
	cFieldF                                                      = lipgloss.AdaptiveColor{Light: "#E0E7FF", Dark: "#33335A"}
)

func defaultPalette() {
	cViolet = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#C084FC"}
	cPink = lipgloss.AdaptiveColor{Light: "#DB2777", Dark: "#F472B6"}
	cCyan = lipgloss.AdaptiveColor{Light: "#0E7490", Dark: "#22D3EE"}
	cGreen = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	cYellow = lipgloss.AdaptiveColor{Light: "#A16207", Dark: "#FDE047"}
	cOrange = lipgloss.AdaptiveColor{Light: "#C2410C", Dark: "#FB923C"}
	cBlue = lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#60A5FA"}
	cRed = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
	cText = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"}
	cDim = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	cFaint = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#4B5563"}
	cOption = cViolet
}

// contrastPalette has stronger colors: full white or black text, and
// accents that stand further from the background.
func contrastPalette() {
	cViolet = lipgloss.AdaptiveColor{Light: "#5B21B6", Dark: "#E9D5FF"}
	cPink = lipgloss.AdaptiveColor{Light: "#9D174D", Dark: "#FBCFE8"}
	cCyan = lipgloss.AdaptiveColor{Light: "#155E75", Dark: "#A5F3FC"}
	cGreen = lipgloss.AdaptiveColor{Light: "#14532D", Dark: "#BBF7D0"}
	cYellow = lipgloss.AdaptiveColor{Light: "#713F12", Dark: "#FEF08A"}
	cOrange = lipgloss.AdaptiveColor{Light: "#7C2D12", Dark: "#FED7AA"}
	cBlue = lipgloss.AdaptiveColor{Light: "#1E3A8A", Dark: "#BFDBFE"}
	cRed = lipgloss.AdaptiveColor{Light: "#7F1D1D", Dark: "#FECACA"}
	cText = lipgloss.AdaptiveColor{Light: "#000000", Dark: "#FFFFFF"}
	cDim = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"}
	cFaint = lipgloss.AdaptiveColor{Light: "#4B5563", Dark: "#9CA3AF"}
	cOption = cViolet
}

// ApplyTheme sets the colors: theme is "auto" (follow the terminal's
// background), "dark", "light" or "contrast", and colors overrides roles
// ("accent", "danger"…) with hex colors. Call it after UseRenderer.
func ApplyTheme(r *lipgloss.Renderer, theme string, colors map[string]string) {
	defaultPalette()
	switch theme {
	case "dark":
		r.SetHasDarkBackground(true)
	case "light":
		r.SetHasDarkBackground(false)
	case "contrast":
		contrastPalette()
	}
	roles := map[string]*lipgloss.AdaptiveColor{
		"accent": &cViolet, "command": &cPink, "header": &cCyan, "option": &cOption,
		"value": &cYellow, "argument": &cGreen, "path": &cBlue, "group": &cOrange,
		"danger": &cRed, "text": &cText, "dim": &cDim, "faint": &cFaint,
	}
	accent, hasOption := colors["accent"], colors["option"] != ""
	for role, hex := range colors {
		if c, ok := roles[role]; ok && hex != "" {
			*c = lipgloss.AdaptiveColor{Light: hex, Dark: hex}
		}
	}
	if accent != "" && !hasOption {
		cOption = cViolet // option names follow the accent unless set
	}
	setStyles()
}

var (
	sText    lipgloss.Style
	sDim     lipgloss.Style
	sFaint   lipgloss.Style
	sBold    lipgloss.Style
	sName    lipgloss.Style
	sNameB   lipgloss.Style
	sValue   lipgloss.Style
	sArg     lipgloss.Style
	sCmd     lipgloss.Style
	sOn      lipgloss.Style
	sHeader  lipgloss.Style
	sGroup   lipgloss.Style
	sKey     lipgloss.Style
	sErr     lipgloss.Style
	sOK      lipgloss.Style
	sCursor  lipgloss.Style
	sMatchHL lipgloss.Style
)

func init() {
	defaultPalette()
	setStyles()
}

// UseRenderer makes the styles render for r's output: the terminal the
// form runs on, or stdout for --explain, which may be a pipe (no color).
// Styles keep the renderer they were made with, so they are remade here.
func UseRenderer(r *lipgloss.Renderer) {
	lipgloss.SetDefaultRenderer(r)
	setStyles()
}

func setStyles() {
	sText = lipgloss.NewStyle().Foreground(cText)
	sDim = lipgloss.NewStyle().Foreground(cDim)
	sFaint = lipgloss.NewStyle().Foreground(cFaint)
	sBold = lipgloss.NewStyle().Foreground(cText).Bold(true)
	sName = lipgloss.NewStyle().Foreground(cOption)
	sNameB = lipgloss.NewStyle().Foreground(cOption).Bold(true)
	sValue = lipgloss.NewStyle().Foreground(cYellow)
	sArg = lipgloss.NewStyle().Foreground(cGreen)
	sCmd = lipgloss.NewStyle().Foreground(cPink).Bold(true)
	sOn = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	sHeader = lipgloss.NewStyle().Foreground(cCyan).Bold(true)
	sGroup = lipgloss.NewStyle().Foreground(cOrange).Bold(true)
	sKey = lipgloss.NewStyle().Foreground(cPink).Bold(true)
	sErr = lipgloss.NewStyle().Foreground(cRed).Bold(true)
	sOK = lipgloss.NewStyle().Foreground(cGreen)
	sCursor = lipgloss.NewStyle().Foreground(cPink).Bold(true)
	sMatchHL = lipgloss.NewStyle().Foreground(lipgloss.Color("#111111")).Background(cYellow)
	kindColor = map[string]lipgloss.AdaptiveColor{
		"flag": cGreen, "choice": cCyan, "number": cYellow, "path": cBlue, "text": cOrange,
	}
}

var kindColor map[string]lipgloss.AdaptiveColor

func badge(label string, c lipgloss.TerminalColor) string {
	return lipgloss.NewStyle().Foreground(lipgloss.Color("#111111")).Background(c).Bold(true).Padding(0, 1).Render(label)
}

// gradient renders s with a left-to-right color gradient.
func gradient(s string, from, to [3]int) string {
	rs := []rune(s)
	var b strings.Builder
	for i, r := range rs {
		t := 0.0
		if len(rs) > 1 {
			t = float64(i) / float64(len(rs)-1)
		}
		c := fmt.Sprintf("#%02X%02X%02X",
			from[0]+int(t*float64(to[0]-from[0])),
			from[1]+int(t*float64(to[1]-from[1])),
			from[2]+int(t*float64(to[2]-from[2])))
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(c)).Bold(true).Render(string(r)))
	}
	return b.String()
}

// fit pads or truncates a styled string to exactly w cells.
func fit(s string, w int) string {
	if w <= 0 {
		return ""
	}
	sw := ansi.StringWidth(s)
	if sw > w {
		return ansi.Truncate(s, w, "…")
	}
	return s + strings.Repeat(" ", w-sw)
}

// box draws a rounded panel with a title in the top border. body lines are
// fitted to the inner width; the box is exactly w x h cells.
func box(title string, body []string, w, h int, color lipgloss.TerminalColor) string {
	if w < 4 || h < 2 {
		return ""
	}
	bc := lipgloss.NewStyle().Foreground(color)
	inner := w - 4
	top := "╭─"
	if title != "" {
		t := " " + title + " "
		if ansi.StringWidth(t) > w-4 {
			t = ansi.Truncate(t, w-4, "…")
		}
		top += t
	}
	top = bc.Render("╭─") + strings.TrimPrefix(top, "╭─")
	if n := w - 1 - ansi.StringWidth(top); n > 0 {
		top += bc.Render(strings.Repeat("─", n))
	}
	top += bc.Render("╮")

	lines := make([]string, 0, max(0, h))
	lines = append(lines, top)
	side := bc.Render("│")
	for i := 0; i < h-2; i++ {
		l := ""
		if i < len(body) {
			l = body[i]
		}
		lines = append(lines, side+" "+fit(l, inner)+" "+side)
	}
	lines = append(lines, bc.Render("╰"+strings.Repeat("─", w-2)+"╯"))
	return strings.Join(lines, "\n")
}

// wrap word-wraps plain text to width w, returning lines.
func wrap(s string, w int) []string {
	if w < 8 {
		w = 8
	}
	var out []string
	for _, para := range strings.Split(s, "\n") {
		if strings.HasPrefix(para, "  ") {
			// preformatted example line
			out = append(out, para)
			continue
		}
		words := strings.Fields(para)
		if len(words) == 0 {
			out = append(out, "")
			continue
		}
		cur := ""
		for _, word := range words {
			for ansi.StringWidth(word) > w {
				if cur != "" {
					out = append(out, cur)
					cur = ""
				}
				out = append(out, ansi.Truncate(word, w, ""))
				word = string([]rune(word)[w:])
			}
			switch {
			case cur == "":
				cur = word
			case ansi.StringWidth(cur)+1+ansi.StringWidth(word) <= w:
				cur += " " + word
			default:
				out = append(out, cur)
				cur = word
			}
		}
		if cur != "" {
			out = append(out, cur)
		}
	}
	return out
}
