package cards

import (
	"fmt"
	"math"
	"strconv"

	"ghstats/internal/github"
)

type statRow struct {
	Key   string
	Icon  string
	Label string
	Value string
}

// StatsView is the fully-resolved model handed to the stats templ component.
type StatsView struct {
	Opts   Options
	Title  string
	Width  int
	Height int
	Rows   []statRow
	RowY   []int // baseline Y for each row, parallel to Rows

	ShowRank       bool
	RankLevel      string
	RankCircleR    float64
	RankCircumf    float64
	RankDashOffset float64
	RankCenterX    float64
	RankCenterY    float64
}

// BuildStatsView assembles the view model. commitYear labels the (annual)
// commit row; showRank toggles the right-hand percentile circle.
func BuildStatsView(s *github.Stats, opts Options, commitYear int, showRank, includeAllCommits bool) StatsView {
	commitLabel := fmt.Sprintf("Total Commits (%d):", commitYear)
	if includeAllCommits {
		commitLabel = "Total Commits:"
	}

	candidates := []statRow{
		{Key: "stars", Icon: Icons["star"], Label: "Total Stars Earned:", Value: fmtInt(s.TotalStars)},
		{Key: "commits", Icon: Icons["commits"], Label: commitLabel, Value: fmtInt(s.TotalCommits)},
		{Key: "prs", Icon: Icons["prs"], Label: "Total PRs:", Value: fmtInt(s.TotalPRs)},
		{Key: "issues", Icon: Icons["issues"], Label: "Total Issues:", Value: fmtInt(s.TotalIssues)},
		{Key: "contribs", Icon: Icons["contribs"], Label: "Contributed to (last year):", Value: fmtInt(s.ContributedTo)},
	}

	width := opts.CardWidth
	if width < 300 {
		width = 495
	}

	v := StatsView{
		Opts:        opts,
		Title:       firstNonEmpty(opts.CustomTitle, s.Name+"'s GitHub Stats"),
		Width:       width,
		ShowRank:    showRank,
		RankLevel:   s.Rank.Level,
		RankCircleR: 40,
		RankCenterX: float64(width) - 70,
		RankCenterY: 85,
	}

	const rowStartY = 0
	const rowGap = 25
	y := rowStartY
	for _, r := range candidates {
		if opts.hidden(r.Key) {
			continue
		}
		v.Rows = append(v.Rows, r)
		v.RowY = append(v.RowY, y)
		y += rowGap
	}

	// Card height: header + rows + padding. Keep room for the rank circle.
	contentHeight := len(v.Rows)*rowGap + 75
	if showRank {
		contentHeight = maxInt(contentHeight, 195)
	}
	v.Height = contentHeight

	// Rank circle progress: better percentile => more of the ring filled.
	v.RankCircumf = 2 * math.Pi * v.RankCircleR
	progress := (100 - s.Rank.Percentile) / 100
	v.RankDashOffset = v.RankCircumf * (1 - progress)

	return v
}

func fmtInt(n int) string { return strconv.Itoa(n) }

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
