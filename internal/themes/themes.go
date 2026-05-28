package themes

import "strings"

// Theme is a card color palette. Colors are stored as raw hex (no leading #);
// use Color() to normalise user-supplied or stored values for SVG output.
type Theme struct {
	TitleColor  string
	IconColor   string
	TextColor   string
	BgColor     string
	BorderColor string
}

// Registry holds the built-in themes, ported from github-readme-stats.
var Registry = map[string]Theme{
	"default":      {TitleColor: "2f80ed", IconColor: "4c71f2", TextColor: "434d58", BgColor: "fffefe", BorderColor: "e4e2e2"},
	"transparent":  {TitleColor: "006AFF", IconColor: "0579C3", TextColor: "417E87", BgColor: "ffffff00", BorderColor: "e4e2e2"},
	"dark":         {TitleColor: "fff", IconColor: "79ff97", TextColor: "9f9f9f", BgColor: "151515", BorderColor: "e4e2e2"},
	"radical":      {TitleColor: "fe428e", IconColor: "f8d847", TextColor: "a9fef7", BgColor: "141321", BorderColor: "e4e2e2"},
	"merko":        {TitleColor: "abd200", IconColor: "b7d364", TextColor: "68b587", BgColor: "0a0f0b", BorderColor: "e4e2e2"},
	"gruvbox":      {TitleColor: "fabd2f", IconColor: "fe8019", TextColor: "8ec07c", BgColor: "282828", BorderColor: "e4e2e2"},
	"tokyonight":   {TitleColor: "70a5fd", IconColor: "bf91f3", TextColor: "38bdae", BgColor: "1a1b27", BorderColor: "e4e2e2"},
	"onedark":      {TitleColor: "e4bf7a", IconColor: "8eb573", TextColor: "df6d74", BgColor: "282c34", BorderColor: "e4e2e2"},
	"cobalt":       {TitleColor: "e683d9", IconColor: "0480ef", TextColor: "75eeb2", BgColor: "193549", BorderColor: "e4e2e2"},
	"synthwave":    {TitleColor: "e2e9ec", IconColor: "ef8539", TextColor: "e5289e", BgColor: "2b213a", BorderColor: "e4e2e2"},
	"dracula":      {TitleColor: "ff6e96", IconColor: "79dafa", TextColor: "f8f8f2", BgColor: "282a36", BorderColor: "e4e2e2"},
	"nord":         {TitleColor: "81a1c1", IconColor: "88c0d0", TextColor: "d8dee9", BgColor: "2e3440", BorderColor: "e4e2e2"},
	"vue":          {TitleColor: "41b883", IconColor: "41b883", TextColor: "273849", BgColor: "fffefe", BorderColor: "e4e2e2"},
	"github_dark":  {TitleColor: "58a6ff", IconColor: "1f6feb", TextColor: "c3d1d9", BgColor: "0d1117", BorderColor: "30363d"},
	"catppuccin_latte":   {TitleColor: "137980", IconColor: "8839ef", TextColor: "4c4f69", BgColor: "eff1f5", BorderColor: "e4e2e2"},
	"catppuccin_mocha":   {TitleColor: "89b4fa", IconColor: "cba6f7", TextColor: "cdd6f4", BgColor: "1e1e2e", BorderColor: "e4e2e2"},
	"rose_pine":   {TitleColor: "9ccfd8", IconColor: "ebbcba", TextColor: "e0def4", BgColor: "191724", BorderColor: "e4e2e2"},
}

// Get returns the named theme (case-insensitive), falling back to default.
func Get(name string) Theme {
	if t, ok := Registry[strings.ToLower(strings.TrimSpace(name))]; ok {
		return t
	}
	return Registry["default"]
}

// Color normalises a hex value for SVG: trims whitespace, strips a leading #,
// and re-adds it. Returns the fallback if the value is empty.
func Color(value, fallback string) string {
	v := strings.TrimSpace(value)
	if v == "" {
		v = fallback
	}
	v = strings.TrimPrefix(v, "#")
	if v == "" {
		return ""
	}
	return "#" + v
}
