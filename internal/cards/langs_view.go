package cards

import (
	"fmt"

	"ghstats/internal/github"
)

type langRow struct {
	Name     string
	Color    string
	Percent  float64
	BarWidth float64
}

// LangsView is the resolved model for the top-languages card (list layout).
type LangsView struct {
	Opts   Options
	Title  string
	Width  int
	Height int
	Rows   []langRow
	RowY   []int
}

// BuildLangsView keeps the top `limit` languages and computes each one's share
// of total bytes across all returned languages.
func BuildLangsView(langs []github.Language, opts Options, limit int) LangsView {
	if limit <= 0 {
		limit = 5
	}
	var total int
	for _, l := range langs {
		total += l.Size
	}
	if total == 0 {
		total = 1
	}

	width := opts.CardWidth
	if width < 200 {
		width = 300
	}
	const padX = 25
	barMax := float64(width - 2*padX)

	v := LangsView{
		Opts:  opts,
		Title: firstNonEmpty(opts.CustomTitle, "Most Used Languages"),
		Width: width,
	}

	shown := langs
	if len(shown) > limit {
		shown = shown[:limit]
	}

	y := 0
	const rowGap = 40
	for _, l := range shown {
		pct := float64(l.Size) / float64(total) * 100
		v.Rows = append(v.Rows, langRow{
			Name:     l.Name,
			Color:    normalizeHex(l.Color),
			Percent:  pct,
			BarWidth: barMax * pct / 100,
		})
		v.RowY = append(v.RowY, y)
		y += rowGap
	}

	v.Height = len(v.Rows)*rowGap + 70
	return v
}

func pctLabel(p float64) string { return fmt.Sprintf("%.2f%%", p) }
