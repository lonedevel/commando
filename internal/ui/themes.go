package ui

import (
	"strconv"
	"strings"
)

// palette is a named theme's colors, by role. Named themes assume the
// terminal uses the same scheme, so they aren't adaptive.
type palette struct {
	accent, command, header, option, value, argument, path, group, danger string
	text, dim, faint                                                      string
}

// namedThemes are well-known color schemes, mapped onto commando's roles
// from their published palettes.
var namedThemes = map[string]palette{
	// https://draculatheme.com/contribute
	"dracula": {
		accent: "#BD93F9", command: "#FF79C6", header: "#8BE9FD", option: "#BD93F9",
		value: "#F1FA8C", argument: "#50FA7B", path: "#8BE9FD", group: "#FFB86C", danger: "#FF5555",
		text: "#F8F8F2", dim: "#BFC3D9", faint: "#6272A4",
	},
	// https://www.nordtheme.com/docs/colors-and-palettes
	"nord": {
		accent: "#B48EAD", command: "#88C0D0", header: "#8FBCBB", option: "#81A1C1",
		value: "#EBCB8B", argument: "#A3BE8C", path: "#5E81AC", group: "#D08770", danger: "#BF616A",
		text: "#ECEFF4", dim: "#D8DEE9", faint: "#616E88",
	},
	// https://github.com/morhetz/gruvbox
	"gruvbox-dark": {
		accent: "#D3869B", command: "#FE8019", header: "#8EC07C", option: "#D3869B",
		value: "#FABD2F", argument: "#B8BB26", path: "#83A598", group: "#FE8019", danger: "#FB4934",
		text: "#EBDBB2", dim: "#A89984", faint: "#7C6F64",
	},
	"gruvbox-light": {
		accent: "#8F3F71", command: "#AF3A03", header: "#427B58", option: "#8F3F71",
		value: "#B57614", argument: "#79740E", path: "#076678", group: "#AF3A03", danger: "#9D0006",
		text: "#3C3836", dim: "#7C6F64", faint: "#A89984",
	},
	// https://ethanschoonover.com/solarized/
	"solarized-dark": {
		accent: "#6C71C4", command: "#D33682", header: "#2AA198", option: "#6C71C4",
		value: "#B58900", argument: "#859900", path: "#268BD2", group: "#CB4B16", danger: "#DC322F",
		text: "#93A1A1", dim: "#839496", faint: "#586E75",
	},
	"solarized-light": {
		accent: "#6C71C4", command: "#D33682", header: "#2AA198", option: "#6C71C4",
		value: "#B58900", argument: "#859900", path: "#268BD2", group: "#CB4B16", danger: "#DC322F",
		text: "#586E75", dim: "#657B83", faint: "#93A1A1",
	},
	// https://catppuccin.com/palette
	"catppuccin-mocha": {
		accent: "#CBA6F7", command: "#F5C2E7", header: "#89DCEB", option: "#B4BEFE",
		value: "#F9E2AF", argument: "#A6E3A1", path: "#89B4FA", group: "#FAB387", danger: "#F38BA8",
		text: "#CDD6F4", dim: "#A6ADC8", faint: "#6C7086",
	},
	"catppuccin-latte": {
		accent: "#8839EF", command: "#EA76CB", header: "#04A5E5", option: "#7287FD",
		value: "#DF8E1D", argument: "#40A02B", path: "#1E66F5", group: "#FE640B", danger: "#D20F39",
		text: "#4C4F69", dim: "#6C6F85", faint: "#9CA0B0",
	},
	// https://github.com/folke/tokyonight.nvim
	"tokyo-night": {
		accent: "#9D7CD8", command: "#7AA2F7", header: "#7DCFFF", option: "#BB9AF7",
		value: "#E0AF68", argument: "#9ECE6A", path: "#2AC3DE", group: "#FF9E64", danger: "#F7768E",
		text: "#C0CAF5", dim: "#A9B1D6", faint: "#565F89",
	},
}

// rgb parses "#RRGGBB" for the logo's gradient.
func rgb(hex string) ([3]int, bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return [3]int{}, false
	}
	var out [3]int
	for i := range out {
		v, err := strconv.ParseUint(hex[2*i:2*i+2], 16, 8)
		if err != nil {
			return [3]int{}, false
		}
		out[i] = int(v)
	}
	return out, true
}
