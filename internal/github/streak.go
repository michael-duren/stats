package github

import (
	"context"
	"sort"
	"time"
)

// Streak holds the contribution-streak data backing the streak card: a lifetime
// contribution total, the current consecutive-day streak, and the longest one.
type Streak struct {
	Username           string
	Name               string
	TotalContributions int
	FirstContribution  time.Time

	CurrentStreak      int
	CurrentStreakStart time.Time
	CurrentStreakEnd   time.Time

	LongestStreak      int
	LongestStreakStart time.Time
	LongestStreakEnd   time.Time
}

// streakQuery pulls one year's contribution calendar. GitHub caps a single
// contributionsCollection window at one year, so FetchStreak loops year-by-year
// from the account's creation date.
const streakQuery = `
query streak($login: String!, $from: DateTime!, $to: DateTime!) {
  user(login: $login) {
    name
    login
    createdAt
    contributionsCollection(from: $from, to: $to) {
      contributionCalendar {
        weeks {
          contributionDays {
            date
            contributionCount
          }
        }
      }
    }
  }
}`

type streakResponse struct {
	User struct {
		Name                    string `json:"name"`
		Login                   string `json:"login"`
		CreatedAt               string `json:"createdAt"`
		ContributionsCollection struct {
			ContributionCalendar struct {
				Weeks []struct {
					ContributionDays []struct {
						Date  string `json:"date"`
						Count int    `json:"contributionCount"`
					} `json:"contributionDays"`
				} `json:"weeks"`
			} `json:"contributionCalendar"`
		} `json:"contributionsCollection"`
	} `json:"user"`
}

// FetchStreak aggregates a user's full daily contribution history and derives
// total, current, and longest streaks. The current streak counts back from
// today; a zero-contribution today does not break it (the streak is measured to
// yesterday), matching github-readme-streak-stats.
func (c *Client) FetchStreak(ctx context.Context, username string) (*Streak, error) {
	now := time.Now().UTC()
	days := map[string]int{}
	var name, login, createdAt string

	fetchWindow := func(from, to time.Time) error {
		var resp streakResponse
		vars := map[string]any{
			"login": username,
			"from":  from.Format(time.RFC3339),
			"to":    to.Format(time.RFC3339),
		}
		if err := c.graphql(ctx, streakQuery, vars, &resp); err != nil {
			return err
		}
		name = resp.User.Name
		login = resp.User.Login
		createdAt = resp.User.CreatedAt
		// Calendars return whole weeks, so boundary days (e.g. Dec 31) can appear
		// in two adjacent yearly windows. Keep the max to avoid double-counting
		// without zeroing a real count seen in another window.
		for _, w := range resp.User.ContributionsCollection.ContributionCalendar.Weeks {
			for _, d := range w.ContributionDays {
				if d.Count > days[d.Date] {
					days[d.Date] = d.Count
				}
			}
		}
		return nil
	}

	// Fetch the current year first so we learn createdAt, then back-fill prior
	// years down to account creation.
	startCurrent := time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)
	if err := fetchWindow(startCurrent, now); err != nil {
		return nil, err
	}
	created, err := time.Parse(time.RFC3339, createdAt)
	if err != nil {
		created = startCurrent
	}
	for y := created.Year(); y < now.Year(); y++ {
		from := time.Date(y, 1, 1, 0, 0, 0, 0, time.UTC)
		to := time.Date(y, 12, 31, 23, 59, 59, 0, time.UTC)
		if err := fetchWindow(from, to); err != nil {
			return nil, err
		}
	}

	dates := make([]string, 0, len(days))
	for d := range days {
		dates = append(dates, d)
	}
	sort.Strings(dates) // ISO dates sort chronologically as strings

	parse := func(s string) time.Time {
		t, _ := time.Parse("2006-01-02", s)
		return t
	}

	s := &Streak{Username: username, Name: firstNonEmpty(name, login, username)}

	for _, d := range dates {
		s.TotalContributions += days[d]
		if s.FirstContribution.IsZero() && days[d] > 0 {
			s.FirstContribution = parse(d)
		}
	}

	// Longest streak: the longest run of consecutive days with contributions.
	var runLen int
	var runStart string
	for _, d := range dates {
		if days[d] > 0 {
			if runLen == 0 {
				runStart = d
			}
			runLen++
			if runLen > s.LongestStreak {
				s.LongestStreak = runLen
				s.LongestStreakStart = parse(runStart)
				s.LongestStreakEnd = parse(d)
			}
		} else {
			runLen = 0
		}
	}

	// Current streak: walk back from the most recent day. If today has no
	// contributions yet, start from yesterday so the streak isn't prematurely
	// broken.
	if n := len(dates); n > 0 {
		start := n - 1
		if days[dates[start]] == 0 {
			start--
		}
		for j := start; j >= 0; j-- {
			if days[dates[j]] == 0 {
				break
			}
			if s.CurrentStreak == 0 {
				s.CurrentStreakEnd = parse(dates[j])
			}
			s.CurrentStreak++
			s.CurrentStreakStart = parse(dates[j])
		}
	}

	return s, nil
}
