package cards

import (
	"ghstats/internal/github"
	"ghstats/internal/render"
)

// RepoCardView is the shared model for the pin and gist cards (same visual
// layout: title, wrapped description, language dot, stars/forks).
type RepoCardView struct {
	Opts          Options
	Title         string
	DescLines     []string
	Width         int
	Height        int
	Language      string
	LanguageColor string
	Stars         string
	Forks         string
	StarIcon      string
	ForkIcon      string
}

// BuildRepoView builds the pinned-repo card model.
func BuildRepoView(r *github.Repo, opts Options) RepoCardView {
	width := opts.CardWidth
	if width < 300 {
		width = 400
	}
	desc := firstNonEmpty(r.Description, "No description provided")
	lines := wrapText(desc, float64(width-50), 14.5, 2)

	return RepoCardView{
		Opts:          opts,
		Title:         firstNonEmpty(opts.CustomTitle, r.Name),
		DescLines:     lines,
		Width:         width,
		Height:        120 + maxInt(0, len(lines)-2)*18,
		Language:      r.Language,
		LanguageColor: normalizeHex(r.LanguageColor),
		Stars:         fmtInt(r.Stars),
		Forks:         fmtInt(r.Forks),
		StarIcon:      Icons["star"],
		ForkIcon:      Icons["fork"],
	}
}

// BuildGistView builds the gist card model (same layout as the repo card).
func BuildGistView(g *github.Gist, opts Options) RepoCardView {
	width := opts.CardWidth
	if width < 300 {
		width = 400
	}
	desc := firstNonEmpty(g.Description, "No description provided")
	lines := wrapText(desc, float64(width-50), 14.5, 2)

	return RepoCardView{
		Opts:          opts,
		Title:         firstNonEmpty(opts.CustomTitle, g.Name),
		DescLines:     lines,
		Width:         width,
		Height:        120 + maxInt(0, len(lines)-2)*18,
		Language:      g.Language,
		LanguageColor: normalizeHex(g.LanguageColor),
		Stars:         fmtInt(g.Stars),
		Forks:         fmtInt(g.Forks),
		StarIcon:      Icons["star"],
		ForkIcon:      Icons["fork"],
	}
}

// wrapText greedily wraps a string into at most maxLines lines that each fit
// within maxWidth pixels at the given font size. The final line is ellipsized
// if content overflows.
func wrapText(s string, maxWidth, fontSize float64, maxLines int) []string {
	words := splitWords(s)
	var lines []string
	var cur string
	for _, w := range words {
		try := w
		if cur != "" {
			try = cur + " " + w
		}
		if render.MeasureText(try, fontSize) > maxWidth && cur != "" {
			lines = append(lines, cur)
			cur = w
			if len(lines) == maxLines {
				break
			}
		} else {
			cur = try
		}
	}
	if len(lines) < maxLines && cur != "" {
		lines = append(lines, cur)
	}
	if len(lines) == 0 {
		return []string{s}
	}
	// Ellipsize if we ran out of lines but had more words.
	last := lines[len(lines)-1]
	if render.MeasureText(s, fontSize) > maxWidth*float64(maxLines) {
		for render.MeasureText(last+"...", fontSize) > maxWidth && len(last) > 0 {
			last = last[:len(last)-1]
		}
		lines[len(lines)-1] = last + "..."
	}
	return lines
}

func splitWords(s string) []string {
	var words []string
	cur := ""
	for _, r := range s {
		if r == ' ' || r == '\n' || r == '\t' {
			if cur != "" {
				words = append(words, cur)
				cur = ""
			}
		} else {
			cur += string(r)
		}
	}
	if cur != "" {
		words = append(words, cur)
	}
	return words
}
