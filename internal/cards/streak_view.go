package cards

import (
	"strconv"
	"time"

	"github.com/michael-duren/stats/internal/github"
)

// StreakView is the resolved model for the streak card: three columns —
// total contributions, current streak (ringed), and longest streak.
type StreakView struct {
	Opts   Options
	Width  int
	Height int

	Total        string
	TotalRange   string
	Current      string
	CurrentRange string
	Longest      string
	LongestRange string

	Col1X int
	Col2X int
	Col3X int
	Div1X int
	Div2X int

	RingCY    int
	RingR     float64
	RingColor string
}

// BuildStreakView assembles the streak card model and its column geometry.
func BuildStreakView(s *github.Streak, opts Options) StreakView {
	width := opts.CardWidth
	if width < 300 {
		width = 495
	}

	v := StreakView{
		Opts:      opts,
		Width:     width,
		Height:    195,
		Total:     strconv.Itoa(s.TotalContributions),
		Current:   strconv.Itoa(s.CurrentStreak),
		Longest:   strconv.Itoa(s.LongestStreak),
		Col1X:     width / 6,
		Col2X:     width / 2,
		Col3X:     width * 5 / 6,
		Div1X:     width / 3,
		Div2X:     width * 2 / 3,
		RingCY:    78,
		RingR:     40,
		RingColor: opts.TitleColor,
	}

	v.TotalRange = streakRange(s.FirstContribution, time.Time{}, true)
	v.CurrentRange = streakRange(s.CurrentStreakStart, s.CurrentStreakEnd, false)
	v.LongestRange = streakRange(s.LongestStreakStart, s.LongestStreakEnd, false)
	return v
}

// streakRange formats a start–end date label. openEnded renders "… – Present"
// for the running total; an empty start renders an em dash.
func streakRange(start, end time.Time, openEnded bool) string {
	const layout = "Jan 2, 2006"
	if start.IsZero() {
		return "—"
	}
	if openEnded {
		return start.Format(layout) + " – Present"
	}
	if end.IsZero() || end.Equal(start) {
		return start.Format(layout)
	}
	return start.Format(layout) + " – " + end.Format(layout)
}
