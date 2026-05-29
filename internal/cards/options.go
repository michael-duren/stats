package cards

import "github.com/michael-duren/stats/internal/themes"

// Options is the resolved, per-request rendering configuration shared by all
// card types. Colors here are final hex strings (with leading #), already
// merged from theme + explicit query overrides.
type Options struct {
	TitleColor  string
	IconColor   string
	TextColor   string
	BgColor     string
	BorderColor string

	HideBorder  bool
	HideTitle   bool
	ShowIcons   bool
	CustomTitle string
	CardWidth   int
	Hide        map[string]bool
}

// ResolveOptions merges a base theme with explicit color overrides. Empty
// override strings fall back to the theme value. Color overrides are raw hex
// (with or without a leading #).
func ResolveOptions(themeName, titleColor, iconColor, textColor, bgColor, borderColor string) Options {
	t := themes.Get(themeName)
	return Options{
		TitleColor:  themes.Color(titleColor, t.TitleColor),
		IconColor:   themes.Color(iconColor, t.IconColor),
		TextColor:   themes.Color(textColor, t.TextColor),
		BgColor:     themes.Color(bgColor, t.BgColor),
		BorderColor: themes.Color(borderColor, t.BorderColor),
		Hide:        map[string]bool{},
	}
}

func (o Options) hidden(key string) bool { return o.Hide[key] }
