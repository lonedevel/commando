package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

// Palette. Bright accents read well on dark iTerm/Terminal profiles; the
// adaptive neutrals keep light profiles legible.
var (
	cViolet = lipgloss.AdaptiveColor{Light: "#7C3AED", Dark: "#C084FC"}
	cPink   = lipgloss.AdaptiveColor{Light: "#DB2777", Dark: "#F472B6"}
	cCyan   = lipgloss.AdaptiveColor{Light: "#0E7490", Dark: "#22D3EE"}
	cGreen  = lipgloss.AdaptiveColor{Light: "#15803D", Dark: "#4ADE80"}
	cYellow = lipgloss.AdaptiveColor{Light: "#A16207", Dark: "#FDE047"}
	cOrange = lipgloss.AdaptiveColor{Light: "#C2410C", Dark: "#FB923C"}
	cBlue   = lipgloss.AdaptiveColor{Light: "#1D4ED8", Dark: "#60A5FA"}
	cRed    = lipgloss.AdaptiveColor{Light: "#B91C1C", Dark: "#F87171"}
	cText   = lipgloss.AdaptiveColor{Light: "#1F2937", Dark: "#E5E7EB"}
	cDim    = lipgloss.AdaptiveColor{Light: "#6B7280", Dark: "#9CA3AF"}
	cFaint  = lipgloss.AdaptiveColor{Light: "#9CA3AF", Dark: "#4B5563"}
	cSelBg  = lipgloss.AdaptiveColor{Light: "#EDE9FE", Dark: "#2E1F4F"}
	cField  = lipgloss.AdaptiveColor{Light: "#F3F4F6", Dark: "#24243A"}
	cFieldF = lipgloss.AdaptiveColor{Light: "#E0E7FF", Dark: "#33335A"}
)

var (
	sText    = lipgloss.NewStyle().Foreground(cText)
	sDim     = lipgloss.NewStyle().Foreground(cDim)
	sFaint   = lipgloss.NewStyle().Foreground(cFaint)
	sBold    = lipgloss.NewStyle().Foreground(cText).Bold(true)
	sName    = lipgloss.NewStyle().Foreground(cViolet)
	sNameB   = lipgloss.NewStyle().Foreground(cViolet).Bold(true)
	sValue   = lipgloss.NewStyle().Foreground(cYellow)
	sArg     = lipgloss.NewStyle().Foreground(cGreen)
	sCmd     = lipgloss.NewStyle().Foreground(cPink).Bold(true)
	sOn      = lipgloss.NewStyle().Foreground(cGreen).Bold(true)
	sHeader  = lipgloss.NewStyle().Foreground(cCyan).Bold(true)
	sGroup   = lipgloss.NewStyle().Foreground(cOrange).Bold(true)
	sKey     = lipgloss.NewStyle().Foreground(cPink).Bold(true)
	sErr     = lipgloss.NewStyle().Foreground(cRed).Bold(true)
	sOK      = lipgloss.NewStyle().Foreground(cGreen)
	sCursor  = lipgloss.NewStyle().Foreground(cPink).Bold(true)
	sMatchHL = lipgloss.NewStyle().Foreground(lipgloss.Color("#111111")).Background(cYellow)
)

var kindColor = map[string]lipgloss.AdaptiveColor{
	"flag": cGreen, "choice": cCyan, "number": cYellow, "path": cBlue, "text": cOrange,
}

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

	lines := make([]string, 0, h)
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
